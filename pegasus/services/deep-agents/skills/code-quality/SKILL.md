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
