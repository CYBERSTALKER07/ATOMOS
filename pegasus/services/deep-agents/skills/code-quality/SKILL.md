---
name: code-quality
description: Go package ownership, tests, dead paths, duplicate writers, composition-root hygiene.
---

# Code quality

## Ownership

- One canonical writer package per aggregate (no duplicate Spanner write paths)
- `bootstrap/` is composition root only — domains receive narrow Deps
- Prefer `outbox.SpannerTxnBuffer` for domains that write via Spanner directly

## Tests & hygiene

- Focused `*_test.go` in the touched package
- SSMR marker when user-visible / cross-role
- No silent dead consumers (constructed but never `Start`ed)
- Fail closed on money/tenant; fail open only where documented (e.g. AR after cash)

## Anti-patterns

- `context.Background()` for long-lived workers (use app cancel ctx)
- Placeholder / seed fallbacks that mask IDOR
- Float money; wall-clock stamps ahead of Spanner emulator

## Evidence

`apps/backend-go/**`, `runtime_workers.go`, `bootstrap/bootstrap.go`, package `*_test.go`


# Universal Agent & Engineering Guidelines
When developing, designing, or planning, always ensure to account for:
- Gaps, edge cases, and comprehensive feature validation.
- Best practices and optimized integration for Kafka, Redis, Backend, Optimizers, AI, and UI.
- Real-time concepts including WebSockets, webhooks, and their native app equivalents.
- Thorough business logic for features, understanding how the role, app, and ecosystem work together, and engagements with other roles and features.
- Best practices for backend, frontend, and infrastructure libraries/packages. Always prefer existing, high-quality open-source libraries and packages that best suit our features before creating our own.
- Optimal UI infrastructure and UX patterns (e.g., optimal screen positioning for drivers during an active route), applying the same high standards to backend and cloud architecture.
- ALWAYS search the web to find open-source code, libraries, packages, math, algorithms, approaches, and best practices for anything we are doing. If none exist, then create our own.
- Always search the web to get the correct logic, and incorporate edge cases, business logic for features, operations (ops), workflow, data consistency, finance, and AI into everything we do.
