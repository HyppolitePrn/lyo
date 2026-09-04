# ADR 013 — Per-address rate limiting and self-service account deletion

## Status

Accepted — 2026-09-04.

## Context

Two gaps stood between the platform and a public deployment.

**Nothing limited how often a client could call the API.** `POST /auth/login`
compares a bcrypt hash on every attempt, so an unthrottled endpoint is both a
credential-guessing oracle and a cheap way to saturate the CPU of a
single-VPS deployment. Nothing else in the stack filled the gap either: Caddy
caps request *bodies* (ADR 012), not request *rates*.

**A user could create an account but never remove one.** Beyond being a GDPR
obligation, the absence was structural: `tracks` cascades from `users`, so
even an operator deleting a row by hand would leave the audio objects those
rows pointed at orphaned in S3 — personal data with nothing left to identify
it by.

## Decision

### Rate limiting in the application, not at the edge

A token bucket per client address, in `pkg/middleware.RateLimit`, in two tiers:
a general quota (120/min) and a much tighter one for `/auth/*` (10/min).

- **In the app, not in Caddy.** The limiter has to distinguish `/auth/login`
  from browsing, and its 429 has to be the same `{code, message}` shape as
  every other error the mobile client parses. Expressing that in the Caddyfile
  would split one policy across two languages.
- **A token bucket, not a fixed window.** A fixed window lets a caller spend a
  full quota at the end of one window and again at the start of the next —
  twice the intended rate exactly when it matters. The bucket still permits a
  burst (a screen opening fires several calls at once) while holding the
  sustained rate to the average.
- **No new dependency.** `go-chi/httprate` would do this, but the whole
  mechanism is ~80 lines and the two-tier routing would need custom code
  anyway.
- **Keyed on `RemoteAddr`, which means it depends on `TRUSTED_PROXY`.** Behind
  Caddy the socket peer is always the proxy; chi's `RealIP` rewrites
  `RemoteAddr` from `X-Forwarded-For`, and main.go installs it only when the
  deployment declares a proxy in front. On a directly exposed socket that
  header is caller-controlled — trusting it would hand every request a fresh
  quota. `APP_ENV=production` already refuses to start without
  `TRUSTED_PROXY=true`, and now also without `RATE_LIMIT_ENABLED=true`.
- **`/health` and `/internal/*` are exempt.** They are infrastructure: the
  probe polls far above any per-client quota, and a throttled alert webhook
  would drop the incident it exists to report.

### `DELETE /users/me`, audio first

The account a caller can delete is the one their own JWT names — never a path
or body parameter — which keeps the route from becoming an account-deletion
primitive for other people's accounts. The usecase erases S3 objects **before**
the row:

- track rows cascade away with the user, and they are the only record of which
  objects belonged to the account;
- so deleting the row first, then failing on storage, leaves undeletable
  personal data behind — unrecoverable;
- doing it in this order can at worst leave the account intact after a storage
  failure, which the caller simply retries.

`user.ErrNotFound` answers 204: the caller wanted the account gone, and it is.

## Consequences

- Every response may now be a 429 with `Retry-After`; `ApiException` already
  carries the status code, so the client needs no new error type.
- The limiter holds one small struct per client address, capped at 50 000
  entries. Reaching the cap clears the table — buckets are soft state, and
  refilling from empty only grants tokens clients were about to earn.
- Buckets are per-process. A second backend replica would double the effective
  quota; making the limit exact across replicas means moving the state to
  Redis, which this single-VPS deployment does not justify.
- Account deletion is gated behind the `account_deletion` flag, so it can be
  turned off without a deploy.
- Deleting an account with many tracks issues one S3 delete per object inside a
  15 s request budget. If a broadcaster ever accumulates enough audio to
  exceed it, this moves to a background job.

See also [ADR 010](010-s3-presigned-upload.md) and
[ADR 012](012-tls-reverse-proxy.md).
