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
| Mobile           | Flutter 3.41, Riverpod, just_audio |
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
- `providers/<feature>_notifier.dart` — `Notifier<State>` with a `copyWith` state class
- `services/<feature>_service.dart` — raw API calls via `ApiClient`
- `screens/` + `widgets/` — consume providers via `ref.watch`

`ApiClient` (`core/api/api_client.dart`) is a thin HTTP wrapper. It derives the base URL from `--dart-define=API_BASE_URL` at build time (default: `http://10.0.2.2:8080` for Android emulator → host). WebSocket URIs are derived from the same base via `.wsUri()`.

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
final flags = ref.watch(featureFlagsProvider);
if (!flags.isEnabled('flag_name')) return const SizedBox.shrink();
```

### Available flags
| Flag | Default | Description |
|------|---------|-------------|
| `live_streaming` | true | Live broadcast feature |
| `chat_websocket` | false | Live chat between listeners |
| `recommendations` | false | Listen-history recommendations |
| `offline_mode` | false | Playlist caching for offline use |
| `transcoding` | false | Adaptive bitrate transcoding |

---

## Roles

| Role | Permissions |
|------|-------------|
| `anonymous` | Browse public streams (no token) |
| `user` | Listen, favorites, playlists |
| `broadcaster` | Create live streams, upload audio |
| `admin` | All above + user management + feature flags |

Role hierarchy is ordinal — `claims.Role.AtLeast(auth.RoleBroadcaster)` is the standard check.

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
- Target ≥ 80% coverage on `internal/` packages
- Integration tests: `backend/internal/*_integration_test.go` with build tag `//go:build integration`
- Mobile: `flutter test` for unit + widget tests

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
