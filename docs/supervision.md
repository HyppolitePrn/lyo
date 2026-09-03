# Supervision, alerting and incident response

How to run the observability stack, what each piece answers, and what to do when something fires.

The design decisions behind all of this are in [ADR 009](adr/009-observability-otlp.md) (why
OpenTelemetry is the only export protocol, and why metrics are split into two families) and
[ADR 011](adr/011-alerting-incidents.md) (why alerts become incidents owned by the platform).

---

## Running it

```bash
docker compose -f docker/docker-compose.yml up -d
```

| Service | URL | What it is |
|---|---|---|
| Grafana | http://localhost:3000 (`admin` / `admin`) | Dashboards, alert rules, alert state |
| Prometheus | http://localhost:9090 | Metric storage; useful for checking a raw query |
| Loki | http://localhost:3100 | Log storage (queried through Grafana) |
| Tempo | http://localhost:3200 | Trace storage (queried through Grafana) |
| Collector | OTLP on `:4317` (gRPC) and `:4318` (HTTP) | The only endpoint the application talks to |

The backend needs three variables, all in `.env.example`:

| Variable | Purpose |
|---|---|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Where to export traces, metrics and logs |
| `PROMETHEUS_URL` | Powers `GET /admin/supervision`; blank serves incidents without live metrics |
| `ALERT_WEBHOOK_SECRET` | Shared secret Grafana presents on `POST /internal/alerts`; **blank disables the endpoint** |

Under docker-compose, `ALERT_WEBHOOK_SECRET` is *not* read from `.env`: the backend and Grafana both take
it from Compose's own environment, from a single shared expression, so the two ends of the webhook cannot
be configured differently. It defaults to a dev value; override it with

```bash
export ALERT_WEBHOOK_SECRET=$(openssl rand -hex 32)
docker compose -f docker/docker-compose.yml up -d
```

In production the variable is required and Compose refuses to start without it — a mismatch would reject
every alert notification with a 401, silently.

`OTEL_ENABLED=false` keeps stdout logging and creates no exporter — use it to run the backend without
the stack. Exporters connect lazily, so a collector that is down never delays startup.

---

## Where each signal goes

```
backend ──OTLP──> collector ─┬─> Prometheus  (metrics, scraped from the collector)
                             ├─> Loki        (logs)
                             └─> Tempo       (traces)
                                     ↓
                                  Grafana ──alert──> POST /internal/alerts ──> incidents table
                                                                                    ↓
                                                                         admin app + email
```

Every log line carries the `trace_id` of its request, and Grafana turns it into a link to the trace.
That link is the fastest path from "this request failed" to "here is why".

---

## The two dashboards, and which one to open

**Lyo — API Performance** answers *is something broken?* Throughput, 5xx rate, latency percentiles,
slowest routes, Go runtime, and the error log stream.

**Lyo — Streaming Experience** answers *are users suffering?* Live streams, active listeners, dropped
audio chunks, audio in versus out, disconnect reasons, broadcast duration.

These are different questions. A platform can be at a 0% error rate and still be dropping audio for every
listener on an overloaded stream, and only the second dashboard will say so.

### The metrics that carry the most meaning

| Metric | Read it as |
|---|---|
| `lyo_chunks_dropped_total` | Listeners are hearing gaps *right now*. Nothing is broken; the hub is shedding load exactly as [ADR 003](adr/003-streaming-hub-pattern.md) designed. |
| `lyo_listener_disconnect_total{reason}` | `client` is someone closing the app. `write_failed` is the platform cutting a listener off. Only the second is a problem. |
| `lyo_listeners_active` | Must return to zero when streams end. A count that only rises is a leak, and it makes every capacity reading wrong. |
| `lyo_stream_bytes_total{direction}` | Egress should track ingress times the listener count. Flat egress under rising ingress is a fan-out failure. |

---

## When an alert fires

1. **Grafana evaluates** the rules in `docker/grafana/provisioning/alerting/rules.yml` (as code — UI
   edits are disabled and overwritten on restart).
2. **It POSTs** to `/internal/alerts` with a Bearer secret.
3. **The backend records an incident**, deduplicated on Grafana's fingerprint, and emails every admin —
   but only for a genuinely new incident, never for a re-fire.
4. **An admin acknowledges or resolves it** from the app (Profile → Supervision) or through
   `PATCH /admin/incidents/{id}`.

Grafana also resolves the incident automatically when the condition clears.

### Reading an incident

Each one carries a **family**, and it tells you what kind of work is next:

- `technical` — the platform failed. There is a bug, a dependency is down, or a resource ran out.
- `experience` — nothing failed and users suffered anyway. Usually a capacity decision:
  `STREAM_BUFFER_SIZE`, listener count per stream, or network headroom.

### The rules

| Rule | Family | Fires when |
|---|---|---|
| Backend unreachable | technical | `up == 0` for 2m |
| Telemetry export failing | technical | The collector is dropping what it cannot forward |
| API error rate above 5% | technical | 5xx share over 5% for 5m |
| API latency p95 above 1s | technical | p95 over 1s for 10m |
| Goroutine count climbing | technical | Over 1000 goroutines for 15m — the signature of a leak |
| Listeners are losing audio | experience | Chunks dropped for 5m |
| Listeners dropped by the server | experience | `reason=write_failed` disconnects for 5m |

---

## Checking it end to end

```bash
# Does the collector have the application's metrics?
curl -s 'http://localhost:9090/api/v1/label/__name__/values' | grep lyo_

# Are logs reaching Loki, with trace correlation?
curl -s --get 'http://localhost:3100/loki/api/v1/query_range' \
  --data-urlencode '{service_name="lyo-backend"}' --data-urlencode 'limit=1'

# Are the rules healthy? (health should be "ok" for every rule)
curl -s -u admin:admin 'http://localhost:3000/api/prometheus/grafana/api/v1/rules'

# Simulate an alert without waiting for one
curl -X POST http://localhost:8080/internal/alerts \
  -H "Authorization: Bearer $ALERT_WEBHOOK_SECRET" \
  -H 'Content-Type: application/json' \
  -d '{"alerts":[{"status":"firing","fingerprint":"manual-test",
       "labels":{"alertname":"Manual test","severity":"warning","family":"technical"},
       "annotations":{"summary":"Manual webhook test"}}]}'
```

**If a dashboard is empty**, check in this order: the backend is exporting
(`OTEL_ENABLED`, `OTEL_EXPORTER_OTLP_ENDPOINT`), the collector is up
(`docker compose logs otel-collector`), and Prometheus is scraping it
(http://localhost:9090/targets). A metric that has never been recorded — no stream has ever gone live, so
`lyo_streams_live` does not exist yet — has no series at all, which looks identical to a broken pipeline
until you check the first two.
