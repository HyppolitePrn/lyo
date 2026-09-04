# ADR 012 — TLS Terminated at a Reverse Proxy, Not in the Application

**Status:** Accepted — implemented 2026-09-03
**Date:** 2026-09-03

## Context

Until now the production stack published three plaintext ports: the API on 8080, Grafana on 3000, and
whatever a `docker compose` override happened to add. Every one of them carried credentials. The mobile
client sends a `Bearer` JWT on every request; the WebSocket endpoints fall back to `?token=` in the query
string because a browser or mobile WebSocket cannot set headers ([ADR 006](006-coder-websocket.md)) — over
plain HTTP that token is readable by anything on the path, and it is a token that names a **role**. An
intercepted broadcaster token is a hijacked stream; an intercepted admin token is the supervision surface.
Transport encryption is therefore not a checkbox here, it is what makes the whole role model meaningful.

Three ways to get there:

1. **TLS in the Go server** (`ListenAndServeTLS`). No extra container, but the application then owns
   certificate loading, renewal, the HTTP→HTTPS redirect, cipher policy and security headers — none of
   which is application logic, all of which is deployment policy that would need a redeploy to change.
2. **nginx + certbot.** The conventional answer, and a well-understood one. It costs two moving parts (a
   renewal cron plus a reload hook), a hand-written TLS block that ages badly, and a config that must be
   kept in step with certbot's output paths.
3. **Caddy.** Obtains and renews certificates over ACME on its own, redirects HTTP to HTTPS by default,
   and ships a modern TLS profile with no cipher list to maintain. WebSocket upgrades are proxied without
   configuration.

## Decision

**Caddy terminates TLS and is the only container with published ports.** Everything else — the API,
Postgres, Grafana, the collector, Prometheus, Loki, Tempo — is reachable only on the Compose network.
There is no plaintext path into the platform, so a misconfigured client cannot silently downgrade.

Consequences of that single choice, all in `docker/Caddyfile`:

- **Security headers live at the edge.** HSTS, `X-Content-Type-Options`, `X-Frame-Options`,
  `Referrer-Policy`, `Permissions-Policy` and `-Server` are set once for every route, including the ones
  that are not in the OpenAPI contract. Setting them in the application would mean setting them in three
  places (the generated handlers, the WebSocket handlers, the alert webhook) and forgetting one.
- **`/internal/*` returns 404 from outside.** The Grafana alert webhook authenticates with a shared
  secret, but it is infrastructure-to-infrastructure and has no reason to be reachable publicly. Grafana
  still calls it directly over the Compose network.
- **WebSocket upgrades are matched before any body policy.** The API's 2 MB request-body cap must never
  apply to a hijacked connection carrying a live broadcast. Audio bytes do not transit the API at all
  ([ADR 010](010-s3-presigned-upload.md)), so 2 MB is generous for the JSON that remains.
- **Grafana moves to `grafana.<domain>`** with `GF_SERVER_ROOT_URL` and secure, `SameSite=strict` session
  cookies — it stops being an open port and becomes a TLS host like any other.

The application learns exactly one thing about this arrangement: `TRUSTED_PROXY`. When set, chi's `RealIP`
middleware reads the client address from `X-Forwarded-For`. It is **false by default** and deliberately
not inferred: that header is caller-controlled on a directly reachable socket, so trusting it
unconditionally would let anyone write their own address into the audit log. `APP_ENV=production` refuses
to start without it — which is also how the configuration refuses to run production outside the proxy.

## Consequences

**What this buys.** One place to reason about transport security. Certificate renewal is not an operational
task. The blast radius of a compromised container is smaller: nothing but Caddy listens on a public
interface.

**What it costs.** A dependency on Caddy's ACME behaviour and one more image to keep current. The
`caddy_data` volume is now stateful and load-bearing — losing it means re-requesting every certificate and
meeting Let's Encrypt's rate limits at the worst possible moment. Both DNS records (`<domain>` and
`grafana.<domain>`) must resolve to the host **before** the first start, or the ACME challenge fails and
Caddy serves nothing.

**What it does not cover, stated plainly.**

- Traffic between containers is still plaintext, `DATABASE_URL` included (`sslmode=disable`). That is
  acceptable only because the Compose network is a single host's private bridge; it stops being acceptable
  the moment any service moves to another machine.
- Caddy without a plugin has **no rate limiting**. Nothing here throttles credential stuffing against
  `POST /auth/login` or `POST /auth/forgot-password`. That is a known, unmitigated gap, not an oversight.
- Certificate issuance requires reachable inbound 80/443. Behind a NAT without port forwarding the stack
  starts but serves an untrusted internal certificate.
