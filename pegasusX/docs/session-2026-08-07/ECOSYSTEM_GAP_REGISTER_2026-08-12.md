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
                       └─ 8 WS hubs (retailer, supplier, driver, payload, warehouse, factory, telemetry, platform-admin)
                            cross-pod via Redis Pub/Sub
                            └─ clients: every role has live WS on at least one platform
```

**The architecture you described is already the one that exists.** Spanner is the source of truth, the outbox is transactional, Kafka is real (not in-process), WS hubs exist per role, webhooks are signed with retry+DLQ, push is FCM. The system is not missing a data-flow paradigm — remaining work is **ops enablement, legal cert, and a few flag/scale residuals**, not a new bus.

### 1.2 Why it *felt* unmaintainable — root causes (and 2026-08-12 status)

1. **Pipeline coverage was uneven.** Order/payment/fiscal/allocation/wms/dispatch already emitted in-transaction events. **AR + Payout now emit** `AR_*` / `PAYOUT_BATCH_*` in the same Spanner txn (P0-4/5 ✅). Driver location is on the bus (P1-10 ✅). Twin consumer started (P2-11 ✅). Residual: some domains still dual-write or best-effort post-commit (e.g. FlywheelDemandFeed).

2. **Behavior changed by run mode without warning.** In `PEGASUSX_RUN_MODE=api` the relay/consumers historically did not start → FCM/inbox could silently stop while direct WS kept working. **Mitigated (P1-9 ✅):** worker heartbeat + api-tier notification consumer when no live worker. Mode docs still matter; silent dual-truth is closed for push/inbox.

3. **Cross-role loops were half-wired in clients.** Those Class A loops are **closed (W2):** factory loading bay ↔ payload (P1-18/P2-25), warehouse cold-chain + labor-capacity + twin Live Ops Map (P2-23), supplier planning web (P2-26), retailer AR/HQ mobile (P1-17), warehouse WMS screens (P2-24). Residual Class B/C: thin admin billing analytics; portal-only warehouse Control Tower on mobile; payload `seal-all` + capacity API-only. **R4.1 (2026-08-12):** warehouse Android/iOS cold-chain + labor-capacity parity with portal.

**Conclusion:** keep the **coverage rule** (every state mutation emits an event; every event has a declared consumer). Checklist items that were open at audit time are tracked as ✅ in Part 2; open residuals are owner keys, Soliq EDS, optimizer prod replicas, and auto-order place flip.

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
| P2-1 ✅ | **RESOLVED (2026-08-12).** Optimizer cold-chain e2e soft-skips only when sidecar unreachable; HTTP error / decode error / cold stop on non-reefer → hard `PX_E2E_OPTIMIZER_CONSTRAINT_FAIL` (no more SKIPPED on constraint violation). | Planning | `cmd/ssmr-smokecheck/e2e_optimizer_constraint.go` |
| P2-2 ✅ | **RESOLVED (2026-08-12).** Certification harness: `testdata/cert/*.json` + `test_certification_harness.py` (cold-chain/capacity/max-stops/multi-depot). Rust greedy VRP/CPSAT now report `HEURISTIC` (proto + contract), never `OPTIMAL`; Rust unit test asserts. | Planning | `services/optimizer-core/testdata/cert/`, `server/test_certification_harness.py`, `server-rust` solver + proto `HEURISTIC=5` |
| P2-3 ✅ | **RESOLVED (2026-08-12).** MEIO transfer recommendations remain greedy donor→receiver per SKU, but now enforce `Σ(qty × unit_value) <= MEIO_CAPITAL_CAP_MINOR` (0 = unlimited). Unit value from `Products.PriceMinor` with `MEIO_UNIT_VALUE_MINOR` fallback; CRITICAL/lowest days-cover prioritized; summary exposes capital used/skipped. | Planning | `replenishment/mei_engine.go` |
| P2-4 ✅ | **RESOLVED (2026-08-12).** Durable `PlanningScenarios` (DRAFT/PUBLISHED/SUPERSEDED) with run→persist, clone, compare, CAS publish + `planning.scenario.published.v1` outbox. RaR uses `Products.PriceMinor` (`SCENARIO_UNIT_VALUE_MINOR` fallback); heuristic path included. Publish is planning-baseline only (no sealed-manifest rewrite). | Planning | `planning/scenarios.go`, `projector.go`, `schema/migrations/20260812_planning_scenarios.ddl`, PlanningBrainPanel |
| P2-5 ✅ | **RESOLVED (2026-08-12).** S&OP keeps `FactoryCapacityUnits` from production-lines model (`factories × SOP_LINES_PER_FACTORY × daily × SOP_HORIZON_DAYS`); supply-request projections go to `ProjectedDemandUnits` only; utilization = demand/capacity; alert on demand > factory or inbound. | Planning | `planning/service.go`, `planning/sandop_test.go`, PlanningBrainPanel |
| P2-6 ✅ | **RESOLVED (2026-08-12).** Density worker computes H3 order hotspots → `EVENT_DENSITY` (+ extreme `EVENT`); flywheel peer skew → `COMPETITOR_PRESSURE`; sensing blends STORE_POS flywheel into base velocity (65/35); PAYDAY DemandSignals override hard-coded calendar (DOW fallback retained). Worker started in `runtime_workers`. | Planning | `demand/density_worker.go`, `demand/worker_sensing.go`, `runtime_workers.go` |
| P2-7 ✅ | **RESOLVED (2026-08-12).** `AUTO_ORDER_SOAK_GATE_DISABLED` added to money-flag set (dual-control + reason); approve audits `FLAG_AUTO_ORDER_SOAK_GATE`; runtime bypass (env/flag) writes deduped `AUTO_ORDER_SOAK_GATE_BYPASS` via PlatformAdminAudit. | Planning | `featureflags/service.go`, `retailer/auto_order_soak_gate.go`, bootstrap `SetSoakBypassAuditor` |
| P2-8 ✅ | **RESOLVED (2026-08-12).** AR invoices take ISO-4217 from order (credit leave) or fee schedule (billing); reject invalid codes; payment paths AssertSameCurrency when currency provided. Empty currency still falls back to UZS as last resort. | Money | `ar/service.go` OpenFromCreditLeaveRequest; order/driver_edges + shop-closed + cash collect; billing invoice_worker |
| P2-9 ✅ | **RESOLVED (2026-08-12).** `payment.NewService` no longer invents `dev-*` webhook secrets; production strips empty/`dev-*` so verify fails closed even if bootstrap validation is skipped. Local/SSMR still via bootstrap `envOr` + `ValidateProductionProfile`. | Money | `payment/service.go` `normalizeWebhookSecret`, `payment/webhook_secret_test.go` |
| P2-10 ✅ | **RESOLVED (W3 2026-08-12).** Global Pay refund `action=RF` proven against in-repo simulator (auth→perform). Live merchant confirm remains ops checklist. See [`GLOBAL_PAY_REFUND_PROOF.md`](../GLOBAL_PAY_REFUND_PROOF.md). | Money | `payment/gp_simulator_refund_test.go`; `simulator/global_pay.go` |
| P2-11 ✅ | **RESOLVED (W1 2026-08-12).** Twin Kafka consumer started (`void-digital-twin` on TwinConsumerTopics); telemetry envelope parsing fixed. TopicWebhooks **retired** (payment/partner stay on TopicMain/Orders; const kept for infra compat, no producers). | Data-flow |
| P2-12 ✅ | **RESOLVED (W1 2026-08-12).** Sell-through row + DEMAND_SIGNAL outbox emit in the same Spanner RW txn; FlywheelDemandFeed remains best-effort post-commit. | Data-flow |
| P2-13 ✅ | **DECIDED deferred (W1 2026-08-12).** Keep Spanner LIKE/prefix search; no ES/OpenSearch until scale/SLO evidence. See [`SEARCH_DECISION.md`](../SEARCH_DECISION.md). | Data-flow |
| P2-14 ✅ | **RESOLVED (2026-08-12).** Partner webhook URLs validated: https-required, no credentials, block localhost/metadata/private IPs after DNS; optional host allowlist. SSMR: `PARTNER_WEBHOOK_ALLOW_HTTP/PRIVATE`. | Data-flow | `partner/webhook_url.go` |
| P2-15 ✅ | **RESOLVED (2026-08-12).** `platform-admin` WS hub + `/v1/platform-admin/ws-session`; broadcasts on tenant/flag/MFA audit; admin console live refresh. Push/FCM N/A for break-glass desktop (WS is the path). | Data-flow | `ws/handler.go`, `platformadmin`, `admin-portal/lib/use-admin-ws-refresh.ts` |
| P2-16 ✅ | **RESOLVED (2026-08-12).** Flag set + approve write `PlatformAdminAudit` (fail-closed); `auth.ActorLabel` Subject→phone→role; mint-dev-jwt `--role PLATFORM_ADMIN --subject`. | Tenancy | `featureflags/handlers.go`, `auth/claims.go` ActorLabel |
| P2-17 ✅ | **RESOLVED (2026-08-12).** PLATFORM_ADMIN TOTP enroll/confirm/verify; JWT `mfa_verified`; step-up on tenants/flags; prod requires `PLATFORM_ADMIN_MFA_REQUIRED`; admin console MFA gate. HS256 JWT remains (not RS256). | Tenancy | `mfa/`, `auth` claim, `admin-portal` |
| P2-18 ✅ | **RESOLVED (2026-08-12).** CI jobs `enterprise-gates` (phase2→5c + analytics-tenancy; needs backend-spanner) + `admin-portal` (typecheck/build). `scripts/ci_enterprise_gates.sh` + `make ci-enterprise-gates` / `admin-portal-ci`. | Tenancy | `.github/workflows/ci.yml`, `scripts/ci_enterprise_gates.sh` |
| P2-19 ✅ | **RESOLVED (2026-08-12).** Metrics + TF alerts for relay restarts (`void_outbox_relay_restarts_total`), DLQ depth (`void_outbox_dlq_depth`), partner webhook success (`void_partner_webhook_success_ratio`). Phase-3 gate asserts stubs. | Tenancy | `telemetry/slo_metrics.go`, `outbox/metrics.go`, `infra/terraform/observability.tf` |
| P2-20 ✅ | **RESOLVED (W5 2026-08-12).** EDI-lite breadth: PRICAT/INVRPT/SLSRPT/RECADV/ORDCHG/DELFOR/REMADV build+parse; PRICAT→price upsert, INVRPT→stock upsert; others ledger+ACK. Still not certified EDIFACT/Drummond. | Integration | `partner/edi/breadth.go`, `partner/edi_inbound.go` |
| P2-21 ✅ | **RESOLVED (W5 2026-08-12).** Sandbox keys `pxs_*` (`environment=SANDBOX`, `RateLimitClass=partner_sandbox`); retailer/supplier self-serve key routes; sandbox orders tagged `PARTNER_SANDBOX`. | Integration | `partner/keys.go`, `partner/routes.go`, `partner/service.go` |
| P2-22 ✅ | **RESOLVED (2026-08-12).** Partner OpenAPI 1.6: admin/supplier/retailer key revoke + `/v1/supplier/partner-*` aliases (keys/sftp/webhooks/as2/coa) + admin SFTP; gate asserts. | Integration | `contracts/partner.openapi.yaml`, `Makefile` partner-openapi-gate |
| P2-23 ✅ | **RESOLVED (2026-08-12).** Warehouse `/cold-chain` + `/labor-capacity`; supplier `/labor-capacity` + Live Ops Map twin routes (inventory fetch on select). Twin was already Class A via ops map. | Client apps | `apps/warehouse-portal/app/cold-chain`, `*/labor-capacity`, `supplier-portal/.../ops/map` |
| P2-24 ✅ | **RESOLVED (W2 2026-08-12, already shipped).** Dedicated warehouse portal screens + nav for pick-waves, bins, cycle-counts (not TransferActions-only). | Client apps |
| P2-25 ✅ | **RESOLVED (2026-08-12).** Native Android/iOS payload apps already had full loading-bay UI; remaining gap vs Expo was factory merge (P1-18). Wired `/v1/factory/manifests` list + start-loading + seal alongside payloader/supplier; batch seal stays payloader-only. Expo remains field SoT; natives are Class A peers for factory↔payload. | Client apps | `payload-app-android` PayloadApi/Repository/HomeViewModel; `payload-app-ios` APIClient/HomeViewModel |
| P2-26 ✅ | **RESOLVED (W2 2026-08-12).** Supplier web `/planning` page with PlanningBrainPanel (S&OP + scenarios) + analytics-section nav; settings/planning remains overrides. | Client apps |

### P3 — polish / acknowledged-by-design

~~WhatsApp dunning channel absent (SMS/email only)~~ ✅ **2026-08-12** Twilio WhatsApp via `DUNNING_WHATSAPP_PROVIDER=twilio` + Content SID (`ar/dunning_channels.go`; owner: approved template + WhatsApp sender) · ~~WS JWT in URL query param (non-Firebase path)~~ ✅ **2026-08-12** query `?token=` accepts only `token_use=ws` short-lived tickets; session JWT via `Authorization: Bearer` (natives) or role `/ws-session` then query (browsers) · ~~ECC200 comment said 52×52~~ ✅ **2026-08-12** comments match 44×44 encoder cap · ~~control-tower scenariosData hardcoded empty~~ ✅ supplier control-tower ← planning scenarios · ~~Pulse/reports-export desktop-only~~ ✅ **2026-08-12** Android/iOS Reports Pro CSV share via `/v1/retailer/reports/export` + Dashboard network pulse strip via `/v1/retailer/pulse` (desktop remains primary deep-reports surface) · ~~POS-holds desktop-only~~ ✅ **2026-08-12** Android/iOS POS park/list/resume/void via `/v1/retailer/pos/holds` (**pilot default on**; set `POS_HOLDS_ENABLED=false` to hide / 404) · quantity negotiation intentionally disabled · ~~stale optimizer/soak docs (Part 3)~~ ✅ · ~~dead `optimizationjobs/` Go package~~ ✅ deleted (Spanner `OptimizationJobs` DDL retained).

### Security follow-through (from Domain 6.2 audit, re-verified 2026-08-12 · **W0 closed 2026-08-12**)

`driver.HandleOrderGet` was already FIXED (fail-closed ownership). **W0 Trust (2026-08-12) closed the remaining detail IDORs + JWT revocation:**

| Item | Status |
|------|--------|
| `payment.HandleListPayers` | Fixed — auth required; retailer/admin scoped to self; platform-admin may list |
| `payment.HandleGetPayer` / `HandleUpdatePayer` / `HandleCreatePayer` | Fixed — self-only for retailer/supplier ADMIN; platform-admin may any; create no longer allows ADMIN to mint foreign `payer_id` |
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
| `OPTIMIZER_AND_ROUTING_RUNTIME.md` | ~~SSMR optimizer "replicas 0"~~ | ✅ Synced 2026-08-12 — SSMR patch `replicas: 1`; prod still 0 until AR |
| `workers-kafka.md` | ~~optimizer-core "not in SSMR overlay"~~ | ✅ Synced 2026-08-12 — in SSMR overlay at replicas 1 |
| `DOMAIN3_PLANNING_PROGRESS.md` | ~~"no in-repo optimizer source/Dockerfile"~~ | ✅ Synced — `services/optimizer-core/` in-tree |
| `DOMAIN2_AUTONOMY_PROGRESS.md` | ~~soak 0.60 / Evaluate unwired~~ | ✅ Synced — default unmodified 0.80; `placeAllowedForRetailer` calls `Evaluate` (W4) |
| `AUTO_ORDER_PLACE_FLIP.md` | flip needs 30-day ≥80% artifact + audit | Aligned with runtime (0.80) + soak artifact scripts + `FLAG_AUTO_ORDER_PLACE` (W4); place env still false until flip |
| `PHASE4_COMPLETION.md:13` | "optimizer replicas ≥1 SSMR+staging OK" | Caveat remains: gate greps YAML intent; does not prove a healthy live optimizer pod |
| `PARTNER_EDI.md` | datamatrix / breadth notes | Updated W5 — FNC1 DataMatrix + PRICAT…REMADV |
| `END_PRODUCT_REALITY_REPORT` §5 | admin/SDK/datamatrix stubs | Historical snapshot only (banner on report); use gap register + master alignment |

---

## Part 4 — What "done" looks like (the coverage rule)

Closing condition (invariant):

> **Every Spanner state mutation emits an in-transaction outbox event; every event has a declared consumer (WS hub and/or webhook and/or push); no cross-role loop ends at an API with no client.**

**Checklist vs Part 2 (2026-08-12):**

| Step | Status |
|------|--------|
| 1. AR + Payout on outbox (P0-4/5) + AR invoice pay-down (P0-1) | ✅ |
| 2. P0 tenant + routing (P0-3, P0-6); payout bank-file decision (P0-2) | ✅ |
| 3. Run-mode push/inbox parity (P1-9); driver location on bus (P1-10) | ✅ |
| 4. Class A client loops (P1-17/18, P2-23/24/25/26) | ✅ |
| 5. W0 Trust IDOR + JWT revocation (P1-11 + security follow-through) | ✅ |
| 6. W3–W5 money/autonomy/partner gates in-tree | ✅ code/simulator; **owner residuals:** live Soliq EDS, GP merchant, dunning provider keys, prod optimizer replicas ≥1, auto-order place flip |

---

## Part 5 — Master alignment pointer (same-day re-verify)

Planning SoT for docs↔code↔role×platform + desktop stack decision:

- [`MASTER_ALIGNMENT_DATAFLOW_2026-08-12.md`](./MASTER_ALIGNMENT_DATAFLOW_2026-08-12.md)
- **Post–W0–W5 ordered residuals:** [`../PROD_READINESS_SEQUENCE.md`](../PROD_READINESS_SEQUENCE.md) (R0–R6 enterprise prod readiness)

Re-verify notes (2026-08-12 afternoon):

- AR/payout outbox emits **confirmed** in live `ar/service.go` + `payout/store.go`.
- Admin portal is a **real** Tenants/Flags/Audit console (not a stub).
- Twin consumer **started** (W1); TopicWebhooks **retired** (unused); search **Spanner LIKE by decision** — [`SEARCH_DECISION.md`](../SEARCH_DECISION.md).
- **W2 Class A loops closed 2026-08-12:** factory↔payload (P1-18), admin match/partner/dunning (P1-16), retailer AR/HQ mobile (P1-17), warehouse WMS screens (P2-24), supplier planning web (P2-26).
- **W3 money legality closed 2026-08-12:** EDS sign→submit→poll contract ([`FISCAL_EDS_PROOF.md`](../FISCAL_EDS_PROOF.md)); payout bank-file decision ([`PAYOUT_RAIL_DECISION.md`](../PAYOUT_RAIL_DECISION.md)); Global Pay `RF` simulator proof ([`GLOBAL_PAY_REFUND_PROOF.md`](../GLOBAL_PAY_REFUND_PROOF.md)).
- **W4 autonomy closed 2026-08-12:** optimizer-core no longer remapped to backend-go; soak artifact path + 80% threshold aligned; forecast/accuracy CronJobs on prod/SSMR; shadow soak flags on (place off); dual-control `Evaluate` + `FLAG_AUTO_ORDER_PLACE` audit. See [`AUTO_ORDER_PLACE_FLIP.md`](../AUTO_ORDER_PLACE_FLIP.md).
- **W5 partner cert closed 2026-08-12:** GS1 FNC1 DataMatrix + label path; AS2 MDN/MIC verify; SDK in go.work; EDI-lite breadth; sandbox keys.
- **P1-12 schema drift closed 2026-08-12:** 14 migration-only tables mirrored into `spanner.ddl`; offline + live schema-drift gate broadened.
- **P2-14 / P2-19 closed 2026-08-12:** webhook URL SSRF validation; SLO alerts for relay restarts / DLQ depth / partner webhook success.
- **P2-23 closed 2026-08-12:** warehouse cold-chain + labor-capacity UIs; supplier labor-capacity; twin Live Ops Map (inventory on select). Class A client loops for those APIs.
- **R4.1 closed 2026-08-12:** warehouse Android/iOS cold-chain + labor-capacity screens + nav (portal parity); Control Tower remains portal-primary.
- **R4.2 closed 2026-08-12:** retailer desktop `/control-tower` added to `RetailerShell` nav (was orphan page).
- **Docs alignment pass 2026-08-12:** historical reports/roadmaps bannered; living map in [`../DOCS_SOURCE_OF_TRUTH.md`](../DOCS_SOURCE_OF_TRUTH.md); no `.docx` in repo.
- **P2-16 closed 2026-08-12:** flag set/approve → PlatformAdminAudit fail-closed; ActorLabel for empty-Subject tokens.
- **P2-17 closed 2026-08-12:** PLATFORM_ADMIN TOTP MFA + `mfa_verified` step-up (HS256 JWT retained).
- **P2-18 closed 2026-08-12:** phase2→5c + analytics-tenancy in CI (`enterprise-gates`); admin-portal typecheck/build job.
- **P2-15 closed 2026-08-12:** platform-admin WS hub + admin console live refresh (push deferred by design for break-glass).
- **P2-22 closed 2026-08-12:** partner OpenAPI covers supplier/admin key revoke + SFTP + partner-* JWT aliases.
- Desktop stack recommendation: **keep Next.js + Tauri 2** (no Electron migration).

*Generated by consolidated audit, 2026-08-12; Part 5 appended after live re-verify.*
