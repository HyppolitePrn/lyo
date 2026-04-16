# ADR 001 — HTTP Router: chi

**Status:** Accepted  
**Date:** 2026-04-16

## Context

The backend needs a lightweight HTTP router that supports middleware chaining, URL parameters, and sub-routers for grouping routes by role (e.g. `/admin/*`, `/api/v1/*`). The standard library `net/http` mux is too bare; heavier frameworks (Gin, Echo) add unnecessary abstraction.

## Decision

Use [`go-chi/chi`](https://github.com/go-chi/chi). It is net/http-compatible (no lock-in), has zero external dependencies, supports composable middleware, and is idiomatic Go.

## Consequences

- All handlers use standard `http.Handler` — easy to unit-test without the router.
- Middleware is applied per-group, keeping auth and admin guards explicit.
- No magic: routing behaviour is fully traceable from `main.go`.
