# Backend programming + system design — audit

**Date:** 2026-08-18  
**Tree:** `pegasusX/apps/backend-go`  
**Kind:** how the monolith is shaped vs golang-project-layout / 12-factor / Spanner discipline. **Not** a microservices proposal.

**Related:** [`SURFACE_AUDITS.md`](./SURFACE_AUDITS.md) · [`KAFKA_AUDIT.md`](./KAFKA_AUDIT.md) · [`REDIS_AUDIT.md`](./REDIS_AUDIT.md)

---

## 0. Verdict

```
VERDICT: PARTIAL
KEEP the one-process monolith + domain packages + *routes
DO NOT extract microservices, cmd/internal rewrite, or Vercel
checkout_reads_this still false — auth/market_pack.go:138-139
NEXT: split bootstrap.NewApp in-package; finish leftover Client.Apply; JWT-only API groups
```

---

## 1. Architecture (as built — keep)

```text
main.go (route mounts)
  → chi + SessionAuth (HS256)
  → *routes packages (cycle break)
  → domain packages (order, payment, factory, warehouse, …)
bootstrap.NewApp  → Spanner, Redis, Kafka publisher, outbox relay (worker), WS hubs

SoT:  Spanner ReadWriteTransaction + outbox.EmitJSON
Bus:  Kafka at-least-once
Hot:  Redis (not ledger)
Live: WS rooms + optional FCM
```

**Laws already in code**

| Law | Live |
|-----|------|
| Tenant `SupplierId` | JWT claims + Gate 5; seed fail-closed when `TenantContextEnforced` |
| Dual manifests | `SupplierTruckManifests` vs `FactoryTruckManifests` — `schema/spanner.ddl` |
| Integer money | `AmountMinor` / `PriceMinor` INT64. Leftover: billing meters convert to float (`internal/services/billing/amount.go`) |
| Unkeyed ≠ success | `catalogHonestyExecutor` → `no_live_keys` 501 |
| Memory fallback | Forced **off** when production or `REQUIRE_INFRA_ADAPTERS` (`bootstrap.go:376-379`). Default adapters **true** (`:351`) |

Payout: `IsLivePayoutRailImplemented` always **false** — `auth/payout_pack.go:45-48`.

---

## 2. Layout vs `golang-project-layout`

Skill wants `cmd/server` + `internal/` + thin main.

This tree: **flat domain monolith** (one `go.mod`). Server is module-root `main.go` (~525 lines of mounts). `cmd/` is tools (`ssmr-smokecheck`, `gen-contracts`, `setup`, …). `internal/` is billing + web helpers only.

**That is valid.** `*routes` exist specifically to break import cycles (`routing/localsearch.go` comment: dispatch ↛ routing ↛ dispatch).

**Do not** spend a phase on `cmd/backend/main.go` + moving 100 packages under `internal/`. Cosmetic vs Class A.

**Do** split `bootstrap/bootstrap.go` (~2900 lines) into same-package files (`infra.go`, `services.go`, `workers.go`) so `NewApp` is a table of constructors.

12-factor: config via env (`bootstrap` `envOr`), logs stdout JSON in k8s, backing services attached, migrate Job — **REAL** shape. `PEGASUSX_RUN_MODE` api|worker|all is the process split, not extra repos.

---

## 3. Transactions (Spanner discipline)

**Right:** `ReadWriteTransaction` + `BufferWrite` + `outbox.EmitJSON` (order create, payload/factory Spanner repos).

**Leftover `spanner.Client.Apply`** (no abort retry; easy to omit outbox):

- `factory/auth_register.go` user/factory stamp
- `factory/planning_service.go` SupplyLanes batch
- warehouse register/setup/ops siblings

`Apply` is OK only for single-row idempotent heartbeats. Register/planning should use existing `RunTx` if they emit events or write multiple rows.

Factory in-memory overlay (`ensureDemoDataLocked` / `HandleFleet`) is **PARTIAL** vs Spanner live-map list — collapse onto DB when touching factory fleet.

---

## 4. Auth module

| Path | Role |
|------|------|
| `SessionAuth` | API SoT |
| Firebase `VerifyIDToken` | Login OTP only |
| `FirebaseAuth` middleware | **Unmounted.** Pass-through only (`auth/firebase.go:192-203`); tests in `firebase_auth_test.go`. Do not remount as session. |
| Auth0 | Explicitly **not** wrapping router (`main.go`) |

See [`FIREBASE_AUDIT.md`](./FIREBASE_AUDIT.md).

---

## 5. Modularization rules (for other agents)

1. New HTTP: `*routes` mount + domain service + tests. Do not add a second binary for one resource.
2. New event: `events/` type + `outbox.EmitJSON` in the **same** txn + consumer case **or** documented no-op. Run `make gen-contracts-gate`.
3. New cache: `cache` key helper + Invalidate **after commit**. Do not invent a process-global Redis client.
4. Factory vs supplier trucks: never join tables for “one fleet map.”
5. Pack reads: session/catalog already bind display; **do not flip** `CheckoutReadsThis` in a drive-by.
6. Fail closed: missing infra → boot error when `REQUIRE_INFRA_ADAPTERS`. No `loggingOutboxPublisher` in ssmr/prod.

---

## 6. Ranked improvements

| # | Change | Why |
|---|--------|-----|
| 1 | JWT-only on role API groups | Avoid Firebase Bearer as session |
| 2 | `Apply` → RW+outbox where multi-row / events | Spanner abort + bus |
| 3 | Split `bootstrap.go` in-package | Reviewability |
| 4 | Factory fleet list from Spanner | Drop demo overlay |
| 5 | Billing meters stay labeled non-money | Do not “fix” VU into INT64 blindly |

**INTEGRATE microservices: NO.** One module, one Spanner, one outbox, one JWT.

---

## 7. Next slice (when asked)

In-package bootstrap split **or** factory `Apply` → `RunTx`. Not a service extraction. Not Layer B.
