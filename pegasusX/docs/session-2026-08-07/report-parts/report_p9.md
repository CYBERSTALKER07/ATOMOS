# 8. Recommendations

## 8.1 Product scope — what to change

1. **Sell deep-single-distributor, not thin-marketplace — for now.** The runtime is single-supplier by construction; the vertical chain (factory→warehouse→fleet→store) is the rare asset. Position as the operating system for one large distributor's whole chain, with the integration layer (§4) as the sales wedge into retail chains. Defer marketplace commerce (Phase 3–5) until Gates below produce evidence.
2. **Monetize credit, not distribution margin.** The market evidence (MaxAB-Wasoko, Udaan, Jumbotail) and the codebase agree: the credit spine is built and dormant. Completing AR activation + off-app dunning + fee schedule + payouts is the revenue path.
3. **Rename or build the "factory" app.** Today it is a dispatch hub; either scope it honestly or fund MES-lite (§6.5).
4. **Treat the Control Tower honestly.** The live-network map broadcasting random data (`simulator/control_tower.go:53-79`) must be env-gated to demo builds only; the real twin projection (`twin/consumer.go`) should feed it.

## 8.2 Architecture — what to change

1. **Route every status write through the validator.** Make `ValidateStatusTransition` the only path (repository-level guard), and add DB-level defense: Spanner CHECK on `Orders.Status` membership at minimum. Today: 4 call sites vs ~65 direct writes.
2. **DB-enforce financial idempotency.** Unique indexes on `OrderPaymentLegs.IdempotencyKey`, `PaymentLedgerEntries` idempotency column; the 24h Redis TTL is a performance layer, not the guarantee.
3. **Kafka HA before enterprise promises.** 3 brokers, RF=3, min.insync.replicas=2; add an outbox-relay DLQ (exhausted publishes currently stay unpublished with only a log line, `outbox/relay.go:135-152`).
4. **Kill the stub-success philosophy.** Any PSP executor returning fabricated success (`gp_*_stub_`) must hard-fail outside explicitly-marked dev envs; the dormant `WebhookReconciler` must refuse stub refs.
5. **Collapse client duplication.** ~12k lines mobile and ~8k web are copy-paste (per repo measurement); fold AutoUpdater/WS clients into `mobile-android-kit`/`mobile-ios-kit` and grow `ui-kit` — it is the prerequisite for accessibility/localization ever being fixed once instead of 4–6 times.
6. **Generate the API client from the OpenAPI contracts** instead of maintaining 164 hand-written methods against 255 paths (drift is inevitable; the contracts now exist).

## 8.3 Prioritized gap list

**P0 — correctness & legality (days-to-2-weeks each; do first):**

| # | Action | Evidence |
|---|---|---|
| P0-1 | Fix capture routing key (`GLOBALPAY`→`GLOBAL_PAY`); remove the in-txn optimistic `CAPTURED` leg — record legs only after PSP confirmation; make backorder sweeper capture path synchronous | `payment/service.go:653`; `order/service.go:1899-1929` |
| P0-2 | Inject an EDS signer into MY_SOLIQ (or hard-block the provider until injectable); obtain Soliq sandbox credentials; prove `PX_E2E_SOLIQ_SANDBOX_OK` | `order/fiscal_provider.go:129,232-234` |
| P0-3 | Remove/gate all `gp_*_stub_*` success paths; guard `WebhookReconciler` against stub refs | `payment/global_pay_executor.go:112-320`; `payment/reconciliation.go:57` |
| P0-4 | Shop-closed worker: fail the transition (not warn) when credit profile unreadable; use `credit.Available()` math incl. `ReservedMinor` | `order/worker_shop_closed.go:91-165` |
| P0-5 | Unique indexes on payment-leg/ledger idempotency keys; honor `Idempotency-Key` on partner order create | `spanner.ddl:1667`; `partner/handlers.go:37-63` |
| P0-6 | Enable `AR_INVOICES_ENABLED`/`AR_DUNNING_ENABLED` wherever credit delivery is enabled — or block credit leave when AR is off (never allow debt without aging) | `ar/service.go:88-96` |
| P0-7 | Kafka RF=3; outbox relay DLQ | `infra/k8s/kafka.yaml:42-47`; `outbox/relay.go:135-152` |
| P0-8 | Fail loudly on negative stock instead of clamping; reconcile report for existing clamps | `inventory/repository.go:166-169` |
| P0-9 | Role-gate `POST /v1/payers`; fail-closed ownership on payer GET/PUT | `payment/crud_handlers.go:52-81` |
| P0-10 | Rotate secrets in committed `.env.local`; purge tracked binaries; delete stale `.bak`/`.orig` artifacts and root patch scripts | repo root |
| P0-11 | Warehouse Android scanner: wire to `WarehouseApi` or delete; CI grep against `TODO: Inject` | `warehouse-app-android/.../ScannerViewModel.kt:22,47` |
| P0-12 | Gate the random-data Control Tower simulator to demo builds | `simulator/control_tower.go:53-79` |

**P1 — structural product truth (2–8 weeks):**

1. **Platform admin console** (tenant lifecycle, KYB, suspension, flags, fees, observability) + `PLATFORM_ADMIN` role — prerequisite for any second tenant and for trust.
2. **Freeze multi-supplier registration** until Gate 5 Phase 1 lands (registration currently mints tenants the runtime misattributes, `supplier/service.go:433-447`).
3. **Master-data sync API** (partner upsert of products/prices/stock) + webhook coverage expansion beyond 4/155 events + EDI CONTRL/APERAK.
4. **WMS mobile floor execution** (pick waves, putaway, counts on Android/iOS) + FEFO/cold-chain operator surfaces.
5. **Off-app dunning transports** (SMS/email/WhatsApp) — the field-agent-displacement blocker for collections.
6. **Payout execution + refund initiation + fee schedule + invoices** (the monetization loop).
7. **Enable the shipped-off integration transports** (AS2/SFTP) in manifests; SFTP host-key pinning.
8. **Supplier mobile offline**; warehouse Room-parity queue; iOS telemetry buffer; server telemetry ACK.
9. **Server-side offline POS acceptance + fiscalization** (close the client-queue/server-deferred contradiction).
10. **Pricing authority engine** (rules + approval + margin floors).

**P2 — planning quality (4–10 weeks):**

1. **Deploy optimizer-core for real** (AR image + replicas ≥1) so routing is optimization, not heuristic; keep the H3/bin-pack fallback as fallback.
2. **Default-on safety-stock v2 and forecast algo** after shadow evidence; **auto-flip auto-order to `place`** only at ≥80% shadow acceptance with human+env signoff (the harness exists: `RetailerAutoOrderShadowProposals` 30-day stats).
3. **Partial allocation / backorder queue** — insufficient stock is currently a hard error and a lost sale (`allocation/service.go`).
4. **POS demand feed + real weather-driven demand adjustments** into forecasting (weather ingest exists; POS feed is the residual).
5. **Real S&OP capacity model** replacing `factories × 700 × 7`; remove `sku-projection-%d` literals.
6. **Serial tracking**; DataMatrix for UZ marking; EPCIS.
7. **Credit risk scoring v2** — only as a product decision (it was deliberately removed); if re-added, data-driven from the now-accumulating `DelinquencyCount`/repayment history, and delete the stale runbook references to the dead flag.

**P3 — scale / enterprise (quarters):**

1. **Multi-tenancy Phase 1** (request-scoped tenancy, ~150–250 files; fail-closed middleware; per-tenant rate limits; outbox partition by tenant) per the accepted ADR — then Phase 2 cross-supplier cart/split.
2. **Certified EDIFACT / certified 1C exchange package / Drummond AS2.**
3. **Global product master** (GTIN-keyed + offers + match queue) as the marketplace prerequisite.
4. **BI/data sink** (BigQuery/parquet) and Excel exports.
5. **Marketplace commerce (Phases 3–5)** — decided on evidence from the above, not on the premise that no competitor exists.

## 8.4 Process — keep the honesty, fix the drift

The repo's audit culture is an asset (its Substance Gate found real theatre). Three mechanical fixes: **(1)** status claims must carry dates and get auto-staled by CI (three runbooks currently instruct enabling a flag that no longer exists); **(2)** evidence artifacts cited by gates must exist at referenced paths (the Substance Gate's marker-gate log is a dangling reference); **(3)** one canonical tree — the docs name `/Users/shakhzod/ATOMOS/pegasusX` canonical while the live, newer tree is this one (202 backend files diverge).

---

# 9. Closing Assessment

The engineering is not the problem. The transactional outbox, the retry-safe closures, out-of-band webhook verification, integer-money discipline with `math/big` FX, idempotency mirrored across three client platforms, the FSM with battle-scar comments, a 25%-test backend with 1 TODO — that is careful, rare work.

The problems are threefold and each is fixable:

**First, the money path has live bugs worse than any tracked gap.** Broken card capture with optimistic ledger writes, a legal fiscal provider that cannot sign, stub-success PSP paths, and AR that ships inert. These are days-to-weeks of work and they gate everything — legality, trust, and monetization.

**Second, the autonomy stack is built and switched off.** Forecast, safety stock, auto-order, dunning, WMS execution, EDI/AS2 — implemented, flag-gated, default-off. The unlock is evidence, not code: run shadow mode, publish acceptance/accuracy numbers, then flip flags with discipline. The shadow harness already exists for exactly this.

**Third, the platform property is missing.** Single-supplier runtime, no admin console, no tenant lifecycle. The schema was designed multi-tenant and the runtime was not; closing that is the largest single investment (150–250 files for Phase 1) and the correct sequencing is after the money path is fixed and the integration layer is complete — building tenant isolation before partner APIs stabilize would rebuild them twice.

Against the six questions this report set out to answer: **(1)** field-agent displacement is real for the order pad (~35–40% today, ~65% at P1-complete) and structural-human for cash and exceptions — a hybrid future, not a wipe-out; **(2)** versus O9/Kinaxis/Blue Yonder/ERP+WMS/marketplaces, PegasusX loses on planning sophistication and wins on vertical factory→shelf transactional depth no category leader has; **(3)** alignment with incumbent 1C/SAP estates is real but incomplete — file-drop and pull-export today, with specific P0/P1 gaps (idempotent writes, master-data push, webhook coverage, certified formats) before a chain adopts without re-keying; **(4)** the exact ideal of a unified transactional platform with near-zero-human routine replenishment plus physical execution roles does not exist publicly — PegasusX has ~70% of its substance and ~10% of its platform property; **(5)** per-role detail shows deeply wired clients with concentrated, fixable weak points; **(6)** the recommendations above sequence the repair: P0 money-path correctness first, then structural truth (admin console, integration completeness, mobile floor execution), then planning quality, then scale.

Fix the money path. Prove the autonomy in shadow. Then decide the marketplace question on evidence.
