# ADR 009 — Observability: Structured Logs Now, OpenTelemetry as the Single Export Protocol

**Status:** Accepted — implemented 2026-09-03
**Date:** 2026-08-10

## Context

A streaming backend fails in ways that are invisible from a request log alone. The two questions that
actually matter in operation are "is something broken?" and "are users suffering?" — and they are
different questions with different answers and different remedies. A 500 response means code failed. A
dropped audio chunk or an abrupt listener disconnect means the system worked **exactly as designed** and
the user still had a bad time. Conflating them leads to fixing the wrong thing.

Answering both requires three signals — logs, metrics, traces — and a decision about how they leave the
process. The options were: a vendor SDK (fast to adopt, hard to leave), Prometheus scraping plus a
separate tracing SDK (two mechanisms, two configurations), or OpenTelemetry as a single vendor-neutral
protocol.

## Decision

**OpenTelemetry (OTLP) is the only export protocol.** The application talks to an OpenTelemetry Collector
and knows nothing about what is behind it; the Collector routes metrics to Prometheus, logs to Loki and
traces to a trace backend. Swapping any of those is a Collector configuration change, not an application
change.

Structured logging is `log/slog` in JSON, which is already in place and requires no dependency.

Two metric families are declared **explicitly and separately**, because the distinction above is the
whole point:

| Family | Source | Examples |
|---|---|---|
| **Technical** | `otelhttp` middleware, automatic | request duration, status code, route |
| **Business / experience** | Hand-instrumented in `internal/streaming` | `lyo_listeners_active`, `lyo_streams_live`, `lyo_chunks_dropped_total`, `lyo_stream_bytes_total`, `lyo_listener_disconnect_total{reason}`, `lyo_broadcast_duration_seconds` |

`lyo_chunks_dropped_total` is the keystone metric: it is the only observable proof that the back-pressure
policy of [ADR 003](003-streaming-hub-pattern.md) is engaging, and it is an experience indicator that
would never appear in an error rate.

## Status and consequences

**Implemented**

- Structured JSON logging via `slog`, with a request identifier for correlation.
- The full collector stack (OTel Collector, Prometheus, Loki, Tempo, Grafana) is deployed in both Compose
  files, and the application exports all three signals to it over OTLP.
- `otelhttp` server spans and HTTP metrics, with the span renamed to the chi route pattern so path
  parameters do not multiply metric series.
- The six business metrics named above, recorded where the events happen in `internal/streaming`.
- `trace_id` and `span_id` on every log record, and a Loki derived field that turns them into a link to
  the trace in Tempo — the log↔trace correlation this ADR promised.
- Two provisioned Grafana dashboards, split along exactly the line this record draws: *API Performance*
  for the technical family, *Streaming Experience* for the business one.
- Alerting on those metrics, and the incident feed built on it — see [ADR 011](011-alerting-incidents.md).

**What the gap cost**

This record spent a release describing infrastructure that received nothing, and said so rather than
claiming otherwise. The lesson it carried is unchanged now that it is closed: **observability
infrastructure without application instrumentation is worth nothing**, and instrumenting a handler while
writing it is far cheaper than instrumenting thirty-six of them afterwards. The gap was closed in one
pass precisely because the metric names, their two families and their meanings had already been decided
here — the expensive part of the work was the design, and it was already written down.

**Consequences**

- Vendor neutrality: no rewrite to change backend tooling.
- One trace from the mobile app to the database, provided the client propagates `traceparent`. The
  backend accepts and continues an incoming trace context; the Flutter client does not yet send one, so
  today the trace starts at the backend and the "from the app to the database" property is still not met.
  That remains open, and is now the only part of this record that is.
- Trace identifiers in log lines, giving log↔trace correlation in Grafana.
- Alerting built on metrics that mean something operationally, rather than on host-level proxies.
