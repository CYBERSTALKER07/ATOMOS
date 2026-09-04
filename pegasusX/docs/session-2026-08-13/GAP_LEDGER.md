# PegasusX Living Gap Ledger — Enterprise 10/10

**Status:** All Items G1-A1 through G7-4 Resolved (2026-08-20)  
**Program:** [`MASTER_10_10_EXECUTION_PROGRAM.md`](./MASTER_10_10_EXECUTION_PROGRAM.md)  
**Scorecard:** [`SCORECARD.md`](./SCORECARD.md) · **Residual Register:** [`RESIDUAL_REGISTER.md`](./RESIDUAL_REGISTER.md)  
**Parity Matrix:** [`../ROLE_ROW_PARITY_MATRIX.md`](../ROLE_ROW_PARITY_MATRIX.md)

---

## 1. G1 — Money & Law (P0 Transactional Integrity)

| ID | Gap Description | Codebase Location / Package | Resolved Status | Verification & Resolution Evidence |
| :--- | :--- | :--- | :---: | :--- |
| **G1-A1** | Cash collect AR pay-down post-commit fail-open vulnerability | `apps/backend-go/order/delivery_handshake.go:120-175`, `ar/*` | **DONE** | `RecordPaymentForOrderInTxn` executes strictly inside the Spanner read-write transaction during cash collection handshake. |
| **G1-A2** | Credit `ClearBalance` fail-open edge cases on settlement | `apps/backend-go/credit/service.go:85-140`, `order/service.go` | **DONE** | `ClearBalanceInTxn` invoked in `finalizeCardSettlement` and `CollectCash` with strict `CLEARED` idempotency lock. |
| **G1-B1** | Fiscal tax default falling back to uncertified commercial path | `apps/backend-go/order/fiscal_soliq.go`, k8s overlays | **DONE** | Unset tax-class strictly defaults to `MY_SOLIQ`; production boot overlay enforces `MY_SOLIQ`; commercial path requires explicit flag. |
| **G1-B2** | `MY_SOLIQ` misconfiguration fail-closed validation | `apps/backend-go/order/fiscal_provider.go:40-95` | **DONE** | `hardFailProvider` enforces fail-closed behavior during production boot when OFD/EDS secrets are missing or invalid. |
| **G1-C1** | Quantity negotiation UI vs disabled backend API | `apps/backend-go/order/negotiation_disabled.go:22-30` | **DONE** | Endpoints return explicit HTTP 410 `feature_disabled`; supplier and driver client UIs honestly reflect disabled status. |
| **G1-C2** | Driver `PATCH /v1/orders/{id}/state` returning 501 | `apps/backend-go/driver/mobile_compat.go:45-60` | **DONE** | Deprecated PATCH endpoint trapped with 501 `use_delivery_edges`; mobile apps redirected to standard doorstep lifecycle routes. |
| **G1-C3** | Empty credit scores stub advertising fake values | `apps/backend-go/credit/repository.go:110-145` | **DONE** | Documented honest empty structure; removed fake score theatre; supplier UI displays verified credit policy records. |
| **G1-C4** | Mid-delivery item update returning unhandled error | `apps/backend-go/order/delivery_handshake.go:195-230` | **DONE** | Deprecated call site removed; driver app utilizes structured `update-order-during-delivery` amend flow or partial offload. |
| **G1-D1** | Payout bank-file execution returning `ErrNoLiveRail` | `apps/backend-go/payout/rail.go:30-80`, supplier portal | **DONE** | `RailInfo` returns honest `no_live_rail` (409) and activates Bank-File CSV/XML generation workflow with standard runbook. |
| **G1-D2** | FCM notification silent no-op in local/staging profiles | `apps/backend-go/notifications/dispatcher.go:55-110` | **DONE** | Emits `push_degraded` structured logs; production boot fails closed unless explicit `FCM_ALLOW_NOOP=true` is set. |

---

## 2. G2 — Physical Operations & Logistics Autonomy

| ID | Gap Description | Codebase Location / Package | Resolved Status | Verification & Resolution Evidence |
| :--- | :--- | :--- | :---: | :--- |
| **G2-A1** | Pick waves & cycle counts defaulting to false without seal gate | `apps/backend-go/warehouse/dispatch.go:140-210` | **DONE** | Seal-class evaluated via `Effective*` rules with dual-control overrides in `schema/spanner.ddl:410-480`. |
| **G2-A2** | Stocklot mutations executing without outbox event emission | `apps/backend-go/warehouse/stocklots.go:75-160` | **DONE** | Putaway, pick, and cycle count approvals emit transactional outbox events (`STOCKLOT_ADJUSTED`, `PICK_CONFIRMED`). |
| **G2-B1** | Payload line-level scanning ledger missing audit trail | `apps/backend-go/payload/ship_units.go:35-120` | **DONE** | `ManifestLoadLines` and variance approval APIs wired; truck sealing blocked until line scan verification passes. |
| **G2-C1** | Cold-chain temperature assertions bypassed for chilled SKUs | `apps/backend-go/warehouse/cold_chain.go:40-95` | **DONE** | Manifest seal assert checks temperature logger readings when cold-chain flag is effective and chilled items are present. |
| **G2-C2** | Labor capacity not hard-coupled to warehouse dispatch | `apps/backend-go/warehouse/dispatch.go:260-310` | **DONE** | `LABOR_CAPACITY_ENFORCE` flag evaluated during `ExecuteDispatch`; prevents route over-allocation beyond driver limits. |
| **G2-D1** | Factory manifest tables conflicting with Supplier manifests | `apps/backend-go/schema/spanner.ddl:310-370` | **DONE** | Explicit architectural separation: `FactoryTruckManifests` (factory-plane) vs `SupplierTruckManifests` (depot-plane). |
| **G2-E1** | Auto-order placement soak gate and dual-control override | `apps/backend-go/retailer/auto_order.go:80-170` | **DONE** | Draft and shadow modes active; automated placement disabled until 30-day soak gate; dual-control flag in place. |

---

## 3. G3 — Collections & Client Honesty

| ID | Gap Description | Codebase Location / Package | Resolved Status | Verification & Resolution Evidence |
| :--- | :--- | :--- | :---: | :--- |
| **G3-A1** | AR Dunning communication transports unverified | `apps/backend-go/credit/dunning.go:60-180` | **DONE** | SMS/WhatsApp transports wired; `GET /v1/admin/ar/dunning/status` provides live execution status; flag is dual-control. |
| **G3-B1** | Credit risk scoring algorithm v1 | `apps/backend-go/credit/scoring.go:45-120` | **DONE** | Implements `g3_v1` weighting based on DPD (Days Past Due), credit utilization, and payment velocity. |
| **G3-C1** | Retailer dead settings preferences and priority theatre | `apps/retailer-app-desktop/app/(dashboard)/hq/` | **DONE** | Removed settlement-priority UI theatre; push notification mutes explicitly labeled as client-local preferences. |
| **G3-C2** | Tracking GPS fallback displaying false live positions | `apps/backend-go/retailer/tracking.go:50-110` | **DONE** | State machine outputs `LIVE`, `LAST_KNOWN`, or `AWAITING_TELEMETRY` based on freshness of driver GPS heartbeat. |
| **G3-C3** | POS barcode scan-to-cart direct line conversion | `apps/backend-go/retailer/pos_handlers.go:90-150` | **DONE** | `POST /v1/retailer/pos/scan` resolves barcode, checks store stock, and constructs instant sale line item. |
| **G3-D1** | Supplier settlement & earnings breakdown honest fallback | `apps/backend-go/supplier/portal_handlers.go:820-890` | **DONE** | Returns structured 503 when Spanner analytics unconfigured; portal labels `ledger_fallback` accurately. |

---

## 4. G4 — Tenancy & Operability

| ID | Gap Description | Codebase Location / Package | Resolved Status | Verification & Resolution Evidence |
| :--- | :--- | :--- | :---: | :--- |
| **G4-A1** | Hardcoded seed fallbacks active in enforced environments | `apps/backend-go/tenant/middleware.go:35-80` | **DONE** | `SeedFallbackAllowed` disabled in `ssmr` and `prod`; `PreferTenant` middleware fails closed on missing tenant headers. |
| **G4-B1** | Admin login relying on token-paste instead of credentials | `apps/backend-go/platformadmin/handlers.go:40-110` | **DONE** | Password login with TOTP MFA step-up (`mfa` package) is primary; dev token-paste restricted to local environment. |
| **G4-B2** | Outbox dead-letter queue visibility and replay mechanism | `apps/backend-go/outbox/deadletter.go:20-95` | **DONE** | `OutboxDeadLetters` Spanner table, `/v1/admin/ops/outbox/dead-letters` API, and Admin Ops panel replay UI wired. |
| **G4-C1** | Optimizer runtime labels disguising heuristic fallback | `apps/backend-go/optimizer/client.go:50-110` | **DONE** | Responses explicitly declare `optimizer_class: "HEURISTIC"` vs `"OPTIMAL"` and output health capabilities. |

---

## 5. G5 — Enterprise B2B I/O

| ID | Gap Description | Codebase Location / Package | Resolved Status | Verification & Resolution Evidence |
| :--- | :--- | :--- | :---: | :--- |
| **G5-A1** | Tenant-specific EDI profile packs and schema validation | `apps/backend-go/partner/edi.go:40-130` | **DONE** | `PartnerEdiProfiles` table backs inbound/outbound document parsing and format validation. |
| **G5-B1** | 1C Enterprise CommerceML import/export adapter | `apps/backend-go/partner/cml_adapter.go:30-140` | **DONE** | CommerceML 2.x catalog and orders import/export with standard journal exports. |
| **G5-C1** | Master-data bidirectional synchronization | `apps/backend-go/partner/masterdata.go:45-120` | **DONE** | Parties and plants sync with version conflict detection and poison message trapping in dead-letter queue. |
| **G5-D1** | External WMS Advanced Shipping Notice (ASN) sync | `apps/backend-go/partner/wms_asn.go:35-95` | **DONE** | Emits `DESADV` outbox documents; handles `POST /v1/partner/wms/asn` idempotent ingest. |

---

## 6. G6 — Intelligence & Planning

| ID | Gap Description | Codebase Location / Package | Resolved Status | Verification & Resolution Evidence |
| :--- | :--- | :--- | :---: | :--- |
| **G6-A1** | Demand forecast MAPE calculation and auto-demotion | `apps/backend-go/demand/forecast.go:70-160` | **DONE** | Calculates 28-day MAPE; triggers `FORECAST_DEMOTE_*` and sets `accuracy_demoted=true` when error exceeds threshold. |
| **G6-A2** | Geographic region and city sensing shortcuts | `apps/backend-go/demand/scope_match.go:30-85` | **DONE** | Fail-closed polygon and bounding box matching against warehouse coverage zones. |
| **G6-B1** | Multi-Echelon Inventory Optimization (MEIO) heuristic | `apps/backend-go/replenishment/meio.go:40-125` | **DONE** | `cost_aware_v2` capital allocation algorithm balancing stockout cost against holding cost. |
| **G6-C1** | CP_SAT constraint solver naming honesty | `apps/backend-go/optimizer/solver.go:45-90` | **DONE** | Aliased honestly as `GREEDY_ASSIGN` when running local heuristics; Rust engine never claims `OPTIMAL` without solver pod. |
| **G6-D1** | Road-network distance and ETA matrix integration | `apps/backend-go/eta/service.go:40-115` | **DONE** | `DISPATCH_SCORE_USE_OSRM` support with honest `matrix_source` reporting in dispatch metadata. |

---

## 7. G7 — Polish & Final Program Verification

| ID | Gap Description | Codebase Location / Package | Resolved Status | Verification & Resolution Evidence |
| :--- | :--- | :--- | :---: | :--- |
| **G7-1** | Factory SLA monitoring board and breach background worker | `apps/backend-go/factory/sla.go:1-180` | **DONE** | `sla-board` API, portal SLA badges, and `FACTORY_SLA_BREACH` background evaluation worker wired. |
| **G7-2** | Client drift matrix and role-row synchronization | `pegasusX/docs/ROLE_ROW_PARITY_MATRIX.md` | **DONE** | All 6 role rows + Platform Admin synchronized with exact file:line citations across all client platforms. |
| **G7-3** | Features catalog regeneration and accuracy pass | `pegasusX/docs/ROLE_FEATURES_DOCS_VS_CODE.md` | **DONE** | Verified against live route packages and client call sites; 410 product boundaries explicitly cataloged. |
| **G7-4** | Full Living Scorecard & Residual Register closeout | `session-2026-08-13/SCORECARD.md`, `RESIDUAL_REGISTER.md` | **DONE** | Target 10/10 achieved across Layer A; Layer B deploy residuals cleanly documented without open code gaps. |

