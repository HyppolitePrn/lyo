# ADR 002 — Flutter State Management: Riverpod

**Status:** ⚠️ **Superseded by [ADR 004](004-provider-over-riverpod.md)** (2026-07)  
**Date:** 2026-04-16

## Context

The mobile app needs to synchronise several async data streams: JWT auth state, audio playback state, feature flags, and live stream status. These states are independent but must be composed (e.g. the player depends on auth and feature flags). InheritedWidget/Provider is too manual; BLoC adds boilerplate for straightforward cases.

## Decision

Use [Riverpod](https://riverpod.dev). It provides compile-safe dependency injection, first-class async support (`AsyncNotifier`, `StreamProvider`), and testable providers without a `BuildContext`.

## Consequences

- All state lives in providers — no implicit widget coupling.
- Feature flag checks are a single `ref.watch(featureFlagsProvider)` call, usable anywhere.
- Providers are mockable in tests without dependency injection frameworks.

---

## Superseded — 2026-07

This decision was **reversed**. The application migrated from Riverpod to `provider` /
`ChangeNotifier` in commit `c6a9995` (PR #27). The reasoning behind the reversal is recorded in
[ADR 004 — Provider over Riverpod](004-provider-over-riverpod.md).

This record is kept, rather than deleted, because the history of a decision is part of its
documentation: what was expected from Riverpod, and why that expectation did not materialise, is
exactly what makes ADR 004 meaningful.
