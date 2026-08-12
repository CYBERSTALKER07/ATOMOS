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
