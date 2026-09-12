# 01 Backend Core Correctness

> **HISTORICAL / FROZEN — do not plan from this file alone.**
> Current SoT: [`PROD_READINESS_SEQUENCE.md`](../../PROD_READINESS_SEQUENCE.md) · [`ECOSYSTEM_GAP_REGISTER_2026-08-12.md`](../ECOSYSTEM_GAP_REGISTER_2026-08-12.md) · [`FEATURES_BY_APP_ROLE.md`](../../FEATURES_BY_APP_ROLE.md).
> Body is a point-in-time snapshot; re-verify any claim against code before acting.


_Source: subagent `cf2e366c-bf7d-4302-9926-1e16b52e1d48` from End-Product Reality Report session (2026-08-07)._

# END-PRODUCT REALITY REPORT — PegasusX / ATOMOS `apps/backend-go`

**Ground truth from code only.** Repo: `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go`. Scale: 812 non-test Go files, 265 test files (~25% test ratio), 644 HTTP handler functions, 155-table Spanner DDL (2,624 lines), 87 migration files, 2,485-line `bootstrap/bootstrap.go`. All paths below are relative to `apps/backend-go/`.

---

## 1. ORDER STATE MACHINE — **LIVE & ENFORCED (in code; not at DB level)**

A real central FSM validator exists: `ValidateStatusTransition` at `order/state_machine.go:14-81`, pure Go switch over canonical statuses. Statuses (18) defined at `order/service.go:51-69`:

`PENDING, LOADED, IN_TRANSIT, ARRIVED, SHOP_CLOSED_PENDING, AWAITING_PAYMENT, PENDING_CASH_COLLECTION, DELIVERED_ON_CREDIT, FISCALIZING, FISCAL_FAILED, COMPLETED, CANCELLED, CANCEL_REQUESTED, RECONCILIATION_REQUIRED, DELAYED, BACKORDERED, SCHEDULED, AUTO_ACCEPTED`

Enforcement points:
- `order/service.go:1523` — generic `UpdateStatus` (admin/retailer PATCH), plus ADR-009 fiscal hard-gate at `1527-1538` (COMPLETED requires `FiscalStatus==SUCCESS || FORCE_SKIPPED`).
- `order/service.go:2153` — `transitionDriverOrder`, the funnel for **all** driver mutations (arrive, deliver, complete, collect-cash…).
- `order/preorder_sweeper.go:168,241` — preorder auto-accept/promote.
- Full transition-matrix test coverage: `order/state_machine_test.go`, `simulator/order_lifecycle_test.go` (~290 lines of matrix tests).

**No DB constraint.** `Orders.Status STRING(32)` at `schema/spanner.ddl:163` — the entire 155-table DDL contains exactly **one** CHECK constraint (`spanner.ddl:1273`, supplier import sessions). Enforcement is Go-only.

Ad-hoc direct mutations that bypass the validator but carry hand-rolled guards (manually FSM-consistent today):
- `order/driver_edges.go:972-1014` — shop-closed bypass offload: requires `SHOP_CLOSED_PENDING` (`:972`), then blind-sets `AWAITING_PAYMENT` (`:1014`).
- `order/repository_spanner.go:1225-1260` — `ClearBackorder`: requires `BACKORDERED` (`:1225`), sets `PENDING` (`:1260`).
- `order/worker_shop_closed.go:91,128-165` — timeout worker: guard at `:91`, sets `DELIVERED_ON_CREDIT` (`:130`) or `CANCELLED` (`:161`).
- `order/shop_closed.go:263`, `order/retailer_shop_closed.go:133,201,246,292`.

Illegal transitions **cannot** happen via `UpdateStatus`/driver paths, but nothing stops a future writer from inserting a new ad-hoc mutation. Terminal states: `COMPLETED` hard-blocked (`state_machine.go:57-58`); `CANCELLED → RECONCILIATION_REQUIRED` is the only exit (`:60`).

---

## 2. MONEY HANDLING — **LIVE & ENFORCED (exemplary)**

- **Representation: int64 minor units everywhere.** `Orders.TotalMinor INT64` (`schema/spanner.ddl:167`), `PaymentSessions.AmountMinor` (`:522`), `OrderPaymentLegs.AmountMinor` (`:1665`), `CreditNotes.TotalNetMinor/TotalVatMinor/TotalGrossMinor` (`:1693-1695`), VAT in basis points (`VatRateBps`, `:1714`), AR `PrincipalMinor/BalanceMinor` (`ar/service.go:42-43`).
- **Zero `float64` in money paths** of `payment/`, `credit/`, `ar/`, `fiscal/`, `tax/`, `pricing/`, `fxrates/`, `cashrecon/` production code (grep-verified; hits are tests doing JSON map assertions only). `fiscal/marshal.go:21` comments on deliberately avoiding float64. Order floats are geo-only (Lat/Lng/DistanceM, `order/service.go:162-163`…).
- **FX**: `fxrates/convert.go` — integer-scaled rates (1e8, `:15`), `math/big` arithmetic with half-away-from-zero rounding (`:114-146`), overflow-checked (`:138-144`), **fail-closed** on missing rate (`ErrRateMissing`, `:110` — "never silent 1:1", `:2`), currency-mismatch guard `AssertSameCurrency` (`:60-71`), inverse-rate fallback (`:101-109`). Webhook currency mismatch rejected at `payment/global_pay_webhook.go:101-105`.
- Only float conversion: billing metering minor→major (`internal/services/billing/amount.go:7-13`) — display/metering, not ledger math.
- **One bad spot**: `inventory/repository.go:166-169` silently clamps negative stock to 0 instead of erroring — masks shrinkage drift.

---

## 3. CONCURRENCY & CONSISTENCY — **LIVE & ENFORCED (Spanner-native)**

- **DB: Google Cloud Spanner** (`cloud.google.com/go/spanner` throughout; emulator support). No Postgres, no `SELECT … FOR UPDATE` (Spanner doesn't have it; RW transactions are serializable with reader/writer locks).
- Central helper `spannerutils/retry.go:24-55`: retries `Aborted`/`Unavailable`, 5 attempts, 20ms→500ms exp backoff; fail-loud on nil client (`:14`). Chunked writer for the 80k mutation limit: `spannerutils/chunker.go:18-40`.
- **Optimistic version CAS inside serializable txns**: Orders `Version` re-read + compared in-txn (`order/repository_spanner.go:226-228` and `:1527-1529` — "optimistic concurrency conflict"), credit profiles (`credit/repository.go:238,299,377,485`), inventory (`inventory/repository.go:155,188,252`), AR dunning CAS (`ar/service.go` `UpdateDunning(..., version)`).
- Inventory reservation is read-check-write inside the order txn: `order/inventory_reservation.go:65-92` (aggregate-by-SKU, `qoh-qr < qty` → `ErrInventoryExhausted`), optional FEFO lot allocation under `WMS_LOTS_ENABLED` (`:36-52`). Cancel releases reservations in the same txn (`order/repository_spanner.go:230-248`).
- **Race risks found**:
  - Fire-and-forget card capture in a `go func()` **after** commit, error log-only: `order/service.go:1921-1929`.
  - Shop-closed worker tolerates credit-profile read failure (warn-only, `order/worker_shop_closed.go:99-102`) then still marks `DELIVERED_ON_CREDIT` — with **no balance marked** if the profile load failed.
  - Same worker re-implements balance math inline (`:136-146`), ignoring `ReservedMinor` — inconsistent with `credit` package's `Available()` and double-counts vs. reserve-at-create.

---

## 4. OUTBOX / EVENTS / KAFKA — **LIVE & ENFORCED (real transactional outbox)**

- **True transactional outbox**: `outbox/outbox.go:91-117` `EmitJSON` buffers an `OutboxEvents` row via `TxnBuffer` in the **same Spanner RW txn** as the state change (e.g. `order/repository_spanner.go:253-258`, `order/driver_edges.go:1000-1026`, `credit/reserve.go:31-40`). Doc comment `outbox.go:1-8` states the invariant. Trace-ID injection `:103-108`.
- **Relay**: `outbox/relay.go` — 250 ms tick, batch 100, 5 tries with exp backoff + full jitter (`:155-203`), watchdog flags events stuck >60 s (`:88-122`). **Multi-replica safe via lease claiming** (`outbox/spanner_store.go:87-170`: `ClaimedBy/ClaimedUntil`, 2-min lease, claimed inside a RW txn — the "lease relay" is real).
- **Kafka is real**: `segmentio/kafka-go`, `RequiredAcks=all`, sync writes, hash balancer on aggregate key, `AllowAutoTopicCreation=false`, SASL_SSL/PLAIN for GCP Managed Kafka (`outbox/kafka_publisher.go:79-90`, `kafkautil/auth.go:35-139`). Known limitation documented at `kafka_publisher.go:59-62`: no broker-side idempotent producer → at-least-once + consumer dedup (`kafka/event_dedup_store.go`, `kafka/redis_event_dedup.go`, `event_id` header at `relay.go:205-211`).
- Dual-write to domain topics: `events/topic_routing.go:122` `RelayPublishTopics` (env `KAFKA_TOPIC_DUAL_WRITE`).
- **DLQ**: consumer-side real — `kafka/dlq_writer.go` + `kafka/consumer.go:148,196-210` (sendToDLQ then commit), plus `cmd/replay-dlq`. Webhook inbox has DEAD-after-5-attempts (`payment/webhook_inbox.go:15-19`). **Gap**: the outbox relay itself has no DLQ — exhausted publishes stay unpublished forever with only an error log (`relay.go:135-152`); recovery relies on retry + watchdog alert.

---

## 5. IDEMPOTENCY — **LIVE (middleware-level; partially DB-enforced)**

- Guard: `idempotency/idempotency.go:52-76` — Load → Acquire (60 s in-flight claim) → Save (24 h). Same key + same body → stored response replayed; same key + different body → `ErrConflict` → **409** (`middleware.go:80-82`); in-progress → 409 (`:84-86`).
- **Global middleware** on all mutating methods when `Idempotency-Key`/`X-Idempotency-Key` present (`idempotency/middleware.go:45-120`), mounted at `main.go:129-131`. Key scoped `principal + method + path` (`middleware.go:65-66`).
- Store: in-memory default (`bootstrap/bootstrap.go:419`), Redis when `REDIS_ADDR` set (`:420-424`); `REQUIRE_INFRA_ADAPTERS` (default **true**, `:310`) makes memory fallback fatal in prod (`:426-430`).
- **Soft by default**: no header = pass through (`middleware.go:57-60`); `GuardStrict` exists (`idempotency.go:79-84`) but the middleware doesn't use it. Payment handlers additionally persist their own records (`payment/service.go:643`).
- **DB uniqueness exists only for**: `ArLedgerEntries(IdempotencyKey)` (`schema/spanner.ddl:2457`), webhook delivery attempts (`:2508`), partner EDI (`:2569`), FX rates (`:2620`). **`OrderPaymentLegs.IdempotencyKey` has NO unique index** (`:1667`) and `PaymentLedgerEntries` has none — a replay after the 24 h Redis TTL can double-record a financial leg.

---

## 6. PAYMENTS — **PARTIAL (one real PSP; three decorative; two live correctness bugs)**

- **Real PSP integration: Global Pay only.** `payment/global_pay_executor.go` — merchant auth (`:178-223`), hosted checkout token, capture `CP` (`:250-309`), refund `RF` (`:112-176`), status check — against `checkout-api.globalpay.uz` / backoffice API (`:63-86`), circuit-breaker wrapped.
- **Decorative executors**: ADYEN, STRIPE, CASH, INTERNAL, CREDIT are `staticProviderExecutor` stubs that fabricate **local** redirect URLs (`payment/execution.go:148-170, 332-360`); AIRWALLEX flag-gated stub returning a fake local URL (`:171-173, 366-399`). Webhook *parsers* for Stripe/Adyen exist (`payment/stripe_webhook.go`, `adyen_webhook.go`) but no real executor behind them.
- **Real inbound webhooks**: Click — MD5 signature per Click protocol (`click_webhook.go:53-69`); Payme — HTTP Basic (`payme_webhook.go:39-56`); Global Pay — replay-guarded + **server-side authoritative status re-verification** that fails closed when creds missing (`global_pay_webhook.go:80-91, 138-141`).
- **Payment state model**: `PaymentSessions` + `PaymentAttempts` (`schema/spanner.ddl:515-546`), status normalized at `payment/service.go:1449-1480`. **Partial payments real**: multi-leg `OrderPaymentLegs` interleaved under Orders (`:1661-1672`), shortfall/overage via `OrderSettlementExceptions` (`:1675-1684`), `order/settlement_hardening.go`. **Refunds**: real for Global Pay (`executeRefund`), partial-refund capping at session amount (`payment/service.go:713-716`); chargeback + reversal ledger with role-gated admin endpoints (`paymentroutes/routes.go:33-40`).
- **BUG A (broken capture routing)**: `CaptureCardPayment` sends `Gateway: "GLOBALPAY"` (`payment/service.go:653`) but the executor map is keyed `"GLOBAL_PAY"` (`execution.go:140`); router does only case/whitespace normalization (`execution.go:224`) → lookup miss → `GatewayPolicyError` "unsupported payment gateway" (`:230-236`), no fallback. **Every call fails.** Callers: post-delivery capture goroutine (`order/service.go:1925`, error only logged `:1926`) and backorder sweeper (`order/backorder_sweeper.go:51-55` — deferred-payment orders stay `BACKORDERED` forever).
- **BUG B (optimistic capture)**: `CompleteOrder` records `PaymentLeg{Method: CARD, Status: PaymentStatusCaptured}` **in-txn before any PSP call** (`order/service.go:1899-1909`) and moves the order to `FISCALIZING`; the actual capture is the broken fire-and-forget above. DB asserts money captured that was never collected.
- **Stub-success footguns when GP creds empty** (all return `nil` error): capture → `gp_capture_stub_` (`global_pay_executor.go:251-258`), refund → `gp_refund_stub_` (`:112-120`), status → `gp_status_stub_paid` (`:312-320`).
- **Dormant landmine**: `WebhookReconciler` maps `gp_status_stub_paid` → `PAID` and writes a `SignatureValid: true` webhook + `EventPaymentCleared` (`payment/reconciliation.go:57,66-108`). Not wired (no `NewWebhookReconciler` call anywhere) — but if wired with empty creds, stuck sessions auto-“pay” themselves after 15 min.
- **Cash reconciliation real**: server-computed expected cash vs driver-declared (`cashrecon/service.go:39-57`), client-supplied expectation must match server (`:49-51`), finance accept/write-off with state guard (`:91-136`), nightly escalation worker (`cashrecon/escalation_worker.go`), shift-close gate `HasAcceptedReconciliation` (`:152-161`).

---

## 7. CREDIT / AR — **LIVE (substantially implemented; flag-gated OFF by default)**

The `CREDIT_COLLECTIONS_ENGINE_PLAN.md` "current state" section (no terms/invoices/dunning) is **stale** — the plan was largely executed:

- **Credit spine**: `RetailerCreditProfiles` (limit/balance/**reserved**/available/risk/delinquency/status/version, `schema/spanner.ddl:339`); `CheckOrder` gate (`credit/service.go:49-94` — `no_credit_limit`, `credit_limit_breached`); reserve-at-create with idempotent `OrderCreditReservations` (`credit/reserve.go:14-41`, DDL `:2408`); same-txn `MarkBalanceInTxn` on credit leave (`reserve.go:67-77` → `credit/repository.go:425`, used at `order/driver_edges.go:304,499`) — plan Phase-0 items 1-3 done.
- **Terms & policy**: `SupplierCreditPrograms`, `RetailerPaymentTerms`, `CreditPolicyAudit` (DDL `:2358,2374`); `credit/policy.go` + policy handlers; dunning auto-hold → `creditPolicySvc.HoldRelationship` (`bootstrap/bootstrap.go:1248-1250`).
- **AR open items**: `ar/service.go` — `OPEN/PARTIAL/PAID/VOID` (`:23-27`), aging buckets `CURRENT/1_30/31_60/61_90/90_PLUS` (`:28-33`), `OpenFromCreditLeave` idempotent per order (`:88-96`, unique `Idx_ArInvoices_ByOrder` DDL `:2441`), `ApplyPayment`/`ApplyCreditNote` with idempotency keys (`:65-66`; unique index `:2457`).
- **Dunning engine**: `ar/dunning.go:8-17,41-74` — step machine `DUE_SOON(T−3)→OVERDUE→ESC1(+7)→ESC2(+14)→CREDIT_HOLD(+21)→COLLECTIONS(+30)` with grace; hourly worker (`ar/dunning_worker.go:127-143`), monotonic advancement (`:89`), delinquency bump on first overdue (`:103-109`), auto-hold (`:110-116`), inbox + FCM notify — all wired in `bootstrap/bootstrap.go:1247-1270`.
- **Flag gates**: `AR_INVOICES_ENABLED` (`ar/service.go:18-21`) — when off, `OpenFromCreditLeave` **silently returns nil** (`:89-91`); `AR_DUNNING_ENABLED` (`dunning_worker.go:13-16,66`). Local `.env.local` enables neither. Plan's `ledger/` package still absent (no `apps/backend-go/ledger` dir); `MasterInvoices` remains schema-only (`schema/spanner.ddl:1313`).
- Residual leak: shop-closed timeout worker bypasses `MarkBalanceInTxn` (see §3).

---

## 8. FISCAL / SOLIQ / TAX — **PARTIAL (framework rigorous; legal fiscalization NOT live)**

- **Order fiscal flow is real and well-hardened**: immutable attempt rows (`OrderFiscalReceipts`, `order/fiscal.go:83-104`), `FISCALIZING/FISCAL_FAILED` states, max 3 failed attempts (`fiscal.go:41`), force-complete with closed reason-code set (`fiscal.go:44-71`) role-gated to admin/warehouse-admin (`orderroutes/routes.go:54`), event-driven processing via Kafka consumer (`order/consumer.go:44-52` → `ApplyFiscalWorkerResult` `fiscal.go:342`), cash-bag freeze while fiscal open (`order/fiscal_open.go:23-65`), late-webhook money guard (`fiscal.go:73-81`).
- **BUT the default provider is not a tax fiscalizer.** `ProviderFromEnv` default = `PEGASUS` (`order/fiscal_provider.go:68-84`) issuing **platform commercial receipts** — payload says `"tax_ofd": false` with `OFDDeferredNote` (`order/fiscal_provider_pegasus.go:78-79`); header comment is explicit: *"These are NOT Soliq/OFD tax fiscal receipts… tax OFD (MY_SOLIQ) remains deferred until credentials arrive"* (`:13-15`). Orders reach `COMPLETED` with no OFD/Soliq receipt.
- **MY_SOLIQ adapter is real HTTP but currently unusable**: `order/fiscal_provider.go:123-256` + `soliq/client.go` (EHF submit/status, idempotency-key header, permanent-vs-retry classification, didox operator variant). Misconfiguration fails closed via `hardFailProvider` (`fiscal_provider.go:54,114-121`). **However `signer` is never injected** — field assigned nowhere (`fiscal_provider.go:129`), so `CreateReceipt` always errors `"mysoliq: no EDSSigner configured"` (`:232-234`). Enabling `FISCAL_PROVIDER=MY_SOLIQ` today → 100% `FISCAL_FAILED`.
- `fiscal/uzbekistan.go` is a **dead mock** (`SubmitDocument` "Mock success", `:17-20`); referenced only by its own test.
- E-invoice corrective chain designed in schema (`CreditNotes.OriginalEhfId/CorrectiveEhfId`, `schema/spanner.ddl:1696-1698`); Soliq client also feeds the buyer-acceptance poller when MY_SOLIQ active (`bootstrap/bootstrap.go:993`, `order/buyer_acceptance_poller.go:21-30`).
- **Tax**: real regime versioning service — country-scoped, effective-dated VAT regimes, overlap validation, role-gated CRUD (`tax/service.go:38-80`), 5-min cache; consumed by order fiscal snapshots and credit notes. Journal entries: **absent** (no GL; only PaymentLedgerEntries / ArLedgerEntries event-style rows).

---

## 9. SCHEMA & SCHEMADRIFT — **LIVE (mature DDL; drift tool is narrow)**

- `schema/spanner.ddl`: **155 tables**, 2,624 lines; 87 incremental migrations in `schema/migrations/`; `SchemaMigrations` checksum ledger refuses drift on re-apply (`spanner.ddl:635-639`); `cmd/apply-migration` applier.
- Design: `STRING(36)` UUID PKs, `allow_commit_timestamp` audit columns, interleaved children with `ON DELETE CASCADE` (`OrderPaymentLegs` `:1671-1672`, `CreditNoteLines` `:1718-1719`), dense secondary indexes incl. `FORCE_INDEX` hints (`outbox/spanner_store.go:110`, `inventory/repository.go:220`), JSON for line items (`Orders.LineItemsJson BYTES(MAX)` `:166`).
- Only 13 UNIQUE indexes and **1 CHECK constraint** in the whole DDL — all enum/status integrity is app-level.
- `schemadrift/shop_closed.go` + `cmd/schema-drift/main.go:50-53` assert **only** the shop-closed columns — not a general drift detector despite the name.
- Stale artifacts: `schema/spanner.ddl.orig` (3-table ancestor) and `bootstrap/bootstrap.go.bak` committed — apply-order/confusion hazards.

---

## 10. WIRING — **LIVE (everything registered; no commented-out routes)**

- `main.go` registers **all** route groups: orders (`:244`), payments (`:236`), credit (`:253`), cashrecon (`:282`), credit notes (`:288`), returns ×3 mounts (`:207-209`), delivery, claims (inside orderroutes), tax regimes, fxrates admin+supplier (`:347-350`), partner API + admin keys (`:341-346`), ws hubs (`:351`), platform, pulse, catalog, demand, laborcapacity, eta, warehouse, supplier, retailer, driver, factory, payload(er), control tower, promotions, telemetry, update. **Zero commented-out registrations found.**
- Middleware chain: trace → metrics → CORS → SessionAuth → partner keys → optional Auth0 (`main.go:108-124`) → reliability (`:126-128`) → idempotency (`:129-131`).
- Workers started: outbox relay, 6 Kafka consumers, webhook-inbox reconciler, AR dunning, cash-recon escalation, shop-closed worker (`main.go:403`), preorder auto-confirm (flag `AUTO_CONFIRM_PREORDERS_ENABLED=1`, `runtime_workers.go:126`), replenishment cron, etc. (`runtime_workers.go:15-154`).
- Feature flags gating core flows: `AR_INVOICES_ENABLED`, `AR_DUNNING_ENABLED`, `WMS_LOTS_ENABLED`, `ALLOW_AUTH_BYPASS` (default false, `bootstrap/bootstrap.go:313`), `REQUIRE_INFRA_ADAPTERS` (default **true** — Spanner/Redis/Kafka mandatory, memory fallback refused, `:310,442-448,499`), `AIRWALLEX_DIRECT_EXECUTION_ENABLED`, `CASH_RECONCILIATION_REQUIRED`, Global Pay simulator mounted only when env ≠ production/staging (`main.go:387-400`).
- Local `.env.local` (committed) runs emulator Spanner + real Kafka, sets **no** FISCAL_PROVIDER (→ PEGASUS default), no AR flags, no GP credentials (→ stub paths active).

---

## MATURITY METRICS

| Metric | Value |
|---|---|
| Non-test Go files | 812 |
| Test files | 265 (~25%) |
| HTTP handler funcs | 644 |
| TODO/FIXME/HACK (non-test) | **1** (`order/driver_edges.go:752`, trivial rename) |
| "not implemented" | **0** |
| panic() (non-test) | **3** — `driverroutes/routes.go:114` & `payload/service.go:222` (fail-fast wiring asserts, acceptable), `cmd/mint-dev-jwt/main.go:20` (dev tool) |
| mock/stub/fake hits (non-test) | 57 across 26 files — concentrated: `payment/global_pay_executor.go` (3 silent stubs), `order/fiscal_provider.go` (FAKE provider SSMR hooks), `fiscal/uzbekistan.go` (dead mock), `simulator/`, `bootstrap/memory/` |

Overall: this is **not** a scaffold — it's a substantially real system with disciplined money/transaction handling. The immaturity is concentrated and specific: payment capture routing, non-tax "fiscal" receipts, flag-gated AR, and a few tolerated-error paths.

---

## TOP 10 CORRECTNESS/LEGALITY RISKS (ranked)

1. **Card capture is permanently broken AND fire-and-forget.** Gateway name mismatch `"GLOBALPAY"` (`payment/service.go:653`) vs executor key `"GLOBAL_PAY"` (`payment/execution.go:140`) → every capture errors; callers swallow it: log-only goroutine after commit (`order/service.go:1921-1929`), backorder sweep skips (`order/backorder_sweeper.go:51-55`). Combined with the leg pre-recorded as `CAPTURED` in-txn (`order/service.go:1899-1909`), the ledger asserts money moved that never did. **Financial loss, silent.**
2. **Legally non-compliant completion path (Uzbekistan).** Default `FISCAL_PROVIDER=PEGASUS` issues platform receipts, explicitly `"tax_ofd": false` (`order/fiscal_provider_pegasus.go:13-15,78-79`); orders complete without Soliq/OFD fiscal receipts while the state machine's "FISCALIZING" hard-gate (`order/service.go:1527-1538`) creates false assurance of fiscal compliance.
3. **Silent PSP stub-success when Global Pay creds empty.** Refund → `gp_refund_stub_` (`payment/global_pay_executor.go:112-120`), capture → `gp_capture_stub_` (`:251-258`), status → `gp_status_stub_paid` (`:312-320`) — all return `nil` error. Refunds reported to users that never happened. If the dormant `WebhookReconciler` is ever wired, stub status becomes a forged `SignatureValid:true` PAID webhook (`payment/reconciliation.go:57,66-108`).
4. **Shop-closed worker can deliver on credit without recording debt.** Profile-load error is warn-only (`order/worker_shop_closed.go:99-102`) yet the order still becomes `DELIVERED_ON_CREDIT` (`:128-134`); inline balance math (`:136-146`) ignores `ReservedMinor` (double-counts vs reserve-at-create) and hardcodes `MaxAutoCreditMinor: 50000000` (`:105-108`).
5. **Payment idempotency not DB-enforced.** `OrderPaymentLegs.IdempotencyKey` lacks a unique index (`schema/spanner.ddl:1667`); replay protection lives only in Redis with 24 h TTL (`idempotency/middleware.go:114`) — a retry after TTL double-records a payment leg. Middleware is header-optional (`middleware.go:57-60`); non-strict mode allows per-pod in-memory store (`bootstrap/bootstrap.go:419`).
6. **AR silently inert behind flags.** `AR_INVOICES_ENABLED` off → `OpenFromCreditLeave` returns success with no invoice (`ar/service.go:89-91`); dunning off by default (`ar/dunning_worker.go:13-16,66`). Credit leave-behind works (debt accrues on profiles) while aging/dunning/collections produce nothing — AR blindness by configuration.
7. **All status integrity is app-level.** 155 tables, one CHECK constraint (`schema/spanner.ddl:1273`); `Orders.Status` unconstrained (`:163`). Multiple writers bypass `ValidateStatusTransition` (e.g. `order/driver_edges.go:1014`, `order/repository_spanner.go:1260`, `order/worker_shop_closed.go:130,161`, `order/shop_closed.go:263`) — one new ad-hoc writer = illegal transitions, invisible to the DB.
8. **MY_SOLIQ provider cannot work as shipped.** `signer` field never assigned anywhere in the codebase (`order/fiscal_provider.go:129`); `CreateReceipt` always fails `"mysoliq: no EDSSigner configured"` (`:232-234`). Flipping the env flag to go legal yields 100% `FISCAL_FAILED` and halted completions.
9. **Inventory negative-stock clamp hides shrinkage.** `AdjustStock` silently floors at 0 (`inventory/repository.go:166-169`) — stock drift masked instead of surfaced; downstream reservation math (`qoh-qr`) then trusts the clamped number.
10. **Fail-open payer authorization + committed secrets.** Payer GET/PUT check ownership only *if claims exist* (`payment/crud_handlers.go:52-57,76-81`) and `POST /v1/payers` has no role gate (`paymentroutes/routes.go:43`) — IDOR-shaped under `ALLOW_AUTH_BYPASS`. Plus hygiene: `.env.local` (with `JWT_SECRET`), `bootstrap.go.bak`, `spanner.ddl.orig`, patch scripts, and a 90 MB compiled `backend-go` binary all committed at repo root.

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
