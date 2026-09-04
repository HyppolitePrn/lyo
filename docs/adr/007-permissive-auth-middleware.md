# ADR 007 — Permissive Authentication Middleware with Handler-Level Authorization

**Status:** Accepted
**Date:** 2026-05-02

## Context

Lyo has genuinely public endpoints. Browsing live streams and listening to them must work with **no
account at all** — that is a product requirement (user story US-01), not a convenience: requiring
registration before a visitor can hear anything would remove the fastest path to understanding what the
service is.

At the same time, the very same endpoints behave better when a user *is* authenticated: the response can
be personalised, and ownership-dependent actions become available.

The conventional design — a middleware that rejects any request without a valid token — forces one of two
unpleasant shapes: either two parallel route trees (a public one and an authenticated one) with the
duplication and divergence that implies, or a growing list of path exceptions inside the middleware,
which is exactly the kind of security logic that rots quietly.

## Decision

Make `pkg/middleware.Authenticate` **permissive**:

- If a valid `Bearer` token is present, decode it and place the claims in the request context.
- If the header is absent, malformed, or the token is invalid, **call the next handler anyway**, without
  claims.
- The middleware never writes a response and never rejects.

Authorization then lives **in each handler**, in an explicit and readable order:

1. `middleware.ClaimsFromContext(ctx)` — absent claims where identity is required gives **401**.
2. `claims.Role.AtLeast(auth.RoleX)` — insufficient role gives **403**.
3. Feature flag check — disabled feature gives **503**.
4. Ownership check (`owner_id == claims.UserID`, or the caller is an admin) — otherwise **403/404**.

`middleware.RequireRole` exists for route groups where a blanket minimum applies.

## Consequences

**Positive**

- One route serves both anonymous and authenticated callers, with no duplication and no exception list.
- Each handler's access rules are visible **at the top of the handler**, next to the logic they protect.
  Answering "who can call this?" is a matter of reading the function, not of reconstructing a middleware
  chain and its exception table.
- The role check is ordinal (`AtLeast`), never an equality test, which structurally prevents the
  recurring "an admin is denied a broadcaster route" class of bug.
- Layering role, feature flag and ownership as independent checks is defence in depth: no single mistake
  opens a resource.

**Negative, and accepted — this is the real cost**

- **A forgotten check in a new handler leaves that endpoint open.** The middleware provides no safety
  net. This is the price of the design and it is stated plainly rather than glossed over.

  Three mitigations, in increasing order of strength:
  1. Every new endpoint starts from the contract (ADR 005), and the contract review is where the access
     rule is decided.
  2. The pattern is uniform across all handlers, so a missing check is visible in review by its absence
     from an otherwise repeated shape.
  3. **Tests are the actual guarantee.** `pkg/middleware` is at 100 % coverage, and the endpoint suite
     asserts the refusal paths explicitly — including a broadcaster being refused access to *another*
     broadcaster's stream (`TestIngest_RejectsForeignBroadcaster`), which is precisely the case a
     role-only middleware would have let through.

- An invalid token is indistinguishable from no token at the middleware level, so a client sending an
  expired token gets 401 from the handler rather than a specific "token expired" signal. The mobile
  client handles this by refreshing on any 401, which is the behaviour it would need regardless.
