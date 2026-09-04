# ADR 005 — API Contract: OpenAPI-first with Code Generation

**Status:** Accepted
**Date:** 2026-04-18

## Context

The backend and the mobile app are developed by the same person but evolve as two separate codebases,
each with its own release cadence: the backend deploys on every merge to `main`, the mobile app ships on
version tags. Any change to a request or response shape has to reach both sides, or the app breaks in
the field — where it is the hardest and slowest thing to fix.

Three approaches were considered: writing handlers by hand and documenting them afterwards; generating
documentation from code annotations; or writing the contract first and generating the server bindings
from it.

Hand-written handlers with after-the-fact documentation is the default path, and it fails predictably:
documentation drifts because nothing forces it to be updated, and drift is discovered by a client at
runtime. Code-first generation from annotations keeps the two closer, but the contract is then a
by-product of the implementation — it cannot be reviewed before the code exists, and it cannot be the
object of a deliberate design discussion.

## Decision

Make **`backend/api/openapi.yaml` the single source of truth**, and generate the server bindings from it
with **`oapi-codegen`** (config: `backend/oapi-codegen.yaml`, output: `internal/api/api.gen.go`).

- Handlers implement the generated `StrictServerInterface`: request and response types are produced by
  the generator, never hand-written.
- `internal/api/api.gen.go` is never edited manually.
- The CI pipeline runs `go generate ./internal/api/` and then compiles. If the contract and the
  implementation have diverged, **the build fails**.
- The two WebSocket endpoints are deliberately left **outside** the contract (see ADR 006).

## Consequences

**Positive**

- Contract drift is structurally impossible to merge, not merely discouraged. The guarantee is mechanical
  rather than procedural — and a rule that is not mechanised is not a rule.
- Adding an endpoint starts with a design decision about the contract rather than with an implementation
  detail, which is the right order.
- The generated types are the client's specification too: the mobile models are written against a
  document, not against observed behaviour.
- Request validation and routing boilerplate disappear from the handler layer.

**Negative, and accepted**

- The generator is a build dependency (pinned in `tools.go`), and its version is part of the build's
  reproducibility surface.
- Anything the OpenAPI specification cannot express — bidirectional streaming being the obvious case —
  needs a separate mechanism and separate documentation.
- Coverage measurement must exclude the generated file, otherwise the metric measures the generator
  rather than the project. The CI job strips `api.gen.go` from the coverage profile before applying the
  80 % threshold, with a comment explaining why.
