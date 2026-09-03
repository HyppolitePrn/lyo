package observability

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

// Config describes how this process identifies itself to the collector.
// It mirrors pkg/config.ObsConfig but is declared here so the observability
// package stays independent of the application's configuration loader.
type Config struct {
	Enabled        bool
	ServiceName    string
	ServiceVersion string
	Environment    string
	OTLPEndpoint   string
	LogLevel       string
}

// Providers holds the OpenTelemetry pipelines owned by the process.
// Shutdown must be called on exit so buffered spans, metrics and logs are
// flushed rather than lost.
type Providers struct {
	// Logger is the application logger: JSON to stdout always, plus OTLP
	// export when telemetry is enabled. Trace and span IDs of the current
	// context are attached to every record.
	Logger *slog.Logger

	shutdowns []func(context.Context) error
}

// metricExportInterval is how often the SDK pushes metrics to the collector.
// It must stay below Prometheus' scrape_interval (15s) so every scrape sees a
// fresh sample rather than a stale one.
const metricExportInterval = 10 * time.Second

// Setup builds the trace, metric and log pipelines and registers them
// globally, so that any package can reach them through otel.Tracer /
// otel.Meter without being handed a dependency.
//
// When cfg.Enabled is false — tests, or a deployment with no collector — no
// exporter is created and the returned Providers only carries a stdout
// logger. Instrumented code keeps calling the same API and gets the no-op
// implementations, so nothing needs to be conditional at the call sites.
func Setup(ctx context.Context, cfg Config) (*Providers, error) {
	p := &Providers{}

	if !cfg.Enabled {
		p.Logger = newSlogLogger(cfg.LogLevel, nil)
		return p, nil
	}

	res, err := newResource(cfg)
	if err != nil {
		return nil, err
	}

	// Propagate W3C trace context so a traceparent header sent by the mobile
	// client continues the same trace instead of starting a new one.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	if err := p.initTracing(ctx, cfg, res); err != nil {
		return nil, p.abort(ctx, err)
	}
	if err := p.initMetrics(ctx, cfg, res); err != nil {
		return nil, p.abort(ctx, err)
	}
	logProvider, err := p.initLogging(ctx, cfg, res)
	if err != nil {
		return nil, p.abort(ctx, err)
	}

	p.Logger = newSlogLogger(cfg.LogLevel, logProvider)

	// Go runtime metrics (goroutines, GC, heap) — the host-level half of the
	// supervision dashboard.
	if err := runtime.Start(runtime.WithMinimumReadMemStatsInterval(metricExportInterval)); err != nil {
		return nil, p.abort(ctx, fmt.Errorf("runtime metrics: %w", err))
	}

	// Export failures must never take the process down: log and carry on.
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		p.Logger.Warn("opentelemetry export error", slog.Any("err", err))
	}))

	return p, nil
}

func newResource(cfg Config) (*resource.Resource, error) {
	res, err := resource.Merge(resource.Default(), resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(cfg.ServiceName),
		semconv.ServiceVersion(cfg.ServiceVersion),
		semconv.DeploymentEnvironmentNameKey.String(cfg.Environment),
	))
	if err != nil {
		return nil, fmt.Errorf("otel resource: %w", err)
	}
	return res, nil
}

func (p *Providers) initTracing(ctx context.Context, cfg Config, res *resource.Resource) error {
	exp, err := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpointURL(cfg.OTLPEndpoint))
	if err != nil {
		return fmt.Errorf("otlp trace exporter: %w", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	p.shutdowns = append(p.shutdowns, tp.Shutdown)
	return nil
}

func (p *Providers) initMetrics(ctx context.Context, cfg Config, res *resource.Resource) error {
	exp, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithEndpointURL(cfg.OTLPEndpoint))
	if err != nil {
		return fmt.Errorf("otlp metric exporter: %w", err)
	}
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exp,
			sdkmetric.WithInterval(metricExportInterval))),
	)
	otel.SetMeterProvider(mp)
	p.shutdowns = append(p.shutdowns, mp.Shutdown)
	return nil
}

func (p *Providers) initLogging(ctx context.Context, cfg Config, res *resource.Resource) (*sdklog.LoggerProvider, error) {
	exp, err := otlploggrpc.New(ctx, otlploggrpc.WithEndpointURL(cfg.OTLPEndpoint))
	if err != nil {
		return nil, fmt.Errorf("otlp log exporter: %w", err)
	}
	lp := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exp)),
	)
	p.shutdowns = append(p.shutdowns, lp.Shutdown)
	return lp, nil
}

// abort tears down whatever was already started when a later stage fails, so
// a partial pipeline never survives a failed Setup.
func (p *Providers) abort(ctx context.Context, cause error) error {
	if err := p.Shutdown(ctx); err != nil {
		return errors.Join(cause, err)
	}
	return cause
}

// Shutdown flushes and stops every pipeline, in reverse order of creation.
func (p *Providers) Shutdown(ctx context.Context) error {
	var errs []error
	for i := len(p.shutdowns) - 1; i >= 0; i-- {
		if err := p.shutdowns[i](ctx); err != nil {
			errs = append(errs, err)
		}
	}
	p.shutdowns = nil
	return errors.Join(errs...)
}
