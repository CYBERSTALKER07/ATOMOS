# End-Product Reality Report — Money / Fiscal / Orders / Concurrency / Outbox

**SoT:** `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go`  
**Verdict:** Delivery money-path (cash/card/credit leave → payment legs → fiscal hard-gate → COMPLETED) is substantially implemented in Spanner + transactional outbox. Tax-authority OFD, live payout rails, doorstep credit negotiation, escrow/wallet, and true AP are not product-complete. Several money side-effects remain post-commit / fail-open.

---

## 1. Order state machine

### Wired & live

**Canonical graph** enforced in `ValidateStatusTransition` (`order/state_machine.go:14-80`):

| From | Allowed to |
|------|------------|
| `PENDING` | `LOADED`, `CANCELLED`, `DELAYED` |
| `LOADED` | `IN_TRANSIT`, `CANCELLED`, `CANCEL_REQUESTED`, `DELAYED`, `PENDING` |
| `DELAYED` | `PENDING` |
| `IN_TRANSIT` | `ARRIVED`, `CANCELLED`, `CANCEL_REQUESTED`, `PENDING` |
| `ARRIVED` | `AWAITING_PAYMENT`, `PENDING_CASH_COLLECTION`, `DELIVERED_ON_CREDIT`, `CANCEL_REQUESTED`, `SHOP_CLOSED_PENDING` |
| `SHOP_CLOSED_PENDING` | payment/credit/cancel/`ARRIVED`/`CANCEL_REQUESTED` |
| `DELIVERED_ON_CREDIT` | `FISCALIZING` only |
| `AWAITING_PAYMENT` | `FISCALIZING`, `PENDING_CASH_COLLECTION`, `DELIVERED_ON_CREDIT` |
| `PENDING_CASH_COLLECTION` | `FISCALIZING` |
| `FISCALIZING` | `COMPLETED`, `FISCAL_FAILED` |
| `FISCAL_FAILED` | `FISCALIZING`, `COMPLETED` |
| `CANCEL_REQUESTED` | `CANCELLED`, `LOADED`, `IN_TRANSIT`, `ARRIVED` |
| `COMPLETED` | none |
| `CANCELLED` | `RECONCILIATION_REQUIRED` |
| preorder | `SCHEDULED` / `AUTO_ACCEPTED` / `BACKORDERED` paths |

Status constants: `order/service.go:50-68`.

**Who triggers (enforced at handlers, not in the graph):**

| Actor | Path | Evidence |
|-------|------|----------|
| Admin | generic `UpdateStatus` | `order/service.go:1581-1583` (ADMIN or RETAILER) |
| Retailer | cancel only via `UpdateStatus` | `order/service.go:1584-1592` |
| Driver | arrive / offload / cash / credit / complete | `transitionDriverOrder` `order/service.go:2238+`; credit leave `order/driver_edges.go:202-210` |
| Warehouse | `warehouseTransition` | `order/warehouse_ops.go:95-162` |
| Fiscal consumer | `FISCALIZING` → `COMPLETED` / `FISCAL_FAILED` | `order/consumer.go:44-51` → `ApplyFiscalWorkerResult` |
| Force-complete | ADMIN / WAREHOUSE_ADMIN | `order/fiscal.go:754-768` |

**Hard gates that are real (not decorative):**

- No soft `COMPLETED` without fiscal `SUCCESS` or `FORCE_SKIPPED` on generic status patch (`order/service.go:1599-1609`).
- ADR-009: `ARRIVED` cannot go directly to `COMPLETED` (`order/state_machine.go:33-35`).
- Order optimistic concurrency CAS on `Version` (`order/repository_spanner.go:220-268`, `1543-1591`).

### Present but decorative / weak

- `TransitionOpts` fields (`Actor`, `PhotoURL`, `SupervisorToken`, `SkipProximity`) are **unused** inside `ValidateStatusTransition` (`order/state_machine.go:5-20`). Proximity/supervisor are enforced only in specific service paths (e.g. credit leave `ensureProximityUnlocked`).
- Generic admin `UpdateStatus` can walk any allowed edge **without** money/fiscal side-effects that dedicated money APIs attach — graph-only mutation (`order/service.go:1560-1656`).
- Timeline table `OrderStatusTransitions` is **best-effort, separate Apply** — not same-txn with order (`order/status_timeline.go:44-62`).

### Missing

- No first-class **field-agent / sales-rep** role on the order graph.
- No doorstep **credit-terms negotiation** state (only quantity negotiation — see §6).

---

## 2. Money handling

### Wired & live

| Capability | Reality | Refs |
|------------|---------|------|
| **Cash collection** | Driver geofence/proximity → `OrderPaymentLegs` CAPTURED (`cash-<orderId>`) → `FISCALIZING` + outbox | `CollectCash` `order/service.go:2029-2235`; leg key `2171-2182` |
| **Card capture** | PENDING leg `card-capture-<orderId>` then sync PSP capture; CAPTURED only after provider confirm | `CompleteOrder` `1901-2026`; `settleOutstandingCardPayment` `settlement_hardening.go:397-449`; `finalizeCardSettlement` `315-343` |
| **Credit leave** | ARRIVED → `DELIVERED_ON_CREDIT`; credit balance in-txn; payment leg `credit-leave-<orderId>` | `driver_edges.go:202-329`; `credit/reserve.go:64-77`; `credit_guard.go:9-21` |
| **AR open items** | Invoice at credit leave; pay-down on cash collect; aging/dunning | `ar/service.go:43-76`, `109-156`, `166-198`; cash pay-down `order/service.go:2188-2199` |
| **Payment ledger** | Immutable `PaymentLedgerEntries` with sessions/chargebacks/webhooks | `payment/repository_spanner.go:341+`, `575+`; unique index migration `schema/migrations/20260816_payment_idempotency_indexes.ddl:11-16` |
| **Payment legs idempotency** | Unique index on `OrderPaymentLegs.IdempotencyKey` | same DDL `:8-9` |
| **PSP webhooks** | Global Pay, Adyen, Stripe, Click, Payme | `payment/global_pay_webhook.go`, `adyen_webhook.go`, `stripe_webhook.go`, `click_webhook.go:28`, `payme_webhook.go:25` |
| **Checkout / COD select** | Retailer cash → `PENDING_CASH_COLLECTION` | `SelectCashAtDelivery` `order/service.go:3817+`; `payment/retailer_checkout.go:204+` |
| **Cash recon (driver shift)** | Expected vs declared; accept/write-off | `cashrecon/service.go:26-136` |
| **Refunds / settlement exceptions** | Card refund path + cash shortfall exceptions | `order/refunds.go`; shortfall in `CollectCash` `2151-2165` |
| **FX settlement helpers** | Present | `payment/settlement_fx.go` |
| **Platform SaaS billing → AR** | Monthly PLATFORM→supplier AR via same AR engine | `internal/services/billing/invoice_worker.go:17-115` |

### Present but decorative / stub / broken-for-live

| Item | Evidence |
|------|----------|
| **Default “fiscal” is not tax OFD** | `PegasusReceiptProvider`: `legal_class=platform_receipt`, `tax_ofd=false` (`order/fiscal_provider_pegasus.go:12-15`, `75-79`) |
| **GLOBAL_PAY stub mode** | Fabricated capture/refund when non-prod + `GLOBAL_PAY_STUB_MODE` (`payment/global_pay_executor.go:64-74`, `278-285`); production refuses stub (`money_path_gate_test.go`) |
| **Payout live rail** | `railByName` always returns `BankFileRail`; comment: “No live rail is implemented yet” (`payout/rail.go:54-64`). Live dispatch fails closed (`ErrNoLiveRail` `:33-35`, `:90-93`) |
| **Click/Payme** | Webhooks exist; **not** registered in `ProviderExecutionRouter` executors (only GLOBAL_PAY, ADYEN, STRIPE, CASH, INTERNAL, CREDIT, AIRWALLEX — `payment/execution.go:140-174`) |
| **Credit risk scoring** | Explicit no-op stub (`credit/repository.go:616-620`) |
| **AR open after credit leave** | **Separate txn after order commit**; failures only logged (`driver_edges.go:353-365`) — order can be on credit with no invoice |
| **AR pay-down on cash** | Fail-open post-commit (`order/service.go:2188-2199`) |
| **Credit ClearBalance after card settle from credit** | Post-commit, log-only on failure (`order/service.go:2008-2013`) |
| **OpenFromCreditLeave when AR disabled** | Returns empty invoice, nil error (`ar/service.go:111-113`); credit leave path separately rejects if AR off (`driver_edges.go:265-270`) |

### Missing

- **Escrow** — no matches in backend-go.
- **Wallet** — no matches.
- **Accounts payable (AP)** as a domain — none; partner “journals” are export mappings, not an AP subsystem (`partner/export_journals.go:34-61`).
- **Live bank/payout API** moving supplier funds.
- Doorstep **credit limit negotiation** (limits are supplier-portal policy, not stop-level bargain).

---

## 3. Fiscal reality

### Wired & live

- Hard-gate statuses + receipt attempts + worker (`order/fiscal.go:420-588`; consumer `order/consumer.go:44-51`).
- Provider selection via `FISCAL_PROVIDER` (`order/fiscal_provider.go:14-86`).
- **MY_SOLIQ** HTTP adapter + mandatory EDS signer from env (`fiscal_provider.go:123-201`; `fiscal/signer_pkcs12.go`, `signer_env.go`).
- Tax regime snapshots on success (`stampTaxRegimeTxn` `order/fiscal.go:713-751`; package `tax/`).
- Force-complete audited skip (`ForceCompleteOrder` `fiscal.go:754+`).
- Partner journal export from AR + payment ledgers with COA mapping (`partner/export_journals.go:94+`).

### Present but decorative / not tax-complete

| Claim | Evidence |
|-------|----------|
| Product default completes on **platform commercial receipts**, not Soliq | `ProviderFromEnv` default → PEGASUS (`fiscal_provider.go:68-84`); pegasus provider self-documents OFD deferred (`fiscal_provider_pegasus.go:12-15`) |
| FAKE provider for SSMR | `FakeFiscalProvider` `fiscal_provider.go:88-112` |
| GLOBAL_PAY receipt secondary is best-effort | Comment `fiscal_provider.go:21-25` |
| “E-invoice / ЭФактура” as Uzbek legal e-invoice stack | Not found as a dedicated product path; MY_SOLIQ is OFD receipt HTTP, not a full e-factura lifecycle |

### Missing / hard human remaining

- Production **my.soliq.uz** credentials + proven EDS + sandbox→prod contract (adapter ready; default path bypasses it).
- Full **VAT e-invoice / factura** lifecycle beyond line VAT snapshots.
- Guaranteed AR invoice co-commit with credit leave (see gaps).

---

## 4. Outbox pattern

### Wired & live

- Package doctrine: same Spanner RW txn for domain + `OutboxEvents` (`outbox/outbox.go:1-8`, `EmitJSON:107-141`).
- `SpannerTxnBuffer` + `Flush` (`outbox/spanner_txn_buffer.go:9-40`).
- Relay → Kafka, at-least-once, DLQ after attempts (`outbox/relay.go:68-105`, `145-208`; `RecordPublishFailures`).
- Bootstrap wires relay; `runtime_workers.go:21-22` starts it; order consumer started `:40`.
- Topics: `pegasusx-main` (+ optional dual-write to `pegasusx-orders` etc.) (`events/events.go:11-21`; `events/topic_routing.go:9-24`, `99-118`).
- Money-path emits generally use txn buffer (order UpdateOrder, payment writeWithOutbox, AR OpenInvoice flush).

**Consumers (money-relevant):**

- `order.EventConsumer` — fiscal + external payment settle (`order/consumer.go:25-71`)
- Notification dispatcher (fan-out) (`kafka/notification_dispatcher.go`)
- Partner / twin / billing tier consumers (`runtime_workers.go:142-150`)

### Gaps (non-atomic / fail-open)

| Gap | Refs |
|-----|------|
| AR invoice open **after** credit-leave commit | `driver_edges.go:353-365` |
| AR pay-down / credit ClearBalance **after** cash/card settle | `service.go:2008-2013`, `2188-2199` |
| Status timeline / some shop-closed emits ignore errors (`_ = outbox.EmitJSON`) | `status_timeline.go:44`; `retailer_shop_closed.go:150+`; `worker_shop_closed.go:124` |
| Credit `MarkBalanceInTxn` memory fallback is best-effort outside txn | `credit/reserve.go:74-76` |
| Dual-write / domain-topic cutover flags can leave consumers on `TopicMain` only | `topic_routing.go:26-36` |
| Publish then MarkPublished is classic at-least-once (duplicate Kafka possible if mark fails) | `relay.go:173-179` |

---

## 5. Concurrency / idempotency

### Wired & live

| Mechanism | Where |
|-----------|--------|
| Order `Version` CAS | `order/repository_spanner.go:220-268` |
| Credit profile / terms version CAS | `credit/policy.go:39-56`, `:433` |
| AR invoice `Version` on dunning updates | `ar/service.go:60`, repo `UpdateDunning` |
| HTTP `Idempotency-Key` on order money edges & payment checkout | `order/service.go:2467+`, `payment/retailer_checkout.go:99+`, `payment/service.go:1429+` |
| Stable money keys: `cash-`, `card-capture-`, `credit-leave-` | `service.go:1999`, `2178`; `driver_edges.go:307` |
| DB unique indexes on legs + ledger refs | `20260816_payment_idempotency_indexes.ddl` |
| Fiscal attempt SUCCESS short-circuit | `fiscal.go:448-458` |
| Proximity unlock / settlement radius on cash | `CollectCash` `2036-2064` |
| Payout batch idempotency key | `payout/handlers.go:52-73` |

### Weak / decorative

- Redis/HTTP idempotency is TTL-backed; money correctness relies on Spanner unique keys (good) — HTTP layer alone is not enough.
- `FiscalMaxFailedAttempts = 3` logged on retry path (`fiscal.go:52-53`, `661-662`) — worker still sets `FISCAL_FAILED` per failure; ops must retry/force.

---

## 6. Field-agent tradition vs code today

Traditional B2B field agent: **pitch → take order → negotiate credit → collect payment**.

| Traditional job | Code can do today? | Evidence |
|-----------------|-------------------|----------|
| Pitch / relationship sell | **No** dedicated agent CRM/pitch flow | No sales-rep role; `retailer/assist` is **retailer floor help tickets**, not van sales (`retailer/assist.go:29-48`) |
| Take order | **Yes** — retailer/unified/multi-supplier checkout, auto-order, EDI | `order/unified_checkout.go`, `multi_supplier_checkout.go`, `partner/edi/orders.go` |
| Negotiate **qty** at door | **Partial** — driver proposes, supplier resolves (feature-flaggable) | `order/negotiation.go:25-172` (`/v1/delivery/negotiate`) — **not credit** |
| Negotiate **credit terms/limit** at door | **No** — supplier portal policy only | `credit/policy_handlers.go:206-267` (enable/patch terms) |
| Leave goods on credit | **Yes** (driver, proximity, limit check, AR flag) | `driver_edges.go:202-329` |
| Collect cash / card | **Yes** (driver) | `CollectCash`, `CompleteOrder` |
| End-of-day cash bag recon | **Yes** | `cashrecon/service.go` |
| Act as salesman placing orders for shops | **No** first-class “order on behalf of retailer as field agent” money role found |

**Net:** Product is **retailer-self-serve + driver last-mile money**, not a field-sales OS.

---

## Structured summary

### Wired & live

1. Enforced order lifecycle graph + ADR-009 fiscal gate (`order/state_machine.go`, `order/service.go:1599-1609`).
2. Driver money edges: cash / card / credit leave with Spanner payment legs + unique idempotency keys.
3. Same-txn outbox on core order/payment/AR open; Kafka relay + fiscal consumer.
4. AR invoices, aging, dunning; platform SaaS billing reuses AR.
5. Payment ledger + multi-PSP webhook surface; Global Pay capture path with production stub ban.
6. Cash reconciliation; settlement exceptions; refunds.
7. Optimistic concurrency on orders/credit/AR; HTTP + DB idempotency on money paths.
8. MY_SOLIQ adapter + EDS signing **code** exists; tax regime snapshots on fiscal success.

### Present but decorative / stub / broken for “real” fiscal/money

1. Default PEGASUS receipts = platform commercial, **not** tax OFD (`fiscal_provider_pegasus.go`).
2. Payout = bank CSV file rail only; no live money rail (`payout/rail.go:54-64`).
3. Click/Payme webhooks without execution-router capture path.
4. Credit scoring removed/stubbed.
5. AR / credit balance side-effects often **post-commit fail-open**.
6. `TransitionOpts` in state machine unused; timeline audit non-atomic.
7. Assist ≠ field agent; quantity negotiation ≠ credit negotiation.

### Missing

1. Escrow / wallet.
2. True AP subsystem.
3. Live payout / bank API settlement.
4. Field-agent order+credit negotiation product surface.
5. Guaranteed co-atomic credit-leave ↔ AR invoice.
6. Production-proven Soliq OFD as default (credentials + cutover).
7. Full Uzbek e-factura stack beyond OFD receipt adapter + VAT snapshots.

### Hard human requirements remaining

1. Configure and certify **MY_SOLIQ** (TIN, API, EDS) and switch `FISCAL_PROVIDER` off PEGASUS for tax-legal markets.
2. Operate **force-complete / FISCAL_FAILED** queues and compliance exports (`order/compliance_audit.go`).
3. Finance ops for **cash recon** variances and **payout CSV → bank → MarkPaid** (no automated rail).
4. Supplier finance must set credit programs/limits in portal before drivers can leave on credit; AR flag must stay on.
5. Reconcile post-commit AR/credit bookkeeping failures (monitoring — code will not roll back delivery).
6. No software replacement for a human pitcher: pitching and doorstep credit bargaining are outside the codebase.

---

### Exact module index (claim anchors)

| Domain | Paths |
|--------|--------|
| Order SM | `apps/backend-go/order/state_machine.go`, `order/service.go` |
| Money edges | `order/service.go` (CollectCash/CompleteOrder), `order/driver_edges.go`, `order/settlement_hardening.go` |
| Payment | `apps/backend-go/payment/*` especially `execution.go`, `global_pay_executor.go`, `*_webhook.go`, `repository_spanner.go` |
| Credit | `credit/reserve.go`, `credit/policy.go`, `credit/policy_handlers.go`, `order/credit_guard.go` |
| AR | `ar/service.go`, `ar/dunning_worker.go` |
| Fiscal | `order/fiscal.go`, `order/fiscal_provider*.go`, `fiscal/*`, `order/consumer.go` |
| Tax | `tax/*`, `order/fiscal.go:stampTaxRegimeTxn` |
| Outbox | `outbox/outbox.go`, `spanner_txn_buffer.go`, `relay.go`, `kafka_publisher.go` |
| Kafka topics | `events/events.go`, `events/topic_routing.go` |
| Payout | `payout/rail.go`, `payout/store.go`, `payout/handlers.go` |
| Cash recon | `cashrecon/service.go` |
| Journals export | `partner/export_journals.go` |
| Idempotency DDL | `schema/migrations/20260816_payment_idempotency_indexes.ddl` |
| Billing AR | `internal/services/billing/invoice_worker.go` |
| Assist (not field sales) | `retailer/assist.go` |

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
