## 6.4 Payload / Loading — Android (61 kt) · iOS (41 Swift) · Terminal (Expo RN, 33 files)

> **HISTORICAL / FROZEN — do not plan from this file alone.**
> Current SoT: [`PROD_READINESS_SEQUENCE.md`](../../PROD_READINESS_SEQUENCE.md) · [`ECOSYSTEM_GAP_REGISTER_2026-08-12.md`](../ECOSYSTEM_GAP_REGISTER_2026-08-12.md) · [`FEATURES_BY_APP_ROLE.md`](../../FEATURES_BY_APP_ROLE.md).
> Body is a point-in-time snapshot; re-verify any claim against code before acting.


**Maturity: ~85%. Zero mock/TODO hits. 38/34/15 endpoints across android/iOS/terminal.**

### What exists and works

| Feature | Evidence |
|---|---|
| Payloader auth with token refresh | `PayloadApi.kt:49,52`; `TokenRefreshAuthenticator.kt` |
| Trucks sidebar, manifest list/detail | `PayloadApi.kt:56,81,87` |
| Start loading / seal / seal-completed / **seal-all**; SSCC mint per ship unit at seal | `PayloadApi.kt:90-143`; `gs1/checkdigit.go:142-171` |
| Inject order into manifest; recommend/execute reassign | `PayloadApi.kt:68,74,108,135` |
| Loading checklist per order | `OrderChecklist.kt`; iOS `OrderChecklistSection.swift` |
| Manifest exceptions; missing-items report | `PayloadApi.kt:149-161` |
| Inbound returns (sessions/scan) incl. on terminal | `PayloadApi.kt:201-212`; terminal `inboundReturns.tsx` |
| Offline queue (Room android; OfflineQueue iOS); push | `PayloadDatabase.kt`, `QueuedActionDao.kt` |
| Dual scope: warehouse + supplier manifests | `PayloadApi.kt:115-135` |

### Incomplete / decorative / broken

- **Terminal is a strict subset** (15 endpoints vs 38): no reassign, no seal-all on shared bay devices.
- **Checklist verification is manual taps** — no barcode scan per item; a mis-load is indistinguishable from a correct load.
- **In-memory dev overlay persists with comment-only gating** — `payload/service.go:41-48` keeps an in-memory overlay; the `PAYLOAD_DEV_OVERLAY` env gate exists only as a comment, no code reads it (doc-claimed control that doesn't exist).
- iOS payload app lacks `.xcodeproj` for SPM integration of the shared kit (per repo status docs); minimal test coverage (`ExampleTests.swift` only).
- No weight/temperature capture at loading despite cold-chain backend support.

### Missing features that matter (Payload)

1. **Item-level scan verification at loading.** *Purpose:* loading accuracy — the cheapest place to stop the most expensive errors. *Why:* every mis-load becomes a driver-reported condition report after the truck has driven; scan-at-load catches it at zero transport cost. *Logic:* expected set = manifest lines (sku, qty); scan events decrement expected; seal blocked (or soft-warned, mirroring the existing `pick_wave_warning` soft-warn pattern) while residual > 0; variance auto-creates a manifest exception (endpoint exists). *E2E:* loader scans each case → checklist auto-checks → seal gate → SSCC label → DESADV carries real ship units.
2. **Per-line quantities + hardware scanner on the terminal.** *Purpose:* the shared bay device must do real verification, not taps. *Why:* keyboard-wedge/DataWedge scanning already proven in warehouse kit. *Logic/E2E:* wedge input → line match → qty confirm → same seal gate as mobile.
3. **Cold-chain capture at loading.** *Purpose:* chain-of-custody for temperature before transit. *Why:* backend ingests readings and auto-raises `TEMPERATURE_BREACH` (`stocklots/coldchain.go`); loading-time baseline missing. *Logic:* record bay temp + product band at seal; breach clock starts at seal. *E2E:* seal → baseline reading → in-transit readings → delivery dock reading → quarantine decision already implemented downstream.
4. **Split the 1,700-line god view; generate the iOS Xcode project** (hygiene, unblocks shared-kit adoption).

## 6.5 Factory — Portal (365 files) · Android (77 kt) · iOS (67 Swift)

**Maturity: ~85% as a dispatch hub. This is NOT a manufacturing execution system — no BOM, work orders, or line management anywhere (a scope decision, stated honestly).**

### What exists and works

| Feature | Evidence |
|---|---|
| Auth, dashboard/pulse | `factory-portal/lib/auth.ts:132`; `DashboardScreen.kt` |
| Transfers full lifecycle (create/move/detail/driver assignment) | `factory-portal/app/transfers/page.tsx:39`; `CreateTransferScreen.kt` |
| Loading bay (grid/controls) | `LoadingBayScreen.kt`, `LoadingBayGrid.kt` |
| Manifests + rebalance/cancel; exceptions | `ManifestLifecycle.kt`; portal `manifests/[id]/page.tsx` |
| Supply requests + fulfill options | `SupplyRequestsScreen.kt`; `/v1/factory/supply-requests/fulfill-options` |
| Fleet + live map + drivers/vehicles; staff; payload overrides | `FleetScreen.kt`; `StaffScreen.kt`; `PayloadOverrideScreen.kt` |
| Analytics/insights; handoff timeline; notifications; WS realtime | `AnalyticsScreen.kt`; `FactoryRealtimeClient.kt` |
| Android offline queue | `FactoryOfflineQueue.kt` |

### Incomplete / decorative / broken

- **Factory service holds manifest state in in-memory maps** (`factory/service.go:63-65,271-273`) — restart loses state; flagged in the hardening plan (E6) and still open.
- **S&OP numbers are stub math** (shared `planning/service.go:252` stub).
- iOS/portal online-only; analytics shallow (single overview endpoint).
- `GetSAndOP` + planning-brain screens present a planning capability the backend does not have.

### Missing features that matter (Factory)

1. **Durable factory manifest state.** *Purpose:* correctness across restarts/deploys. *Why:* in-memory maps lose in-flight manifests. *Logic:* persist to the existing Spanner manifest tables; overlay reads become authoritative reads. *E2E:* deploy mid-shift → bay state intact.
2. **Production-decision: MES or not.** *Purpose:* if "factory" remains a dispatch hub, rename it honestly; if it must plan production, that is a new subsystem. *Why:* today the label outruns the substance. *Logic (if MES):* work orders from demand plan (`DemandForecastBaseline` exists per SKU-day), simple capacity check `Σ(work_order_hours) ≤ line_hours`; BOM explosion only if multi-component products are in scope. *E2E:* forecast → weekly production proposal → confirm → material reservation → completion posts finished-goods inventory.
3. **Real S&OP feed.** *Purpose:* replace `factories × 700 × 7`. *Logic:* capacity = Σ active production lines × shifts × rated throughput (needs a `ProductionLines` table); demand = `DemandForecastBaseline` 13-week sum; gap = demand − capacity. *E2E:* S&OP screen shows real gap, drives supply-request urgency.
4. **Transfer lead-time capture completeness.** *Purpose:* σ_L for safety stock v2. *Why:* `FactoryInternalTransfers.ReceivedAt` exists and feeds `ObservedLeadStats` when ≥10 samples; mobile receive-confirm UX ensures every transfer produces a sample. *E2E:* warehouse receive confirm → `ReceivedAt` written → lead stats mature → safety stock stops using assumed σ_L.

## 6.6 Warehouse — Portal (597 files, deepest portal) · Android (110 kt) · iOS (96 Swift)

**Maturity: ~82% overall — portal ~90%, mobile ~70%. The gap between backend capability and floor-worker tooling is the widest in the platform.**

### What exists and works

| Feature | Evidence |
|---|---|
| Auth, dashboard/pulse, orders + ops actions | `warehouse-portal/lib/auth.ts:90`; `OrdersScreen.kt` |
| **Pick waves (create/confirm/waive) + seal gate** — portal | `warehouse-portal/app/pick-waves/page.tsx:34,79,103`; backend `stocklots/picking.go`, `seal_gate.go` |
| **Cycle counts + adjustments (apply-on-approve, ABC)** — portal | `portal/app/cycle-counts/page.tsx:30-238`; backend `stocklots/counting.go` |
| **Bins/lots/putaway (FEFO)** — portal | `portal/app/bins/page.tsx:27-54`; backend `stocklots/fefo.go` |
| **Cold chain ingest + quarantine** — backend | `stocklots/coldchain.go` (`WMS_COLD_CHAIN_ENABLED`) |
| Inventory, stock commitments, replenishment, demand forecast | `InventoryScreen.kt`, `ReplenishmentScreen.kt`, `DemandForecastScreen.kt` |
| Dispatch + locks + rescues + auto-dispatch settings | `DispatchScreen.kt`; portal `dispatch-locks`, `dispatch/rescues` |
| Fleet live map/drivers/vehicles; returns; claims; supply requests; transfers (incl. pick-wave create inside `TransferActionsScreen.kt:340-366`) | portal + android |
| Treasury/payment config; preorders/tomorrow board; CRM; broadcasts; staff; control tower (portal) | portal routes |

### Incomplete / decorative / broken

- **The Android barcode scanner is a dead stub** — `ScannerViewModel.kt:22` (`// TODO: Inject API when available`), `:47` (`// TODO: Dispatch telemetry event to backend` — marks every scan SUCCESS without any call), and `BarcodeScannerScreen.kt` has **zero references from navigation** — an orphaned screen that cannot even be reached. This is the single clearest DECORATIVE feature in the client set.
- **WMS execution is portal-only**: pick waves, cycle counts, bins/lots/putaway have no mobile screens (Android has API methods for some, consumed only inside a transfers screen; iOS has none). Floor workers on mobile cannot execute the WMS that the backend implements.
- **No FEFO/cold-chain UI in any client** (zero hits across apps) — backend capability invisible to operators.
- Android offline queue is SharedPreferences-based (`WarehouseOfflineQueue.kt:64`) — fragile vs Room peers; iOS has no offline store.
- Serial tracking absent (backend too).

### Missing features that matter (Warehouse)

1. **Mobile floor execution (pick waves, putaway, cycle counts) on Android/iOS.**
 *Purpose:* the warehouse is run by people walking aisles with phones/scanners, not by people at desks.
 *Why:* all three flagship WMS capabilities are desk-bound today; without mobile execution the WMS is an aspirational console and the Android scanner stub is a liability.
 *Logic:* pick tasks sorted by zone + serpentine `PickSequence` (backend PR-5 exists); worker claims task → scan lot/location → confirm qty (`QuantityPicked` vs `QuantityRequested`); short-pick → exception; wave complete → seal gate clears. Cycle count: ABC cadence (A monthly/B quarterly/C annually by annual movement value) enqueues counts; variance `> threshold` → `InventoryAdjustments` with mandatory reason + approval; accuracy KPI `1 − Σ|variance|/Σexpected` per warehouse.
 *E2E:* wave released → picker mobile list → serpentine walk with scans → seal → manifest dispatch; count scheduled → mobile count → variance approval → lot QoH + roll-up adjust in one txn (apply-on-approve backend exists).
2. **Fix or delete the scanner stub.** *Purpose:* trust. *Logic:* wire `ScannerViewModel` to `WarehouseApi` (96 endpoints exist) or remove the screen; either way, add a CI grep failing on `TODO: Inject` in main sources.
3. **FEFO/cold-chain operator surfaces.** *Purpose:* expiry-driven allocation must be visible/overridable. *Logic:* lots near expiry list (`ExpiryDate − today < threshold`); allocation preview shows chosen lots; breach alerts route to quarantine actions (backend `TEMPERATURE_BREACH` auto-raise exists).
4. **Room-based offline queue parity + iOS offline store.** *Logic:* adopt `mobile-android-kit` queue contract as driver/payload already do.
5. **Serial tracking.** *Purpose:* pharma/electronics. *Logic:* `SerialNumbers` table keyed to lot + order line; scan at pick and at delivery; warranty/returns by serial.

## 6.7 Platform Admin — dedicated app ABSENT

**Maturity: ~40%.** `apps/admin-portal/` is a retired redirect stub (3 files; `redirect.mjs` exits 1). `apps/supplier-app-desktop/` likewise. Admin capability is real but scattered: ~17 `/v1/admin/*` endpoints (partner keys, FX rates, planning run-once, AR dunning run-once, credit disables) exercised through supplier-portal/warehouse-portal under ADMIN JWT.

**Absent entirely:** tenant/org lifecycle (no approval queue, no suspension, no offboarding — a supplier can self-register and **nobody can approve or remove them**, `supplier/service.go:433-447`), user/role administration UI, feature-flag console, system health/observability, audit-log viewer, support tooling, fee schedule management. No `PLATFORM_ADMIN` role exists in `auth/` at all.

### Missing features that matter (Admin)

1. **Platform admin console.** *Purpose:* the platform cannot be governed. *Why:* multi-tenancy Phase 5 and basic operations both depend on it; today tenant trust is implicit and unenforceable. *Logic:* `PLATFORM_ADMIN` break-glass role; tenant states `PENDING→APPROVED→SUSPENDED→OFFBOARDED` with KYB document collection; every admin action audit-rowed. *E2E:* supplier registers → KYB review → approve → tenant activated; incident → suspend → all tenant tokens denied at middleware.
2. **Feature-flag console.** *Purpose:* the entire autonomy stack is env-flag-gated; operators need runtime control per tenant. *Logic:* flags table + middleware resolver (env default → tenant override); audit + change approval for money-affecting flags (AR, auto-order place, fiscal provider).
3. **Fee schedule + billing ops.** *Purpose:* monetization. *Why:* billing meter schema + event decode are wired (`internal/services/billing/meter_worker.go`) but no fee schedule or invoices exist. *Logic:* fee rules `(per-order fixed | GMV bps | subscription)` per tier; nightly meter → invoice → AR open item (reuse `ArInvoices`) → dunning reuse. *E2E:* tier assign → usage meters → monthly invoice → collection → payout net-of-fees.
4. **Observability & audit surfaces.** *Purpose:* ops trust. *Logic:* outbox lag, relay watchdog state (`outbox/relay.go:88-122` already computes stuck events), DLQ depth, fiscal failure rate, capture success rate — all already measurable from existing tables.
