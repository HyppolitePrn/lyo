# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Lyo**.

| Layer            | Technology |
|------------------|-----------|
| Backend API      | Go 1.26, chi router, JWT auth |
| Streaming engine | Goroutines + channels (hub/fan-out pattern) |
| Database         | PostgreSQL (pgx/v5), migrations via golang-migrate (embedded FS) |
| Observability    | slog (JSON), OpenTelemetry, Grafana + Loki |
| Mobile           | Flutter 3.41, provider (ChangeNotifier), just_audio |
| CI/CD             | GitHub Actions |
| Containers       | Docker multi-stage (alpine), Docker Compose |

---

## Commands

### Backend

```bash
# Run server
cd backend && go run ./cmd/server

# Build
cd backend && go build ./...

# Lint (REQUIRED before any PR — see CI rule below)
cd backend && golangci-lint run ./...

# All tests (race detector always on)
cd backend && go test -race ./...

# Single package
cd backend && go test -race ./internal/streaming/...

# Regenerate OpenAPI bindings (after editing api/openapi.yaml)
cd backend && go generate ./internal/api/...
```

### Mobile

```bash
# Run on connected device/emulator
cd mobile && flutter run

# Override API URL (e.g. physical device or custom host)
cd mobile && flutter run --dart-define=API_BASE_URL=http://192.168.x.x:8080

# Lint
cd mobile && flutter analyze

# Tests
cd mobile && flutter test

# Single test file
cd mobile && flutter test test/features/auth/auth_notifier_test.dart
```

### Full stack

```bash
docker compose -f docker/docker-compose.yml up -d
```

---

## CI Simulation Rule — MANDATORY

**Before completing any code edit, Claude MUST run the appropriate check:**

- **Backend changes:** `cd backend && golangci-lint run ./... && go build ./... && go test -race ./... 2>&1 | tail -20`
- **Mobile changes:** `cd mobile && flutter analyze && flutter test 2>&1 | tail -20`

Fix all failures before presenting code. Never present code that fails `golangci-lint` or `flutter analyze`.

---

## Backend Architecture

### OpenAPI-first API layer

The REST API is contract-first. The source of truth is `backend/api/openapi.yaml`. Running `go generate ./internal/api/...` invokes `oapi-codegen` (config: `backend/oapi-codegen.yaml`) and writes `internal/api/api.gen.go` — do not edit that file manually.

Handler logic lives in `internal/api/handlers.go`, implementing the generated `StrictServerInterface`. Route mounting and WebSocket endpoints are wired in `cmd/server/main.go`.

WebSocket endpoints (`GET /streams/{id}/ingest` and `GET /streams/{id}/listen`) are **outside the OpenAPI spec** and registered directly on the chi router.

### Auth middleware behavior

`pkg/middleware.Authenticate` is **permissive**: it stores JWT claims in the request context if a valid `Bearer` token is present, but never rejects a request on its own. Individual handlers call `middleware.ClaimsFromContext(ctx)` and enforce role requirements themselves. This lets anonymous users hit public endpoints.

For WebSocket upgrades (which cannot send custom headers), auth falls back to `?token=` query param — handled in `streaming/ingest.go` and `streaming/listen.go`.

### Streaming data flow

```
Broadcaster app  →  WS /streams/{id}/ingest  →  Hub.Broadcast()
                                                       ↓  (fan-out to N buffered channels)
Listener app(s)  ←  WS /streams/{id}/listen  ←  Hub.Subscribe()
```

- `streaming.Service` owns the in-memory `map[streamID]*Hub`.
- Each `Hub` fans chunks out in parallel shards of 100 listeners (`hub.go:shardSize`).
- If a listener's buffer is full the chunk is **dropped for that listener** (never blocks the broadcaster).
- `Hub.Done()` channel fires when `EndStream` is called, cleanly terminating ingest loops.

### Database migrations

Migrations live in `backend/migrations/` as numbered SQL files and are embedded via `embed.go`. They run automatically at server startup (`runMigrations` in `main.go`). To add a migration, create the next numbered `*.up.sql` / `*.down.sql` pair.

---

## Mobile Architecture

### State management pattern

All features follow the same structure:
- `providers/<feature>_notifier.dart` — a `ChangeNotifier` subclass holding mutable state fields directly (no separate immutable state class), calling `notifyListeners()` after each mutation
- `services/<feature>_service.dart` — raw API calls via `ApiClient`
- `screens/` + `widgets/` — consume notifiers via `context.watch<T>()` (rebuilds on change) or `context.read<T>()` (one-off calls, e.g. inside callbacks/`initState`)

Notifiers are registered once in `main.dart` under a `MultiProvider` (`ChangeNotifierProvider` per notifier, plus a plain `Provider<FeatureFlags>`). Notifiers that need the current auth token (e.g. `PlayerNotifier.connect`, `BroadcasterNotifier.startBroadcast`) take it as a method parameter — read it from `AuthNotifier` via `context.read<AuthNotifier>().accessToken` at the call site — since notifiers don't reach into each other directly.

`ApiClient` (`core/api/api_client.dart`) is a thin, stateless HTTP wrapper (`const ApiClient()`), constructor-injected into each notifier for testability. It derives the base URL from `--dart-define=API_BASE_URL` at build time (default: `http://10.0.2.2:8080` for Android emulator → host). WebSocket URIs are derived from the same base via `.wsUri()`.

### Audio pipeline

**Player:** `PlayerNotifier` opens a WebSocket, pipes binary AAC frames through a `StreamController<Uint8List>`, and feeds them to `just_audio` via a custom `_WsAudioSource` (a `StreamAudioSource` subclass). `setAudioSource/play()` is fire-and-forget to avoid blocking the UI in "connecting" state.

**Broadcaster:** `BroadcasterNotifier` uses the `record` package to capture mic audio as AAC-LC ADTS (44100 Hz, mono, 128 kbps) and sends each chunk as a binary WebSocket message to the ingest endpoint.

### Auth state

`AuthState` holds `accessToken` (raw JWT) and `role` (decoded in-app from the JWT payload without a library — see `_jwtRole` in `auth_notifier.dart`). `isBroadcaster` is true for both `broadcaster` and `admin` roles.

---

## Feature Flag System

Every new feature MUST be gated behind a feature flag. Flags are DB-backed and seeded at startup via `internal/features/seed.go`.

### Backend
```go
// seed.go — add a row
// handler — gate at the top
if !h.featureSvc.IsEnabled(ctx, "flag_name") {
    return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "feature disabled"}
}
```

### Mobile
```dart
final flags = context.watch<FeatureFlags>();
if (!flags.isEnabled('flag_name')) return const SizedBox.shrink();
```

### Available flags
| Flag | Default | Description |
|------|---------|-------------|
| `live_streaming` | true | Live broadcast feature |
| `track_uploads` | true | Broadcaster track audio upload via presigned S3 URLs |
| `playlists` | true | Playlist creation and management |
| `favorites` | true | Favoriting tracks, streams, and playlists |
| `admin_supervision` | true | Admin supervision dashboard, incident feed and alert notifications |
| `chat_websocket` | false | Live chat between listeners |
| `recommendations` | false | Listen-history recommendations |
| `offline_mode` | false | Playlist caching for offline use |
| `transcoding` | false | Adaptive bitrate transcoding |

This table mirrors `internal/features/seed.go` — update both together.

---

## Production Topology — Reverse Proxy

`docker/docker-compose.prod.yml` publishes ports from **one** container: Caddy (`docker/Caddyfile`), which
terminates TLS with automatically renewed ACME certificates. The backend, Grafana, Postgres and the
observability stack expose ports only on the Compose network.

Consequences when changing anything deployment-related:

- **Never add a `ports:` mapping to a service in `docker-compose.prod.yml`.** Route it through the
  Caddyfile instead — a published port is a plaintext bypass of TLS and of the security headers.
- Security headers (HSTS, `X-Frame-Options`, …) are set **at the proxy**, once, for every route including
  the WebSocket and webhook endpoints that are outside the OpenAPI contract. Do not add per-handler
  header middleware.
- `/internal/*` is 404 at the edge. A new infrastructure-only endpoint belongs under that prefix.
- New WebSocket routes are covered by the `@websocket` matcher and bypass the 2 MB request-body cap;
  anything else must fit under it (audio never transits the API — presigned S3 upload, ADR 010).
- `TRUSTED_PROXY=true` is what allows `X-Forwarded-For` to be believed. `APP_ENV=production` refuses to
  start without it. Never make it default to true.
- Android release builds deny cleartext (`usesCleartextTraffic` is set in the debug/profile manifests
  only), and the release workflow fails if `VPS_URL` is not `https://`.

See [ADR 012](docs/adr/012-tls-reverse-proxy.md).

---

## Roles — No Route Writes a Role

| Role | Permissions |
|------|-------------|
| `anonymous` | Browse public streams (no token) |
| `user` | Listen, favorites, playlists |
| `broadcaster` | Create live streams, upload audio |
| `admin` | All above + user management + feature flags |

Role hierarchy is ordinal — `claims.Role.AtLeast(auth.RoleBroadcaster)` is the standard check.

**Invariant: no HTTP request may raise the role of an account.** Registration always creates
`user.selfServiceRole` (= `RoleUser`); `Repository.Update` writes username and email only;
`POST /auth/refresh` re-reads the role from the database rather than copying it out of the token being
exchanged, so a demoted or deleted account cannot renew its former privileges. Promotion is an operator
action on the database.

If a role-management endpoint is ever added, it must gate on `checkAdmin(ctx)` (`internal/api/supervision.go`)
like every other `/admin/*` route, and refuse to grant a role above the caller's own. The regression tests
that pin this invariant live in `internal/api/privilege_escalation_test.go` — extend them, do not weaken
them.

---

## Request Timeouts — MANDATORY

Every handler must derive a timeout context at the top:

```go
ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
defer cancel()
```

| Operation | Timeout |
|-----------|---------|
| DB query (simple) | 5s |
| DB query (complex) | 10s |
| External HTTP call | 10s |
| Audio stream init | 15s |

- Return `503 Service Unavailable` (not 500) on `context.DeadlineExceeded`.
- Log at `warn` level with route and elapsed time.

---

## Testing Conventions

- Race detector is always on: `go test -race ./...`
- Target ≥ 80% coverage on `internal/` packages, enforced by the CI coverage gate
- Handler-level tests go through `net/http/httptest` (`internal/api/handlers_*_test.go`); the WebSocket
  chain is covered end to end against a real connection in `internal/streaming/ws_test.go`
- Mobile: `flutter test` for unit + widget tests

There is no database integration-test suite today: repositories are tested against a `PgxPool` double.
Adding one is tracked in [the test plan](docs/plan-de-tests.md) — do not document a convention for it
here until the tests actually exist.

---

## Architecture Decision Records

Non-trivial architectural choices are recorded in `docs/adr/` as numbered markdown files. Format: Title, Status, Context, Decision, Consequences.

---

## Environment Variables

All config via env vars — use `pkg/config/config.go`, never hardcode. See `.env.example` for the full list.

Key vars: `DATABASE_URL`, `JWT_SECRET` (min 32 chars), `SERVER_PORT` (default 8080), `STREAM_BUFFER_SIZE`, `OTEL_EXPORTER_OTLP_ENDPOINT`, `LOG_LEVEL`.

---

## Commit Conventions

Format: `type(scope): message`

Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `ci`
