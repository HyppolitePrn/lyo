package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Trace instruments every request with an OpenTelemetry server span and the
// standard HTTP metrics (duration, request/response size, status code) that
// the performance dashboard is built on.
//
// otelhttp cannot know the route pattern — chi only resolves it while
// routing — so the span starts with a placeholder name and is renamed to the
// pattern (e.g. "GET /streams/{id}") once the handler returns. Without this
// every path parameter would create its own metric series and blow up
// cardinality.
func Trace(serviceName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		instrumented := otelhttp.NewHandler(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)
				nameSpanFromRoute(r)
			}),
			serviceName,
		)
		return instrumented
	}
}

func nameSpanFromRoute(r *http.Request) {
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		return
	}
	pattern := rctx.RoutePattern()
	if pattern == "" {
		return
	}
	span := trace.SpanFromContext(r.Context())
	if !span.IsRecording() {
		return
	}
	span.SetName(r.Method + " " + pattern)
	span.SetAttributes(attribute.String("http.route", pattern))
}
