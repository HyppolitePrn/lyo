package observability_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/trace"

	"github.com/hyppoliteprn/lyo/internal/observability"
)

func TestSetup_DisabledStillProvidesLogger(t *testing.T) {
	p, err := observability.Setup(context.Background(), observability.Config{
		Enabled:  false,
		LogLevel: "debug",
	})
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if p.Logger == nil {
		t.Fatal("expected a logger even with telemetry disabled")
	}
	if !p.Logger.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("expected debug level to be honoured")
	}
	if err := p.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown: %v", err)
	}
}

func TestSetup_UnreachableCollectorDoesNotBlockStartup(t *testing.T) {
	// The gRPC exporters connect lazily, so Setup must succeed even with
	// nothing listening — the server has to boot when the collector is down.
	p, err := observability.Setup(context.Background(), observability.Config{
		Enabled:      true,
		ServiceName:  "lyo-test",
		OTLPEndpoint: "http://127.0.0.1:1",
		LogLevel:     "info",
	})
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // do not wait on a flush that can never reach the collector
	_ = p.Shutdown(ctx)
}

// TestLogger_AddsTraceCorrelation is the log↔trace link the Grafana dashboards
// rely on: a log line without trace_id cannot be pivoted to its trace.
func TestLogger_AddsTraceCorrelation(t *testing.T) {
	out := captureStdout(t, func() {
		logger := observability.NewLogger("info")

		traceID, _ := trace.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
		spanID, _ := trace.SpanIDFromHex("0102030405060708")
		ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    traceID,
			SpanID:     spanID,
			TraceFlags: trace.FlagsSampled,
		}))
		logger.InfoContext(ctx, "hello")
	})

	var rec map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &rec); err != nil {
		t.Fatalf("log line is not JSON (%q): %v", out, err)
	}
	if got := rec["trace_id"]; got != "0102030405060708090a0b0c0d0e0f10" {
		t.Errorf("trace_id = %v, want the span context trace ID", got)
	}
	if got := rec["span_id"]; got != "0102030405060708" {
		t.Errorf("span_id = %v, want the span context span ID", got)
	}
}

func TestLogger_NoTraceContextNoCorrelationFields(t *testing.T) {
	out := captureStdout(t, func() {
		observability.NewLogger("info").Info("hello")
	})
	if strings.Contains(out, "trace_id") {
		t.Errorf("unexpected trace_id on an untraced record: %q", out)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	fn()
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe: %v", err)
	}
	os.Stdout = orig

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	return buf.String()
}
