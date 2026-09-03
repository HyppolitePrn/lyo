package streaming

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/hyppoliteprn/lyo/internal/observability"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// testMetrics binds the instruments to the global no-op meter: these tests
// assert streaming behaviour, not telemetry. Use collectingMetrics when the
// recorded values themselves are what is under test.
func testMetrics(t *testing.T) *observability.Metrics {
	t.Helper()
	m, err := observability.NewMetrics()
	if err != nil {
		t.Fatalf("NewMetrics: %v", err)
	}
	return m
}

// collectingMetrics returns instruments backed by a manual reader, plus a
// function that reads back the current value of one counter by name.
func collectingMetrics(t *testing.T) (*observability.Metrics, func(name string) int64) {
	t.Helper()

	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	prev := otel.GetMeterProvider()
	otel.SetMeterProvider(provider)
	t.Cleanup(func() {
		otel.SetMeterProvider(prev)
		_ = provider.Shutdown(context.Background())
	})

	m, err := observability.NewMetrics()
	if err != nil {
		t.Fatalf("NewMetrics: %v", err)
	}

	sum := func(name string) int64 {
		var rm metricdata.ResourceMetrics
		if err := reader.Collect(context.Background(), &rm); err != nil {
			t.Fatalf("collect: %v", err)
		}
		var total int64
		for _, scope := range rm.ScopeMetrics {
			for _, metric := range scope.Metrics {
				if metric.Name != name {
					continue
				}
				switch data := metric.Data.(type) {
				case metricdata.Sum[int64]:
					for _, dp := range data.DataPoints {
						total += dp.Value
					}
				}
			}
		}
		return total
	}
	return m, sum
}

func TestHub_SubscribeUnsubscribeCount(t *testing.T) {
	h := NewHub(4, testLogger(), testMetrics(t))

	if got := h.ListenerCount(); got != 0 {
		t.Fatalf("count = %d, want 0", got)
	}

	h.Subscribe("a")
	h.Subscribe("b")
	if got := h.ListenerCount(); got != 2 {
		t.Fatalf("count = %d, want 2", got)
	}

	h.Unsubscribe("a")
	if got := h.ListenerCount(); got != 1 {
		t.Fatalf("count = %d, want 1", got)
	}

	// Unsubscribing an unknown listener is a no-op, not a panic.
	h.Unsubscribe("does-not-exist")
	if got := h.ListenerCount(); got != 1 {
		t.Fatalf("count = %d, want 1", got)
	}
}

func TestHub_UnsubscribeClosesChannel(t *testing.T) {
	h := NewHub(4, testLogger(), testMetrics(t))
	ch := h.Subscribe("a")

	h.Unsubscribe("a")

	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("expected the listener channel to be closed")
		}
	case <-time.After(time.Second):
		t.Fatal("listener channel was not closed")
	}
}

func TestHub_BroadcastReachesEveryListener(t *testing.T) {
	h := NewHub(4, testLogger(), testMetrics(t))
	a := h.Subscribe("a")
	b := h.Subscribe("b")

	h.Broadcast(context.Background(), Chunk("audio"))

	for name, ch := range map[string]<-chan Chunk{"a": a, "b": b} {
		select {
		case got := <-ch:
			if string(got) != "audio" {
				t.Errorf("%s received %q, want %q", name, got, "audio")
			}
		case <-time.After(time.Second):
			t.Errorf("%s received nothing", name)
		}
	}
}

// Broadcast spawns one goroutine per shardSize listeners; go past that boundary
// to exercise the sharding path.
func TestHub_BroadcastAcrossMultipleShards(t *testing.T) {
	const n = shardSize*2 + 7
	h := NewHub(2, testLogger(), testMetrics(t))

	chans := make([]<-chan Chunk, n)
	for i := range chans {
		chans[i] = h.Subscribe(string(rune('a'+i%26)) + string(rune('0'+i/26)))
	}
	if got := h.ListenerCount(); got != n {
		t.Fatalf("count = %d, want %d", got, n)
	}

	h.Broadcast(context.Background(), Chunk("x"))

	for i, ch := range chans {
		select {
		case got := <-ch:
			if string(got) != "x" {
				t.Fatalf("listener %d received %q", i, got)
			}
		case <-time.After(time.Second):
			t.Fatalf("listener %d received nothing", i)
		}
	}
}

// A listener that never drains must not block the broadcaster: chunks are
// dropped for that listener once its buffer is full.
func TestHub_BroadcastDropsChunksForSlowListener(t *testing.T) {
	h := NewHub(1, testLogger(), testMetrics(t))
	slow := h.Subscribe("slow")

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 100; i++ {
			h.Broadcast(context.Background(), Chunk("x"))
		}
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Broadcast blocked on a slow listener")
	}

	// Exactly one chunk fits in the buffer; the other 99 were dropped.
	if got := len(slow); got != 1 {
		t.Fatalf("buffered chunks = %d, want 1", got)
	}
}

func TestHub_BroadcastStopsOnCancelledContext(t *testing.T) {
	h := NewHub(0, testLogger(), testMetrics(t)) // unbuffered: sends block until ctx cancels
	h.Subscribe("a")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		h.Broadcast(ctx, Chunk("x"))
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Broadcast did not return after its context was cancelled")
	}
}

func TestHub_CloseFiresDone(t *testing.T) {
	h := NewHub(4, testLogger(), testMetrics(t))

	select {
	case <-h.Done():
		t.Fatal("Done fired before Close")
	default:
	}

	h.Close()

	select {
	case <-h.Done():
	case <-time.After(time.Second):
		t.Fatal("Done did not fire after Close")
	}
}

// Subscribe/Unsubscribe/Broadcast are called concurrently in production; the
// race detector should stay quiet.
func TestHub_ConcurrentSubscribeBroadcastUnsubscribe(t *testing.T) {
	h := NewHub(8, testLogger(), testMetrics(t))

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := string(rune('a' + i))
			ch := h.Subscribe(id)
			go func() {
				for range ch { //nolint:revive // drain until Unsubscribe closes it
				}
			}()
			h.Broadcast(context.Background(), Chunk("x"))
			h.Unsubscribe(id)
		}(i)
	}
	wg.Wait()

	if got := h.ListenerCount(); got != 0 {
		t.Fatalf("count = %d, want 0", got)
	}
}

// TestHub_DropsAreCounted pins the keystone metric of ADR 009: a full listener
// buffer must be observable, because the broadcaster keeps running and the
// dropped audio shows up in no error rate.
func TestHub_DropsAreCounted(t *testing.T) {
	metrics, sum := collectingMetrics(t)
	h := NewHub(1, testLogger(), metrics)
	h.Subscribe("slow") // never drained: capacity 1, so chunk 2 onwards drop

	ctx := t.Context()
	for range 4 {
		h.Broadcast(ctx, Chunk("x"))
	}

	if got := sum("lyo.chunks.dropped"); got != 3 {
		t.Errorf("lyo.chunks.dropped = %d, want 3", got)
	}
}

func TestHub_NoDropsWhenListenersKeepUp(t *testing.T) {
	metrics, sum := collectingMetrics(t)
	h := NewHub(8, testLogger(), metrics)
	ch := h.Subscribe("fast")

	ctx := t.Context()
	for range 4 {
		h.Broadcast(ctx, Chunk("x"))
		<-ch
	}

	if got := sum("lyo.chunks.dropped"); got != 0 {
		t.Errorf("lyo.chunks.dropped = %d, want 0", got)
	}
}
