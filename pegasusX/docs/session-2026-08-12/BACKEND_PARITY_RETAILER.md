# Backend Parity — RETAILER (A2)

**Date:** 2026-08-12  
**Agent:** A2-RETAILER  
**Tree:** `pegasusX` only  
**Phase:** Backend Class A audit (no code changes)  
**Protocol:** [`BACKEND_PARITY_PROTOCOL.md`](./BACKEND_PARITY_PROTOCOL.md)  
**Clients (later):** `retailer-app-desktop`, `retailer-app-android`, `retailer-app-ios` — one HTTP/WS contract.

## Definition of done (reminder)

JWT-scoped mutation → Spanner RW txn + in-txn outbox → relay → Kafka → declared consumer(s) → WS hub room and/or FCM and/or partner webhook; cache invalidate after commit; idempotency on mutators; no silent Spanner writes; edge cases covered by tests or documented intentional deferral.

---

## 1. Feature inventory (route → service → Class A)

### 1.1 Auth gate surface

| Mount | Gate | Evidence |
|-------|------|----------|
| `retailerroutes.RegisterRoutes` protected group | `auth.ProtectMutations` + `auth.RequireRole(auth.RoleRetailer)` | `retailerroutes/routes.go:260-267` |
| Public auth | login/refresh/register; memberships + select-org **without** RequireRole (PendingOrgSelect) | `retailerroutes/routes.go:39-44` |
| switch-org | full JWT (RequireRole rejects PendingOrgSelect) | `retailerroutes/routes.go:47-48` |

### 1.2 RETAILER-gated routes by package

#### A. `retailerroutes` (`apps/backend-go/retailerroutes/routes.go`)

| Area | Methods / paths | Handler package | Mutator? |
|------|-----------------|-----------------|----------|
| Auth | `POST /v1/auth/retailer/{login,refresh,register}` | `retailer` | partial |
| Multi-org | `GET …/memberships`, `POST …/select-org`, `POST …/switch-org` | `retailer` | yes (token) |
| WS ticket | `GET /v1/retailer/ws-session` | `auth.WSSessionHandler` | no |
| Identity | `GET /v1/retailer/me`, capabilities enable/disable | `retailer` | enable/disable mut |
| Control tower | `GET /v1/retailer/control-tower/pulse` | `retailer` | no |
| Org / locations | members*, locations*, switch-location | `retailer` | yes |
| Store stock | stock list/SKU, receive, transfer, adjust, counts, commit | `retailer` | yes |
| POS | registers, sessions, sales, void/refund, holds, catalog | `retailer` | yes |
| Local SKUs | CRUD | `retailer` | yes |
| Time / shifts | clock-in/out, shifts close | `retailer` | yes |
| Sections | CRUD + skus/staff | `retailer` | yes |
| Reports / HQ / insights | GET only | `retailer` | no |
| Assist tickets | create/claim/complete/cancel | `retailer` | yes |
| Setup / profile | setup, profile GET/PUT | `retailer` | yes |
| Suppliers | list, attach/detach actions | `retailer` | yes |
| Cart | `GET/POST cart/sync`, clear | `retailer` | yes |
| Checkout quote / promo watch | promotion service (if wired) | `promotion` | quote=no; watch mut |
| Orders list | `GET /v1/retailers/{id}/orders`, `GET /v1/orders` | `retailer` | no |
| Cancel | `POST /v1/orders/request-cancel`, `POST /v1/order/cancel` | **order** if wired else **stub** | yes |
| Shop-closed respond | shop-closed-response paths | **order** if wired else stub | yes |
| Auto-order | settings, patch, run, shadow, soak | `retailer` | yes |
| AI / preorder | predictions, confirm/reject AI, preorder lifecycle | `retailer` (+ order lifecycle) | yes |
| Delivery proposals | accept/reject | `retailer` | yes |
| Pending pay / tracking | GETs | `retailer` | no |
| Cash/card checkout | `POST /v1/order/{cash,card}-checkout` | **payment** if wired else stub | yes |
| Cards | list + initiate/confirm/deactivate/default | **stubs** | silent OK |
| Notifications | list + mark read | `retailer` / notifications | mark-read mut |

#### B. `orderroutes` (RETAILER roles)

| Path | Method | Service | Notes |
|------|--------|---------|-------|
| `/v1/order/create` | POST | `order.HandleCreate` | Class A core | 
| `/v1/order/currencies` | GET | order | read |
| `/v1/retailer/parent-orders/{parentOrderID}` | GET | order | multi-supplier rollup |
| `/v1/order/{orderID}/status` | PATCH | order | retailer may **CANCEL only** |
| `/v1/retailer/orders/{orderID}/receipt` | GET | order | |
| `/v1/order/{orderID}/timeline` | GET | order | |
| `/v1/order/{orderID}/status-context` | GET | order | |
| `/v1/order/{orderID}/qr-payload` | GET | order | |
| `/v1/delivery/confirm-cash` | POST | order | doorstep cash confirm |
| `/v1/delivery/report-condition` | POST | order | shared |
| `/v1/order/{orderID}/condition-reports` | GET | order | |
| `/v1/orders/{orderID}/claims` | POST/GET | claims | file + list |
| `/v1/orders/{orderID}/claim-eligibility` | GET | claims | |

Evidence: `orderroutes/routes.go:36-78`.

#### C. `paymentroutes` (RETAILER)

| Path | Method | Service | Class A notes |
|------|--------|---------|---------------|
| `/v1/checkout/unified` | POST | `payment.HandleUnifiedCheckout` → cart body delegates to `order.HandleUnifiedCheckout` | create path Class A |
| `/v1/checkout/preview` | POST | order preview | dry-run |
| `/v1/checkout/b2b` | POST | **410 Gone** pre-delivery removed | intentional |

Evidence: `paymentroutes/routes.go:27-29`; B2B gone at `payment/service.go:1343-1350`.

#### D. `creditroutes` (RETAILER read)

| Path | Method | Service |
|------|--------|---------|
| `/v1/retailer/credit-profile` | GET | credit |
| `/v1/retailer/credit-relationships` | GET | credit policy |
| `/v1/retailer/ar/invoices` | GET | ar |

**No retailer HTTP “pay AR invoice” mutator.** AR open = driver credit-leave; AR pay-down = driver cash collect (`order/service.go:2186-2196`).

#### E. Other shared

| Path | Role | Package |
|------|------|---------|
| `GET /v1/catalog/*` | (auth, not role-locked retailer-only) | catalogroutes |
| `GET /v1/retailer/pulse` | RETAILER | pulseroutes |
| Platform client-policy | multi-role incl. RETAILER | platformroutes |

### 1.3 Class A checklist — money / order core mutators

| # | Mutation | Auth scope | Idempotency | Spanner RW | In-txn outbox | Cache | Realtime | Edge cases / tests | Class A |
|---|----------|------------|-------------|------------|---------------|-------|----------|--------------------|---------|
| 1 | `POST /v1/order/create` | RoleRetailer; **uses `claims.Subject` as retailerID** | yes (`guardIdempotency`) | CreateOrder RW | `EventOrderCreated` (+ preorder notify) | invalidate retailer/supplier/catalog keys | dispatcher → `retailer:{id}` | zone/acceptance/inventory/currency | **PARTIAL** — org-id bug (P0) |
| 2 | `POST /v1/checkout/unified` (cart items) | RoleRetailer; **Subject again** | yes | via Create / multi Create | per child Create | per Create | per child ORDER_CREATED | empty cart rejected; multi-supplier split | **PARTIAL** |
| 3 | Multi-supplier parent insert/update | same | n/a (parent not keyed) | separate `Apply` | **no outbox on ParentOrders** | no | no parent event | compensate cancel on leg fail | **FAIL** (silent parent writes) |
| 4 | `POST /v1/order/cancel` | Subject must match `order.RetailerID`; cancel only | yes | UpdateOrder RW | `ORDER_STATUS_CHANGED` | yes | hub fanout | cancel locked (preorder); request-cancel hard 403 | **PARTIAL** (Subject vs OrgID) |
| 5 | `POST /v1/orders/request-cancel` | RoleRetailer | n/a | none | none | n/a | n/a | intentional 403 | **N/A** (disabled by design) |
| 6 | File claim | org via `ResolveRetailerOrgID` + `PermClaimFile` | yes | CreateClaim RW | CLAIM_FILED (+ reverse logistics) | no explicit cache | driver-edge → retailer room | window expired, residual qty, photo, hold fail | **PASS** (strong) |
| 7 | Card checkout | Subject as retailerID | yes | session + outbox via initCheckoutSession | payment events | payment keys | finance → retailer | pay-only ARRIVED/AWAITING_PAYMENT | **PARTIAL** (scope) |
| 8 | Cash checkout | Subject | yes (record only) | **no Spanner write** | **none** | none | none | status gate via snapshot only | **FAIL** (ack-only / silent) |
| 9 | Credit leave | DRIVER (not retailer) | yes (driver path) | status + AR open | CREDIT_LEAVE | — | shop-closed handler → retailer | proximity, fiscal, AR invoices | **PASS** (driver surface; retailer consumer) |
| 10 | AR list | RoleRetailer | n/a | read | n/a | n/a | AR events **not** in dispatcher switch | no pay endpoint | **READ-ONLY** |
| 11 | Cart sync POST | `retailerIDFromRequest` (org) | **no** | CartItems RW | **none** | `retailer:cart:{rid}` | EventCartSyncUpdated never emitted | empty/partial OK | **FAIL** (silent write) |
| 12 | Auto-order place | PermOrderPlace + manager role for place | **no** on HTTP run | via `order.Create` | via Create | via Create | via Create | geo missing, credit skip soft, per-supplier continue | **PARTIAL** |
| 13 | POS sale | PermPosSell + org | yes (+ client_sale_id) | multi-step (stock then sale) | **separate** emitPosEvent txn | none | POS events **not** dispatched | offline cash-only, tender match | **PARTIAL** |
| 14 | Store stock mutators | org + perms | partial | applyDelta RW | in-txn STORE_STOCK_* | version bump | STORE_STOCK_* **not** dispatched | claim hold | **PARTIAL** (realtime hole) |

---

## 2. Gaps (P0 / P1 / P2) with file:line

### P0 — money / auth / silent state-machine risk

| ID | Gap | Evidence | Risk |
|----|-----|----------|------|
| **R-P0-1** | ~~**Retailer org scope on order create/checkout uses `claims.Subject`**~~ **FIXED Wave B3 (2026-08-13)** — money paths use `auth.ResolveRetailerOrgID`. | Create/unified/cancel/card/cash/confirm-cash + ownership gates | Was: staff tokens created orders under wrong RetailerId |
| **R-P0-2** | **Cash checkout is ack-only** — no Spanner mutation, no outbox, no status advance. Clients may believe cash path was booked. | `payment/retailer_checkout.go:191-251` builds response only; contrast card which calls `initCheckoutSession` | Silent “mutation” / money-path theatre |
| **R-P0-3** | **Fallback stubs if OrderService/PaymentService nil** return fake success without persistence. Routes prefer real services, but stubs remain live as else-branch. | Fake cancel OK: `retailer/core_handlers.go:601-617`; Fake create/unified: `retailer/mobile_compat.go:16-43`; Fallback wire: `retailerroutes/routes.go:179-188`, `235-241` | api-only / mis-bootstrap silent success |
| **R-P0-4** | ~~**ParentOrders insert/update bare Apply**~~ **FIXED Wave B3** — RW txn + `PARENT_ORDER_CREATED` / `PARENT_ORDER_UPDATED` outbox + dispatcher. | `order/multi_supplier_checkout.go` insert/update; kafka `handleParentOrderEvent` | Was: silent parent aggregate |

### P1 — realtime / incomplete transitions / contract split

| ID | Gap | Evidence | Risk |
|----|-----|----------|------|
| **R-P1-1** | ~~**POS / store-stock not in dispatcher**~~ **FIXED Wave B3 (bus)** — `handleRetailerOpsEvent` fans to RetailerHub. (Sale still multi-step txn — R-P1-4 remains.) | `kafka/notification_dispatcher.go` | Was: WS miss on POS/stock |
| **R-P1-2** | **AR invoice lifecycle events (`AR_INVOICE_*`) not fanned to RetailerHub.** | Events: `events/events.go:263-266`; Emit: `ar/service.go:442+`, `598+`; Dispatcher: no AR case | Credit/AR screens stale across platforms |
| **R-P1-3** | ~~**Cart sync write without outbox**~~ **FIXED Wave B3** — `CART_SYNC_UPDATED` emitted in cart Upsert/Clear txns. | `retailer/repository_cart.go` | Was: multi-device cart desync |
| **R-P1-4** | **POS sale not single-txn Class A:** stock decrement loop → sale persist → separate outbox txn. Partial failure can leave stock down without sale (or sale without event). | `retailer/pos.go:575-627` | Inventory corruption under concurrent POS |
| **R-P1-5** | **Multi-supplier checkout is not one atomic Spanner txn** across suppliers: sequential Create + compensate cancel; parent may end CANCELLED with compensated children races. | `order/unified_checkout.go:282-395`; compensate: `multi_supplier_checkout.go:210-223` | Partial multi-supplier states under failure |
| **R-P1-6** | **Credit reserve at create is soft-skip** (create proceeds if not allowed); reserve after commit can fail with only log. | Soft-skip: `order/service.go:1319-1329`; Post-commit reserve warn: `1432-1436` | Credit hold / headroom not enforced on cash/card path (by design soft) — document; credit-leave still gated separately |
| **R-P1-7** | **Saved cards API is theatre** — empty list + mutation always OK. | `retailer/mobile_compat.go:101-117`; routes: `retailerroutes/routes.go:250-254` | Platform contract lies for all 3 clients |
| **R-P1-8** | **AI preorder / correct-prediction / family residual** — Gone/empty/silent OK surfaces still mounted. | AI preorder 410: `mobile_compat.go:80-89`; correct prediction silent OK: `92-98`; request-cancel stub if OrderService nil already P0 | Contract noise |
| **R-P1-9** | **Auto-order `POST …/run` lacks Idempotency-Key** while place mode creates real orders. | `retailer/auto_order_worker.go:90-134` | Double place on retry |
| **R-P1-10** | **Unified checkout does not clear server cart** after success — client-only clear. | UnifiedCheckout ends at response `unified_checkout.go:384-395`; no ClearCart call | Stale cart re-checkout risk |
| **R-P1-11** | **B2B checkout 410** while FEATURES still lists path — clients must treat unified + pay-at-delivery only. | `payment/service.go:1348-1350`; FEATURES: `docs/FEATURES_BY_APP_ROLE.md:83` | Contract doc drift |

### P2 — polish / naming / dead paths

| ID | Gap | Evidence |
|----|-----|----------|
| **R-P2-1** | Supplier name hard-coded `"Supplier"` / `"pegasusX Supplier"` in multi-supplier + mobile tracking maps | `unified_checkout.go:310-313`; `mobile_compat.go:279-280` |
| **R-P2-2** | Notification list errors soft-return empty 200 | `mobile_compat.go:137-139` |
| **R-P2-3** | `HandleMarkNotificationsRead` returns ok even if mark fails | `mobile_compat.go:167-170` |
| **R-P2-4** | Auto-order placeholder unit price 100 (quoted later if Spanner products) | `auto_order_worker.go:533` |
| **R-P2-5** | `EventCartSyncUpdated` consumer dead without producer | see R-P1-3 |

---

## 3. Event / consumer matrix (retailer-relevant)

| Event | Producer (txn?) | Kafka consumer path | RetailerHub room | FCM / inbox | Notes |
|-------|-----------------|---------------------|------------------|-------------|-------|
| `ORDER_CREATED` | order.Create in-txn | `handleOrderEvent` | `retailer:{RetailerID}` | yes | `service.go:1393-1414`; fan: `notification_dispatcher.go:116-117`, `311-322`, `671-697` |
| `ORDER_STATUS_CHANGED` | UpdateOrder in-txn | same | same | yes | cancel included |
| `ORDER_ASSIGNED` / `REASSIGNED` | assign path | `handleOrderAssignmentEvent` | same | yes | |
| `ORDER_FINALIZED` / amended / validation failed | order | order event handlers | same | yes | |
| `PRE_ORDER_*` | create scheduled / cancel | parity prefix | retailer | yes | |
| `SHOP_CLOSED*` / `CREDIT_LEAVE` / proximity / partial offload | order/driver | `handleShopClosedEvent` | retailer | yes | `notification_dispatcher.go:125-128` |
| `CLAIM_FILED` / resolved / reverse logistics | claims CreateClaim in-txn | `handleDriverEdgeEvent` | retailer (if retailer_id in envelope) | yes | `claims/service.go:500-538` |
| `CREDIT_LEAVE` / AR open | driver credit leave + ar.OpenFromCreditLeave | shop-closed + AR open event | CREDIT_LEAVE → hub; **AR open gap** | partial | AR: R-P1-2 |
| `RETAILER_CREDIT_PROFILE_CHANGED` / limit breached | credit | `handleCreditEvent` | retailer | yes | |
| `PAYMENT_REQUIRED` / cleared / settlement / fiscal* | payment/order | `handleSupplierFinanceEvent` | retailer | yes | |
| `PROMOTION_CHANGED` | promotion | `handlePromotionChanged` | allowlist per-retailer OR `supplier-promo:{id}` | allowlist only | watch attaches room |
| `CART_SYNC_UPDATED` | **none found on cart POST** | `handleSyncEvent` | would retailer | — | dead producer |
| `POS_SALE_*` / `POS_SESSION_*` | emitPosEvent (separate txn) | **no handler** | **none** | **none** | R-P1-1 |
| `STORE_STOCK_*` | applyDelta in-txn | **no handler** | **none** | **none** | R-P1-1 |
| `AR_INVOICE_*` | ar service in-txn | **no handler** | **none** | **none** | R-P1-2 |

**RetailerHub wiring:** bootstrap binds hub; dispatcher `RetailerHub.Broadcast(ctx, "retailer:"+retailerID, payload)` at `kafka/notification_dispatcher.go:671-677`. Multi-user FCM expands via `RetailerActors.ListActiveUserIDs` (`678-697`). WS session: `GET /v1/retailer/ws-session` (`retailerroutes/routes.go:49`).

**Promo room:** `ws.SupplierPromoRoom(supplierID)` via watch + ALL-scope promotions (`notification_dispatcher_parity.go:231-233`).

---

## 4. Edge-case matrix

| Edge | Behavior | Evidence | Verdict |
|------|----------|----------|---------|
| **Empty cart** | UnifiedCheckout rejects `items must not be empty` | `unified_checkout.go:201-203` | Covered |
| **Invalid line qty/price** | 4xx validation | `unified_checkout.go:218-225`, `436+` | Covered |
| **Inventory exhausted** | `ErrInventoryExhausted` → 422 | Create `1332-1333`; handler maps inventory errors | Covered |
| **All backorder** | BACKORDERED order, no empty primary | `service.go:1336-1362` | Covered |
| **Zone / warehouse miss** | 422 / 503 | `1249-1260`, HandleCreate maps | Covered |
| **Order acceptance closed** | 422 + code | `1276-1278` | Covered |
| **Multi-supplier one leg fails** | compensate cancel prior children; 422 + `supplier_errors` | `unified_checkout.go:341-350`; handler `163-168` | Covered logic; non-atomic (P1) |
| **Multi-supplier N=1** | still ParentOrders when flag on | comment `196`; path always inserts parent | OK when flag |
| **Flag off multi-supplier** | single Create under JWT tenant | `214-233` vs legacy `235+` | Env contract: `MULTI_SUPPLIER_CHECKOUT_ENABLED` / ssmr default `multi_supplier_checkout.go:23-36` |
| **Cancel after dispatch** | state machine + cancel locked; request-cancel always 403 | `retailer_request_cancel.go:5-14`; `PreorderCancelLocked` | Intentional |
| **Cancel ownership multi-user** | Subject vs RetailerID | `service.go:1584-1586` | **Broken for staff JWT (P0)** |
| **Claim window expired** | `ErrClaimWindowExpired` | `claims/service.go:412-416`; eligibility `claims/eligibility.go`; window snapshot at complete `claim_window.go` | Covered + tests |
| **Claim over residual qty** | CapAmount + prior claimed | `claims/service.go:424-446` | Covered |
| **Claim without photo (damage)** | `ErrEvidenceRequired` | `371-392` | Covered |
| **Claim stock hold fail** | compensate claim | `543-547` | Covered |
| **Credit profile hold / disabled** | credit-leave `CanLeaveOnCredit` / CheckCreditPath; create soft-skips reserve | `order/credit_guard.go:9-21`; create soft-skip | Create not hard-blocked (doc) |
| **Pay before delivery** | card/cash snapshot rejects non ARRIVED/AWAITING_PAYMENT | `unified_checkout.go:111-119`; payment error map `254-266` | Covered |
| **Backorder payment** | deferred error | `ErrBackorderPaymentDeferred` | Covered |
| **Double submit order** | Idempotency-Key on create/unified | `HandleCreate`/`HandleUnifiedCheckout` | Covered when client sends key |
| **Double auto-order place** | bucket marks; **no HTTP idem** | `auto_order_worker.go:565-567` vs missing guard on run | Partial |
| **POS offline card** | forbidden | `pos.go:463-474` | Covered |
| **POS tender mismatch** | 400 | `548-554` | Covered |
| **Cart nil repo** | GET empty / POST no-op success | `core_handlers.go:459-461`, `490` | Soft degrade (api-only hole) |
| **AR pay after credit leave** | driver cash collect RecordPaymentForOrder fail-open | `order/service.go:2186-2196` | Intentional fail-open log |

---

## 5. One backend contract for all 3 platforms

### 5.1 Canonical paths (desktop / Android / iOS must share)

| Capability | Canonical HTTP | Notes |
|------------|----------------|-------|
| Login / refresh / multi-org | `/v1/auth/retailer/*` | intermediate memberships → select-org → full JWT |
| Identity | `/v1/retailer/me`, capabilities, switch-location | org-scoped |
| Catalog | `/v1/catalog/*` | not retailer demo stubs |
| Cart | `GET/POST /v1/retailer/cart/sync?scope=all` | multi-supplier lines carry `supplier_id` |
| Place order (single) | `POST /v1/order/create` **or** cart unified | prefer unified for multi-supplier |
| Place order (cart / multi) | `POST /v1/checkout/unified` with `items[]` | response: `parent_order_id`, `supplier_orders[]` |
| Parent rollup | `GET /v1/retailer/parent-orders/{id}` | on-read status rollup |
| Pay at delivery | `POST /v1/order/card-checkout` or cash-checkout | **not** pre-delivery B2B/unified payment mode |
| Cancel | `POST /v1/order/cancel` | request-cancel is always forbidden |
| Claims | `POST/GET /v1/orders/{id}/claims`, eligibility GET | needs `claim.file` perm |
| Credit / AR read | credit-profile, relationships, ar/invoices | **no pay AR REST** — settle via delivery cash/credit lifecycle |
| Tracking / QR / cash confirm | tracking GET, qr-payload, confirm-cash | |
| Auto-order | settings + `…/run?mode=shadow|draft|place` | place = manager+ |
| Retail OS | stock, POS, shifts, sections, assist, reports, HQ | capability packs |
| Realtime | `GET /v1/retailer/ws-session` → hub room `retailer:{orgId}` | also promo room via watch |
| Notifications | `/v1/user/notifications*` | dual inbox org + staff |

### 5.2 Contract invariants platforms must not diverge on

1. **RetailerId for orders = org id**, never staff user id — **backend currently violates this on create/checkout (R-P0-1)**; fix is backend-first before client work.  
2. **Empty cart → 422**, not empty 200 order.  
3. **Payment only after ARRIVED / AWAITING_PAYMENT.**  
4. **Multi-supplier response shape** always includes `supplier_orders[]`; `parent_order_id` when multi flag on.  
5. **Idempotency-Key** required client-side for create, unified, cancel, claims, POS sale, card checkout.  
6. **No reliance on stub handlers** (cards, AI correct, request-cancel success, cash-checkout as ledger).  
7. **WS room key = org RetailerId** matching order.RetailerID field.

Client SoT nav inventory: `docs/FEATURES_BY_APP_ROLE.md` §1 (retailer).

---

## 6. Proposed fixes (audit only — do not implement here)

### P0

1. **Org scope helper for order/payment money paths**  
   - Replace `claims.Subject` with `auth.ResolveRetailerOrgID(claims)` (and forbid empty) in:  
     - `order.HandleCreate` / `Create` caller sites  
     - `order.HandleUnifiedCheckout` / cancel ownership  
     - `payment` card/cash checkout retailerID  
   - Align cancel/update ownership and CheckoutOrderContext retailer match.  
   - Add unit tests with `Subject=user-1`, `RetailerOrgID=ret-1`.

2. **Cash checkout**  
   - Either implement real session/intent + outbox (mirror card), or return **409/501** with explicit `cash_collection_driver_only` so clients do not treat response as ledger.  
   - Prefer driver collect-cash as SoT (already Class A-ish).

3. **Delete or hard-fail stubs** when real services wired:  
   - `HandleCancelOrder` / `HandleRequestCancel` / `HandleCreateOrder` / `HandleUnifiedCheckout` in `mobile_compat` / core stub cancel — return 503 if reached.  
   - Fail bootstrap if PaymentService/OrderService nil in production run-mode.

4. **ParentOrders**  
   - Emit parent lifecycle outbox in same txn as insert/update, **or** document intentional deferral + stop on-read silent update (or make update evented).

### P1

5. Dispatcher cases for `POS_*`, `STORE_STOCK_*`, `AR_INVOICE_*` → `broadcastRetailer`.  
6. Cart POST: emit `EventCartSyncUpdated` in RW txn (or document “pull-only cart” and remove dead consumer).  
7. POS: single RW txn stock + sale + outbox.  
8. Multi-supplier: evaluate single parent-scoped saga / stronger compensation + parent outbox.  
9. Auto-order run: Idempotency-Key + optional clear cart after unified success.  
10. Cards API: 501/empty with `not_implemented` or wire PSP.  
11. FEATURES / client docs: B2B 410, cash ack semantics, no AR pay route.

### P2

12. Replace demo supplier/warehouse strings in mobile tracking maps with Spanner names.  
13. Fail-closed notification mark-read errors.

---

## 7. Test / E2E coverage notes

| Area | Coverage signal |
|------|-----------------|
| Claims window / residual / hold | `claims/service_test.go`, `eligibility_test.go`, `handlers_idempotency_test.go`; smokecheck `PX_E2E_STORE_STOCK_CLAIM_HOLD_OK` |
| Retailer cancel request disabled | `order/retailer_request_cancel_test.go` |
| Unified / multi-supplier | `order/unified_checkout_test.go` (verify flag paths) |
| Money path gate | `order/money_path_gate_test.go` |
| Retailer mutation idempotency | `retailer/mutation_idempotency_test.go` (profile, preorder) — **not create/checkout** |
| POS / stock / HQ | package tests under `retailer/` |
| Dispatcher retailer fanout | `kafka/notification_dispatcher_test.go` |

**Missing tests for Class A closure:** multi-user org Subject vs OrgID on create/checkout; cash checkout non-persistence assertion; ParentOrders outbox; POS atomicity; dispatcher POS/STORE_STOCK fanout.

---

## 8. Summary scorecard

| Domain | Class A |
|--------|---------|
| Order create (+ outbox + cache + hub) | **PARTIAL** (org scope P0) |
| Unified / multi-supplier checkout | **PARTIAL** (scope + parent silent + non-atomic) |
| Cancel | **PARTIAL** (real path good; org scope; request-cancel intentional 403) |
| Claims file | **PASS** |
| Card checkout session | **PARTIAL** (scope) |
| Cash checkout | **FAIL** |
| Credit leave / AR open (driver) | **PASS** producer; AR pay-down on cash collect **partial fail-open** |
| AR list (retailer) | **PASS** read; no pay mutator (document) |
| Cart | **FAIL** silent |
| POS / store stock | **PARTIAL** (writes + outbox; realtime hole; POS multi-txn) |
| Auto-order place | **PARTIAL** |
| RetailerHub for orders/claims/shop-closed/credit/payment | **PASS** |
| One contract × 3 platforms | **Blocked on P0 org scope + cash/cards honesty** |

**Audit complete.** Implementation deferred to fix phase.
