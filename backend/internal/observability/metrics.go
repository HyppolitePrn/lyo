package observability

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// meterName scopes every instrument below to this application.
const meterName = "github.com/hyppoliteprn/lyo"

// Metrics holds the business — as opposed to technical — instruments of the
// platform. The distinction matters operationally: a 500 means code broke, a
// dropped audio chunk means the system worked exactly as designed and the
// listener still heard a gap. Only the second kind tells us whether users are
// suffering, and it appears in no error rate.
//
// Deliberately low cardinality: no stream, user or listener identifier is
// ever used as an attribute, since each distinct value would create its own
// Prometheus series for the lifetime of the retention window.
type Metrics struct {
	listenersActive    metric.Int64UpDownCounter
	streamsLive        metric.Int64UpDownCounter
	chunksDropped      metric.Int64Counter
	streamBytes        metric.Int64Counter
	listenerDisconnect metric.Int64Counter
	broadcastDuration  metric.Float64Histogram
}

// DisconnectReason labels why a listener's session ended. "client" is normal
// (the app was closed); the others are the ones worth alerting on.
type DisconnectReason string

const (
	// ReasonClient — the listener closed the connection or the network dropped.
	ReasonClient DisconnectReason = "client"
	// ReasonStreamEnded — the broadcaster ended the stream, so every listener
	// is disconnected at once. Expected, but it explains a cliff in the graph.
	ReasonStreamEnded DisconnectReason = "stream_ended"
	// ReasonWriteFailed — the server could not push audio to the listener.
	ReasonWriteFailed DisconnectReason = "write_failed"
)

// NewMetrics registers the instruments on the global meter provider. When no
// provider is configured the meter is a no-op and every record is discarded,
// so callers never need to check whether telemetry is on.
func NewMetrics() (*Metrics, error) {
	m := otel.Meter(meterName)
	var errs []error
	collect := func(err error) { errs = append(errs, err) }

	metrics := &Metrics{}
	var err error

	metrics.listenersActive, err = m.Int64UpDownCounter("lyo.listeners.active",
		metric.WithDescription("Listeners currently connected to a live stream"),
		metric.WithUnit("{listener}"))
	collect(err)

	metrics.streamsLive, err = m.Int64UpDownCounter("lyo.streams.live",
		metric.WithDescription("Streams currently broadcasting"),
		metric.WithUnit("{stream}"))
	collect(err)

	// The keystone metric: the only observable proof that the hub's
	// back-pressure policy is engaging and that some listener heard a gap.
	metrics.chunksDropped, err = m.Int64Counter("lyo.chunks.dropped",
		metric.WithDescription("Audio chunks dropped because a listener's buffer was full"),
		metric.WithUnit("{chunk}"))
	collect(err)

	metrics.streamBytes, err = m.Int64Counter("lyo.stream.bytes",
		metric.WithDescription("Audio bytes ingested from broadcasters and delivered to listeners"),
		metric.WithUnit("By"))
	collect(err)

	metrics.listenerDisconnect, err = m.Int64Counter("lyo.listener.disconnect",
		metric.WithDescription("Listener sessions ended, by reason"),
		metric.WithUnit("{disconnect}"))
	collect(err)

	metrics.broadcastDuration, err = m.Float64Histogram("lyo.broadcast.duration",
		metric.WithDescription("Wall-clock duration of a completed broadcast"),
		metric.WithUnit("s"))
	collect(err)

	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return metrics, nil
}

// ListenerConnected and ListenerDisconnected bracket a listening session.
func (m *Metrics) ListenerConnected(ctx context.Context) {
	m.listenersActive.Add(ctx, 1)
}

func (m *Metrics) ListenerDisconnected(ctx context.Context, reason DisconnectReason) {
	m.listenersActive.Add(ctx, -1)
	m.listenerDisconnect.Add(ctx, 1,
		metric.WithAttributes(attribute.String("reason", string(reason))))
}

// StreamStarted and StreamEnded bracket a broadcast. StreamEnded also records
// how long it lasted, which is what distinguishes a healthy broadcast from one
// that died seconds after starting.
func (m *Metrics) StreamStarted(ctx context.Context) {
	m.streamsLive.Add(ctx, 1)
}

func (m *Metrics) StreamEnded(ctx context.Context, durationSeconds float64) {
	m.streamsLive.Add(ctx, -1)
	m.broadcastDuration.Record(ctx, durationSeconds)
}

// ChunkDropped records one chunk the hub could not hand to a listener.
func (m *Metrics) ChunkDropped(ctx context.Context) {
	m.chunksDropped.Add(ctx, 1)
}

// BytesIngested counts audio arriving from a broadcaster; BytesDelivered
// counts audio successfully written to a listener. Comparing the two is how a
// fan-out problem shows up as a shape on the dashboard.
func (m *Metrics) BytesIngested(ctx context.Context, n int) {
	m.streamBytes.Add(ctx, int64(n),
		metric.WithAttributes(attribute.String("direction", "ingest")))
}

func (m *Metrics) BytesDelivered(ctx context.Context, n int) {
	m.streamBytes.Add(ctx, int64(n),
		metric.WithAttributes(attribute.String("direction", "egress")))
}
