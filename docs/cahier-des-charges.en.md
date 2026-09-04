# Requirements Specification — Lyo / StreamPulse

**Version:** 1.0 · **Date:** 2026-09-03 · **Author:** Hyppolite Pernot
**French version:** [`cahier-des-charges.fr.md`](cahier-des-charges.fr.md)

> **Reading convention.** ✅ = delivered and verifiable in the repository · 🚧 = planned, tracked in the
> roadmap (§ 10). Nothing in this document is presented as achieved when it is not: the credibility of
> the whole documentation set depends on that discipline.

## Contents

1. [Context and objectives](#1-context-and-objectives)
2. [Feasibility study](#2-feasibility-study)
3. [Technology and competitive watch](#3-technology-and-competitive-watch)
4. [High-level architecture](#4-high-level-architecture)
5. [Functional specifications](#5-functional-specifications)
6. [Technical specifications](#6-technical-specifications)
7. [Accessibility and inclusion](#7-accessibility-and-inclusion)
8. [Sustainable computing](#8-sustainable-computing)
9. [Risk analysis](#9-risk-analysis)
10. [Roadmap and project governance](#10-roadmap-and-project-governance)
11. [Key performance indicators](#11-key-performance-indicators)
12. [Compliance: GDPR, ANSSI, ITIL](#12-compliance-gdpr-anssi-itil)
13. [Review and reflective analysis](#13-review-and-reflective-analysis)

---

## 1. Context and objectives

### 1.1 Context

Live audio is the only media format consumed **while doing something else** — driving, walking, working.
That property explains the vitality of the market (podcasts, online radio, social audio), but also its
main friction point: existing platforms are either too heavy for an individual broadcaster (server setup,
encoder, hardware) or too tolerant of latency to deliver a genuine live experience.

Two observations underpin the project:

- **Latency is a product characteristic, not a technical detail.** A 20-second delay makes any
  interaction with the audience impossible: the reaction arrives after the topic has moved on.
- **The broadcaster's entry barrier is technical, not editorial.** Going live from a phone in three taps
  is far from obvious today for a non-technical user.

### 1.2 Problem statement

> How can an individual broadcaster start a live audio stream from their phone within seconds and with no
> configuration, while guaranteeing listeners sub-two-second latency and a service that does not degrade
> globally when a single listener degrades individually?

### 1.3 Objectives

| Objective | Type | Success indicator |
|---|---|---|
| O1 — Go live from a phone with no configuration | Product | Fewer than 3 actions between opening the app and broadcasting |
| O2 — Perceived latency below 2 s | Technical | End-to-end measurement during acceptance testing |
| O3 — Fault isolation: a slow listener degrades only itself | Technical | Proven by an automated test |
| O4 — Bounded, predictable memory cost per listener | Technical | Linear, capped, measurable |
| O5 — Continuous and reversible delivery | Industrial | Every merge to `main` deploys automatically; one-command rollback |
| O6 — Observability separating technical errors from experience degradation | Operations | Dashboard splitting the two families 🚧 |
| O7 — Usable with assistive technology | Inclusion | Main journey validated with a screen reader 🚧 |

### 1.4 Target audience

| Audience | Need | Pilot volume |
|---|---|---|
| **Individual broadcaster** (creator, host, association, place of worship) | Broadcast without hardware or server skills | 10–50 accounts |
| **Listener** | Listen immediately, no mandatory account | 100–1,000 sessions |
| **Administrator** | Operate, monitor, switch off a failing feature | 1–2 accounts |

### 1.5 Scope

| In scope (MVP) | Out of scope, and why |
|---|---|
| One-way live audio broadcasting | Video — an order of magnitude more transcoding and bandwidth cost |
| Recorded tracks, playlists, favourites | Two-way conversation — would require WebRTC and SFU/TURN infrastructure |
| Accounts, roles, password reset | Licensed music catalogue — copyright is the top business risk |
| Android application | iOS — feasible with Flutter, but developer account and signing chain are outside the pilot budget |
| Observability and continuous deployment | Kubernetes — see § 2.1 |

---

## 2. Feasibility study

### 2.1 Technical feasibility

| Technical obstacle | Analysis | Conclusion |
|---|---|---|
| Fan out one stream to N listeners without cross-degradation | Goroutines and buffered channels allow lock-free fan-out; chunk dropping bounds degradation to the affected listener | **Cleared** — implemented and proven by test |
| Sub-two-second latency | WebSocket carrying AAC frames relayed without transcoding; no segmentation, no intermediate buffering | **Cleared** |
| Predictable memory cost | One goroutine (a few KB of stack) and one bounded channel per listener; the buffer dominates and is configurable (`STREAM_BUFFER_SIZE`) | **Cleared** — linear and capped |
| Horizontal scalability | Hub state lives **in process memory**: two instances do not share listeners | **Not cleared — accepted limitation.** A message bus (Redis Pub/Sub, NATS) with per-stream routing is the prerequisite. Tracked in the roadmap. |
| One codebase, two platforms | Flutter provides a shared base; microphone capture and background playback rely on mature packages | **Cleared** for Android |

> **Why not Kubernetes.** Orchestration is not this system's limiting factor: horizontal scaling is
> blocked by the Hub's in-memory state, not by the ability to start replicas. Adding Kubernetes would
> introduce a control plane to operate without removing the real constraint. The prerequisite for
> Kubernetes here is not orchestration — it is making the Hub distributable. Docker Compose on a single
> VPS is therefore the proportionate tool.

### 2.2 Organisational feasibility

| Resource | Availability | Consequence |
|---|---|---|
| Team | **1 developer** | Main risk factor (*bus factor* of 1). Mitigated by systematic decision records, maximal delivery automation, and strict naming and commit conventions. |
| Duration | April to September 2026, part-time | Tight MVP scope, value-driven prioritisation |
| Skills | Go, Flutter, Docker, CI/CD acquired; observability and network security being acquired | These two areas hold most of the identified debt |
| Tooling | GitHub (repository, Actions, Releases, Projects), shared VPS | Zero marginal cost for the pilot |

### 2.3 Financial feasibility and return on investment

**Infrastructure costs** (pilot assumption: 50 broadcasters, 500 peak concurrent listeners)

| Item | Assumption | Monthly estimate |
|---|---|---|
| VPS (4 vCPU, 8 GB RAM, 200 Mbit/s) | Backend + PostgreSQL + observability | €20–40 |
| Object storage | 100 GB of recorded tracks | €2–5 |
| **Egress bandwidth** | 500 listeners × 128 kbit/s × 3 h/day ≈ **2.6 TB/month** | €0 if bundled, **€20–250** otherwise |
| Domain + TLS certificate | Let's Encrypt free | ~€1 |
| CI/CD, image registry | GitHub Actions, public repository | €0 |
| **Total** | | **≈ €25–300 / month** |

**The dominant cost is egress bandwidth, not compute.** This finding drives architecture directly:
optimising CPU would yield nothing, whereas edge distribution (CDN) or bitrate reduction are the only
real levers under growth. It also justifies excluding video, which would multiply this item by 10 to 50.

**Development cost (valuation; the project was carried out as coursework)**

| Item | Estimate |
|---|---|
| Backend design and development | ≈ 200 h |
| Mobile development | ≈ 150 h |
| Industrialisation (CI/CD, Docker, observability) | ≈ 60 h |
| Testing and documentation | ≈ 90 h |
| **Total** | **≈ 500 h** — €20,000–25,000 at an average developer cost |

**Return on investment.** The pilot has no revenue model; the return is educational and asset-building.
Under a commercial hypothesis:

| Scenario | Assumption | Break-even |
|---|---|---|
| Broadcaster subscription | €5 / month / broadcaster | **6 to 60 paying broadcasters** cover infrastructure; ≈ 350 over 12 months amortise development |
| Marginal cost of one extra listener | ≈ 0.17 GB per listening hour | **Near zero in compute, non-zero in bandwidth** — hence a model that must charge the broadcaster, never the listener |

**Conclusion.** The project is technically feasible within its MVP scope, with a documented horizontal
scaling limitation. It is financially feasible at pilot scale. The dominant risk is organisational: a
single person.

---

## 3. Technology and competitive watch

Covered in full in **[`veille-technologique.fr.md`](veille-technologique.fr.md)** (French, with an English
summary): a seven-axis watch plan with sources, tooling and a three-question triage grid; a competitive
analysis (Twitch, Spotify Live, Clubhouse, Mixlr, Icecast/Shoutcast); a transport architecture comparison
(WebSocket / HLS / WebRTC / RTMP); a decision log; and four documented cases where the watch changed
actual code.

**Founding trade-off.** WebSocket is selected: latency is the core value proposition (ruling out HLS),
the stream is unidirectional (making WebRTC's complexity unprofitable), and the protocol traverses
proxies and firewalls on port 443. The accepted trade-off is recorded openly: no CDN distribution, so a
message bus is a prerequisite for any horizontal scaling.

---

## 4. High-level architecture

Full detail, diagrams and text alternatives in **[`architecture/`](architecture/)**.

| Structural principle | Concrete translation |
|---|---|
| Contract first | `backend/api/openapi.yaml` is the source of truth; server types are **generated**, never hand-written; CI regenerates and compiles, making contract/code drift impossible to merge |
| Layered separation | presentation → domain → infrastructure, strictly downward dependencies |
| Real time does not go through the REST contract | Both WebSocket endpoints sit outside OpenAPI, a specification that does not model persistent bidirectional streams; they are mounted directly on the router and documented separately |
| Local degradation, never global | Bounded per-listener buffer plus chunk dropping |
| Critical business rules enforced by the database | Partial unique index for "one live stream per broadcaster" |
| Inactive code is deployable | Database-backed feature flags: shipping is not enabling |
| No hardcoded values | 100 % environment-variable configuration, fail-fast startup when a secret is missing |

---

## 5. Functional specifications

Detailed requirements are formalised as user stories with acceptance criteria in
**[`user-stories.md`](user-stories.md)** (20 stories, 5 epics, traceability to endpoints, acceptance
scenarios and feature flags).

### 5.1 Permission matrix

| Capability | `anonymous` | `user` | `broadcaster` | `admin` |
|---|:---:|:---:|:---:|:---:|
| Browse and listen to public live streams | ✅ | ✅ | ✅ | ✅ |
| Listen to published tracks | ✅ | ✅ | ✅ | ✅ |
| Create an account, manage profile | — | ✅ | ✅ | ✅ |
| Favourites | — | ✅ | ✅ | ✅ |
| Playlists | — | ✅ | ✅ | ✅ |
| Start / end a live stream | — | — | ✅ | ✅ |
| Publish / delete a track | — | — | ✅ (own) | ✅ (any) |
| Manage feature flags | — | — | — | 🚧 |
| Manage accounts, view global metrics | — | — | — | 🚧 |

The hierarchy is **ordinal**: checks always read "at least this role", never "exactly this role", so an
administrator mechanically inherits every lower privilege.

### 5.2 Target performance levels

| Indicator | Target | Measurement | Status |
|---|---|---|---|
| End-to-end audio latency | **< 2 s** | Acceptance test R-06 | ✅ met under nominal conditions |
| Time to first sound | < 3 s | R-06 | ✅ |
| Concurrent listeners per instance | **≥ 500** | `STREAM_MAX_LISTENERS` default; to be confirmed by load test | 🚧 |
| Memory footprint per listener | ≤ 70 KB | `pprof` profile | 🚧 |
| API latency (p95) | < 300 ms | HTTP metrics | 🚧 |
| Availability | 99 % during pilot | `/health` probe | 🚧 |
| Dropped-chunk rate | < 1 % of broadcast chunks | `lyo_chunks_dropped_total` | 🚧 |
| UI smoothness | 60 FPS while listening | Flutter DevTools profile capture | 🚧 |

### 5.3 Business rules

| # | Rule | Enforcement point |
|---|---|---|
| BR-01 | A broadcaster may have only one active live stream | PostgreSQL partial unique index |
| BR-02 | Only the stream owner may push audio to it | Ingest handler check, tested |
| BR-03 | An ended stream accepts neither ingest nor listening | Status check, tested |
| BR-04 | A track may be deleted only by its owner or an administrator | Ownership check in handler |
| BR-05 | A playlist is private by default | `is_public` default `false` |
| BR-06 | Playlist track order is the playback queue | Ordered `track_ids` array |
| BR-07 | A reset token is single-use and expiring | `used_at` + `expires_at` |
| BR-08 | A disabled feature returns 503, never 404 or 500 | Flag check at handler entry |
| BR-09 | A saturated listener loses its own chunks, never anyone else's | `default` branch of the `select` in `Hub.Broadcast` |

---

## 6. Technical specifications

### 6.1 Stack and rationale

| Layer | Choice | Rejected alternative | Rationale |
|---|---|---|---|
| Backend language | **Go 1.26** | Node.js, Python | Native concurrency (a goroutine starts at ~2 KB of stack against ~1 MB for an OS thread), no global interpreter lock, a few-megabyte static binary enabling a minimal alpine image, low-latency garbage collector suited to real time |
| HTTP router | **chi** | Gin, Echo, bare `net/http` | `http.Handler`-compatible: no vendor lock-in, the whole standard middleware ecosystem works (ADR 001) |
| API contract | **OpenAPI + oapi-codegen** | Hand-written handlers | Single versioned contract, generated types: silent drift between mobile and backend becomes impossible (ADR 005) |
| Real-time transport | **WebSocket (`coder/websocket`)** | HLS, WebRTC, RTMP | See § 3, ADR 003 and ADR 006 |
| Database | **PostgreSQL 17 + pgx/v5** | ORM (GORM) | Explicit SQL, no hidden or N+1 queries, efficient pooling, and genuine use of advanced PostgreSQL types (arrays, ENUM, partial indexes) (ADR 008) |
| Migrations | **golang-migrate + `embed.FS`** | Manual scripts | Embedded in the binary, applied at startup, each one reversible |
| Audio storage | **S3 / MinIO with presigned URLs** | Upload through the API | The backend never carries audio bytes (ADR 010) |
| Mobile | **Flutter 3.41 + `provider`** | Native Swift/Kotlin, Riverpod | Single codebase; `provider` adopted after field experience (ADR 004) |
| Mobile audio | `just_audio`, `audio_service`, `record` | — | Off-UI-thread decoding, background playback, AAC-LC capture |
| Observability | **slog JSON ✅ + OpenTelemetry 🚧** | Proprietary format | Vendor-neutral standard (ADR 009) |
| Containers | Multi-stage alpine Docker | Full base image | Minimal final image, reduced attack surface |
| CI/CD | GitHub Actions | Jenkins, GitLab CI | Repository-native, encrypted secrets, free for public repositories |

### 6.2 Technical constraints

| Constraint | Origin | Handling |
|---|---|---|
| A WebSocket handshake carries no custom header | RFC 6455 / browser API | `?token=` query-parameter auth fallback with identical token verification — tested |
| Hub state lives in process memory | Deliberate design choice | Single instance; message bus required before replication |
| Android blocks cleartext HTTP from API 28 | Platform | **Makes TLS mandatory for real distribution** — Caddy reverse proxy with automatic ACME certificates; cleartext is allowed in debug/profile builds only |
| Egress bandwidth saturates before CPU | Cost study result (§ 2.3) | No transcoding: AAC frames are relayed as-is |
| Per-request timeouts are mandatory | Internal policy | `context.WithTimeout` at handler entry, 503 on deadline |

---

## 7. Accessibility and inclusion

**Commitment.** The application targets **WCAG 2.1 AA**, the reference standard RGAA aligns with.
Accessibility is treated as a functional requirement (user story US-18, acceptance scenario R-14), not as
a finishing touch.

| Requirement | Product translation | Status |
|---|---|---|
| Screen reader support | Accessible label on every icon-only control (play/pause, favourite, microphone, mini player) | 🚧 |
| State-change announcements | Play/pause transitions and live connection are announced | 🚧 |
| Contrast | ≥ 4.5:1 on text, verified against design tokens | 🚧 |
| Touch targets | ≥ 48 × 48 dp | 🚧 |
| Text scaling | Usable at 200 %, no truncation or overlap | 🚧 |
| Never colour alone | The "live" indicator pairs a dot **with** a text label | 🚧 |
| Consistent navigation | Stable screen structure, predictable back behaviour | ✅ |
| Automated proof | Flutter `meetsGuideline` test (contrast, tap targets) run in CI | 🚧 |

**Documentation accessibility** — already implemented:

- **Every diagram is paired with a full text alternative**, making the architecture documentation usable
  with a screen reader.
- Diagrams are written in Mermaid, therefore in **text**: readable, indexable and diffable, unlike an
  exported image.
- Strict heading hierarchy with no skipped levels; tables with explicit headers.
- No information conveyed by colour alone: statuses are spelled out as well as symbolised.
- Documentation available in **French and English**.

---

## 8. Sustainable computing

### 8.1 Architectural frugality

| Lever | Effect | Status |
|---|---|---|
| **Stream mutualisation** | One incoming stream is fanned out to N listeners: **a single** ingest and a single processing path instead of N independent pipelines. This is the architecture's largest frugality gain. | ✅ |
| **No transcoding** | AAC frames are relayed as-is. Adaptive transcoding is the dominant CPU cost of streaming platforms; we do not pay it. | ✅ by design |
| **Go rather than an interpreted runtime** | Memory and CPU per request an order of magnitude below an equivalent Node.js or Python stack | ✅ |
| **Multi-stage alpine image** | A few tens of megabytes: less storage, less transfer per deployment, smaller attack surface | ✅ |
| **One VPS rather than a cluster** | No permanently running Kubernetes control plane for a single-process service | ✅ |
| **Bounded buffers** | Memory consumption is capped and predictable; no uncontrolled growth under load | ✅ |
| **Direct-to-object-storage upload** | Audio bytes are not copied twice through the backend | ✅ |
| **Resource teardown at stream end** | `Hub.Close()` releases goroutines and channels; nothing outlives its purpose | ✅ |

### 8.2 Device-side frugality

Audio only (one to two orders of magnitude below video in energy and data), a fixed 128 kbit/s bitrate
(≈ 57 MB per listening hour), off-UI-thread decoding, and targeted widget rebuilds. Offline playlist
caching would further avoid re-downloads 🚧.

### 8.3 What is missing

No real carbon or power measurement has been performed: the claims above are argued by design, not
measured. There is also no data retention policy — ended streams and tracks accumulate indefinitely,
which affects both storage and GDPR exposure (§ 12).

---

## 9. Risk analysis

Probability (P) and Impact (I) rated 1 (low) to 4 (critical); Criticality = P × I.

| # | Risk | Type | P | I | Crit. | Mitigation | Status |
|---|---|---|:-:|:-:|:-:|---|---|
| **R1** | **Unencrypted traffic**: JWTs travel in cleartext, and Android blocks cleartext HTTP from API 28 | Technical / security | 4 | 4 | **16** | Caddy reverse proxy with automatic Let's Encrypt TLS, `wss://`, port 8080 no longer published, release APK denied cleartext | ✅ mitigated ([ADR 012](adr/012-tls-reverse-proxy.md)) |
| **R2** | **Copyright** on user-broadcast audio | Legal / business | 3 | 4 | **12** | Terms of use assigning responsibility to the broadcaster, notice-and-takedown procedure, broadcast logging, no platform-provided licensed catalogue | 🚧 |
| **R3** | **Bus factor of 1** | Human | 4 | 3 | **12** | Systematic documentation (ADRs, this specification, guides), full delivery automation, strict conventions | ✅ mitigated |
| **R4** | **No horizontal scalability**: Hub state is in memory | Technical | 3 | 4 | **12** | Documented and accepted; Redis Pub/Sub or NATS with per-stream routing; vertical scaling only for now | 🚧 documented |
| **R5** | **Brute force and spam** on authentication endpoints | Security | 3 | 3 | **9** | Rate limiting on login, registration and password reset | 🚧 |
| **R6** | **Runaway bandwidth cost** on success | Financial | 2 | 4 | **8** | Cost is modelled (§ 2.3); listener cap, egress alerting, broadcaster-side revenue model | ✅ partial |
| **R7** | **Undetected dependency vulnerability** | Security | 3 | 3 | **9** | `govulncheck`, `gosec`, Trivy and Dependabot in CI | 🚧 |
| **R8** | **Undetected bad deployment** (container crash-looping unnoticed) | Operations | 3 | 3 | **9** | Post-deploy `/health` smoke test, failing job, rollback to the previous SHA tag | 🚧 |
| **R9** | **Data loss** on VPS failure | Operations | 2 | 4 | **8** | Persistent volume ✅; automated backup and tested restore 🚧 | 🚧 |
| **R10** | **GDPR non-compliance**: no account deletion, no retention policy | Regulatory | 3 | 3 | **9** | `DELETE /users/me`, expired-token purge, privacy policy (§ 12) | 🚧 |
| **R11** | **Application unusable with assistive technology** | Regulatory / inclusion | 3 | 3 | **9** | Accessible labels, contrast, tap targets, automated test (§ 7) | 🚧 |
| **R12** | **Documentation/code drift** | Quality | 2 | 3 | **6** | OpenAPI regenerated and compiled in CI; ADRs updated or superseded on every reversal; explicit ✅/🚧 statuses throughout | ✅ mitigated |
| **R13** | **Single-vendor dependency** (VPS, GHCR) | Strategic | 2 | 2 | **4** | Everything is containerised and described in Compose files: migration is mechanical | ✅ |
| **R14** | **Object storage outage** making tracks unavailable | Technical | 2 | 3 | **6** | Live streaming keeps working when storage is down: partial, not total, degradation | ✅ by design |

---

## 10. Roadmap and project governance

### 10.1 Method

**Agile / DevOps adapted to a one-person team**: one branch per unit of work named `type/description`,
conventional commits (**66 commits**, 29 `feat`, 7 `fix`), one pull request per unit of work with CI as
the systematic automated reviewer (**35 pull requests**), semantic version tags driving automated GitHub
Releases (**v1.0.0 to v1.1.0**), and architecture decisions recorded as they are made.

### 10.2 Delivered

| Milestone | Period | Content | State |
|---|---|---|---|
| M1 — Foundation | April 2026 | Bootstrap, database schema, OpenAPI contract, backend and mobile CI | ✅ |
| M2 — Authentication | April–May 2026 | JWT, roles, registration and login, mobile screens | ✅ |
| M3 — Live streaming | May–June 2026 | Hub, fan-out, ingest/listen WebSockets, mobile broadcaster and player | ✅ |
| M4 — Continuous deployment | June 2026 | GHCR, VPS rollout, signed mobile releases | ✅ |
| M5 — Content | July–August 2026 | Tracks, presigned S3 upload, playlists, favourites, profile, password reset | ✅ |
| M6 — Quality | September 2026 | 293 tests, ≥ 80 % coverage on every domain package, blocking CI gate (PR #35) | ✅ |
| M7 — Documentation | September 2026 | Specification, architecture, user stories, test plan, guides, ADRs | ✅ this batch |

### 10.3 Remaining work, by value

| Priority | Batch | Content | Risk / criterion cleared | Estimate |
|:-:|---|---|---|---|
| ~~**1**~~ | ~~**TLS and transport security**~~ | Caddy reverse proxy, `wss://`, unpublished port 8080, CORS moved to an environment variable | R1 | ✅ delivered |
| **2** | **OpenTelemetry instrumentation** | Traces down to the database (`otelpgx`), technical **and business** metrics (active listeners, dropped chunks, broadcast bytes, abrupt disconnects), `traceparent` propagation from the mobile app | O6 | 3 d |
| **3** | **Dashboards and alerting** | Tempo trace backend, provisioned Grafana dashboard (listeners, throughput, error rate, latency), alert rules, log shipping to Loki with trace correlation | O6, R8 | 2 d |
| **4** | **Accessibility** | Accessible labels, contrast, tap targets, state announcements, automated test | O7, R11 | 2 d |
| **5** | **Supply-chain security** | `govulncheck`, `gosec`, Trivy, Dependabot, rate limiting | R5, R7 | 1.5 d |
| **6** | **Administration** | Feature-flag handler implementation, account management, global metrics, mobile admin screen, **API-driven feature flags on mobile** | US-15 to US-17 | 3 d |
| **7** | **GDPR compliance** | `DELETE /users/me`, data export, expired-token purge, privacy policy | R10 | 1.5 d |
| **8** | **Deployment reliability** | Post-deploy smoke test, automatic rollback, staging environment | R8 | 1.5 d |
| **9** | **Player polish** | Volume control, audio interruption handling, playlist queue with reordering | US-06, US-08, US-10 | 2 d |
| **10** | **Load measurement** | Fan-out benchmarks at 10/100/1000 listeners, `pprof` profiles, load test, 60 FPS capture | O4, § 5.2 | 2 d |
| 11 | WebSocket chat *(bonus)* | A second, text-oriented Hub reusing the same pattern | Extra feature | 3 d |
| 12 | Recommendations *(bonus)* | Listening history → co-occurrence recommendations: collection, storage, analysis | Extra feature | 4 d |

### 10.4 Governance tooling

GitHub is the single tool: issues for units of work, Projects (Kanban: *To do* → *In progress* → *In
review* → *Done*) for flow visualisation, numbered pull requests for decision traceability, and Releases
for version communication. **The Git history is itself the evidence of governance**: every feature can be
traced to its branch, pull request, review and deployment.

---

## 11. Key performance indicators

### 11.1 Project KPIs

| Indicator | Definition | Current | Target |
|---|---|---|---|
| Merged pull requests | Units of work delivered | 35 | — |
| Commit type distribution | Share of `feat` | 29 / 66 ≈ 44 % | ≥ 40 % |
| Fix-to-feature ratio | `fix` ÷ `feat` | 7 / 29 ≈ **0.24** | < 0.3 — a low ratio indicates stable delivered features |
| Lead time | Merge to production | < 10 min (automated) | < 15 min |
| CI success rate | Green builds ÷ total | to be measured | > 90 % |
| Backend test coverage | Excluding generated code | **≥ 80 % on every domain package** | ≥ 80 %, **blocking** |
| Automated tests | Backend test functions | **293** | growing |
| Documented decisions | ADRs written | 10 | one per structural decision |
| Deployment frequency | Production deployments per month | ≈ 6 | ≥ 4 |

### 11.2 Product and operations KPIs

Active listeners, live streams, **dropped-chunk rate**, abrupt disconnects, egress throughput (which
drives cost directly), average broadcast duration, technical error rate, API latency percentiles,
availability, and mobile crash-free rate. All are 🚧 pending the observability batch.

> **The essential distinction.** A 500 response is a **technical failure**: code broke. An abrupt
> disconnect or a dropped chunk is an **experience degradation**: the system behaved as designed, but the
> user suffered. The two families call for different decisions — a fix versus a capacity change — and must
> therefore be visualised separately.

---

## 12. Compliance: GDPR, ANSSI, ITIL

### 12.1 GDPR

| Data | Purpose | Legal basis | Retention |
|---|---|---|---|
| Email address | Identification, password reset | Contract performance | Account lifetime |
| Username | Public display | Contract performance | Account lifetime |
| Password hash (bcrypt) | Authentication | Contract performance | Account lifetime |
| Role | Authorisation | Contract performance | Account lifetime |
| Favourites, playlists | Service personalisation | Contract performance | Account lifetime |
| Published content | The service itself | Contract performance | Until deleted by its author |
| Reset tokens | Recovery flow security | Legitimate interest | Short expiry, then purge 🚧 |

**Minimisation.** No real name, date of birth, location data, advertising tracker or third-party
identifier. Collection is limited to what the service strictly requires — a design decision, not a
by-product.

**Data subject rights.** Access ✅ (`GET /users/me`), rectification ✅ (`PATCH /users/me`), **erasure 🚧**
(`DELETE /users/me`, made atomic and complete by the cascading foreign keys), portability 🚧 (JSON
export), objection/restriction 🚧, information 🚧 (privacy policy to be written and surfaced in the app).

**Processing security.** Bcrypt password hashing; hashed, expiring, single-use reset tokens; database not
exposed publicly; secrets outside source control; passwords never logged; transport encrypted end to end
by the TLS reverse proxy, the live audio stream included.

**Processors** (Art. 28): VPS host, object storage provider, SMTP provider, GitHub for the build chain.

**Breach procedure** 🚧: detection through alerting, qualification, CNIL notification within 72 hours,
notification of data subjects when the risk is high.

### 12.2 ANSSI recommendations applied

Secrets outside source control ✅ · least privilege (ordinal roles, `packages: write`-scoped CI token,
unpublished database port) ✅ · network segmentation ✅ · event logging with correlation identifiers ✅ ·
defence in depth (role **and** ownership **and** feature flag checked independently, plus a
database-level business constraint) ✅ · traffic encryption 🚧 · security maintenance through automated
dependency analysis 🚧 · backup and tested restore 🚧.

### 12.3 ITIL-inspired practices

| ITIL process | Project transposition |
|---|---|
| Change management | Every change goes through a pull request, CI, and a traced automated deployment |
| Release management | Semantic version tags, GitHub Releases, SHA-tagged images |
| Incident management | Alerting 🚧 → dashboard and log diagnosis → fix or feature-flag kill switch → post-mortem issue |
| Problem management | Every fixed defect ships with a **regression test**: the root cause is locked, not just the symptom |
| Configuration management | Fully environment-variable driven, documented in `.env.example` |
| Continuity management | Rollback by SHA-tagged image, reversible migrations, functional kill switch by flag |

---

## 13. Review and reflective analysis

### 13.1 Planned versus delivered

| Item | Planned | Delivered | Gap analysis |
|---|---|---|---|
| Live streaming | MVP | ✅ Delivered, 96.6 % covered | On target — the technical core was tackled first, which was the right order |
| Content features | MVP | ✅ Delivered | On target |
| CI/CD | MVP | ✅ Delivered in April | **Ahead of plan** — early automation paid compound interest across the project |
| Observability | MVP | 🚧 Infrastructure deployed, **application not instrumented** | **The largest gap.** See § 13.3 |
| Testing | ≥ 80 % | ✅ Reached late (September) | **Schedule gap.** See § 13.3 |
| Transport security (TLS) | Implicit | ✅ Delivered late (September) | Underestimated: filed as "infrastructure" when it is a blocking product requirement on Android |
| Accessibility | Implicit | 🚧 Not done | Not explicitly planned, therefore not done — the project's main lesson |
| Administration | MVP | 🚧 Contract defined, implementation missing | Wrongly deprioritised behind core features: it is also an operations capability |

### 13.2 Three deliberate course corrections

**① Riverpod → Provider migration** (commit `c6a9995`). Riverpod had been chosen on reputation and
recorded in ADR 002. In practice the app's state proved simple and feature-local: Riverpod's power went
unused while its conventions taxed every addition. What matters is not the migration but how it was
documented — ADR 002 was marked **Superseded** and ADR 004 explains the reversal. An ADR documenting an
owned change of mind is worth more than an ADR that has quietly become false.

**② Moving the "one live stream" rule into the database** (migration 000006). The rule lived in code as
`HasLive()` followed by `Create()`; two concurrent requests could slip between them. Rather than adding
an application lock, the constraint became a PostgreSQL partial unique index: the rule became
unbreakable, even across instances, and the code got simpler.

**③ Keeping playback alive when leaving the player screen** (commit `fb25144`). A defect found in use
revealed coupling between UI lifecycle and playback lifecycle. Decoupling them fixed the bug and directly
enabled the persistent mini player — a defect that produced a feature.

### 13.3 What I would do differently

**Instrument observability alongside the first feature, not afterwards.** Deploying the observability
stack early created an illusion of progress: containers were running and data sources were provisioned,
but the application emitted nothing. The lesson is that observability infrastructure without application
instrumentation is worthless, and that instrumenting a handler as you write it is far cheaper than
instrumenting thirty-six of them later. The right move would have been to treat the first end-to-end
trace as an acceptance criterion of the first endpoint.

**Write tests in the same pull request as the code, from day one.** Coverage went from under 15 % to over
80 % in a single late batch (PR #35). The outcome is good, the method is not: for months every refactor
happened without a safety net, and some tests had to be written by rediscovering the code's intent. The
blocking coverage gate added by that same PR is exactly the mechanism that should have existed from the
first commit — a rule that is not mechanised is not a rule.

**Treat accessibility as a functional requirement, not a finishing touch.** Not one line of accessibility
code was written because no user story demanded it. That is a clean demonstration that an unwritten
requirement is an undelivered requirement. The correction is structural: accessibility is now user story
US-18 with verifiable acceptance criteria, acceptance scenario R-14, and an automated test — treated like
any other feature.

**Put TLS in place before distributing the first APK.** Filing transport encryption under
"infrastructure" pushed it to the bottom of the list. It is in fact a blocking product requirement:
Android refuses cleartext traffic from API 28, so an APK pointing at an HTTP URL is not merely less
secure, it is functionally dead. "First production deployment" should have had TLS in its exit criteria.

### 13.4 What I would do again

Contract-first OpenAPI (zero mobile/backend drift in six months, made structurally impossible by the
generator and CI); CI/CD in week one (every subsequent day benefited, and deployment was never a dreaded
event); database-backed feature flags (shipping inactive code then enabling it without redeployment is a
kill switch no rollback procedure can match for speed); the chunk-drop policy (the architectural decision
I am most satisfied with — simple, bounding degradation to whoever suffers it, and proven by test); and
recording decisions when they are made rather than reconstructing them afterwards, which would have
produced rationalisations rather than records.
