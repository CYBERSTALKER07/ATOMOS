# Backend Parity — A0 Spine (Cross-Role Bus)

**Date:** 2026-08-12  
**Agent:** A0-SPINE  
**Tree:** `pegasusX` only  
**Phase:** Audit only (no production `*.go` edits)  
**Protocol:** [`BACKEND_PARITY_PROTOCOL.md`](./BACKEND_PARITY_PROTOCOL.md)

## Scope packages

| Package | Role in spine |
|---------|----------------|
| `order`, `orderroutes` | Order create/transition, cash, fiscal hard-gate, shop-closed, refunds |
| `payment`, `paymentroutes`, `webhookroutes` | Checkout (pay-at-delivery), gateway webhooks, ledger |
| `outbox` | In-txn emit + relay → Kafka |
| `kafka` | Notification dispatcher, order consumer, dedup |
| `ws` | Hub rooms + Redis relay subscribers |
| `cache` | Post-commit invalidate + Pub/Sub fanout |
| `idempotency` | Guard/replay for public mutators + webhooks |
| `ar` | AR open / pay / dunning |
| `payout` | Supplier payout batches |
| `fiscal` | OFD strategy/signer (mutations live in `order`) |
| `events` | Event types + domain topic dual-write |
| `notifications` | Inbox format + FCM bridge |
| `runtime_workers.go`, `bootstrap` (workers/relay) | Run-mode: api / worker / all |
| `claims` (file/approve edges only) | CLAIM_FILED / CLAIM_RESOLVED on spine |

---

## 1. Feature inventory (route → service → Class A)

Legend: **Y** = present and correct · **P** = partial · **N** = missing · **—** = N/A

| # | Feature | Route / entry | Auth | Idem | Spanner RW | Outbox same txn | Cache post-commit | Realtime (WS/FCM/inbox) | Class A |
|---|---------|---------------|------|------|------------|-----------------|-------------------|-------------------------|---------|
| 1 | Order create | `POST /v1/order/create` → `HandleCreate` → `Create` | Y `RoleRetailer`; retailer = `claims.Subject` (`orderroutes/routes.go:40`, `service.go:2474-2475`) | P optional key (`guardIdempotency` no-ops if empty, `idempotency_guard.go:48-50`) | Y `CreateOrder` RW (`repository_spanner.go:59`) | Y `ORDER_CREATED` (+ preorder) (`service.go:1393-1427`) | Y retailer/supplier/catalog (`service.go:1460-1465`) | Y via dispatcher `ORDER_CREATED` (`notification_dispatcher.go:116-117`) | **PASS*** |
| 2 | Order status patch | `PATCH /v1/order/{id}/status` | Y admin / retailer cancel-only (`service.go:1581-1590`) | P optional | Y `UpdateOrder` | Y `ORDER_STATUS_CHANGED` (`service.go:1635-1654`) | Y (`service.go:1661-1665`) | Y | **PASS*** |
| 3 | Driver deliver / offload | `POST /v1/order/deliver`, `confirm-offload` | Y driver assignment | P optional | Y `transitionDriverOrder` | Y status + `SETTLEMENT_REQUIRED` / `PAYMENT_REQUIRED` (`service.go:1848-1856`, `1876-1880`) | Y `afterOrderMutation` (`service.go:2403-2404`) | Y | **PASS*** |
| 4 | Cash collect | `POST /v1/order/collect-cash` | Y driver | P optional HTTP; **leg key unstable** (`service.go:2175`) | Y + payment leg + exceptions | Y status + cash variance + `PAYMENT_CLEARED` + `FISCAL_RECEIPT_REQUESTED` (`service.go:2111-2128`) | Y | Y finance + order | **PASS*** (P0 leg key) |
| 5 | Card complete | `POST /v1/order/complete` | Y driver | P optional; leg key stable `card-capture-{orderID}` (`service.go:1997`) | Y PENDING leg | PAYMENT_CLEARED deferred to post-provider (`service.go:1958-1961`, `settlement_hardening.go`) | Y | Y after capture settle | **PASS*** |
| 6 | Fiscal worker | Kafka `FISCAL_RECEIPT_REQUESTED` → `ApplyFiscalWorkerResult` | worker | Y attempt SUCCESS short-circuit (`fiscal.go:448-458`) | Y | Y fail/success + finalized + buyer pending (`fiscal.go:526-578`) | Y `afterOrderMutation` | Y fiscal events in primary switch (`notification_dispatcher.go:110-114`) | **PASS** |
| 7 | Fiscal retry | `POST /v1/order/{id}/fiscal/retry` | Y driver/admin/wh | **N** no `guardIdempotency` (`service.go:3033-3058`) | Y | Y status + `FISCAL_RECEIPT_REQUESTED` (`fiscal.go:686-697`) | Y | Y | **PARTIAL** |
| 8 | Force complete | `POST /v1/order/{id}/force-complete` | Y admin/wh admin + reason enum | **N** no HTTP idem | Y | Y status + `ORDER_FORCE_COMPLETED` + finalized (`fiscal.go:865-878`) | Y | Y `ORDER_FORCE_COMPLETED` | **PARTIAL** |
| 9 | Shop closed report | `POST /v1/delivery/shop-closed` | Y driver assigned + ARRIVED | P optional | Y attempts/log + order status | Y `SHOP_CLOSED` (`shop_closed.go:237-291`) | Y `invalidateOrderCache` (`shop_closed.go:323`) | Y + local broadcast | **PASS*** |
| 10 | Shop closed resolve | `POST /v1/supplier/shop-closed/resolve` | Y admin | P optional | Y | Y resolve events | Y | Y | **PASS*** |
| 11 | Credit leave (dedicated) | `HandleCreditLeave` | Y driver | P; **stable** leg key `credit-leave-{orderID}` (`driver_edges.go:307`) | Y | Y status only — **not** `CREDIT_LEAVE` (`driver_edges.go:320-328`) | Y | partial (status only) | **PARTIAL** |
| 12 | Credit delivery edge | `HandleCreditDelivery` | Y driver | P; **unstable** leg key embeds `newID()` (`driver_edges.go:509`) | Y | Y `CREDIT_DELIVERY_MARKED` + `CREDIT_LEAVE` (`driver_edges.go:489-496`) | Y | Y shop-closed family | **PARTIAL** |
| 13 | AR open (from credit leave) | post-commit `OpenFromCreditLeave` | caller-gated | Y per-order get-or-create (`ar/service.go:114-118`) | Y + ledger OPEN | Y `AR_INVOICE_OPENED` (`ar/service.go:440-452`) | **N** | **N orphan** (dispatcher default nil) | **PARTIAL** |
| 14 | AR pay-down | post cash collect `RecordPaymentForOrder` | internal | Y idem key `ar-cash-collect-{orderID}` (`service.go:2192-2193`) | Y | Y `AR_INVOICE_PAYMENT` / `SETTLED` (`ar/service.go:598-610`) | **N** | **N orphan** | **PARTIAL** |
| 15 | AR aging pass | `RunAgingPass` / dunning worker | worker | N | Y `Apply` bulk (`ar/service.go:704-738`) | **N silent** | N | N | **FAIL** (silent) |
| 16 | Payment webhook GP/Adyen/Stripe/Payme/Click | `webhookroutes` `POST /v1/webhooks/*` | secret/signature | Y webhook key + body hash (`global_pay_webhook.go:74-77`, `service.go:1409-1425`) | Y `SaveWebhook` | Y `PAYMENT_*` (`service.go:1353-1373`) | Y payment keys (`service.go:1383-1392`) | Y / P (`PAYMENT_FAILED` via parity path) | **PASS*** |
| 17 | Retailer card/cash checkout | payment order checkout handlers | retailer | P optional header | Y session+attempt | Y finance event (`retailer_checkout.go:358+`) | Y | Y | **PASS*** |
| 18 | B2B/unified pre-delivery checkout | `POST /v1/checkout/*` | retailer | — | — | — | — | — | **GONE** (`service.go:1348-1350`) intentional |
| 19 | Payout generate | `POST /v1/supplier/payouts/batches` | **P** `RoleAdmin` only; **body `supplier_id` not tenant-bound** (`payout/handlers.go:62`) | Y period + key (`payout.go:121-129`) | Y | Y `PAYOUT_BATCH_GENERATED` (`store.go:23-24`) | N | **N orphan** | **PARTIAL** (auth P0) |
| 20 | Payout export / dispatch / mark-paid | payout routes | admin | state-machine | Y | Y exported/dispatched/paid (`store.go:128-135`) | N | **N orphan** | **PARTIAL** |
| 21 | Claim file | `POST /v1/orders/{id}/claims` | retailer/admin | P optional | Y | Y dual TopicExceptions+Main (`claims/service.go:517-520`) | N | Y claim cases (`notification_dispatcher.go:137-140`) | **PASS*** |
| 22 | Claim approve | `POST /v1/claims/{id}/approve` | admin/wh | **N** HTTP; CAS + deterministic chargeback id | Y; **UNDER_REVIEW silent** (`claims/service.go:763`) then resolved+outbox (`808-831`) | Y only on terminal resolve | N | Y `CLAIM_RESOLVED` | **PARTIAL** |
| 23 | Claim reject | `POST /v1/claims/{id}/reject` | admin/wh | N HTTP | Y | Y | N | Y | **PARTIAL** |
| 24 | Refund initiate | `POST /v1/order/{id}/refunds` | admin/wh | body/DB key (`refunds.go:135-145`) | Y | Y `REFUND_*` + corrective | N | **N orphan** | **PARTIAL** |

\*PASS with optional Idempotency-Key (not enforced via `guardIdempotencyStrict` — defined but **unused** anywhere in tree).

---

## 2. Gaps (P0 / P1 / P2)

### P0 — money, auth IDOR, silent state, data corruption risk

| ID | Gap | Evidence | Impact |
|----|-----|----------|--------|
| P0-1 | **Cash payment leg idempotency key is non-stable** (`cash-{orderID}-{newID}`) | `order/service.go:2169-2178` | Without HTTP `Idempotency-Key`, double-submit can double-record cash legs despite unique index on different keys |
| P0-2 | **Credit-delivery path unstable leg key** (`credit-leave-{orderID}-{newID}`) | `order/driver_edges.go:503-512` | Same double-credit risk; contrasts with stable key at `driver_edges.go:307` |
| P0-3 | **Payout generate trusts body `supplier_id` without tenant/home-node binding** | `payout/handlers.go:47-62` | Supplier admin can generate batches for another supplier if JWT role is ADMIN and body is attacker-controlled |
| P0-4 | **AR open after credit leave is fail-open post-commit** | `driver_edges.go:353-365`, `547-558`; cash AR pay-down fail-open `service.go:2186-2195` | Order can be `DELIVERED_ON_CREDIT` with captured credit leg but **no AR invoice** (or unpaid AR after cash) — bookkeeping hole |
| P0-5 | **Claim approve intermediate `UNDER_REVIEW` is silent Spanner write** | `claims/service.go:758-765` (`emit` nil) | Concurrent observers / partners see status change with no outbox; if settlement fails, claim stuck UNDER_REVIEW with no event |

### P1 — missing realtime fanout, incomplete transitions, run-mode/contract

| ID | Gap | Evidence | Impact |
|----|-----|----------|--------|
| P1-1 | **Orphan events (emitted, dispatcher no-ops)** | See §4 | No WS/FCM/inbox for AR, payout, refunds, buyer acceptance, fiscal corrective |
| P1-2 | **`PAYMENT_FAILED` not in primary switch** — only parity string path | `notification_dispatcher.go:110-114` vs `notification_dispatcher_parity.go:114-116` | Works today via default→parity; fragile if parity default changes |
| P1-3 | **Domain dual-write omits AR / payout / buyer acceptance** | `events/topic_routing.go:101-143` | Domain consumers never see AR/payout even when `KAFKA_TOPIC_DUAL_WRITE=true` |
| P1-4 | **Fiscal retry / force-complete / claim approve-reject / refund HTTP lack idempotency guard** | `service.go:3033-3086`, `claims/handlers.go:59-109`, `refund_handler.go:17-56` | Double-submit UX; force-complete mints new FORCE attempt rows |
| P1-5 | **HandleCreditLeave emits only `ORDER_STATUS_CHANGED`**, not `CREDIT_LEAVE` / `CREDIT_DELIVERY_MARKED` | `driver_edges.go:320-328` vs `489-496` | Two credit-leave entry points diverge for consumers |
| P1-6 | **Sibling-driver WS is in-process only** | `service.go:2414-2433` `driverHub.Broadcast` without outbox | Multi-pod drivers miss split-shipment payment-complete signals |
| P1-7 | **Optional Idempotency-Key on money mutators** | `idempotency_guard.go:48-50`; `guardIdempotencyStrict` unused | Class A requires guard on public mutators; clients can omit keys |
| P1-8 | **api-only + no worker + Redis heartbeat miss** | `runtime_workers.go:219-236`; outbox relay only in `startBackgroundWorkers` | If worker down and heartbeat still “live”, **no** notification consumer; if worker absent, api safety-net starts **consumer only**, not outbox relay — stuck events |
| P1-9 | **AR list-only HTTP** — no retailer/admin pay-AR mutator | `ar/handlers.go` list only; pay via cash collect internal | Manual AR settlement API missing for non-cash paths |

### P2 — polish / dead code / naming

| ID | Gap | Evidence |
|----|-----|----------|
| P2-1 | Pre-delivery checkout endpoints return 410 | `payment/service.go:1348-1350` |
| P2-2 | `TopicWebhooks` marked retired | `events/topic_routing.go:16-19` |
| P2-3 | `guardIdempotencyStrict` dead | only defined `order/idempotency_guard.go:96-103` |
| P2-4 | Webhook inbox Enqueue is Spanner-only (no outbox) — acceptable ops table | `payment/webhook_inbox.go:48-59` |
| P2-5 | Cache invalidate best-effort (logged, not returned) | `cache/cache.go:114-130` by design |

---

## 3. Event / consumer matrix (spine-critical)

| Event type | Producer (file:line) | Topic(s) | Consumers | Dispatcher fanout |
|------------|----------------------|----------|-----------|-------------------|
| `ORDER_CREATED` | `order/service.go:1394` | Main (+ Orders dual) | twin, partner?, dispatcher | primary → order parties |
| `ORDER_STATUS_CHANGED` | `service.go:1636`, `emitOrderStatusChanged:2384` | Main (+ Orders) | dispatcher | primary |
| `PAYMENT_REQUIRED` / `SETTLEMENT_REQUIRED` | `service.go:3182-3187` | Main (+ Orders) | dispatcher | primary finance |
| `PAYMENT_CLEARED` | cash `emitPaymentCaptureFiscal` `fiscal.go:413`; webhook `payment/service.go:1356`; settlement | Main (+ Orders) | **order consumer** `order/consumer.go:32` → `SettleExternalPayment`; dispatcher | primary |
| `PAYMENT_FAILED` | webhook `payment/service.go:1357-1358` | Main (+ Orders) | order consumer `consumer.go:53` | **parity only** |
| `FISCAL_RECEIPT_REQUESTED` | cash/card/retry | Main (+ Orders) | **order consumer** `consumer.go:44` → OFD | primary |
| `FISCAL_RECEIPT_SUCCEEDED` / `FAILED` | `fiscal.go:534,570` | Main (+ Orders) | dispatcher | primary |
| `ORDER_FORCE_COMPLETED` / `ORDER_FINALIZED` | `fiscal.go:876-878` | Main (+ Orders) | dispatcher | primary / order |
| `CASH_SHORTFALL` / `CASH_OVERAGE` | `fiscal.go:258` emitCashVariance | Main (+ Orders) | dispatcher | primary finance |
| `BUYER_ACCEPTANCE_*` | `fiscal.go:573,620` | Main only | **none declared** | **orphan** |
| `SHOP_CLOSED*` / `PROXIMITY_UNLOCKED` / `PARTIAL_OFFLOAD` / `CREDIT_LEAVE` | shop_closed / driver_edges | Main (+ Orders) | dispatcher | primary shop-closed |
| `CLAIM_FILED` / `CLAIM_RESOLVED` | `claims/service.go` dual Main+Exceptions | Main + Exceptions | dispatcher | primary driver-edge |
| `AR_INVOICE_*` | `ar/service.go:442,602,677` | Main only | **none** | **orphan** |
| `PAYOUT_BATCH_*` | `payout/store.go:23,128-134` | Main only | **none** | **orphan** |
| `REFUND_*` / `FISCAL_CORRECTIVE_REQUESTED` | `order/refunds.go:259+` | Main (+ Orders for refund) | **none realtime** | **orphan** |
| `payment.webhook.*` sources | webhook persist | Main as finance type | order mutator on cleared/failed | as above |

### Outbox relay

- Loop: 250ms tick, batch 100, publish retry + DLQ after max attempts (`outbox/relay.go:13-33`, `89-105`, `145-208`).
- Dual-write: `RelayPublishTopics` (`events/topic_routing.go:146-161`).
- Started only when `RunsWorkers()` (`runtime_workers.go:21-23`, `main.go:100-101`).

### Notification dispatcher switch coverage vs declared types

**Primary switch** covers core order/payment-cleared/fiscal/shop-closed/claims/credit-profile (`notification_dispatcher.go:99-207`).

**Not in primary or parity (silent drop `return nil`)** — see §4 orphan list.

**Parity catch-all** handles legacy aliases + `PAYMENT_FAILED` string (`notification_dispatcher_parity.go:82-180`).

### Worker vs api run-mode (push/inbox)

| Mode | Outbox relay | Notification consumer | WS hub Redis relay | Cache invalidate sub |
|------|--------------|----------------------|--------------------|----------------------|
| `all` | yes | yes | yes | yes |
| `worker` | yes | yes | no public API | yes |
| `api` | **no** | only if no worker heartbeat (`runtime_workers.go:223-236`) | yes | **no** (subscriber is worker-only) |

**Implication:** api-tier alone without worker loses **outbox drain** (events stuck unpublished). Cache cross-pod invalidate subscriber also worker-only (`runtime_workers.go:31-33`).

### Redis cache invalidate patterns

Canonical API: post-commit `cache.Invalidate` → local DEL + PUBLISH `cache:invalidate` (`cache/cache.go:114-130`).  
Order keys: `retailerOrdersKey` / `supplierOrdersKey` / sometimes `catalog:products:{supplier}` (`service.go:1461-1464`, `afterOrderMutation:3159-3168`).  
Payment: `paymentOrderKey` / session / retailer (`payment/service.go:1383-1391`, `retailer_checkout.go:378`).  
AR/payout/claims: **no** cache invalidation (may be acceptable if uncached).

### Webhook idempotency

| Gateway | Key shape | Replay | Outbox |
|---------|-----------|--------|--------|
| Global Pay | `webhook:global_pay:{txn}:{status}` | `writeWebhookReplayIfExists` | yes |
| Stripe / Click / Payme | similar | yes | yes |
| Adyen | `isWebhookReplay` per item | yes | yes |
| Failure path | enqueue `WebhookInbox` + reconciler worker | process re-enters `persistWebhookWithOutbox` | yes on success |

Auth: shared secret / signature validation before persist (e.g. `global_pay_webhook.go:43-59`).

---

## 4. Silent mutations and orphan events (file:line)

### Silent Spanner mutations (state write without outbox)

| Location | What | Why silent |
|----------|------|------------|
| `ar/service.go:704-738` `RecomputeAging` | Updates `AgingBucket`/`Version` via `client.Apply` | No outbox; aging changes invisible to bus |
| `claims/service.go:763` | CAS claim → `UNDER_REVIEW` with `emit=nil` | Money path still pending; no event until resolve |
| `payment/webhook_inbox.go:48-59` | `WebhookInbox` insert | Operational retry table (acceptable) |
| `payment/webhook_inbox.go:116-118` | Inbox delete after success | Same |
| `order/service.go:1432-1437` | Credit reserve post-create | Separate package txn (has own outbox if credit emits); not same as order create txn |
| `order/driver_edges.go:353-365` etc. | AR open **after** order commit | Separate AR txn (has outbox) but decoupled from leave txn |

### Orphan events (outbox/Kafka produced; notification dispatcher drops)

| Event | Producer | Dispatcher fate |
|-------|----------|-----------------|
| `AR_INVOICE_OPENED` | `ar/service.go:442` | default → parity default → **nil** |
| `AR_INVOICE_PAYMENT` | `ar/service.go:598-602` | orphan |
| `AR_INVOICE_SETTLED` | `ar/service.go:599-601` | orphan |
| `AR_INVOICE_DUNNED` | `ar/service.go:677` | orphan (dunning also has direct notify hook in bootstrap) |
| `PAYOUT_BATCH_GENERATED` | `payout/store.go:23` | orphan |
| `PAYOUT_BATCH_EXPORTED` | `store.go:130` | orphan |
| `PAYOUT_BATCH_DISPATCHED` | `store.go:132` | orphan |
| `PAYOUT_BATCH_PAID` | `store.go:134` | orphan |
| `REFUND_REQUESTED` | `order/refunds.go:259` | orphan |
| `REFUND_SUCCEEDED` / `REFUND_FAILED` | `refunds.go:304-306` finalize path | orphan |
| `FISCAL_CORRECTIVE_REQUESTED` | `order/refunds.go` (~495) | orphan |
| `BUYER_ACCEPTANCE_PENDING` | `order/fiscal.go:573` | orphan |
| `BUYER_ACCEPTANCE_ACCEPTED/REJECTED/EXPIRED` | buyer poller path | orphan if emitted |

### Near-orphan / fragile

| Event | Note |
|-------|------|
| `PAYMENT_FAILED` | Handled only via parity string case, not primary switch (`notification_dispatcher.go:110` vs parity `114`) |
| Direct hub `OTHER_TRUCK_ON_WAY` / `PAYMENT_COMPLETED` | Not Kafka events; single-pod only (`service.go:2419-2433`) |

---

## 5. Edge-case matrix

| Scenario | Expected Class A | Observed | Severity |
|----------|------------------|----------|----------|
| Double cash collect same key | One leg, one fiscal | HTTP idem replay if key present; **without key → multiple legs** | P0 |
| Double card complete | Stable leg key + provider settle | Stable `card-capture-{orderID}` (`service.go:1997`) + settle retry on NoChange (`2013-2017`) | OK |
| Webhook replay | 200 + no double money | Idem store + tests `service_webhook_handlers_test.go` | OK |
| Webhook persist fail | Inbox retry | `persistWebhookWithOutbox` enqueue (`service.go:1375-1378`) | OK |
| Fiscal OFD fail max | `FISCAL_FAILED` + event | `ApplyFiscalWorkerResult` (`fiscal.go:512-540`) | OK |
| Force complete after SUCCESS | Reject | `ErrFiscalAlreadySucceeded` (`fiscal.go:779-785`) | OK |
| Soft COMPLETED via status patch | Blocked without fiscal SUCCESS | `service.go:1600-1607` | OK |
| Credit leave AR disabled | Reject leave | `driver_edges.go:268-270` | OK |
| Credit leave AR open fail | Fail-open log | `driver_edges.go:363-365` | P0 bookkeeping |
| Claim approve settlement fail | Stay UNDER_REVIEW | `claims/service.go:780-783` silent intermediate | P0/P1 |
| api-only deploy | Push works | Safety-net consumer; **relay still missing** | P1 |
| Domain consume without dual-write | Events only on Main | `ConsumeDomainTopics` requires dual-write note (`topic_routing.go:32-36`) | ops |
| Cancel after shop-closed max retries | UpdateStatus cancel | `shop_closed.go:295-297` (admin actor system) | OK |

---

## 6. Proposed fixes (do not implement in audit phase)

1. **Stable money leg keys:**  
   - Cash: `cash-collect-{orderID}` (or include received amount band if partial re-collect allowed).  
   - Credit delivery: align with `credit-leave-{orderID}` (`driver_edges.go:307`).  
2. **Require Idempotency-Key** on money mutators via `guardIdempotencyStrict` (create, cash, complete, claims approve, refunds, fiscal force/retry).  
3. **Payout auth:** bind `supplier_id` from `claims.SupplierID` / `PreferTenantSupplierID`; reject body mismatch.  
4. **AR open in same RW txn as credit leave** (or compensating saga); do not fail-open without alert metric.  
5. **Dispatcher cases** for `AR_INVOICE_*`, `PAYOUT_BATCH_*`, `REFUND_*`, `BUYER_ACCEPTANCE_*`, `FISCAL_CORRECTIVE_*`; promote `PAYMENT_FAILED` into primary switch.  
6. **DomainTopicForEventType:** map AR/payout/buyer to finance/orders domain topics.  
7. **Claim UNDER_REVIEW:** emit `CLAIM_STATUS_CHANGED` or skip intermediate persist and CAS open→resolved after settlement.  
8. **api run-mode:** either start outbox relay on api when worker heartbeat missing, or hard-fail readiness if unpublished outbox > N and no worker.  
9. **Sibling driver signals:** emit outbox event instead of in-process hub only.  
10. **Unify credit leave handlers:** single path emitting `CREDIT_LEAVE` + `CREDIT_DELIVERY_MARKED` + status.  
11. **AR pay HTTP** for non-cash settlement with JWT + idempotency (spine for retailer pay AR).  
12. **Aging pass:** either emit `AR_INVOICE_AGED` or document intentional deferral + metric.

---

## 7. Class A spine scorecard (summary)

| Plane | Status |
|-------|--------|
| Order create/transition outbox | Strong |
| Payment webhook idempotency + outbox | Strong |
| Fiscal hard-gate (ADR-009) worker path | Strong |
| Cash/credit money leg keys | **Weak (P0)** |
| AR/payout bus fanout | **Weak (orphan events)** |
| Claims approve intermediate | Silent mutation |
| Run-mode api without worker | Relay hole |
| Cache post-commit order/payment | Strong |
| Dispatcher coverage spine finance | Incomplete |

**Bottom line:** The order↔payment↔fiscal money spine is largely Class A on happy path (txn + outbox + consumer OFD + webhook idempotency). Cross-role bus gaps concentrate on **AR/payout/refund realtime orphans**, **unstable cash/credit leg idempotency keys**, **payout supplier_id auth**, and **api-only outbox relay absence**.

---

## 8. Evidence index (key anchors)

| Area | Path |
|------|------|
| Protocol DoD | `docs/session-2026-08-12/BACKEND_PARITY_PROTOCOL.md` |
| Order routes | `apps/backend-go/orderroutes/routes.go:40-82` |
| Payment / webhook routes | `paymentroutes/routes.go`, `webhookroutes/routes.go:23-27` |
| Outbox emit | `outbox/outbox.go:114-140` |
| Relay | `outbox/relay.go` |
| Dispatcher | `kafka/notification_dispatcher.go:99-207` |
| Parity | `kafka/notification_dispatcher_parity.go:81-180` |
| Run mode | `bootstrap/run_mode.go`, `main.go:100-110`, `runtime_workers.go` |
| Cache | `cache/cache.go:114-158` |
| Domain topics | `events/topic_routing.go` |
| Event types | `events/events.go:118-271` |

---

*End of A0-SPINE audit.*
