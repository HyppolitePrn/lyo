package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/hyppoliteprn/lyo/pkg/middleware"
)

func TestTrace_SpanNamedAfterRoutePattern(t *testing.T) {
	rec := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { _ = tp.Shutdown(t.Context()) })

	r := chi.NewRouter()
	r.Use(middleware.Trace("lyo-test"))
	r.Get("/streams/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/streams/abc-123", nil)
	r.ServeHTTP(httptest.NewRecorder(), req)

	spans := rec.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	// The concrete ID must not leak into the span name, or every stream would
	// create its own metric series.
	if got := spans[0].Name(); got != "GET /streams/{id}" {
		t.Errorf("span name = %q, want %q", got, "GET /streams/{id}")
	}
}

func TestTrace_UnmatchedRouteKeepsFallbackName(t *testing.T) {
	rec := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { _ = tp.Shutdown(t.Context()) })

	r := chi.NewRouter()
	r.Use(middleware.Trace("lyo-test"))
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/nope", nil)
	r.ServeHTTP(httptest.NewRecorder(), req)

	spans := rec.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	// No pattern matched, so the span keeps otelhttp's own name — what matters
	// is that the unrouted path never becomes part of it.
	if got := spans[0].Name(); got != "GET" {
		t.Errorf("span name = %q, want the otelhttp default %q", got, "GET")
	}
}
