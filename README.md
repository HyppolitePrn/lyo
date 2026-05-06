# Lyo — StreamPulse

Real-time audio streaming platform. Bloc 3 project — RNCP 38822.

A broadcaster streams live audio from a mobile app; listeners tune in with zero configuration. The backend fans audio chunks to N concurrent listeners via a hub/fan-out pattern. Feature access is role-gated and controlled at runtime through a feature flag system.

---

## Stack

| Layer | Technology | Version |
|-------|-----------|---------|
| Backend API | Go + chi router | 1.26 |
| Database | PostgreSQL (pgx/v5) | 17 |
| Mobile | Flutter + Riverpod + just_audio | 3.41 |
| Containers | Docker + Docker Compose | — |
| Observability | slog, OpenTelemetry, Grafana + Loki | — |

The REST API is OpenAPI-first (`backend/api/openapi.yaml`). Server stubs are generated via `oapi-codegen` — never edit `internal/api/api.gen.go` manually.

---

## Live server

| | |
|---|---|
| API base URL | `http://167.235.254.103:8080` |
| Health check | `GET http://167.235.254.103:8080/health` |

To point the mobile app at the live server:

```bash
cd mobile && flutter run --dart-define=API_BASE_URL=http://167.235.254.103:8080
```

---

## Running locally

**Prerequisites:** Go 1.26, Flutter 3.41, Docker + Docker Compose, `golangci-lint` v2.

```bash
# 1. Copy env config
cp .env.example .env   # fill in JWT_SECRET (min 32 chars)

# 2. Start Postgres + observability stack
docker compose -f docker/docker-compose.yml up -d

# 3. Start backend (auto-migrates on boot)
cd backend && go run ./cmd/server

# 4. Run mobile (default: Android emulator → host at 10.0.2.2:8080)
cd mobile && flutter run
```

---

## Architecture overview

### Backend

```
cmd/server/main.go        ← wires everything: DB pool, services, router
internal/api/             ← HTTP handlers (StrictServerInterface from oapi-codegen)
internal/streaming/       ← Hub fan-out engine + WebSocket ingest/listen handlers
internal/auth/            ← JWT sign/verify, role hierarchy
internal/features/        ← DB-backed feature flags, seeded at startup
pkg/middleware/           ← Authenticate (permissive) + RequireRole
backend/migrations/       ← numbered SQL files, auto-applied via golang-migrate
```

The `Authenticate` middleware stores JWT claims in context but never blocks — handlers check roles themselves via `middleware.ClaimsFromContext`. WebSocket endpoints use a `?token=` query param fallback because WebSocket upgrades cannot carry custom headers.

Live streaming data flow:

```
Broadcaster  →  WS /streams/{id}/ingest  →  Hub.Broadcast()
                                                  ↓ fan-out (shards of 100)
Listener(s)  ←  WS /streams/{id}/listen  ←  Hub.Subscribe()
```

### Mobile

Each feature follows the same layout:

```
features/<name>/
  providers/<name>_notifier.dart   ← Notifier<State>, business logic
  services/<name>_service.dart     ← raw ApiClient calls
  screens/ + widgets/              ← UI, reads state via ref.watch
```

`ApiClient` (`core/api/api_client.dart`) derives its base URL from `--dart-define=API_BASE_URL` at build time. The player feeds WebSocket binary AAC frames to `just_audio` via a custom `StreamAudioSource`. The broadcaster captures mic audio with the `record` package (AAC-LC, 44100 Hz, 128 kbps) and sends chunks over WebSocket.

---

## Roles

| Role | Access |
|------|--------|
| `anonymous` | Browse public streams (no token required) |
| `user` | Listen, favorites, playlists |
| `broadcaster` | Start/end live streams |
| `admin` | Everything + user management + feature flags |

---

## Feature flags

All features are gated by DB-backed flags (seeded in `internal/features/seed.go`). Currently active:

| Flag | Default |
|------|---------|
| `live_streaming` | **on** |
| `chat_websocket` | off |
| `recommendations` | off |
| `offline_mode` | off |
| `transcoding` | off |

---

## Deployment

### Backend — automatic on merge to `main`

Every push to `main` that touches `backend/**` triggers the full CI pipeline (lint → test → build), then:

1. Builds a Docker image and pushes it to GHCR as `ghcr.io/hyppoliteprn/lyo-backend:latest`
2. SSHs into the VPS and runs:
   ```bash
   docker compose -f docker-compose.prod.yml pull backend
   docker compose -f docker-compose.prod.yml up -d backend
   ```

No manual step is needed — merge the PR and the server updates itself.

### Mobile release — triggered by a version tag

Pushing a `vX.Y.Z` tag builds a signed APK + AAB and publishes a GitHub Release:

```bash
git tag v1.2.0
git push origin v1.2.0
```

The tag drives the `pubspec.yaml` version automatically (`1.2.0+<run_number>`). The APK is built against the live server URL from the `VPS_URL` secret.

### Required GitHub secrets

These must be set in **Settings → Secrets and variables → Actions** before any deploy can succeed:

| Secret | Used by |
|--------|---------|
| `VPS_HOST` | SSH into the server |
| `VPS_USER` | SSH user |
| `VPS_SSH_KEY` | SSH private key |
| `VPS_DEPLOY_PATH` | Absolute path to `docker-compose.prod.yml` on the VPS |
| `VPS_URL` | Full API base URL baked into the mobile APK |
| `KEYSTORE_BASE64` | Android signing keystore (base64-encoded) |
| `KEYSTORE_PASSWORD` | Keystore password |
| `KEY_PASSWORD` | Key password |
| `KEY_ALIAS` | Key alias |

### VPS one-time setup

The server needs this done once before the first deploy:

```bash
# 1. Install Docker + Docker Compose
# 2. Log in to GHCR so the VPS can pull the image
docker login ghcr.io -u <github-username> --password <personal-access-token>

# 3. Create the deploy directory and drop in the prod compose file + .env
mkdir -p /path/to/deploy
# place docker-compose.prod.yml and .env (with DATABASE_URL, JWT_SECRET, etc.) here
```

After that, all subsequent deploys are fully automated by CI.

---

## Contributing

**Branch naming:** `type/short-description` — mirrors the commit type (e.g. `feat/broadcaster-screen`, `fix/jwt-expiry`, `ci/lint-step`).

**Commit format:** `type(scope): message` — types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `ci`.

**Before opening a PR:**

```bash
# Backend
cd backend && golangci-lint run ./... && go build ./... && go test -race ./...

# Mobile
cd mobile && flutter analyze && flutter test
```

---

## Architecture decisions

Key choices are documented in [`docs/adr/`](docs/adr/):

- [ADR 001 — HTTP Router: chi](docs/adr/001-router-chi.md)
- [ADR 002 — Flutter State Management: Riverpod](docs/adr/002-state-management-riverpod.md)
- [ADR 003 — Streaming Engine: Hub / Fan-out](docs/adr/003-streaming-hub-pattern.md)
