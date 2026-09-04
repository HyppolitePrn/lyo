# Security Policy

## Reporting a vulnerability

Report security issues **privately**, through GitHub's *Report a vulnerability* form on this repository's
Security tab, or by email to the maintainer. **Please do not open a public issue for a vulnerability.**

Please include: what the issue is, how to reproduce it, and what an attacker could achieve. A proof of
concept helps but is not required.

**Response commitments** (best effort, this is a single-maintainer project):

| Stage | Target |
|---|---|
| Acknowledgement | 72 hours |
| Initial assessment | 7 days |
| Fix or documented mitigation | 30 days for high severity |

Reporters are credited in the release notes unless they prefer otherwise.

## Supported versions

Only the latest released version is supported. Fixes ship on `main` and in the next tagged release.

## Security measures in place

| Area | Measure |
|---|---|
| Passwords | bcrypt hashing (`DefaultCost`); never stored or logged in cleartext |
| Tokens | JWT with HMAC; the signing method is **explicitly verified**, which blocks `alg: none` substitution. Access token 15 min, refresh token 7 days |
| Password reset | Tokens stored hashed, expiring, and single-use; identical response whether or not the account exists (no account enumeration) |
| Authorization | Ordinal role hierarchy plus per-resource ownership checks, verified independently of the role check |
| SQL injection | Exclusively parameterised queries through `pgx`; no SQL string concatenation |
| Secrets | Environment variables and GitHub Secrets only; the server fails to start if a required secret is missing, or if `JWT_SECRET` is shorter than 32 characters |
| Transport | TLS terminated by Caddy, certificates issued and renewed over ACME; HTTP redirects to HTTPS; HSTS, `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy` and `Permissions-Policy` set at the edge for every route ([ADR 012](docs/adr/012-tls-reverse-proxy.md)) |
| Network | The reverse proxy is the only container publishing a port in production — the API, PostgreSQL and Grafana are reachable only on the private Docker network. `/internal/*` returns 404 from outside |
| Privilege escalation | No API surface writes a role. Registration always creates a plain `user`; `PATCH /users/me` updates username and email only; token refresh re-reads the role from the database instead of carrying over the one in the token, so a demoted or deleted account cannot renew its former privileges. Promotion is an operator action on the database |
| Client identity | `X-Forwarded-For` is trusted only when `TRUSTED_PROXY` says a proxy terminates traffic — never on a directly exposed socket, where the header is caller-controlled |
| Denial of service by slow clients | Bounded per-listener buffers with chunk dropping: a slow listener cannot stall the broadcaster |
| Panics | `Recoverer` middleware; no stack trace is ever returned to a client |
| Supply chain | Images tagged by commit SHA; CI required before merge |

Full detail, including the OWASP Top 10 mapping and the ANSSI recommendations applied:
[`docs/architecture/securite.md`](docs/architecture/securite.md).

## Known gaps

These are **currently unmitigated** and are stated here deliberately rather than left for a reporter to
discover:

| Gap | Impact | Status |
|---|---|---|
| **No rate limiting** | `/auth/login`, `/auth/register` and `/auth/forgot-password` are open to brute force and mail spam | Roadmap |
| **No automated dependency scanning** | A vulnerable dependency could go unnoticed (`govulncheck`, `gosec`, Trivy, Dependabot not yet in CI) | Roadmap |
| **No account deletion endpoint** | GDPR erasure currently requires an administrator | Roadmap |
| Inter-container traffic is plaintext | Including `DATABASE_URL` (`sslmode=disable`). Acceptable only while every service shares one host's private bridge network | Accepted for the single-host deployment |

Reports about the gaps listed above are appreciated but already known; reports of anything else are very
welcome.
