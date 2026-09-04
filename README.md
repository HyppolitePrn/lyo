# Lyo — StreamPulse

Real-time audio streaming platform.

**Version française :** [`README.fr.md`](README.fr.md) · **Full documentation:** [`docs/`](docs/)

A broadcaster streams live audio from a mobile app; listeners tune in with zero configuration. The backend fans audio chunks to N concurrent listeners via a hub/fan-out pattern. Feature access is role-gated and controlled at runtime through a feature flag system.

---

## Stack

| Layer | Technology | Version |
|-------|-----------|---------|
| Backend API | Go + chi router | 1.26 |
| Database | PostgreSQL (pgx/v5) | 17 |
| Mobile | Flutter + provider + just_audio | 3.41 |
| Containers | Docker + Docker Compose | — |
| Observability | slog (JSON), OpenTelemetry (traces, metrics, logs over OTLP), Grafana + Loki ✅ *(the mobile client does not yet propagate `traceparent`, so a trace starts at the backend — [ADR 009](docs/adr/009-observability-otlp.md))* | — |

The REST API is OpenAPI-first (`backend/api/openapi.yaml`). Server stubs are generated via `oapi-codegen` — never edit `internal/api/api.gen.go` manually.

---

## Live server

| | |
|---|---|
| API base URL | `http://API_IP:8080` |
| Health check | `GET http://API_IP:8080/health` |

To point the mobile app at the live server:

```bash
cd mobile && flutter run --dart-define=API_BASE_URL=http://API_IP:8080
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

**Local service URLs:**

| Service | URL | Credentials |
|---------|-----|-------------|
| Backend API | `http://localhost:8080` | — |
| pgAdmin | `http://localhost:5050` | `some@one.com` / `someone` |
| Grafana | `http://localhost:3000` | `admin` / `admin` |
| Prometheus | `http://localhost:9090` | — |

pgAdmin ships with `PGADMIN_CONFIG_SERVER_MODE: "False"` (desktop mode, no login persistence) — it's a dev-only convenience container, not present in `docker-compose.prod.yml`. Add the `postgres` service as a new server inside pgAdmin using host `postgres`, port `5432`, and the credentials from `.env`.

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
  providers/<name>_notifier.dart   ← ChangeNotifier, business logic
  services/<name>_service.dart     ← raw ApiClient calls
  screens/ + widgets/              ← UI, reads state via context.watch / context.read
```

Notifiers are registered once in `main.dart` under a `MultiProvider`. Notifiers never reach into each
other: one that needs the current auth token takes it as a method parameter, read at the call site from
`AuthNotifier`. See [ADR 004](docs/adr/004-provider-over-riverpod.md) for why `provider` replaced
Riverpod.

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

| Flag | Default | Implemented |
|------|---------|-------------|
| `live_streaming` | **on** | ✅ |
| `track_uploads` | **on** | ✅ |
| `playlists` | **on** | ✅ |
| `favorites` | **on** | ✅ |
| `chat_websocket` | off | ❌ flag only |
| `recommendations` | off | ❌ flag only |
| `offline_mode` | off | ❌ flag only |
| `transcoding` | off | ❌ flag only |

> The mobile client currently reads these from a compile-time constant map
> (`core/features/feature_flags_provider.dart`); loading them from `GET /features` at runtime is on the
> roadmap. Until then, toggling a flag server-side changes API behaviour but not the mobile UI.

---

## Deployment

### TLS and the reverse proxy

In production, **Caddy is the only container with a published port**. It terminates TLS on 443, redirects
80, and proxies to the backend and Grafana over the private Compose network — neither of which publishes a
port any more. Certificates are issued and renewed automatically over ACME, so there is no renewal cron to
own. The rationale, and the gaps this does *not* close, are in
[ADR 012](docs/adr/012-tls-reverse-proxy.md).

| Variable | Meaning |
|---|---|
| `LYO_DOMAIN` | Public hostname. Certificates are issued for it **and** for `grafana.<domain>` |
| `ACME_EMAIL` | Contact address for expiry notices |
| `TRUSTED_PROXY` | Set to `true` behind the proxy so the API reads the client IP from `X-Forwarded-For`. `APP_ENV=production` refuses to start without it |
| `CORS_ALLOWED_ORIGINS` | Comma-separated browser origins; set to the public origin in production |

| URL | Serves |
|---|---|
| `https://<LYO_DOMAIN>` | The API, including the `/streams/{id}/ingest` and `/listen` WebSockets (`wss://`) |
| `https://grafana.<LYO_DOMAIN>` | Grafana |
| `https://<LYO_DOMAIN>/internal/*` | Nothing — 404 at the edge; the alert webhook is reachable only from inside the network |

Both DNS records must resolve to the host **before** the first start, or the ACME challenge fails. Left at
the default `LYO_DOMAIN=localhost`, Caddy signs with its own internal CA, which is what lets the production
stack come up on a laptop.

To verify a deployment:

```bash
curl -sI https://<LYO_DOMAIN>/health | grep -i strict-transport-security
curl -sI http://<LYO_DOMAIN>/health | head -1     # expect 308 -> https
curl -s  -o /dev/null -w '%{http_code}\n' https://<LYO_DOMAIN>/internal/alerts   # expect 404
```

### Backend — automatic on merge to `main`

Every push to `main` that touches `backend/**` triggers the full CI pipeline (lint → test → build), then:

1. Builds a Docker image and pushes it to GHCR as `ghcr.io/hyppoliteprn/lyo-backend:latest`
2. SSHs into the VPS and runs:
   ```bash
   docker compose -f docker-compose.prod.yml pull backend
   docker compose -f docker-compose.prod.yml up -d
   ```
   The second command covers the whole stack, not just the backend: the API no longer publishes a port,
   so it is reachable only once the Caddy edge is up. Compose recreates changed services only.

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
# place docker-compose.prod.yml, Caddyfile, and .env here.
#   .env needs DATABASE_URL, JWT_SECRET (>= 32 chars), POSTGRES_PASSWORD,
#   GRAFANA_PASSWORD, ALERT_WEBHOOK_SECRET, LYO_DOMAIN, ACME_EMAIL.
#   Compose refuses to start if GRAFANA_PASSWORD or ALERT_WEBHOOK_SECRET is unset.

# 4. Open 80 and 443 — ACME validates over both, and 80 also serves the
#    redirect to HTTPS. Nothing else needs to be reachable from outside.
sudo ufw allow 80,443/tcp
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
- [ADR 002 — Flutter State Management: Riverpod](docs/adr/002-state-management-riverpod.md) — *superseded by 004*
- [ADR 003 — Streaming Engine: Hub / Fan-out](docs/adr/003-streaming-hub-pattern.md)
- [ADR 004 — Flutter State Management: Provider over Riverpod](docs/adr/004-provider-over-riverpod.md)
- [ADR 005 — OpenAPI-first with code generation](docs/adr/005-openapi-first-codegen.md)
- [ADR 006 — coder/websocket, WebSockets outside the contract](docs/adr/006-coder-websocket.md)
- [ADR 007 — Permissive auth middleware](docs/adr/007-permissive-auth-middleware.md)
- [ADR 008 — PostgreSQL + pgx, no ORM](docs/adr/008-postgres-pgx-no-orm.md)
- [ADR 009 — OpenTelemetry as the single export protocol](docs/adr/009-observability-otlp.md)
- [ADR 010 — Presigned S3 uploads](docs/adr/010-s3-presigned-upload.md)

---

## Documentation

Full documentation index: **[`docs/README.md`](docs/README.md)** (FR / EN).

| Document | Contents |
|---|---|
| [Requirements specification](docs/cahier-des-charges.en.md) · [FR](docs/cahier-des-charges.fr.md) | Context, feasibility incl. cost/ROI, scope, specifications, risks, roadmap, KPIs, compliance, reflective review |
| [Architecture](docs/architecture/) | UML and BPMN diagrams, data model, security model, deployment — every diagram paired with a text alternative |
| [Test plan & acceptance book](docs/plan-de-tests.md) | Strategy, measured coverage, R-01…R-15 acceptance scenarios |
| [User stories](docs/user-stories.md) | 20 stories with acceptance criteria and traceability |
| [Technology watch](docs/veille-technologique.fr.md) | Watch plan, competitive analysis, transport comparison |
| [User guide](docs/guide-utilisateur.en.md) · [FR](docs/guide-utilisateur.fr.md) | Listener, broadcaster and administrator guides |
| [Training plan](docs/plan-formation.fr.md) | Audience-specific training, including accessibility adaptations |
| [Contributing](CONTRIBUTING.md) · [Security policy](SECURITY.md) | Conventions, quality gates, vulnerability reporting |
