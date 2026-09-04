# ADR 004 — Flutter State Management: Provider over Riverpod

**Status:** Accepted
**Date:** 2026-07-20
**Supersedes:** [ADR 002](002-state-management-riverpod.md)

## Context

[ADR 002](002-state-management-riverpod.md) selected Riverpod for its compile-safe dependency injection,
`AsyncNotifier`/`StreamProvider` async primitives, and `BuildContext`-free testability. Those are real
strengths — but they are strengths for a particular shape of application, and after three months of
building screens it became clear that Lyo does not have that shape.

What the application actually needs to hold is: an auth token and role, a playback state, a list of
streams, a list of tracks, favourites, playlists, and an upload progress. Each of these is owned by
exactly one feature, mutates from a small number of well-identified call sites, and is consumed by the
screens of that same feature. There is very little cross-provider composition, and almost no derived
async state that would justify `AsyncNotifier`.

Against that, Riverpod imposed a fixed cost on every addition: a provider declaration, a notifier class,
a state class, and — for anything mutable — a copy-with ceremony. The abstraction was being paid for
without being used.

A second factor weighed in: this is a one-person project whose main risk is the bus factor. The narrower
and more conventional the mobile idiom, the cheaper it is for a newcomer (or for a future maintainer) to
read.

## Decision

Migrate to **`provider`** with plain `ChangeNotifier` subclasses (commit `c6a9995`, PR #27).

Concretely:

- One `ChangeNotifier` per feature, holding its mutable fields directly, calling `notifyListeners()`
  after each mutation. No separate immutable state class.
- Notifiers registered once in `main.dart` under a `MultiProvider`.
- Screens read state with `context.watch<T>()` (rebuild on change) or `context.read<T>()` (one-off calls
  in callbacks and `initState`).
- Notifiers never reach into each other. A notifier that needs the current auth token takes it as a
  **method parameter**, read at the call site from `AuthNotifier`. This keeps the dependency explicit and
  the notifier trivially testable.
- Services (`ApiClient` and the per-feature service classes) are constructor-injected, which is what
  actually makes the notifiers unit-testable — the injection mechanism was never the hard part.

## Consequences

**Positive**

- Adding a feature costs one class instead of four. The state layer stopped being a tax on every change.
- `provider` is the most widely documented Flutter state solution: onboarding cost is minimal.
- Testability was preserved and is now demonstrated rather than asserted: `AuthNotifier`,
  `FavoritesNotifier`, `PlaylistNotifier` and `UploadTrackNotifier` are covered by unit tests using
  `mocktail` doubles of the injected services.
- Explicit token passing made an implicit dependency visible, which is an improvement in its own right.

**Negative, and accepted**

- No compile-time guarantee that a provider is registered before use: a missing registration is a runtime
  error rather than a build error. Mitigated by the fact that all registrations sit in a single file.
- `ChangeNotifier` is coarse-grained: a `notifyListeners()` rebuilds every watching widget, not just the
  ones whose data changed. At the current screen complexity this is not measurable; if a screen ever
  becomes hot, `Selector` or a split notifier is the remedy.
- No first-class async primitive: loading and error states are held as explicit fields on the notifier.
  This is more verbose per notifier but easier to follow.

**Documentary consequence**

ADR 002 is marked *Superseded* rather than deleted, and the README and CLAUDE.md were corrected to say
`provider`. A stale architecture decision record is worse than none: it makes every other document
suspect. Recording a reversal, with its reasoning, is the point of the practice.
