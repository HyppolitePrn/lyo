# Architecture Decision Records

Format: Title, Status, Context, Decision, Consequences. One record per structural decision, written
**at the moment the decision is taken** — a record reconstructed afterwards is a rationalisation, not a
record.

| # | Decision | Status | Why it matters |
|---|---|---|---|
| [001](001-router-chi.md) | HTTP router: chi | Accepted | `net/http` compatibility, no framework lock-in |
| [002](002-state-management-riverpod.md) | Flutter state: Riverpod | ⚠️ **Superseded by 004** | Kept as the history of a reversed decision |
| [003](003-streaming-hub-pattern.md) | Streaming engine: Hub / fan-out | Accepted | Non-blocking broadcast, per-listener chunk dropping |
| [004](004-provider-over-riverpod.md) | Flutter state: Provider over Riverpod | Accepted | Reversal of 002, with its reasoning |
| [005](005-openapi-first-codegen.md) | OpenAPI-first with code generation | Accepted | Contract drift becomes impossible to merge |
| [006](006-coder-websocket.md) | `coder/websocket`, WebSockets outside the contract | Accepted | Context-first API; query-param auth fallback |
| [007](007-permissive-auth-middleware.md) | Permissive auth middleware, handler-level authorization | Accepted | Public and authenticated callers on one route |
| [008](008-postgres-pgx-no-orm.md) | PostgreSQL + pgx, no ORM | Accepted | Visible SQL, database-enforced business rules |
| [009](009-observability-otlp.md) | OpenTelemetry as the single export protocol | Accepted | Technical vs. business metrics separated by design |
| [010](010-s3-presigned-upload.md) | Presigned S3 uploads | Accepted | Audio bytes never transit the API |
| [011](011-alerting-incidents.md) | Alerts become incidents owned by the platform | Accepted | Durable, acknowledgeable alerting; technical vs. experience families |
| [012](012-tls-reverse-proxy.md) | TLS terminated at a Caddy reverse proxy | Accepted | One public port, role-bearing tokens never travel in clear |
| [013](013-rate-limiting-and-account-deletion.md) | Per-address rate limiting in the app; account deletion erases audio first | Accepted | Credential guessing is throttled; a deleted account leaves no orphaned audio |

**Reading the set.** ADR 002 and 004 are best read together: they are the project's clearest worked
example of a decision made, evaluated in practice, and deliberately reversed with the reasoning written
down. ADR 007 and ADR 009 are the two records that state their own costs and gaps most explicitly — 007
names the failure mode its design permits, and 009 stated what was specified but not yet emitted until 011
closed that gap.
