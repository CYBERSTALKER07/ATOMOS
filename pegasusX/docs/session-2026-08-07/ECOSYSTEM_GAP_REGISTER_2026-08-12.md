# PegasusX — Consolidated Ecosystem Gap Register & Data-Flow Blueprint

> **Prod goal SoT:** [`../PROD_ECOSYSTEM_GOAL.md`](../PROD_ECOSYSTEM_GOAL.md) — north star, pillars, coverage rule, and wave order. This register is the evidence backlog, not the goal.

**Date:** 2026-08-12 · **Evidence base:** live source tree, re-verified by six parallel audits (money/fiscal, autonomy/planning, integration, tenancy/ops, role client apps, data-flow plumbing). This supersedes the gap sections of `END_PRODUCT_REALITY_REPORT_2026-08-11.md` — every claim below was checked against code, not docs.

Legend severity: **P0** breaks revenue/legality/security or strands state · **P1** structural product truth / cross-role loop broken · **P2** enterprise-grade completeness · **P3** polish.

---

## Part 1 — The "perfect data flow" question

### 1.1 The as-built pipeline (verified live)

```
Spanner (source of truth, single schema, 167 tables)
  └─ in-transaction outbox write (outbox.TxnBuffer, SupplierId NOT NULL, _platform sentinel)
       └─ Relay (250 ms poll, batch 100, 2-min claim lease, fair tenant interleave,
                 5-try backoff, DLQ after 20 attempts → OutboxDeadLetters)
            └─ Kafka (segmentio/kafka-go, RequiredAcks=all, GCP Managed Kafka auth seam)
                 └─ Consumers (manual commit + DLQ + Redis dedup):
                      NotificationDispatcher → WS hub + FCM push + inbox persist
                      Order mutator · Warehouse mutator · Returns · Billing tier · Partner webhooks
                       └─ 7 WS hubs (retailer, supplier, driver, payload, warehouse, factory, telemetry)
                            cross-pod via Redis Pub/Sub
                            └─ clients: every role has live WS on at least one platform
```

**The architecture you described is already the one that exists.** Spanner is the source of truth, the outbox is transactional, Kafka is real (not in-process), WS hubs exist per role, webhooks are signed with retry+DLQ, push is FCM. The system is not missing a data-flow paradigm — it has **coverage holes and mode-dependency** in an otherwise correct pipeline.

### 1.2 Why it *feels* unmaintainable — the three real root causes

1. **The pipeline is opt-in per domain, not universal.** Order/payment/fiscal/allocation/wms/dispatch emit in-transaction events (16+ sites). **AR and Payout emit zero outbox events** — finance-critical state mutates silently. Telemetry bypasses the bus entirely. So some parts of the system are real-time and some are invisible, and there's no rule that says which.

2. **Behavior changes by run mode without warning.** In `PEGASUSX_RUN_MODE=api` the relay and all consumers don't start → FCM push and inbox persistence silently stop while direct WS broadcasts keep working. Same code, different runtime truth. That's the "complex and difficult to maintain" feeling — it's not the architecture, it's undocumented mode-dependence.

3. **Cross-role loops are half-wired in clients.** Backend emits the events, but: supplier credit → retailer is wired; factory loading bay → payload terminal is broken; cold-chain temperature readings and twin routes have APIs with **no client consumer at all**; supplier planning scenarios exist on mobile but not web; retailer AR/HQ exist on desktop but not mobile.

**Conclusion:** you don't need a new data-flow paradigm. You need a **coverage rule** (every state mutation emits an event; every event has a declared consumer) and **run-mode parity**. That turns "complex and hard to maintain" into a checklist.

---

## Part 2 — Consolidated Gap Register (deduplicated across all audits + reality report)

### P0 — revenue / legality / security

| # | Gap | Domain | Evidence |
|---|-----|--------|----------|
| P0-1 ✅ | **RESOLVED (commit 9f8d8787).** Added `ar.Service.RecordPayment` / `RecordPaymentForOrder` / `GetByID` / `GetByOrder`; wired pay-down into `order.CollectCash` so credit invoices settle on cash collection. Tests in `ar/service_test.go`. | Money | `ar/service.go`, `order/service.go` |
| P0-2 ✅ | **RESOLVED (commit 9f8d8787 + W3 decision 2026-08-12).** `Rail.IsLive()` fail-closed; **bank-file is permanent settlement for this prod bar** — [`PAYOUT_RAIL_DECISION.md`](../PAYOUT_RAIL_DECISION.md). Live bank/GP payout deferred. | Money | `payout/rail.go`, `docs/PAYOUT_RAIL_DECISION.md` |
| P0-3 ✅ | **RESOLVED (commit 9f8d8787).** `RequireTenant` now exempts `RolePlatformAdmin` (cross-tenant break-glass role has no SupplierID) so the admin console works when tenancy is enforced. | Tenancy | `auth/tenant.go:95` |
| P0-4 ✅ | **RESOLVED (commit 9f8d8787).** Payout `Insert`/`UpdateStatusRef` now emit `PAYOUT_BATCH_GENERATED/EXPORTED/DISPATCHED/PAID` via `outbox.SpannerTxnBuffer` in the same txn. | Data-flow | `payout/store.go`, `outbox/spanner_txn_buffer.go` |
| P0-5 ✅ | **RESOLVED (commit 9f8d8787).** AR `OpenInvoice`/`ApplyPayment`/`UpdateDunning` now emit `AR_INVOICE_OPENED/PAYMENT/SETTLED/DUNNED` atomically in the same Spanner txn. | Data-flow | `ar/service.go` |
| P0-6 ✅ | **RESOLVED (commit 9f8d8787).** Moved `/api/[...path]` and `/api/ws-session` from 11-deep `app/api/api/...` to `app/api/`, restoring the demand pages. | Client apps | `apps/supplier-portal/app/api/` |

### P1 — structural product truth / cross-role loops

| # | Gap | Domain | Evidence |
|---|-----|--------|----------|
| P1-1 ✅ | **RESOLVED (W4 2026-08-12).** Prod optimizer-core remaps to `…/optimizer-core:ssmr` (not backend-go); replicas stay 0 until AR publish; CI gate fails if optimizer image is backend-go. | Planning | `infra/k8s/overlays/prod/kustomization.yaml`; `scripts/ci_fail_placeholder_images.sh` |
| P1-2 ✅ | **RESOLVED (W4 2026-08-12).** Soak artifact generator + `artifacts/forecast-shadow/`; flip-check accepts both rate keys; runtime default unmodified rate **0.80** aligned with flip policy. | Planning | `scripts/generate_auto_order_soak_artifact.sh`; `auto_order_place_flip_check.sh`; `retailer/auto_order_soak_gate.go` |
| P1-3 ✅ | **RESOLVED (W4 2026-08-12).** Forecast + accuracy CronJobs included in prod + SSMR overlays (image remap via existing backend-go digest pins). | Planning | `overlays/prod|ssmr/kustomization.yaml` |
| P1-4 ✅ | **RESOLVED (W4 2026-08-12).** Prod configMap merge enables shadow worker + forecast accuracy/algo(require-gate); **place stays false**. | Planning | `overlays/prod/kustomization.yaml` configMapGenerator |
| P1-5 ✅ | **RESOLVED (W4 2026-08-12).** `placeAllowedForRetailer` calls `featureflags.Evaluate` at runtime; dual-control approve writes `FLAG_AUTO_ORDER_PLACE` audit. | Planning | `retailer/auto_order_soak_gate.go`; `featureflags/handlers.go`; `platformadmin.RecordFlagAudit` |
| P1-6 ✅ | **RESOLVED.** ADR-009 still completes on OFD submit success. Parallel EHF track now works: MySoliq SUCCESS stamps `BuyerAcceptanceStatus=PENDING` + 10-day deadline (was never set → poller was dead); poller emits `BUYER_ACCEPTANCE_*` outbox events; auto credit-note on REJECT defaults ON (`CREDIT_NOTE_AUTO_FROM_BUYER_REJECT=false` to opt out). Poller moved onto App + worker tier (cancelable ctx). | Money | `order/fiscal.go`, `order/buyer_acceptance_poller.go` |
| P1-7 ✅ | **RESOLVED (W3 2026-08-12).** Sign→submit→poll proven in CI (`TestMySoliqContract_SignSubmitPoll`) + SSMR `PX_E2E_SOLIQ_CONTRACT_OK`. Live sandbox is opt-in (`FISCAL_MY_SOLIQ_LIVE_PROOF=1`); prod EDS still needs E-IMZO PKCS#12 procurement. See [`FISCAL_EDS_PROOF.md`](../FISCAL_EDS_PROOF.md). | Money | `order/fiscal_soliq_contract_test.go`; `cmd/ssmr-smokecheck/e2e_soliq.go` |
| P1-8 ✅ | **RESOLVED.** Constructed `payment.WebhookReconciler` in bootstrap and started it every 5m from the worker tier (30s startup jitter). Skips stub gateway refs; only advances sessions stuck >15min. `PAYMENT_RECONCILE_DISABLED=1` to turn off. | Money | `bootstrap/bootstrap.go`, `runtime_workers.go`, `payment/reconciliation.go` |
| P1-9 ✅ | **RESOLVED (commit 3b739a5f).** Worker-liveness gate: worker tier publishes a Redis heartbeat (`bootstrap.StartWorkerHeartbeat`); an api-only tier starts the notification consumer (FCM push + inbox) only when no live worker heartbeat exists (`startNotificationConsumerIfNoWorker`). Restores push/inbox for single-tier api deploys without double-firing alongside a worker; fails open when Redis is down. | Data-flow | `bootstrap/worker_heartbeat.go`, `main.go`, `runtime_workers.go` |
| P1-10 ✅ | **RESOLVED (commit 3b0b9b33).** Driver location now emits a throttled (5s/driver) `DRIVER_LOCATION_UPDATED` to the outbox (`TopicRealtime`) via `telemetryroutes.SpannerLocationBusEmitter`, lighting up dispatcher + twin consumers; `route_id` injected when resolvable. Full fidelity stays on WS/Redis. | Data-flow | `telemetryroutes/bus_emitter.go`, `routes.go`, `main.go` |
| P1-11 ✅ | **RESOLVED (W0 2026-08-12).** JWT `jti` issued; Redis/memory denylist; `POST /v1/auth/logout`; refresh rejects revoked tokens. Legacy tokens without `jti` still accepted until expiry. | Tenancy | `auth/jwt.go`, `auth/refresh.go`, `auth/revoke*.go` |
| P1-12 ✅ | **RESOLVED (2026-08-12).** Mirrored 14 migration-only tables into `schema/spanner.ddl`. Offline migration↔DDL parity gate + live required-table assert (beyond shop-closed). `go run ./cmd/schema-drift -offline`. | Tenancy | `schema/spanner.ddl`; `schemadrift/parity.go`; `cmd/schema-drift` |
| P1-13 ✅ | **RESOLVED (W5 2026-08-12).** `BuildAIElementStringFNC1` emits leading FNC1 + AI digits (no HRI parens); `MultiLabelZPL` appends GS1 DataMatrix `^BX`/`^FH_` when GTIN present. HRI form remains in `BuildAIElementString`. | Integration | `gs1/datamatrix.go`, `gs1/zpl.go` |
| P1-14 ✅ | **RESOLVED (W5 2026-08-12).** Outbound AS2 `Send` fail-closes unless sync MDN disposition is processed and `Received-Content-MIC` matches SHA-256 MIC of EDI body. | Integration | `partner/as2/client.go`, `partner/as2/mdn.go` |
| P1-15 ✅ | **RESOLVED (W5 2026-08-12).** Partner Go SDK module `github.com/pegasusx/pegasusx/sdk/partner/go`; included in `go.work`; gen script pins modulePath; deps current. | Integration | `sdk/partner/go/go.mod`, `go.work`, `scripts/gen_partner_sdk.sh` |
| P1-16 ✅ | **RESOLVED (W2 2026-08-12).** Admin console tabs for product-match queue + partner keys/AS2/SFTP/COA + dunning run-once; `PLATFORM_ADMIN` allowed on those admin routes with `tenant_id` query for partner scope. Billing/platform analytics remain thin (follow-on). | Client apps | `apps/admin-portal` + partner/globalproducts/credit routes |
| P1-17 ✅ | **RESOLVED (W2 2026-08-12).** Retailer Credit & AR + HQ multi-store screens on Android and iOS (sidebar entry + same APIs as desktop). | Client apps | `retailer-app-android` / `retailer-app-ios` |
| P1-18 ✅ | **RESOLVED (W2 2026-08-12).** Payload terminal lists/start-loads/seals factory loading-bay manifests (`RolePayload` on factory loading-bay routes; merge payloader+factory). | Client apps | `factoryroutes`, `payload-terminal` |

### P2 — enterprise-grade completeness

| # | Gap | Domain |
|---|-----|--------|
| P2-1 | Optimizer e2e soft-passes on constraint violation (SKIPPED not FAIL) | Planning |
| P2-2 | No optimizer certification harness (4 unit tests, no benchmark corpus, Rust sidecar mislabels greedy as `Optimal`) | Planning |
| P2-3 | MEIO is a greedy heuristic, not joint multi-echelon optimization; no capital-cap constraint | Planning |
| P2-4 | Scenario workbench: no clone→compare→publish, no versioning, placeholder revenue-at-risk unit value | Planning |
| P2-5 | S&OP: projected demand overwrites factory-capacity field; no ProductionLines model; 7-day horizon only | Planning |
| P2-6 | Demand-sensing producers missing: density worker no-op; EVENT/COMPETITOR_PRESSURE no ingestion; POS flywheel not folded into planning; DOW/payday hard-coded | Planning |
| P2-7 | Soak-gate break-glass bypass unaudited, not in money-flag set | Planning |
| P2-8 | Multi-currency AR unsupported (Currency hardcoded "UZS") | Money |
| P2-9 | Dev-default webhook secrets in `payment.NewService` (only bootstrap validation protects prod) | Money |
| P2-10 ✅ | **RESOLVED (W3 2026-08-12).** Global Pay refund `action=RF` proven against in-repo simulator (auth→perform). Live merchant confirm remains ops checklist. See [`GLOBAL_PAY_REFUND_PROOF.md`](../GLOBAL_PAY_REFUND_PROOF.md). | Money | `payment/gp_simulator_refund_test.go`; `simulator/global_pay.go` |
| P2-11 ✅ | **RESOLVED (W1 2026-08-12).** Twin Kafka consumer started (`void-digital-twin` on TwinConsumerTopics); telemetry envelope parsing fixed. TopicWebhooks **retired** (payment/partner stay on TopicMain/Orders; const kept for infra compat, no producers). | Data-flow |
| P2-12 ✅ | **RESOLVED (W1 2026-08-12).** Sell-through row + DEMAND_SIGNAL outbox emit in the same Spanner RW txn; FlywheelDemandFeed remains best-effort post-commit. | Data-flow |
| P2-13 ✅ | **DECIDED deferred (W1 2026-08-12).** Keep Spanner LIKE/prefix search; no ES/OpenSearch until scale/SLO evidence. See [`SEARCH_DECISION.md`](../SEARCH_DECISION.md). | Data-flow |
| P2-14 | Partner webhook subscription URLs: no scheme/host allowlist / SSRF validation | Data-flow |
| P2-15 | Admin portal REST-only (no WS hub, no push) | Data-flow |
| P2-16 | Flag approvals not written to PlatformAdminAudit; platform-admin actor degrades to "unknown" on empty Subject | Tenancy |
| P2-17 | No MFA anywhere (TOTP for PLATFORM_ADMIN at minimum); hand-rolled HS256 JWT | Tenancy |
| P2-18 | Phase 2–5c gates + analytics-tenancy gate local-only, not in CI; admin-portal no CI job | Tenancy |
| P2-19 | SLO alerts missing for relay restarts / DLQ depth / partner-webhook success | Tenancy |
| P2-20 ✅ | **RESOLVED (W5 2026-08-12).** EDI-lite breadth: PRICAT/INVRPT/SLSRPT/RECADV/ORDCHG/DELFOR/REMADV build+parse; PRICAT→price upsert, INVRPT→stock upsert; others ledger+ACK. Still not certified EDIFACT/Drummond. | Integration | `partner/edi/breadth.go`, `partner/edi_inbound.go` |
| P2-21 ✅ | **RESOLVED (W5 2026-08-12).** Sandbox keys `pxs_*` (`environment=SANDBOX`, `RateLimitClass=partner_sandbox`); retailer/supplier self-serve key routes; sandbox orders tagged `PARTNER_SANDBOX`. | Integration | `partner/keys.go`, `partner/routes.go`, `partner/service.go` |
| P2-22 | OpenAPI incomplete vs routes (missing `/v1/supplier/partner-*` aliases, key revoke, SFTP-config path) | Integration |
| P2-23 | Cold-chain temperature readings + labor capacity + twin routes APIs have no client UI anywhere | Client apps |
| P2-24 ✅ | **RESOLVED (W2 2026-08-12, already shipped).** Dedicated warehouse portal screens + nav for pick-waves, bins, cycle-counts (not TransferActions-only). | Client apps |
| P2-25 | Payload native apps (Android 3 screens / iOS 5 views) are stubs; Expo terminal is the real app | Client apps |
| P2-26 ✅ | **RESOLVED (W2 2026-08-12).** Supplier web `/planning` page with PlanningBrainPanel (S&OP + scenarios) + analytics-section nav; settings/planning remains overrides. | Client apps |

### P3 — polish / acknowledged-by-design

WhatsApp dunning channel absent (SMS/email only) · WS JWT in URL query param (non-Firebase path) · ECC200 ASCII-only, square ≤44×44 (comment says 52) · control-tower scenariosData hardcoded empty · Pulse/reports-export/POS-holds desktop-only · quantity negotiation intentionally disabled · stale docs (`PARTNER_EDI.md:74` "placeholder", `OPTIMIZER_AND_ROUTING_RUNTIME.md` replica counts) · dead `optimizationjobs/` package.

### Security follow-through (from Domain 6.2 audit, re-verified 2026-08-12 · **W0 closed 2026-08-12**)

`driver.HandleOrderGet` was already FIXED (fail-closed ownership). **W0 Trust (2026-08-12) closed the remaining detail IDORs + JWT revocation:**

| Item | Status |
|------|--------|
| `payment.HandleListPayers` | Fixed — auth required; retailer/admin scoped to self; platform-admin may list |
| `driver.HandleGetDriver` / `HandleGetVehicle` | Fixed — tenant supplier match (driver self-only for RoleDriver) |
| `warehouse.HandleGetWarehouse` / `factory.HandleGetFactory` | Fixed — tenant supplier or home-node match |
| `creditnote.HandleOrderLines` / `HandleListReverseTasks` | Fixed — order owned-by-supplier; warehouse home-node required |
| `supplier.HandleAIRecommendations` | Fixed — `scopedSupplierID` / PreferTenantSupplierID (no seed fallback) |
| `scopedSupplierID` seed fallback | Fixed — uses PreferTenantSupplierID |
| `demandroutes` unauthenticated | Fixed — RequireRole on all demand routes |
| P1-11 JWT revocation | Fixed — `jti` on Issue/Parse; Redis/memory denylist; `POST /v1/auth/logout`; refresh rejects revoked |

Helpers: `auth/entity_scope.go`, `auth/revoke.go`, `auth/revoke_redis.go`. Bootstrap installs Redis denylist when Redis is up.

---

## Part 3 — Docs-to-code alignment (stale/contradictory docs found)

| Doc | Claim | Reality |
|-----|-------|---------|
| `OPTIMIZER_AND_ROUTING_RUNTIME.md:74` | SSMR optimizer "replicas 0" | Stale — ssmr overlay sets replicas 1 |
| `big-platform-baseline/technical/workers-kafka.md:10` | optimizer-core "not in SSMR overlay" | False now |
| `DOMAIN3_PLANNING_PROGRESS.md:57-59` | "no in-repo optimizer source/Dockerfile" | False — `services/optimizer-core/Dockerfile` + solver in-tree |
| `AUTO_ORDER_PLACE_FLIP.md` + reality report | flip needs 30-day ≥80% artifact, `FLAG_AUTO_ORDER_PLACE` audit | Runtime default is 60%; artifact never generated; audit action doesn't exist |
| `DOMAIN2_AUTONOMY_PROGRESS.md:60-62` | dual-control governs AUTO_ORDER_* flags | Not wired — no production caller of `featureflags.Evaluate` |
| `PHASE4_COMPLETION.md:13` | "optimizer replicas ≥1 SSMR+staging OK" | Gate greps YAML text; proves intent not a running optimizer |
| `PARTNER_EDI.md` | datamatrix / breadth notes | Updated W5 — FNC1 DataMatrix + PRICAT…REMADV |
| `END_PRODUCT_REALITY_REPORT` §5 | "no admin console UI", "SDK README-only", "datamatrix placeholder", "reatilerapp typo", "App.tsx 2.5k monolith", "redirect stubs" | All fixed since the report (Domains 2-6) — report is now 1 day stale |

---

## Part 4 — What "done" looks like (the coverage rule)

To reach the consistent data flow you described, the closing condition is a single invariant:

> **Every Spanner state mutation emits an in-transaction outbox event; every event has a declared consumer (WS hub and/or webhook and/or push); no cross-role loop ends at an API with no client.**

Concretely that means, in order:
1. Put AR + Payout on the outbox (P0-4/5) and wire AR invoice pay-down (P0-1).
2. Fix the P0 tenant + routing breaks (P0-3, P0-6); payout live-rail remainder **decided bank-file** (P0-2 ✅).
3. Run-mode parity so `api` doesn't silently lose push/inbox (P1-9); put driver location on the bus (P1-10).
4. Close the broken cross-role client loops (P1-18 ✅, P2-24 ✅, P2-26 ✅; P2-23 still open).
5. Then the security backlog and certification items.

---

## Part 5 — Master alignment pointer (same-day re-verify)

Planning SoT for docs↔code↔role×platform + desktop stack decision:

- [`MASTER_ALIGNMENT_DATAFLOW_2026-08-12.md`](./MASTER_ALIGNMENT_DATAFLOW_2026-08-12.md)

Re-verify notes (2026-08-12 afternoon):

- AR/payout outbox emits **confirmed** in live `ar/service.go` + `payout/store.go`.
- Admin portal is a **real** Tenants/Flags/Audit console (not a stub).
- Twin consumer **started** (W1); TopicWebhooks **retired** (unused); search **Spanner LIKE by decision** — [`SEARCH_DECISION.md`](../SEARCH_DECISION.md).
- **W2 Class A loops closed 2026-08-12:** factory↔payload (P1-18), admin match/partner/dunning (P1-16), retailer AR/HQ mobile (P1-17), warehouse WMS screens (P2-24), supplier planning web (P2-26).
- **W3 money legality closed 2026-08-12:** EDS sign→submit→poll contract ([`FISCAL_EDS_PROOF.md`](../FISCAL_EDS_PROOF.md)); payout bank-file decision ([`PAYOUT_RAIL_DECISION.md`](../PAYOUT_RAIL_DECISION.md)); Global Pay `RF` simulator proof ([`GLOBAL_PAY_REFUND_PROOF.md`](../GLOBAL_PAY_REFUND_PROOF.md)).
- **W4 autonomy closed 2026-08-12:** optimizer-core no longer remapped to backend-go; soak artifact path + 80% threshold aligned; forecast/accuracy CronJobs on prod/SSMR; shadow soak flags on (place off); dual-control `Evaluate` + `FLAG_AUTO_ORDER_PLACE` audit. See [`AUTO_ORDER_PLACE_FLIP.md`](../AUTO_ORDER_PLACE_FLIP.md).
- **W5 partner cert closed 2026-08-12:** GS1 FNC1 DataMatrix + label path; AS2 MDN/MIC verify; SDK in go.work; EDI-lite breadth; sandbox keys.
- **P1-12 schema drift closed 2026-08-12:** 14 migration-only tables mirrored into `spanner.ddl`; offline + live schema-drift gate broadened.
- Desktop stack recommendation: **keep Next.js + Tauri 2** (no Electron migration).

*Generated by consolidated audit, 2026-08-12; Part 5 appended after live re-verify.*
