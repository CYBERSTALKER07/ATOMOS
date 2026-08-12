# PegasusX — Consolidated Ecosystem Gap Register & Data-Flow Blueprint

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
| P0-2 ✅ | **RESOLVED (commit 9f8d8787).** `Rail.IsLive()` added; `SubmitForDispatch(live=true)` fails closed with `ErrNoLiveRail` on a non-live rail — a batch can no longer strand in SUBMITTED with an empty rail ref. Live rail still pending a real bank integration (separate scope). | Money | `payout/rail.go` |
| P0-3 ✅ | **RESOLVED (commit 9f8d8787).** `RequireTenant` now exempts `RolePlatformAdmin` (cross-tenant break-glass role has no SupplierID) so the admin console works when tenancy is enforced. | Tenancy | `auth/tenant.go:95` |
| P0-4 ✅ | **RESOLVED (commit 9f8d8787).** Payout `Insert`/`UpdateStatusRef` now emit `PAYOUT_BATCH_GENERATED/EXPORTED/DISPATCHED/PAID` via `outbox.SpannerTxnBuffer` in the same txn. | Data-flow | `payout/store.go`, `outbox/spanner_txn_buffer.go` |
| P0-5 ✅ | **RESOLVED (commit 9f8d8787).** AR `OpenInvoice`/`ApplyPayment`/`UpdateDunning` now emit `AR_INVOICE_OPENED/PAYMENT/SETTLED/DUNNED` atomically in the same Spanner txn. | Data-flow | `ar/service.go` |
| P0-6 ✅ | **RESOLVED (commit 9f8d8787).** Moved `/api/[...path]` and `/api/ws-session` from 11-deep `app/api/api/...` to `app/api/`, restoring the demand pages. | Client apps | `apps/supplier-portal/app/api/` |

### P1 — structural product truth / cross-role loops

| # | Gap | Domain | Evidence |
|---|-----|--------|----------|
| P1-1 | Optimizer absent in prod AND prod image remap points optimizer-core at the backend-go digest (crash-loop if scaled) | Planning | `infra/k8s/overlays/prod/kustomization.yaml:36-39,62-69` |
| P1-2 | 30-day soak artifact never auto-generated; `artifacts/forecast-shadow/` absent; flip-check schema (`unmodified_acceptance_rate`) ≠ runtime (`unmodified_accept_rate`); doc threshold 80% vs code default 60% | Planning | `scripts/auto_order_place_flip_check.sh:8-23` vs `retailer/auto_order_soak_gate.go:30-36` |
| P1-3 | Forecast + accuracy CronJobs orphaned manifests — referenced by no overlay; nightly pass never scheduled | Planning | `infra/k8s/planning_forecast_cronjob.yaml`, `planning_accuracy_cronjob.yaml` |
| P1-4 | Prod autonomy flags all off (FORECAST_ALGO/ACCURACY, SAFETY_STOCK_V2, AUTO_ORDER_WORKER unset) → 30-day soak can never run in prod | Planning | `infra/k8s/backend-go/configmap.yaml:55-61` |
| P1-5 | Dual-control place-flip never executes at runtime: `featureflags.Evaluate` has no production caller; place flag read once at bootstrap; `FLAG_AUTO_ORDER_PLACE` audit action doesn't exist | Planning | `featureflags/service.go`; `bootstrap.go:892` |
| P1-6 | Order marked COMPLETED on Soliq Submit success, before buyer acceptance; rejected EHF leaves order completed unless default-off credit-note fires | Money | `order/fiscal.go` CreateReceipt; `buyer_acceptance_poller.go:114,37` |
| P1-7 | Fiscal sandbox proof = env-presence only; no live sign→submit→poll round-trip; E-IMZO key not yet procured | Money | `cmd/ssmr-smokecheck/e2e_soliq.go`; `fiscal/signer_env.go` |
| P1-8 | WebhookReconciler never started; settlement-vs-captured reconciliation is manual-only | Money | `payment/reconciliation.go:92`, no caller in runtime_workers.go |
| P1-9 | Push + inbox are consumer-side only → `RUN_MODE=api` silently loses FCM + polling fallback while WS keeps working | Data-flow | `main.go:96-104`; push only in `notification_dispatcher.go:735-752` |
| P1-10 | Driver location bypasses the bus (in-process hub + Redis, never outbox/Kafka) → twin + dispatcher consumers of DRIVER_LOCATION_UPDATED are dead paths | Data-flow | `telemetryroutes/routes.go:109-195` |
| P1-11 | No JWT session revocation/denylist; refresh extends any non-expired token | Tenancy | `auth/refresh.go:27-46` |
| P1-12 | Schema drift: 14 tables in migrations absent from `spanner.ddl` (fresh-emulator parity broken); drift gate only checks shop-closed schema | Tenancy | migrations vs `spanner.ddl`; `cmd/schema-drift/main.go:50` |
| P1-13 | GS1 DataMatrix payload non-conformant: encodes literal `(01)` parens, no leading FNC1 → won't scan as GS1 element string; not wired into label path (GS1-128 used) | Integration | `gs1/datamatrix.go:23-46`; `payload/ship_units.go:291` |
| P1-14 | AS2 outbound MDN never verified (no MIC comparison, 2xx = success) | Integration | `partner/as2/client.go:94-104` |
| P1-15 | Go SDK module path `github.com/CYBERSTALKER07/ATOMOS/...` ≠ repo module; excluded from go.work; stale 2021 deps | Integration | `sdk/partner/go/go.mod:1`; `go.work:3-10` |
| P1-16 | Admin portal missing UI for product-match queue, partner-keys/AS2/SFTP/COA, dunning, billing, platform analytics | Client apps | `apps/admin-portal/app/page.tsx` vs `platformadmin/handlers.go` |
| P1-17 | Retailer AR-invoices + HQ multi-store views desktop-only (absent on Android/iOS) | Client apps | `retailer-app-desktop/app/(dashboard)/credit|hq` |
| P1-18 | Factory loading bay → payload terminal loop broken (payload calls only supplier/payloader manifests, not factory manifests) | Client apps | `payload-terminal` API calls |

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
| P2-10 | Global Pay refund action `RF` unverified against live gateway | Money |
| P2-11 | `pegasusx-realtime`/`pegasusx-webhooks` topics declared but unwired; `twin` consumer never instantiated; non-Spanner fallback drops events | Data-flow |
| P2-12 | DEMAND_SIGNAL outbox write not atomic with sell-through mutation (separate txn) | Data-flow |
| P2-13 | No search infra (one SQL-LIKE endpoint; everything else point/list) | Data-flow |
| P2-14 | Partner webhook subscription URLs: no scheme/host allowlist / SSRF validation | Data-flow |
| P2-15 | Admin portal REST-only (no WS hub, no push) | Data-flow |
| P2-16 | Flag approvals not written to PlatformAdminAudit; platform-admin actor degrades to "unknown" on empty Subject | Tenancy |
| P2-17 | No MFA anywhere (TOTP for PLATFORM_ADMIN at minimum); hand-rolled HS256 JWT | Tenancy |
| P2-18 | Phase 2–5c gates + analytics-tenancy gate local-only, not in CI; admin-portal no CI job | Tenancy |
| P2-19 | SLO alerts missing for relay restarts / DLQ depth / partner-webhook success | Tenancy |
| P2-20 | EDIFACT breadth: only 6 types (EDI-lite); missing PRICAT/INVRPT/SLSRPT/RECADV/ORDCHG/DELFOR/REMADV | Integration |
| P2-21 | No partner sandbox / test-key mode; no self-serve onboarding (admin-issued keys only) | Integration |
| P2-22 | OpenAPI incomplete vs routes (missing `/v1/supplier/partner-*` aliases, key revoke, SFTP-config path) | Integration |
| P2-23 | Cold-chain temperature readings + labor capacity + twin routes APIs have no client UI anywhere | Client apps |
| P2-24 | Warehouse pick-waves/bins/cycle-counts only partial (inside TransferActions), no dedicated screens | Client apps |
| P2-25 | Payload native apps (Android 3 screens / iOS 5 views) are stubs; Expo terminal is the real app | Client apps |
| P2-26 | Supplier planning S&OP/scenarios/replenishment on mobile only, absent from web | Client apps |

### P3 — polish / acknowledged-by-design

WhatsApp dunning channel absent (SMS/email only) · WS JWT in URL query param (non-Firebase path) · ECC200 ASCII-only, square ≤44×44 (comment says 52) · control-tower scenariosData hardcoded empty · Pulse/reports-export/POS-holds desktop-only · quantity negotiation intentionally disabled · stale docs (`PARTNER_EDI.md:74` "placeholder", `OPTIMIZER_AND_ROUTING_RUNTIME.md` replica counts) · dead `optimizationjobs/` package · `payment.HandleListPayers` + ~9 detail-IDORs from the tenant audit still open (see below).

### Security follow-through (from Domain 6.2 audit, re-verified 2026-08-12)

`driver.HandleOrderGet` **FIXED** (fail-closed ownership). Still **OPEN**: `payment.HandleListPayers` (cross-tenant list, no filter), `driver.HandleGetDriver` / `HandleGetVehicle`, `warehouse.HandleGetWarehouse`, `factory.HandleGetFactory` (path-param IDORs), `creditnote.HandleOrderLines` / `HandleListReverseTasks`, `supplier.HandleAIRecommendations` (seed-supplier fallback), and the systemic root causes (`scopedSupplierID` seed fallback, `?supplier_id=` honored when scope empty, admin-bypass idiom, `demandroutes` mounted with no auth). Middleware masks most list gaps in ssmr/prod but not detail-IDORs reachable with a valid tenant claim. **This is a dedicated security initiative, ~9 detail endpoints + 7 root causes.**

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
| `PARTNER_EDI.md:74` | datamatrix.go "placeholder modules" | Stale — real ECC200 now (but GS1 layer non-conformant, P1-13) |
| `END_PRODUCT_REALITY_REPORT` §5 | "no admin console UI", "SDK README-only", "datamatrix placeholder", "reatilerapp typo", "App.tsx 2.5k monolith", "redirect stubs" | All fixed since the report (Domains 2-6) — report is now 1 day stale |

---

## Part 4 — What "done" looks like (the coverage rule)

To reach the consistent data flow you described, the closing condition is a single invariant:

> **Every Spanner state mutation emits an in-transaction outbox event; every event has a declared consumer (WS hub and/or webhook and/or push); no cross-role loop ends at an API with no client.**

Concretely that means, in order:
1. Put AR + Payout on the outbox (P0-4/5) and wire AR invoice pay-down (P0-1).
2. Fix the P0 tenant + routing breaks (P0-3, P0-6) and the live payout rail (P0-2).
3. Run-mode parity so `api` doesn't silently lose push/inbox (P1-9); put driver location on the bus (P1-10).
4. Close the broken cross-role client loops (P1-18, P2-23/24/26).
5. Then the security backlog and certification items.

*Generated by consolidated audit, 2026-08-12.*
