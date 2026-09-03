package observability

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/trace"
)

// NewLogger returns a structured JSON logger at the given level, writing to
// stdout only. Used by tests and by any code path that runs before — or
// without — the telemetry pipeline.
func NewLogger(level string) *slog.Logger {
	return newSlogLogger(level, nil)
}

// newSlogLogger builds the application logger. Records always go to stdout as
// JSON (so `docker logs` stays useful even if the collector is down) and, when
// a LoggerProvider is supplied, are additionally exported over OTLP to Loki.
func newSlogLogger(level string, lp *sdklog.LoggerProvider) *slog.Logger {
	lvl := parseLevel(level)

	var h slog.Handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	if lp != nil {
		h = fanOutHandler{h, otelslog.NewHandler("lyo", otelslog.WithLoggerProvider(lp))}
	}
	return slog.New(&traceContextHandler{Handler: h})
}

func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// traceContextHandler stamps every record with the trace and span ID of the
// current context. This is what makes a log line in Loki clickable through to
// its trace in Grafana — without it the two signals cannot be correlated.
type traceContextHandler struct{ slog.Handler }

func (h *traceContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		r.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}
	return h.Handler.Handle(ctx, r)
}

func (h *traceContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &traceContextHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *traceContextHandler) WithGroup(name string) slog.Handler {
	return &traceContextHandler{Handler: h.Handler.WithGroup(name)}
}

// fanOutHandler writes each record to every underlying handler. Errors are
// collected but the remaining handlers still run: a broken OTLP connection
// must not silence stdout logging.
type fanOutHandler []slog.Handler

func (f fanOutHandler) Enabled(ctx context.Context, l slog.Level) bool {
	for _, h := range f {
		if h.Enabled(ctx, l) {
			return true
		}
	}
	return false
}

func (f fanOutHandler) Handle(ctx context.Context, r slog.Record) error {
	var firstErr error
	for _, h := range f {
		if !h.Enabled(ctx, r.Level) {
			continue
		}
		if err := h.Handle(ctx, r.Clone()); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (f fanOutHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	out := make(fanOutHandler, len(f))
	for i, h := range f {
		out[i] = h.WithAttrs(attrs)
	}
	return out
}

func (f fanOutHandler) WithGroup(name string) slog.Handler {
	out := make(fanOutHandler, len(f))
	for i, h := range f {
		out[i] = h.WithGroup(name)
	}
	return out
}
