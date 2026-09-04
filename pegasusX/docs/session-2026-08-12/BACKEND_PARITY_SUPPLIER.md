# Backend Parity — A1 Supplier (`ADMIN` = SUPPLIER product)
> **POINT-IN-TIME SNAPSHOT (2026-08-12) — do not treat as current status.**
> Re-verify any claim against live code before acting. Multiple ecosystem hardening phases have shipped since this audit.


**Date:** 2026-08-12  
**Agent:** A1-SUPPLIER  
**Tree:** `pegasusX` only  
**Phase:** Backend Class A audit (no Go business-logic changes)  
**Protocol:** [`BACKEND_PARITY_PROTOCOL.md`](./BACKEND_PARITY_PROTOCOL.md)

## Scope

| Package / routes | In scope |
|------------------|----------|
| `supplier`, `supplierroutes` | Yes |
| `planning` (via supplier handlers) | Yes |
| `credit`, `creditroutes` | Yes |
| `claims` approve/reject paths | Yes (`orderroutes`) |
| `pulse`, `pulseroutes` | Yes (supplier GET) |
| `controltower`, `controltowerroutes` | Yes |
| `promotion`, `promotionroutes` | Yes (supplier-owned) |
| Clients target later | `supplier-portal`, `supplier-app-android`, `supplier-app-ios` — **not** `admin-portal` |

JWT: `auth.RoleAdmin = "ADMIN"` is the supplier portal/mobile session role  
(`apps/backend-go/auth/claims.go:20`).

Global auth: `SessionAuth` (cookie **or** Bearer) on every route  
(`apps/backend-go/main.go:124`, `auth/jwt.go:232-243`).  
Platform contract for portal + mobile is therefore one HTTP API surface; Bearer works for native apps without a separate supplier API.

**No separate `FleetHub` package/hub exists.** Fleet live map / driver location uses:
- `SupplierHub` room `supplier:{id}`
- telemetry room `telemetry:supplier:{id}` (`ws/handler.go:251`)

---

## 1. Feature inventory (route → service → Class A)

Legend for Class A columns:

| Col | Meaning |
|-----|---------|
| Auth | Role + tenant from claims (not body `supplier_id`) |
| Idem | Guard / required key / domain CAS |
| RW | Spanner RW txn |
| Outbox | Same-txn outbox emit |
| Cache | Invalidate after commit |
| RT | Realtime: outbox→Kafka→dispatcher→SupplierHub/FCM **or** intentional local hub |
| Status | `PASS` / `PARTIAL` / `FAIL` / `N/A` (read-only) |

### 1.1 `supplierroutes` — group auth

All routes below are under:

```
CookieAuth + RequireRole(RoleAdmin)
```

`apps/backend-go/supplierroutes/routes.go:79-80`  
(redundant with global `SessionAuth`; still cookie-friendly.)

Scope helper: `scopedSupplierID` → `PreferTenantSupplierID`  
(`supplier/session_scope.go:11-15`).

#### Auth / setup / org

| Route | Method | Handler | Auth | Idem | RW | Outbox | Cache | RT | Status | Evidence |
|-------|--------|---------|------|------|----|--------|-------|----|--------|----------|
| `/v1/auth/supplier/register` | POST | `HandleRegister` | public | optional | Y | Y (`SUPPLIER_CREATED`) | Y | Kafka→dispatcher | **PASS** | `service.go:769+`, register emit ~535 |
| `/v1/auth/supplier/login` | POST | `HandleLogin` | public | optional | N (read) | N | N | N | **N/A** | `service.go:839` |
| `/v1/auth/supplier/refresh` | POST | `HandleSupplierRefresh` | token | optional | N | N | N | N | **N/A** | `auth_refresh.go` |
| `/v1/supplier/configure` | POST | `HandleConfigure` | ADMIN | optional | Y | Y | Y | Y | **PASS** | `portal_handlers.go:273-292` |
| `/v1/supplier/business/setup` | POST | `HandleSupplierBusinessSetup` | ADMIN | optional | Y | Y | Y | Y | **PASS** | `setup.go:93-110` |
| `/v1/supplier/billing/setup` | POST | `HandleConfigureBilling` | ADMIN | optional | Y | Y | Y | Y | **PASS** | `service.go:886+`, ~731-744 |
| `/v1/supplier/profile` | GET/PUT | `HandleProfile` | ADMIN | optional PUT | Y PUT | Y | Y | Y | **PASS** (PUT) | `portal_handlers.go:309-445` |
| `/v1/supplier/settings` | GET | `HandleProfile` | ADMIN | N/A | N | N | N | N | **N/A** | alias `routes.go:91` |
| `/v1/supplier/topology` | GET/PUT | `HandleTopology` | ADMIN | optional PUT | Y PUT | Y | Y | Y | **PASS** (PUT) | `portal_handlers.go:598-618` |
| `/v1/supplier/org/members` | GET/POST | `HandleOrgMembers` | ADMIN | optional POST | Y POST | Y | Y | Y | **PASS** | `onboarding_handlers.go:248-269` |
| `/v1/supplier/org/members/{userID}` | PATCH/PUT/DELETE | `HandleOrgMemberByID` | ADMIN | optional | Y | Y | Y | Y | **PASS** | `org_member_lifecycle.go:89-148` |
| `/v1/supplier/fleet/drivers` | GET/POST | `HandleFleetDrivers` | ADMIN | optional POST | Y POST | Y (`DRIVER_*`) | Y | Y | **PASS** | `onboarding_handlers.go:326-346` |
| `/v1/supplier/fleet/vehicles` | GET/POST | `HandleFleetVehicles` | ADMIN | optional POST | Y POST | Y (`VEHICLE_*`) | Y | Y | **PASS** | `onboarding_handlers.go:397-415` |
| `/v1/supplier/ws-session` | GET | `HandleWebSocketSession` | ADMIN | N/A | N | N | N | mint WS token | **N/A** | `service.go:983` |

#### Pricing / promotions (supplier)

| Route | Method | Handler | Auth | Idem | RW | Outbox | Cache | RT | Status | Evidence |
|-------|--------|---------|------|------|----|--------|-------|----|--------|----------|
| `/v1/supplier/pricing/rules` | GET/PATCH | `HandlePricingRules` | ADMIN | optional | Y PATCH | Y | Y | Y | **PASS** | `portal_handlers.go:811-835` |
| `/v1/supplier/pricing/retailer-overrides` | GET/POST | `HandleRetailerPricingOverrides` | ADMIN | optional POST | Y | Y | Y | Y | **PASS** | `retailer_pricing.go:220+` |
| `/v1/supplier/pricing/retailer-overrides/preview` | POST | preview | ADMIN | N (no write) | N | N | N | N | **N/A** | `retailer_pricing_preview.go` |
| `/v1/supplier/pricing/retailer-overrides/{overrideID}` | DELETE | `HandleRetailerPricingOverrideDelete` | ADMIN | optional | Y | Y | Y | Y | **PASS** | `retailer_pricing.go:314+` |
| `/v1/supplier/promotions` | GET/POST | promotion service | ADMIN | optional | Y POST | Y `PROMOTION_CHANGED` | Y | Y | **PASS** | `promotionroutes/routes.go:24-25`, `promotion/service.go:134-140` |
| `/v1/supplier/promotions/{id}` | PATCH/DELETE | update/deactivate | ADMIN | optional | Y | Y | Y | Y | **PASS** | `promotion/handlers.go:98-181` |

#### Dashboard / analytics / reads

| Route | Method | Status | Notes |
|-------|--------|--------|-------|
| `/v1/supplier/dashboard` | GET | **N/A** | read |
| `/v1/supplier/activity` | GET | **N/A** | |
| `/v1/supplier/earnings` | GET | **N/A** | |
| `/v1/supplier/orders` | GET | **N/A** | |
| `/v1/supplier/manifests` | GET | **N/A** | |
| `/v1/supplier/exceptions` | GET | **N/A** | |
| `/v1/supplier/manifest-exceptions` | GET | **N/A** | |
| `/v1/supplier/ops/exception-map` | GET | **N/A** | |
| `/v1/supplier/supply-lanes` | GET | **N/A** | |
| `/v1/supplier/inventory` GET | GET | **N/A** | |
| `/v1/supplier/inventory/audit` | GET | **N/A** | **stub** empty entries (`portal_handlers.go:1020`) |
| `/v1/supplier/analytics/*` | GET | **N/A** | velocity/revenue/demand/* |
| `/v1/supplier/empathy/adoption` | GET | **N/A** | |
| `/v1/supplier/fleet/orders` | GET | **N/A** | |
| `/v1/supplier/fleet/live-map` | GET | **N/A** | read projection |
| `/v1/supplier/returns` | GET | **N/A** | |
| `/v1/supplier/dispatch/tracking` | GET | **N/A** | |
| `/v1/supplier/pulse` | GET | **N/A** | `pulseroutes` RoleAdmin |
| `/v1/user/notifications` | GET/POST read | **PARTIAL** mark-read only | inbox |

#### Inventory mutators

| Route | Method | Handler | Auth | Idem | RW | Outbox | Cache | RT | Status | Evidence |
|-------|--------|---------|------|------|----|--------|-------|----|--------|----------|
| `/v1/supplier/inventory` | PATCH | `handleInventoryPatch` | ADMIN | optional | Y (`AdjustStock`) | **NO** | Y | **NO** | **FAIL** | `portal_handlers.go:955-1010`; `inventory/repository.go:152-178` silent UpdateMap |
| `/v1/supplier/inventory/policy` | PATCH | `HandleInventoryPolicy` | ADMIN | optional | Apply (not full RW emit) | **NO** | **NO** | **NO** | **FAIL** | `inventory_policy.go:174-183` |
| `/v1/supplier/inventory/import` | POST | `HandleInventoryImport` | ADMIN | optional | Y | check path | Y | partial | **PARTIAL** | legacy import; staging preferred |
| Import sandbox `…/imports/*` | multi | `RegisterImportRoutes` | ADMIN | approve/apply optional | apply RW | **NO outbox on apply** | Y after apply | **local WS only** | **FAIL** (silent stock) | `import_sessions_apply.go` no EmitJSON; `import_sessions_handlers.go:660-741` post-commit hub Broadcast |

#### Orders / dispatch / exceptions

| Route | Method | Handler | Auth | Idem | RW | Outbox | Cache | RT | Status | Evidence |
|-------|--------|---------|------|------|----|--------|-------|----|--------|----------|
| `/v1/supplier/orders/vet` | POST | `HandleVetOrder` | ADMIN | optional | Y | Y `ORDER_STATUS_CHANGED` | Y | Y via order dispatcher | **PASS** (deprecated header) | `orders_vet.go:165-212`; `portal_handlers.go:1187-1275` |
| `/v1/supplier/dispatch/preview` | GET/POST | preview | ADMIN + warehouse scope | N/A | N write | N | N | N | **N/A** | |
| `/v1/supplier/dispatch/execute` | POST | `HandleDispatchExecute` | ADMIN + WH scope | **required** if store | Y multi | Y ROUTE/MANIFEST/ORDER_ASSIGNED | Y | local SupplierHub + outbox | **PASS** | `dispatch_execute.go:72-177`, emit ~503 |
| `/v1/supplier/exceptions/{kind}/{id}/resolve` | POST | `HandleResolveException` | ADMIN | optional (global MW) | via deps | **depends on kind** | via deps | via deps | **PARTIAL** | `exception_resolve.go:46-91` — credit note Issue / cash Accept / credit UpsertProfile |
| `/v1/supplier/broadcast` | POST | `HandleBroadcast` | ADMIN | optional | **NO Spanner** | **NO** | N | **direct multi-hub WS** | **PARTIAL** | `portal_admin_ops.go:146-203`; no durable bus |
| `/v1/supplier/returns/resolve` | POST | `HandleResolveReturn` | ADMIN | optional | Y | Y `SUPPLIER_RETURN_RESOLVED` | Y | Y | **PASS** | `returns.go:222-363` |
| `/v1/supplier/replenishment/trigger` | POST | `HandleReplenishmentTrigger` | ADMIN | optional | Y | Y supply request | Y | local WS + outbox | **PASS** | `portal_admin_ops.go:275-311` |
| `/v1/supplier/replenishment/policies` | GET/PATCH | policies | ADMIN | optional PATCH | UpsertPolicy | **NO** | **NO** | **NO** | **FAIL** | `plan90_handlers.go:377-385` |
| Replenishment suggestions | GET/POST | `replenishment.SuggestionsAPI` | ADMIN | draft paths | varies | varies | — | — | **PARTIAL** | `routes.go:177-182` |

#### Order-service mounted on supplier group

| Route | Method | Auth | Notes / Class A owner |
|-------|--------|------|------------------------|
| `/v1/supplier/shop-closed/active` | GET | ADMIN | read |
| `/v1/supplier/shop-closed/resolve` | POST | ADMIN | order package (A0/A1 shared) — outbox expected in order |
| `/v1/supplier/orders/payment-bypass` | POST | ADMIN | money-adjacent; order package |
| `/v1/supplier/negotiations/pending` | GET | ADMIN | product-disabled empty |
| `/v1/supplier/negotiate/resolve` | POST | ADMIN | 410 / disabled path |
| `/v1/supplier/route/approve-early-complete` | POST | ADMIN | order package |
| `/v1/compliance/*` | GET | ADMIN | read dashboards |
| `/v1/supplier/reassign-order` | POST | ADMIN | payload service |
| `/v1/supplier/recommend-reassign` | POST | ADMIN | payload service |

(Full Class A for order money paths is **A0-Spine**; listed for inventory completeness.)

#### AI / planning / control-tower (supplier paths)

| Route | Method | Handler | Auth | Idem | RW | Outbox | Cache | RT | Status | Evidence |
|-------|--------|---------|------|------|----|--------|-------|----|--------|----------|
| `/v1/supplier/ai/recommendations` | GET/POST | AI | ADMIN | optional POST | Y POST | Y `AI_RECOMMENDATION_DECIDED` | N | Kafka→SupplierHub | **PASS** (POST) | `ai_recommendations.go:176-187` |
| `/v1/supplier/control-tower/zone-overrides` | GET/POST | planning | ADMIN | optional POST | Y POST | Y `DISPATCH_ZONE_OVERRIDE` | N | outbox + local WS | **PASS** | `planning/service.go:692-723`; `plan90_handlers.go:78-82` |
| `/v1/supplier/planning/scenarios/run` | POST | RunScenario | ADMIN | optional | Y | scenario store | — | — | **PARTIAL** | draft create |
| `/v1/supplier/planning/scenarios/{id}/publish` | POST | PublishScenario | ADMIN | domain CAS | Y | Y `planning.scenario.published.v1` | — | local WS; **Kafka orphan** | **PARTIAL** | `scenarios.go:449`; dispatcher **no case** for type; local WS `plan90_handlers.go:221-226` |
| `/v1/supplier/planning/scenarios/{id}/clone` | POST | Clone | ADMIN | optional | Y | no dedicated bus event | — | — | **PARTIAL** | |
| `/v1/supplier/planning/seasonal-overrides` | GET/POST | seasonal | ADMIN | optional | Y | cache invalidate seasonal | cache Y | — | **PARTIAL** | `seasonal_templates.go:217` |
| `/v1/supplier/planning/signals/ingest` | POST | signal | ADMIN | optional | Y | baseline outbox path | forecast cache | planning consumer | **PARTIAL→PASS** if baseline write | `signal_baseline.go:85` |
| `/v1/supplier/planning/agent/invoke` | POST | agent | ADMIN | body `idempotency_key` | Y | Y agent/broadcast types | — | planning handler | **PARTIAL** | `agents.go:35`, `executor.go` |
| `/v1/supplier/meio/network-summary` | GET | | ADMIN | N/A | | | | | **N/A** | |
| `/v1/supplier/knowledge-graph` | GET | | ADMIN | N/A | | | | | **N/A** | |
| `/v1/supplier/segmentation/*` | GET/POST/PATCH | segment | ADMIN | optional | Y bootstrap | Y bootstrap outbox | — | partial | **PARTIAL** | bootstrap emits; patch path TBD for full Class A |
| Twin routes | GET | | ADMIN | N/A | | | | | **N/A** | |

### 1.2 `controltowerroutes` — RoleAdmin

| Route | Method | Handler | Auth | Idem | RW | Outbox | Cache | RT | Status | Evidence |
|-------|--------|---------|------|------|----|--------|-------|----|--------|----------|
| `/v1/control-tower/exceptions/scored` | GET | scored | ADMIN | N/A | N | N | N | N | **N/A** | |
| `/v1/control-tower/playbooks` | GET/POST | list/create | ADMIN | **NO** | Apply | **NO** | **NO** | **NO** | **FAIL** | `repository_spanner.go:144` Apply; no Emit |
| `/v1/control-tower/playbooks/{id}` | PATCH | update | ADMIN | **NO** | Apply | **NO** | **NO** | **NO** | **FAIL** | `handlers.go:89-100` |
| `/v1/control-tower/playbooks/{id}/deactivate` | POST | deactivate | ADMIN | **NO** | Apply | **NO** | **NO** | **NO** | **FAIL** | |
| `/v1/control-tower/runs` | GET | list | ADMIN | N/A | | | | | **N/A** | |
| `/v1/control-tower/runs/{id}/{action}` | POST | approve/skip | ADMIN | **NO** | Apply + side effects | **NO on run row** | **NO** | **NO on run** | **FAIL** (run state silent) | `engine.go:134-149` UpdateRun then ExecuteRun; actions may call credit outbox indirectly |
| `/v1/control-tower/evaluate` | POST | Evaluate | ADMIN | **NO** | CreateRun Apply | **NO** | **NO** | **NO** | **FAIL** | `engine.go:99` |

### 1.3 `creditroutes` — supplier finance

| Route | Method | Handler | Auth | Idem | RW | Outbox | Cache | RT | Status | Evidence |
|-------|--------|---------|------|------|----|--------|-------|----|--------|----------|
| `/v1/supplier/credit-profiles` | GET | list | ADMIN | N/A | | | | | **N/A** | scope `claims.SupplierID` only `handlers.go:69-72` |
| `/v1/supplier/retailer-credit-profile` | PATCH | upsert profile | ADMIN | optional MW | Y | Y `RETAILER_CREDIT_PROFILE_CHANGED` | N | Kafka→supplier+retailer | **PASS** | `service.go:217-228`; dispatcher `notification_dispatcher.go:176-177` |
| `/v1/supplier/credit-program` | GET/POST | program | ADMIN/WH | enable idempotent domain | UpsertProgram Apply | **NO for program row** | **NO** | **NO** | **FAIL** silent program | `policy.go:550-558`; `policy.go:229-250` |
| `/v1/supplier/credit-program/defaults` | GET/PATCH | defaults | finance roles | **NO** | Apply | **NO** | **NO** | **NO** | **FAIL** | `policy.go:587-594` |
| `/v1/supplier/credit-relationships` | GET | list | finance | N/A | | | | | **N/A** | |
| `…/relationships/{rid}/enable` | POST | enable rel | finance | domain idempotent | UpsertTerms **silent** + profile outbox | profile only | N | partial (profile) | **PARTIAL** | `policy.go:646-668` |
| `…/terms` | PATCH | patch terms | finance | **NO** | silent terms | profile if limit | N | partial | **PARTIAL** | `policy.go:698-712` |
| `…/hold` `…/unhold` | POST | freeze/unfreeze | finance | **NO** | profile outbox | Y profile | N | Y profile | **PARTIAL** | Hold uses `UpsertProfile` (`policy.go:802`) |
| `…/disable` self-serve | POST | | | | | | | | **intentional 403** | `ErrDisableRequiresSupport` |
| `/v1/admin/credit-relationships/…/disable` | POST | admin disable | ADMIN | ticket+reason | silent terms + profile outbox | partial | N | partial | **PARTIAL** | `policy.go:740-754` |
| `/v1/admin/credit-program/{sid}/disable` | POST | program kill | ADMIN | ticket+reason | silent program | **NO** | N | **NO** | **FAIL** | `policy.go:779-785` |
| `/v1/supplier/ar/invoices` | GET | AR | ADMIN/WH | N/A | | | | | **N/A** | |

### 1.4 Claims approve paths (`orderroutes`)

| Route | Method | Handler | Auth | Idem | RW | Outbox | Cache | RT | Status | Evidence |
|-------|--------|---------|------|------|----|--------|-------|----|--------|----------|
| `/v1/supplier/claims` | GET | list | ADMIN, WAREHOUSE_ADMIN | N/A | | | | | **N/A** | `orderroutes/routes.go:80` |
| `/v1/claims/{claimID}/approve` | POST | `HandleApproveClaim` | ADMIN, WAREHOUSE_ADMIN | domain replay + chargeback id; **handler has no Idempotency-Key guard** | multi-step | **terminal** CLAIM_RESOLVED dual-topic; **OPEN→UNDER_REVIEW silent** | stock resolve side-effect | Kafka handleDriverEdgeEvent → fanOrderParties | **PARTIAL** | `handlers.go:59-84` no guard; `service.go:758-834` settlement **outside** final outbox txn |
| `/v1/claims/{claimID}/reject` | POST | `HandleRejectClaim` | same | same gap | Y transition | Y dual-topic | stock restore | Y | **PARTIAL** | `service.go:883-897` |

Money note: `ApproveClaim` settles chargeback **before** final RESOLVED outbox (`service.go:767-834`). Deterministic chargeback ID prevents double ledger (`service.go:847-857`), but a crash between settlement and RESOLVED leaves `UNDER_REVIEW` (ops retry) — intentional fail-closed, document as edge case.

### 1.5 Pulse

| Route | Method | Status |
|-------|--------|--------|
| `GET /v1/supplier/pulse` | RoleAdmin | **N/A** read inbox projection (`pulse/handlers.go:39-54`) |

### 1.6 Related ADMIN surfaces (inventory only, not full A0 audit)

| Mount | Examples | RoleAdmin |
|-------|----------|-----------|
| `paymentroutes` | chargeback, ledger, claim-chargebacks | Yes |
| `catalogroutes` | product/category CRUD | Yes |
| `creditnoteroutes` | credit-notes create/issue | Yes |
| `cashreconroutes` | supplier cash recon group | Yes |
| `partner` | supplier partner-keys/webhooks | Yes |
| `orderroutes` | force-complete, refunds, reconciliation | Yes |

---

## 2. Gaps (P0 / P1 / P2) with file:line

### P0 — money / silent state machine / stock truth

| ID | Gap | Why P0 | Evidence | Proposed fix (audit only) |
|----|-----|--------|----------|---------------------------|
| S-P0-1 | ~~Inventory PATCH silent~~ **FIXED Wave B4** | `INVENTORY_QUANTITY_UPDATED` in AdjustStock RW txn | `inventory/repository.go` | handleWMSStockEvent |
| S-P0-2 | ~~Import apply silent~~ **FIXED Wave B4** | one `INVENTORY_SYNC_COMPLETE` per session | `import_sessions_apply.go` | handleSyncEvent |
| S-P0-3 | ~~Credit program silent~~ **FIXED Wave B4** | `SUPPLIER_CREDIT_PROGRAM_CHANGED` + terms events | `credit/policy.go` | dispatcher supplier (+ retailer for terms) |
| S-P0-4 | **Claims approve money outside outbox txn** | Settlement then separate RESOLVED write; intermediate UNDER_REVIEW has **no** outbox (`TransitionStatus(..., nil)`) | `claims/service.go:758-765`, `767-834` | Prefer single saga doc + optional `CLAIM_UNDER_REVIEW` event; ensure HTTP Idempotency-Key on approve; never double settle (already deterministic ID) |

### P1 — realtime / incomplete transitions / platform contract

| ID | Gap | Evidence | Proposed fix |
|----|-----|----------|--------------|
| S-P1-1 | ~~Control tower silent~~ **FIXED Wave B4** | playbook/run outbox + SupplierHub | was repository Apply |
| S-P1-2 | ~~Replenishment policy silent~~ **FIXED** | `UpsertPolicy` RW + `REPLENISHMENT_POLICY_UPDATED` outbox; dispatcher + cache key | was `plan90_handlers.go` / `replenishment/policies.go` |
| S-P1-3 | ~~Inventory policy silent~~ **FIXED Wave B4** | `INVENTORY_POLICY_UPDATED` + cache | was `inventory_policy.go` |
| S-P1-4 | ~~scenario.published orphan~~ **FIXED Wave B4** | dispatcher → `handlePlanningEvent` | multi-pod SupplierHub |
| S-P1-5 | **Broadcast is WS-only (no durable event)** | `portal_admin_ops.go:178-192` | Optional: outbox `SUPPLIER_BROADCAST` for multi-pod FCM/inbox when hub relay down |
| S-P1-6 | **Claims approve/reject missing handler idempotency guard** | File claim has guard (`handlers.go:41`); approve/reject do not (`handlers.go:59-109`) | Reuse `guardIdempotency` / rely on global MW + document required key for mobile |
| S-P1-7 | ~~Credit relationship dual-write~~ **FIXED** | Spanner `UpsertTermsAndProfile` one RW + dual outbox | was sequential terms then profile |
| S-P1-8 | **Platform client contract split (nav, not API)** | `FEATURES_BY_APP_ROLE.md:168-169` portal-only control-tower/credit policy/segmentation vs Android sections | Backend already one API; mobile should consume same routes — no second supplier API |
| S-P1-9 | **Exception resolve lacks explicit idempotency docs** | `exception_resolve.go:23-93` | Require Idempotency-Key; map errors to 409 for double-resolve |

### P2 — polish / dead / naming

| ID | Gap | Evidence |
|----|-----|----------|
| S-P2-1 | Inventory audit returns empty stub | `portal_handlers.go:1013-1020` |
| S-P2-2 | Vet endpoint deprecated but still Class A | Deprecation headers `portal_handlers.go:1273-1274` |
| S-P2-3 | Double CookieAuth on supplierroutes while SessionAuth global | `supplierroutes/routes.go:79` + `main.go:124` |
| S-P2-4 | No hub named FleetHub; docs should say SupplierHub + telemetry room | protocol wording vs code |
| S-P2-5 | Promotion service holds `retailerHub` field unused in invalidate path | `promotion/service.go:23` |

---

## 3. Event / consumer matrix (supplier-relevant)

| Producer (supplier path) | Event type(s) | Outbox topic | Consumer | WS room / FCM | Status |
|--------------------------|---------------|--------------|----------|---------------|--------|
| Profile/topology/billing/org | `SUPPLIER_*` | TopicMain | `handleSupplierUpdated` | `supplier:{id}` + FCM ADMIN | OK |
| Fleet driver/vehicle create | `DRIVER_*` / `VEHICLE_*` | TopicMain | driver/vehicle handlers | supplier + driver/warehouse | OK |
| Pricing rules / overrides | supplier + `RETAILER_PRICE_OVERRIDE` | TopicMain | supplier + price override | supplier (+ retailer on override) | OK |
| Promotions | `PROMOTION_CHANGED` | TopicMain | `handlePromotionChanged` | supplier (+ promo room tests) | OK |
| Order vet | `ORDER_STATUS_CHANGED` | TopicMain | `handleOrderEvent` | all order parties | OK |
| Dispatch execute | ROUTE/MANIFEST/ORDER_ASSIGNED | TopicMain | route/manifest/order | supplier + driver + WH | OK + local hub |
| Returns resolve | `SUPPLIER_RETURN_RESOLVED` | TopicMain | `handleReturnGateEvent` | supplier chain | OK |
| Replenishment trigger | `WAREHOUSE_SUPPLY_REQUEST_OPENED` | TopicMain | warehouse operational | supplier + WH | OK |
| Zone override | `DISPATCH_ZONE_OVERRIDE` | TopicMain | `handlePlanningEvent` | supplier + WH | OK |
| Scenario publish | `planning.scenario.published.v1` | TopicMain | **none** (orphan) | local SupplierHub only | **ORPHAN** |
| AI decision | `AI_RECOMMENDATION_DECIDED` | TopicMain | `handleAIRecommendationEvent` | supplier | OK |
| Credit profile upsert | `RETAILER_CREDIT_PROFILE_CHANGED` | TopicMain | `handleCreditEvent` | supplier + retailer | OK |
| Claims resolve | `CLAIM_RESOLVED` | TopicMain + TopicExceptions | `handleDriverEdgeEvent` (as OrderEvent fields) | order parties if IDs present | OK if supplier_id/order_id set |
| Claims OPEN→UNDER_REVIEW | — | — | — | — | **silent** |
| Inventory patch / import apply | — | — | — | import: local WS only | **silent / partial** |
| Credit program / terms row | audit table only | — | — | — | **silent** |
| Control tower runs/playbooks | — | — | — | — | **silent** |
| Broadcast | — | — | — | multi-role hubs direct | **no bus** |

Relay path (target Class A):

```
HTTP → Service → Spanner + OutboxEvents
  → Relay → Kafka
  → NotificationDispatcher → SupplierHub / FCM / inbox
```

Hub multi-pod: `ws.Hub` publishes `ws:<hub>:fanout` (`ws/hub.go:6-10,283`).

---

## 4. Edge-case matrix

| Scenario | Behavior | Class A? | Evidence |
|----------|----------|----------|----------|
| Double dispatch execute same key | Replay stored 200 | PASS | `dispatch_execute.go:89-110` |
| Dispatch without Idempotency-Key | 400 if store configured | PASS | `dispatch_execute.go:72-75` |
| Dispatch partial chunk failure | Compensation + 500 payload | documented | `dispatch_execute.go:141-156` |
| Claim double-approve | Idempotent settlement shape | PASS domain | `service.go:719-728` |
| Claim approve after reject | 409 invalid state | PASS | `service.go:730-731` |
| Claim amount > remaining | pricing error | PASS | `service.go:746-750` |
| Credit enable program twice | returns existing | PASS domain | `policy.go:515-516` |
| Credit self-serve disable | 403 requires support | intentional | `policy.go:715-717` |
| Inventory version conflict | 409 | PASS local | `portal_handlers.go:997-999` |
| Inventory negative stock | fail loud | PASS | `inventory/repository.go:167-170` |
| Import re-apply | idempotent APPLIED | PASS | `import_sessions_apply.go:85-87` |
| Import when WMS lots enabled | hard fail | PASS safety | `import_sessions_apply.go:62-63` |
| Vet already assigned | 409 | PASS | `orders_vet.go:123-126` |
| Vet APPROVED without payment | 409 payment_not_cleared | PASS | `orders_vet.go:128-135` |
| Cross-supplier claim settle | forbidden | PASS | `service_test.go` supplier scope; `actorMaySettleClaim` |
| Concurrent control-tower approve | last-write / status check only | weak | `engine.go:142-144` status==SUGGESTED no CAS outbox |
| API-only run mode | workers/outbox may be off | flag hole | protocol flag; control tower silent worse |

---

## 5. Platform contract (portal + mobile)

| Check | Result |
|-------|--------|
| One API for portal + Android + iOS | **Yes** — same `/v1/supplier/*` mounts; `SessionAuth` accepts Bearer (`auth/jwt.go:238-241`) |
| Role gate | `ADMIN` everywhere for supplier product |
| Body `supplier_id` for auth | **Avoided** on scoped paths via `scopedSupplierID` / `claims.SupplierID` |
| Separate mobile backend | **None** |
| Client nav parity | **Incomplete on mobile** for control-tower / credit policy / segmentation / tax (`FEATURES_BY_APP_ROLE.md:168-169`) — **client gap**, not dual API |
| `admin-portal` | PLATFORM_ADMIN break-glass — out of supplier product scope |

---

## 6. Class A summary scorecard (supplier mutators)

| Domain | Overall | Notes |
|--------|---------|-------|
| Onboarding / profile / topology / org / fleet | **PASS** | outbox + cache + dispatcher |
| Pricing / promotions / retailer overrides | **PASS** | |
| Dispatch execute | **PASS** | strongest idempotency |
| Returns resolve / replenishment trigger | **PASS** | |
| AI decisions | **PASS** | |
| Zone overrides | **PASS** | |
| Order vet | **PASS** (deprecated) | |
| Credit **profile** | **PASS** | |
| Credit **program/terms** | **FAIL / PARTIAL** | silent program/terms |
| Inventory patch / policy / import apply | **FAIL** | silent stock |
| Control tower | **FAIL** | silent config/runs |
| Claims approve/reject | **PARTIAL** | terminal outbox OK; money/saga + HTTP idem gap |
| Planning publish | **PARTIAL** | outbox orphan consumer |
| Broadcast | **PARTIAL** | realtime without durability |
| Pulse | **N/A** | read |

---

## 7. Proposed fix priority (do not implement in audit)

1. **S-P0-1 / S-P0-2** — inventory mutations must emit outbox (stock truth).  
2. **S-P0-3** — credit program lifecycle events.  
3. **S-P0-4 / S-P1-6** — claims approve saga docs + HTTP idempotency.  
4. **S-P1-1** — control tower run/playbook bus.  
5. **S-P1-4** — wire `planning.scenario.published.v1` in dispatcher.  
6. **S-P1-2 / S-P1-3** — policy patches.  
7. **S-P1-7** — atomic credit relationship enable.  
8. Mobile client surfaces for existing APIs (outside backend).

---

## 8. Tests already covering Class A fragments

| Area | Test evidence |
|------|----------------|
| Supplier mutation idempotency | `supplier/mutation_idempotency_test.go` |
| Import approve idempotency | `supplier/import_sessions_idempotency_test.go` |
| Dispatch key required | `supplier/dispatch_execute_test.go` |
| Claims approve settle + replay | `claims/service_test.go` (`TestApproveClaimSettlesChargeback`) |
| Claims file idempotency | `claims/handlers_idempotency_test.go` |
| Credit profile outbox (repo) | `credit/service_test.go` |
| Promotion dispatcher | `kafka/notification_dispatcher_test.go` PROMOTION / CLAIM cases |
| Broadcast multi-role | `supplier/broadcast_admin_ops_test.go` |

**Missing tests (recommended):** inventory patch outbox; import apply outbox; credit program outbox; control tower run events; scenario publish consumer.

---

## 9. Inventory of RequireRole(ADMIN) mounts (supplier product)

| File | Pattern |
|------|---------|
| `supplierroutes/routes.go:80` | group `Use(RequireRole(RoleAdmin))` — full supplier portal surface |
| `promotionroutes/routes.go:23` | group RoleAdmin |
| `pulseroutes/routes.go:31` | `GET /v1/supplier/pulse` |
| `controltowerroutes/routes.go:23-30` | all control-tower routes |
| `creditroutes/routes.go:30-56` | credit profiles/program/relationships/AR (+ warehouse finance roles where noted) |
| `orderroutes/routes.go:58,71-82` | shop-closed, reconciliation, claims list/approve/reject |
| `paymentroutes`, `catalogroutes`, `creditnoteroutes`, `partner/routes.go` | additional ADMIN supplier ops |

---

**End of audit.** No Go business logic modified; this report only.
)
