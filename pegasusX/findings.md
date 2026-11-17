# Findings: ATOMOS Enterprise Intelligence Plan vs Current Codebase

<!--
  WHAT: Verified ground-truth mapping of /Users/shakhzod/Desktop/ATOMOS_Enterprise_Intelligence_Plan.md against /Users/shakhzod/Desktop/pegasusx.
  BASELINE: Plan written 2026-11-15 against commit a813de086 (Nov 13).
  CURRENT REPO: HEAD e25490326 + 995 uncommitted/modified paths (per git status -s).
-->

## 1. Gate Zero (`T1–T9`) Verified Ground-Truth Audit (2026-11-16)

Many blockers listed in §2 of `ATOMOS_Enterprise_Intelligence_Plan.md` were already remediated between Nov 13 (`a813de086`) and current working tree (`e25490326` + uncommitted S1/S2 work):

| ID | Plan Item (§2) | Actual Codebase Status (2026-11-16) | Verified Code Evidence | Remaining Work |
| :--- | :--- | :--- | :--- | :--- |
| **T1** | `order/inventory_reservation.go` vs `order/inventory_release.go` (`OrderLotReservations` release + `inventory.lot.released`) | **CLOSED & LIVE-VERIFIED** | `EventInventoryLotReleased = "inventory.lot.released"` registered 3-way across `events/events.go:405-406`, `events/types.go:884-897`, `packages/types/src/envelope.ts:794`, `contracts/events.schema.json`; emitted atomically via `outbox.NewSpannerTxnBuffer(txn)` in `order/inventory_release.go:26-74,200-258`; 11/11 prod callers pass `orderID`; live `order` (`417/417` PASS) & `events` (`12/12` `-strict=true`) verified (`#269`). | None. |
| **T2** | Fail-open `SettlementJournal == nil` (`payout/store.go`) & `s.doorstepShortfall == nil` (`order/service_mutations.go`) | **CLOSED & LIVE-VERIFIED** | `payout/store.go:421-433`, `bootstrap/app.go:655,721`, `order/service_mutations.go:1912-1928`; live `payout` (`29/29` PASS) & `order` (`417/417` PASS) verified (`#269`). | None. |
| **T3** | `ALLOW_AUTH_BYPASS` allowed on `PEGASUSX_ENV=staging` | **CLOSED & LIVE-VERIFIED** | `bootstrap/config_validate.go:18-26` + `bootstrap` `58/58` PASS `-race` (`#269`). | None. |
| **T4** | Missing `stocklots/cold_baseline.go` / broken `payload/service.go:508` build | **CLOSED & LIVE-VERIFIED** | `stocklots/cold_baseline.go`, `payload/service.go:509-521`, `bootstrap/app.go:1787`; live `stocklots` (`50/50`) & `payload` (`56/56`) PASS `-race` (`#269`). | Confirm owner sign-off (gate stays active). |
| **T5** | Unverified WebSocket command ACKs (`driver/ws_ack.go`, `DriverCommands`) | **CLOSED & LIVE-VERIFIED** | `DriverCommands` (`spanner.ddl:3953`), `drivercmd` (`2/2`), `driver` (`75/75`) PASS `-race` (`#269`). | Optional live Spanner emulator WS ack test in future pass. |
| **T6** | Double-entry shortfall ID mismatch (`jentry_shortfall_%s` vs `jentry_doorstep_shortfall_%s`) | **CLOSED & LIVE-VERIFIED** | `payment/double_entry.go:485`, `payment/repository_spanner.go:803-856`; live `payment` (`154 PASS, 1 pre-existing skip, 0 FAIL`) verified (`#269`). | None. |
| **T7** | Timing-unsafe handoff secret compare (`packages/handoff/engine.go:155`) | **CLOSED & LIVE-VERIFIED** | `packages/handoff/engine.go:151-165` (`9/9` PASS), `apps/handoff-service/main.go:81-87,139` (`5/5` PASS) `-race` (`#269`). | None. |
| **T8** | Overwrite-able `PaymentLedgerEntries` via `spanner.InsertOrUpdateMap` | **CLOSED & LIVE-VERIFIED** | `payment/repository_spanner.go:789` (`spanner.InsertMap`), `SaveWebhook` `NULL_FILTERED` emulator hint + `ReferenceId IS NOT NULL` (`:225-226`), `creditnote/repository_spanner.go:51-98` (`InsertMap` + status pre-read), `cmd/seed-supplier-prodsim/main.go:169-196`; `0` remaining `InsertOrUpdateMap("PaymentLedgerEntries"`; live `payment` & `creditnote` (`12/12`) verified (`#269`). | None. |
| **T9** | `DraftAdjustments` / `DraftCreditNotes` use `ProposedAmount NUMERIC`, missing SoD (`ReviewedBy != OperatorId`) | **CLOSED & LIVE-VERIFIED** | `schema/spanner.ddl:3712-3741` & `schema/migrations/20261116_financial_drafts_minor_units.ddl` (`ProposedAmountMinor INT64 NOT NULL DEFAULT (0)`, `ReviewedAt` `allow_commit_timestamp=true`); `order/financial_drafts.go` + `orderroutes/routes.go:46-49` enforce four-eyes SoD across both tables; live `ci_schema_drift_gate.sh` & `order` (`417/417` PASS) verified (`#269`). | None. |
| **L98 Note** | `stocklots.SetLotsEnabled` never called from bootstrap | **ALREADY FIXED** | `bootstrap/app.go:1782` wires `stocklots.SetLotsEnabled(envTrue("WMS_LOTS_ENABLED"))`; tested in `bootstrap/wms_flags_test.go`. | None. |
| **§12 Q6** | `order/substitution.go` dead code / unwired | **ALREADY FIXED** | Wired via `order/allocation_wiring.go:385-555`, `order/unified_checkout.go`, and verified by `order/operating_model_w1_test.go`. | None. |

---

## 2. Full P0 Architecture & Repository Findings (`M0.2–M0.5`)

- **Schema & DDL paths:** Canonical DDL is `apps/backend-go/schema/spanner.ddl`; incremental migrations live in `apps/backend-go/schema/migrations/*.ddl`. Must pass `apps/backend-go/cmd/schema-drift` and `scripts/ci_schema_drift_gate.sh`.
- **Events Lockstep (Invariant 2):** `apps/backend-go/events/events.go` + JSON schemas + `make gen-contracts` must match (`events/schema_lockstep_test.go` checks lockstep).
- **Existing Partial Tables for P0:**
  - `AuditLog` (`spanner.ddl:876`), `PlatformAdminAudit` (`2971`), `CashReconciliations` (`1894`), `PhysicalReconciliationQueue` (`3696`), `OntologyObjects` (`3795`), `OntologyGraphEdges` (`3809`) exist alongside `apps/backend-go/compliance/`, `apps/backend-go/foundry/ontology.go`, and `apps/backend-go/payment/`.
  - **Implemented in M0.2 (`2026-11-16`, live-verified `#296`):** Append-only balanced GL (`GlAccounts`, `JournalEntries` w/ `EntryHash`/`PrevHash`, interleaved `JournalPostings`, sharded `GlBalances`, `PostingRules`), `TopicLedger` (`pegasusx-ledger`), `gl.entry.posted`/`gl.entry.reversed` 3-way contract lockstep, `apps/backend-go/gl/` + `glroutes/`, append-only `ArLedgerEntries` (`InsertMap`), full in-txn GL dual-write across `ar`, `migration`, `payment`, `creditnote`, `payout`, `cashrecon`, `claims`, `order`, and `internal/services/billing`, plus `scripts/ci_immutable_ledger_gate.sh` (`1182` files, `0` violations).
  - **Remaining P0 Modules (`M0.3–M0.5`):** 4-way reconciliation runner/tables (`ReconciliationRuns`, `ReconciliationBreaks`), hash-chained `AuditEvents` v2 + uniform SoD/RBAC policy layer, and typed ontology projector + BigQuery/Vertex/Kafka sink configs & Terraform/emulator adapters.
- **Local Runtime Environment:**
  - Live Docker stack available: `pegasusx-spanner-emulator:9010`, `pegasusx-kafka:9092`, `pegasusx-redis:6379`, `pegasusx-zookeeper:2181`, plus Memgraph / CodeGraph.
  - No live GCP cloud project, BigQuery dataset, Vertex AI endpoint, or Billz/1C live credentials verified locally unless configured via env vars — any cloud/ERP sinks in local/CI runs must follow clean adapter interfaces and emulator/dev toggles without failing Law 0.3 (`no mock theatre in production profile`).
- **Sibling Parity:** `/Users/shakhzod/Desktop/V.O.I.D/pegasusX` exists as the parity mirror per `AGENTS.md`.
