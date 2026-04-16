# ADR 003 — Streaming Engine: Hub / Fan-out Pattern

**Status:** Accepted  
**Date:** 2026-04-16

## Context

A single broadcaster must push audio chunks to N simultaneous listeners with minimal latency and memory overhead. A naive approach (one goroutine per listener, shared mutex on a slice) risks blocking the broadcaster when a slow listener's buffer fills.

## Decision

Implement a `Hub` struct with:
- One broadcaster goroutine writing chunks to the hub.
- Per-listener buffered channels (size configurable via `STREAM_BUFFER_SIZE`).
- Non-blocking fan-out: if a listener's channel is full, the chunk is **dropped for that listener** (not the broadcaster). The listener receives a warning log.

## Consequences

- The broadcaster is never blocked by a slow client — memory usage stays O(listeners × buffer).
- Slow listeners experience audio gaps rather than causing cascading backpressure.
- `Hub.ListenerCount()` is a cheap O(1) metric for Grafana dashboards.
- Context cancellation propagates cleanly through `Broadcast(ctx, chunk)`.
