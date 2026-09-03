package streaming

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestHub_SubscribeUnsubscribeCount(t *testing.T) {
	h := NewHub(4, testLogger())

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
	h := NewHub(4, testLogger())
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
	h := NewHub(4, testLogger())
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
	h := NewHub(2, testLogger())

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
	h := NewHub(1, testLogger())
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
	h := NewHub(0, testLogger()) // unbuffered: sends block until ctx cancels
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
	h := NewHub(4, testLogger())

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
	h := NewHub(8, testLogger())

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
