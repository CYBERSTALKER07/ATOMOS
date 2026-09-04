---
name: backend-mutations
description: Backend-go mutation checklist — Spanner, owner package, outbox, cache invalidation, tests, SSMR.
---

# Backend mutations

## Same change set must include

- Schema/migration if columns change (`schema/spanner.ddl`, `migrations/`)
- Canonical owner package only (no duplicate write paths)
- Outbox emit same RW txn as domain write
- Redis cache invalidation keys when reads are cached
- Role gates on routes (`auth.RequireRole` / tenant scope)
- Focused `*_test.go`
- SSMR marker when user-visible or cross-role (`cmd/ssmr-smokecheck`, `contracts/ssmr_ecosystem_markers.json`)

## Cancel / terminal paths

If status becomes terminal (cancel, reject, vet reject): verify inventory release, payment state, notifications — not only happy path.

## Money

- Amounts: int64 minor units
- Never float64 for currency
- Fiscal/legal: EDS path is cert-blocked until real signer — do not claim Soliq live without evidence

## Evidence roots

`apps/backend-go/**`, `auth/claims.go`, `bootstrap/`, `runtime_workers.go`


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
