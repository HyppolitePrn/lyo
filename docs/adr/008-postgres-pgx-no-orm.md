# ADR 008 — Persistence: PostgreSQL with pgx/v5, No ORM

**Status:** Accepted
**Date:** 2026-04-17

## Context

The data model is small (six tables) but it is not generic: it leans on PostgreSQL-specific features —
UUID arrays for favourites and playlist contents, an `ENUM` type for stream status, and a **partial
unique index** to enforce "one live stream per broadcaster" at the database level.

An ORM (GORM being the usual Go choice) trades SQL visibility for CRUD convenience. That trade is
favourable when a schema is large, uniform and mostly relational plumbing. It is unfavourable here, for
three reasons: the interesting parts of this schema are exactly the parts an ORM abstracts worst; the
queries are few and mostly simple; and the failure mode of an ORM — a hidden query, an unnoticed N+1 —
is invisible in code review and only surfaces under load.

## Decision

Use **`pgx/v5`** directly with hand-written SQL, and `pgxpool` for connection pooling. No ORM, no query
builder.

- Every query is **parameterised**. No SQL string is ever built by concatenation with user input.
- Repositories are the only place SQL appears; services never see it.
- Repositories are defined behind interfaces, so services are tested against doubles and the SQL layer is
  tested separately.
- Schema changes go through numbered `golang-migrate` files embedded via `embed.FS` and applied at
  startup, each with its `down` counterpart.

## Consequences

**Positive**

- Every query that reaches the database is visible in the source. Performance reasoning is possible by
  reading, not by profiling.
- PostgreSQL features are used as first-class design tools rather than worked around. The partial unique
  index in migration 000006 is the clearest example: a business rule that was previously enforced by
  `HasLive()` followed by `Create()` — with a TOCTOU window between them — became unbreakable, including
  across multiple backend instances, and the service code got *simpler* in the process.
- SQL injection is addressed by construction rather than by trusting a library's escaping.
- The interface boundary made the test suite possible: `internal/streaming`, `internal/user`,
  `internal/track` and `internal/playlist` all reach ≥ 96 % coverage with fake repositories, and the SQL
  itself is exercised separately.

**Negative, and accepted**

- More boilerplate: each repository method spells out its scan targets.
- Refactoring a table means editing every query that touches it, with the compiler helping only partially.
- The denormalised UUID arrays (favourites, playlist contents) buy single-query reads and free ordering
  semantics, but give up referential integrity on the array elements and do not scale past a few hundred
  entries. Documented in `docs/architecture/modele-de-donnees.md`; a join table is the known migration
  path if favourites ever need reverse indexing ("who favourited this track?").
