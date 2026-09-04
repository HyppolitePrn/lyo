# ADR 006 — WebSocket Library: coder/websocket, and WebSockets Outside the OpenAPI Contract

**Status:** Accepted
**Date:** 2026-06-05

## Context

Live audio is carried over two WebSocket endpoints: `GET /streams/{id}/ingest` (broadcaster in) and
`GET /streams/{id}/listen` (listeners out). Two decisions had to be made together: which library, and
how these endpoints relate to the OpenAPI contract adopted in [ADR 005](005-openapi-first-codegen.md).

**Library.** `gorilla/websocket` is the historical default in the Go ecosystem and remains widely
deployed, but it went through a period of maintenance uncertainty, and its API predates
`context.Context`: cancellation and deadlines have to be bolted on through the underlying connection.
`coder/websocket` (formerly `nhooyr.io/websocket`) exposes a context-first API — `Read(ctx)`,
`Write(ctx, ...)`, `CloseRead(ctx)` — which matches how the rest of this codebase already handles
cancellation and timeouts.

**Contract placement.** OpenAPI 3 describes request/response HTTP semantics. It has no vocabulary for a
long-lived bidirectional connection: no way to express frame types, no way to express the lifecycle of a
stream, no way to generate anything useful for it. Forcing these endpoints into the specification would
have produced a document that describes them incorrectly.

## Decision

1. Use **`coder/websocket`** for both endpoints.
2. Keep both endpoints **outside `openapi.yaml`**, registered directly on the chi router in
   `cmd/server/main.go`, and document them in `docs/architecture/diagrammes-de-sequence.md`.
3. Authenticate WebSocket upgrades through a **`?token=` query parameter** fallback, because a WebSocket
   handshake — from a browser or from most mobile clients — cannot carry a custom `Authorization`
   header. The token is verified by exactly the same code path as a Bearer header.

## Consequences

**Positive**

- Cancellation composes with the rest of the system: `Hub.Done()` closing propagates through the same
  `ctx` that governs reads and writes, so ending a stream terminates its ingest loop without ad-hoc
  signalling.
- The contract stays honest: it describes what it can describe, and the real-time path is documented
  where it can be documented properly — with sequence diagrams that show frame flow, which is what a
  reader actually needs.
- The authentication fallback is a deliberate, documented exception rather than an undocumented quirk,
  and it is covered by tests (`TestIngest_AcceptsQueryParamToken`,
  `TestIngest_RejectsInvalidQueryToken`, and their `Listen` counterparts).

**Negative, and accepted**

- A token in a query string is more exposed than a header: it can appear in server access logs and in
  intermediary logs. Mitigations: short access-token lifetime (15 minutes), and TLS — which makes the
  query string invisible on the wire — is the top roadmap item precisely because this exposure exists.
- Two documentation surfaces to keep in sync (the generated contract and the hand-written WebSocket
  documentation) instead of one.
- A less common library than `gorilla/websocket`, so fewer third-party examples to copy from.
