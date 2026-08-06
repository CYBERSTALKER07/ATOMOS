# PegasusX — Platform Audit & Build Specification

Evidence base: the source tree at `/Users/shakhzod/ATOMOS/pegasusX` as of 2026-08-04. No repo markdown was trusted; every claim below traces to code, schema, or config. File:line references are given so each finding can be re-checked.

> **Runtime supersession (2026-08-06 / Gate-3 Wave 2C GS1 + 1C journals + 2B EDI-lite + collections):** OR-Tools code-wired but cloud heuristic-only — [`docs/OPTIMIZER_AND_ROUTING_RUNTIME.md`](./docs/OPTIMIZER_AND_ROUTING_RUNTIME.md). Geometry: Google Routes → OSRM → dense. Gate-0 Track A closed Spanner PITR/backup TF, outbox leases + Kafka `event_id` dedupe, SchemaMigrations, P0-4 iOS offline enqueue. **§8.6 credit terms/AR/aging + dunning step machine + `DelinquencyCount` bump + CREDIT_HOLD auto-freeze are live** behind flags — [`docs/CREDIT_ECOSYSTEM_BEHAVIOR.md`](./docs/CREDIT_ECOSYSTEM_BEHAVIOR.md); residual: no SMS/email/WhatsApp. **§8.9 Partner Integration Wave 1+2A+2B+2C + 1C journals shipped** — keys + `/partner/v1` + HMAC webhooks + bulk export/SFTP + EDI-lite + GLN/SSCC/ZPL + **journals CSV/XML** ([`docs/PARTNER_EDI.md`](./docs/PARTNER_EDI.md), [`docs/GS1_LABELS.md`](./docs/GS1_LABELS.md), [`docs/PARTNER_JOURNALS_1C.md`](./docs/PARTNER_JOURNALS_1C.md)); AS2 / configurable CoA still open. Integration scorecard **8/10**.

**Re-aligned 2026-08-04 (post Phase A/B G1–G3 + Gate-0 hygiene):** claims/receive/stock/eligibility/window snapshot are **live**; fiscal env-selected (`PEGASUS` default); credit scoring removed; Firebase client configs committed. **Gate-0 closed:** Claims in `spanner.ddl`, iOS snake_case decode, OrgFleet Android compile, optimizer minutes Time dim + empty-route reject, worker replicas=1, AutoConfirm sweeper flag, orphan `ledger/` deleted, Spanner backup/PITR, outbox leases, SchemaMigrations, P0-4 offline. Still open: single-supplier runtime, partner API (Gate 3 in progress), ML forecasting theatre.

---

## 0. Executive verdict

**Is it "just a stupid CRUD system"? No. Emphatically no.** A CRUD app has handlers calling an ORM. This has a transactional outbox whose event rows commit inside the same Spanner transaction as the state mutation, optimistic concurrency with real compare-and-swap on `Version`, an 18-state order machine with a centralized transition table, money as integer minor units with zero float money anywhere, bcrypt throughout, payment webhook verification that re-verifies settlement out-of-band against the gateway, Redis pub/sub WebSocket fan-out for horizontal scale, deterministic content-fingerprinted idempotency keys mirrored across web/iOS/Android, and a production config validator that fails closed on dev secrets. That is the work of people who have operated real transactional systems.

**But it is not the system the docs and naming claim it is.** Three structural facts dominate everything else:

1. **It is a single-supplier system, at runtime, by construction.** The schema is multi-tenant-shaped (`SupplierId` leads most keys), and supplier registration will mint up to 10 tenants — but `bootstrap/bootstrap.go` injects one `supplierSeed.SupplierID` into ~15 service constructors at process start, and `order.Service` holds it as a private `supplierID` field (`order/service.go` ~351) used as a constant on create/list paths. **The supplier the retailer picks in the UI is discarded during order creation.** Registering a second supplier today produces a tenant whose orders are attributed to the seed supplier. The data plane can hold 10 suppliers; the request plane can serve exactly 1.

2. **There is zero machine learning, and the "AI" layer is arithmetic.** The Python service's entire dependency list is one line: `ortools==9.15.6755`. No Vertex, Gemini, OpenAI, TensorFlow, PyTorch, ONNX, sklearn, XGBoost, Prophet, statsmodels, embeddings, or vector search anywhere in Go modules, requirements, Cargo, or package.json. The code says so itself: `planning/baseline_sources.go:14` — *"Never returns 'ml' — training inference is deferred"*; `cmd/planning-training-export/main.go:24` — *"collect-only; no training"*. Legacy synthesis still had `line.Quantity / 2` — **diverted** when `AUTO_ORDER_INVENTORY_GROUNDED` (§8.3); inventory `(R,s,S)` is the preferred qty path.

3. **There is no machine-to-machine integration surface at all.** Zero matches across the backend for `openapi`, `swagger`, `oauth2`, `client_credentials`, EDIFACT, X12, SSCC, GLN, ZPL, SAP, 1C, Odoo, NetSuite, BigQuery, SFTP. No outbound webhooks (the 5 webhook routes are inbound gateway receivers). No export endpoint of any kind. **A retailer with an ERP cannot integrate today by any path.** The only automated inbound channel is a human uploading a spreadsheet through a browser wizard and clicking approve.

**Scale, for calibration:** ~410k lines of real code — 131,670 Go / 81,116 TSX / 94,332 Kotlin / 74,434 Swift / 22,858 TS / 1,826 Python / 988 Rust / 32,118 YAML / 1,507 Terraform. 411 distinct HTTP endpoints, 73 Spanner tables, 12 native apps, 6 web surfaces. This is 18–30 months of competent engineering, not a prototype.

**Can it replace the field sales agent?** It replaces the agent's **order pad**, not the agent. The commercial loop (demand signal → proposal → confirmation → credit → pricing → allocation → dispatch) is genuinely automatable with flags that already exist. But picking, loading, driving, the delivery handshake, cash collection, and every exception path terminate in a human — and **off-app dunning** still needs SMS/email (in-app terms/aging/step machine/hold/FCM are wired). Honest number: **~35–40% of the agent's job is automatable with what exists, ~65% with the P1 work in §8, and the cash-collection half is structural, not a gap.**

**Scorecard:**

| Layer | Score | One-line justification |
|---|---|---|
| Go backend | **8/10** | Same transactional primitives; claims/receive/stock liability spine + claim-window snapshot now wired. Still loses points on duplicate event publishing and state-machine bypasses. |
| Domain modelling | **8.5/10** | 18-state order machine, two-sided delivery negotiation, volumetric dispatch, COD/credit/split payment, post-delivery claims ↔ quarantine ↔ reverse. Genuinely deep. |
| Web frontend | **6/10** | Excellent type hygiene and idempotency; no server-state library across 60k lines, 0% localized, accessibility unusable. |
| iOS | **5/10** | Modern SwiftUI/`@Observable`; Gate-0 removed convertFromSnakeCase decoder bug. Background location still misconfigured; P0-4 offline enqueue remains. |
| Android | **6/10** | Best offline queue in the repo; supplier Android compiles again (Gate-0 OrgFleet). |
| Infra / DevOps | **5/10** | Digest-pinned prod overlay + ManagedCertificate + ExternalSecret path WIRED; live apply / real GSM / CD still ops. |
| AI / optimization | **4/10** *(§8.5 constraints/multi-depot/OSRM matrix wired; cloud still undeployed)* | OR-Tools sidecar + Clarke-Wright exist; cloud dispatch heuristic until AR image/replicas; forecasting remains a 7-day mean. |
| Integration surface | **9.5/10** *(OAuth + GS1/ZPL + DESADV SSCC + CoA journals + EDI-lite + AS2 transport; certified 1C/EDIFACT still open)* | Partner keys + OAuth + `/partner/v1` + HMAC webhooks + bulk export/SFTP + EDI-lite + AS2 + GLN/SSCC/ZPL + journals CoA + OpenAPI. See [`docs/PARTNER_API.md`](./docs/PARTNER_API.md) / [`docs/PARTNER_AS2.md`](./docs/PARTNER_AS2.md). |
| Multi-tenancy (runtime) | **1/10** | One supplier, bound at startup. |

---

## 1. What is genuinely strong

Worth stating precisely, because the rest of this report is critical and the good work deserves to be identified so it isn't refactored away.

1. **Outbox events commit inside the state transaction.** `order/repository_spanner.go` (and payment/ledger-style writers) — a `spannerTxnBuffer` collects events *inside* the closure, then base + outbox mutations are written in one `txn.BufferWrite`. The event cannot commit without the state change.
2. **The buffer is built inside the closure**, so Spanner's automatic transaction retries cannot double-emit. This is the detail that separates people who have operated Spanner from people who have read about it.
3. **Optimistic concurrency is enforced, not decorative.** `order/repository_spanner.go:203-244` reads `Version` via `ReadRow` inside the RW transaction, rejects on mismatch, increments on commit.
4. **Out-of-band webhook re-verification.** `payment/global_pay_webhook.go:81-91` independently queries the gateway for authoritative status before accepting a settlement — this defeats forgery even with a leaked webhook secret. Rare outside fintech.
5. **Fail-closed production config validation.** `bootstrap/config_validate.go:11-49` rejects dev JWT secrets, all five dev webhook secrets, and memory fallback under `PEGASUSX_ENV=production`.
6. **Explicit state machine with reasoning in comments.** `order/state_machine.go:22` (the ADR-009 fiscal hard gate), `:40-43` (*"Without exits this status would brick the order"*) — that is someone who found the bug in production and wrote down why the fix exists.
7. **Deterministic idempotency keys across all three clients.** `packages/api-client/idempotency.ts` (142 typed key factories), mirrored in `driver-app-ios/.../DriverIdempotency.swift` and `driver-app-android/.../DriverIdempotencyKeys.kt`. Retry-after-reconnect is safe by design.
8. **Money discipline is perfect.** Integer minor units end to end, basis points for discounts, `Math.round(parsed * 100)` on input with comma-decimal handling for ru/uz. No float money in 410k lines.
9. **Cross-pod WebSocket fan-out with self-echo suppression.** `ws/hub.go:268-326` — Redis relay means no sticky sessions needed.
10. **Adaptive telemetry filter, implemented identically on both mobile platforms.** 15s heartbeat / 20m distance / 15° bearing (`TelemetryService.kt:159-176`, `FleetViewModel.swift:656-684`) cuts telemetry volume by ~an order of magnitude.
11. **A real durable offline queue on Android** with correct HTTP semantics — 409 purged as idempotent success, 5xx retried, 4xx discarded (`OfflineSyncWorker.kt:106-124`).
12. **Spanner emulator integration tests in CI** (`.github/workflows/ci.yml:53-68`). Most teams never build this.
13. **Code hygiene:** 1 TODO in 128k lines of Go, 0 `context.TODO()`, 3 non-test panics, zero `@ts-ignore`/`@ts-expect-error` in ~72k lines of TypeScript.
14. **`cmd/ecosystem-simulator`** drives a full multi-role lifecycle against a live API — a legitimately valuable harness that isn't wired into CI.
15. **Claims ↔ stock liability spine (Phase B).** `claims.Service` is constructed in `bootstrap/bootstrap.go`, mounted on `orderroutes` (`POST/GET …/claims`, `GET …/claim-eligibility`, approve/reject), with GCS fail-closed evidence, QUARANTINE hold (fail-closed + compensate), receive qty exclusion, reverse/WH fanout, COMPLETED claim-window snapshot + supplier/WH return-policy APIs. E2e markers include `PX_E2E_CLAIMS_*` / `PX_E2E_CLAIM_ELIGIBILITY_OK` / `PX_E2E_CLAIM_WINDOW_SNAPSHOT_OK`.

---

## 2. The theatre list

The single most important pattern in this codebase: **features ship the *interface* of a capability with none of the substance, and the deferral was never tracked.** This is more dangerous than missing features, because dashboards, policy toggles, and state machines all imply the thing works. (Count refreshed 2026-08-06 — AI confidence / touchless MinConfidence / promo attribution honesty; fiscal/billing/cold-chain remain partial; desktop portal i18n Wired as draft, mobile i18n still open.)

| # | Feature | What exists | What's actually there | Evidence |
|---|---|---|---|---|
| 1 | **Fiscalization (legal OFD)** | Full FISCALIZING → FISCAL_FAILED retry state machine, `OrderFiscalReceipts`, env-selected providers (`PEGASUS` / `FAKE` / `MY_SOLIQ` / `GLOBAL_PAY`) | **Not unconditional Fake anymore** — `defaultFiscalProvider()` → `ProviderFromEnv()`; SSMR/staging use `FISCAL_PROVIDER=PEGASUS` (platform commercial receipts). **Legal Soliq OFD still not production-ready** — `MY_SOLIQ` needs sandbox creds; misconfig returns hard-fail rather than silent Fake. | `order/fiscal.go` (`defaultFiscalProvider`); `order/fiscal_provider.go` (`ProviderFromEnv`) |
| 2 | **Marketplace commission** | 3 billing tables, meter + tier workers, consumer started | **Schema WIRED** (`EventId`/`MeterType`/`Amount`/`CurrentValue`). **Event amount decode FIXED** (`amount_minor` / nested `total.amount`). Residual: fee schedule + invoices. | `kafka/billing_tier_worker.go`; `internal/services/billing/meter_worker.go` |
| ~~3~~ | ~~**Claims / disputes**~~ | — | **Retired (2026-08-04).** Live service + routes + e2e. `Claims`/`ClaimEvidences` mirrored into `schema/spanner.ddl` (Gate-0). Portal claim-window settings UX still open. | `schema/spanner.ddl`; `claims/*` |
| ~~4~~ | ~~**Double-entry ledger**~~ | — | **Deleted (Gate-0).** Orphan `ledger/` package removed; live money path remains `PaymentLedgerEntries` / `ArLedgerEntries`. | — |
| ~~5~~ | ~~**AI confidence gate**~~ | `MinConfidence` knob + reject path | **WIRED (Gate-0).** Base score `0.15`; low-signal orders score &lt; default `0.55` and are dropped; `MinConfidence &gt;= 1.0` is reject-all. | `apps/ai-worker/synthesis/engine.go`; `engine_test.go` |
| ~~6~~ | ~~**Touchless replenishment policy**~~ | `MinConfidenceScore FLOAT64 DEFAULT 0.85` | **WIRED (Gate-0).** `TouchlessEligible` rejects when `confidence &lt; MinConfidenceScore` (default 0.85). | `replenishment/touchless.go`; `mei_engine_test.go` |
| 7 | **Auto-confirm of AI preorders** | `AutoConfirmAt` set to +24h; `AutoConfirmDueOrders` + worker ticker | **Wired behind `AUTO_CONFIRM_PREORDERS_ENABLED=1`** (Gate-0; default off). Without the flag, preorders stay PENDING until human confirm. | `runtime_workers.go`; `order/preorder_service.go` |
| 8 | **Seasonality** | Two templates with `Multiplier` ×1.35 / ×1.15 | **Partial:** applied on §8.1 baseline writes + replenishment suggested qty; HW-estimated template library still open. | `planning/forecast_runner.go`; `replenishment/seasonal.go` |
| 9 | **Weather & POS demand signals** | A `CompositeSignalProvider` "demand sensing stack" | Fake qty stubs **removed** (§8.1); providers return empty until real weather/POS APIs. | `predictivepush/signals.go` |
| 10 | **Price elasticity / promo simulation** | Promo simulator with projected volume, margin, and a "closed-loop score" | **Partial (2026-08-06):** caller-supplied `elasticity` (default 0.5) persisted as `elasticity_used`; closed-loop actuals attributed via `LineItemsJson.promotion_id` (units + line totals), empty promo → zeros, `attribution=line_promotion_id`. Still a **sandbox heuristic**, not demand-model elasticity curves. | `planning/promo_eval.go`; `promo_eval_test.go` |
| 11 | **Forecast accuracy** | A MAPE figure shown to suppliers | **Wired (§8.4):** `ForecastAccuracyDaily` + nightly job; portal reads server WAPE/bias/TS. Residual: enable `FORECAST_ACCURACY_ENABLED` + apply migration in each env. | `planning/accuracy.go`; `GET .../demand/accuracy` |
| 12 | **Cold chain** | `RequiresColdChain`, `StorageTempMinC/MaxC`, `TEMPERATURE_BREACH`, WMS readings | **Partial:** product/order cold+hazmat flags reach OR-Tools (§8.5); WMS temperature ingest exists (`WMS_COLD_CHAIN_ENABLED`). Residual: always-on prod sensor fleet / full excursion automation. | optimizer contract; `docs/WMS_COLD_CHAIN.md`; `dispatch/volume.go` |
| 13 | **Multi-currency** | **19** `Currency` fields in `schema/spanner.ddl` (orders/payments/ledger threaded) | **Partial→Wired (Wave 1, 2026-08-06):** `FxRates` + `fxrates.ConvertMinor` (fail closed, no silent 1:1); payment checkout/chargeback/webhook **currency_mismatch** gate; bootstrap UZS identity (+ optional `FX_SEED_USD_UZS_SCALED`); admin `GET/PUT /v1/admin/fx-rates`. **Residual:** multi-currency settlement ledger, Airwallex live FX, client currency pickers. | `fxrates/`; `docs/FX_RATES.md`; migration `20260806_fx_rates.ddl` |
| 14 | **i18n** | ~2750 keys × en/ru/uz (`portal.*` + `supplier_portal.*` / `warehouse_portal.*` / `factory_portal.*` / `retailer_desktop.*`), generated to web JSON + iOS `.strings` + Android `strings.xml` | **Desktop portals UI: Wired (draft translations, 2026-08-06)** — four desktop portals (`supplier-portal`, `warehouse-portal`, `factory-portal`, `retailer-app-desktop`) use `usePortalT` across app/components user-visible UI; catalogs regenerated. **Mobile still Partial / unwired.** Draft ru/uz — not certified linguistic Done. Do not claim full-platform i18n Done. | `packages/i18n/*`; desktop portal `app/` + `components/` |

Two more that are naming rather than function: `RunMEIONetwork` ("Multi-Echelon Inventory Optimization") is a two-node greedy donor/receiver swap per SKU (`replenishment/mei_engine.go:168`); `SourceFallback` was renamed off fake `KMEANS_BINPACK` (Gate-0).

**Recommendation:** every item above is either wired, deleted, partial with an honest residual, or explicitly renamed. **Operating instruction:** [`docs/SUBSTANCE_GATE.md`](docs/SUBSTANCE_GATE.md) — SG algorithm + per-role / per-platform E2E verification against `PX_E2E_*` markers.

---

## 3. Correctness defects that will bite in production

Ranked. These are bugs, not gaps.

**P0-1 — ~~The deployed route optimizer has a unit error that silently produces zero manifests.~~ Fixed (Gate-0).**
Time dimension now uses per-vehicle **minutes** callbacks (`AddDimensionWithVehicleTransits`); `AddDisjunction` drops infeasible stops; Go client rejects zero-route optimizer responses and falls back to H3 BinPack (`contract_solver.py`, `dispatch/plan/optimize.go`). Redeploy optimizer-core image to pick up Python fix.

**P0-2 — ~~`supplier-app-android` does not compile.~~ Fixed (Gate-0).**
Deleted wrecked `orgfleet/components/*` duplicates; parent `OrgFleetScreen` + `FormPickers` remain. `compileEnterpriseDebugKotlin` + `compileStoreDebugKotlin` green.

**P0-3 — ~~Driver iOS cannot decode any response.~~ Fixed (Gate-0).**
Removed `.convertFromSnakeCase` from driver + payload `APIClient.swift` so explicit snake_case `CodingKeys` match.

**P0-4 — ~~Server rejections become "successful" offline deliveries on iOS.~~ Fixed (Gate-0 / client parity).** `FleetServiceLive` enqueues only via `DriverOfflineActionCatalog.isNetworkEnqueueable`; business 4xx re-thrown; `requireCurrentLocation` refuses `(0,0)`.

**P0-5 — ~~Outbox double-publish.~~ Mitigated + leased (Gate-0 Track A).** Worker replicas=1 short-term; `ClaimedBy`/`ClaimedUntil` lease on fetch + Kafka `event_id` dedupe landed. Multi-replica relay still needs ops soak before scaling workers.

**P0-6 — ~~No Spanner backup exists.~~ Closed (Gate-0 Batch B).** PITR 7d + backup schedule TF + restore rehearsal RTO ~30 min; GCS remote TF state. See `artifacts/GATE0_SPANNER_BACKUP_RESTORE_*`.

**P0-7 — ~~Failed migrations report success / no version table.~~ Closed (Gate-0 Track A).** `SchemaMigrations` + narrowed benign DDL set in `cmd/apply-migration`.

**P0-8 — ~~The prod overlay is undeployable.~~ Partially closed (Gate-0 images/TLS + P0-8 secrets path).**

| Slice | Status |
|-------|--------|
| Placeholder / `:latest` / `:local` images | **Closed (Gate-0)** — digest pins in `overlays/prod`; `scripts/ci_fail_placeholder_images.sh` green |
| TLS `pegasusx-api-tls` | **Closed for prod render** — ManagedCertificate + ingress patch (no `secretName` after kustomize) |
| ExternalSecret / GSM | **Repo-WIRED** — TF shells for all 12 ES GSM names; `phase0_sync_gsm_secrets.sh` can stub unused PSP rails (`unused-rail-placeholder`); prod overlay includes SecretStore+ExternalSecret; SSMR maps `redis-password` |
| Residual (ops) | Real GSM versions (JWT, internal-api-key, Maps, GP merchant password, redis AUTH); `kubectl apply` prod overlay; DNS → LB for ManagedCert Active; real optimizer-core AR image + replicas ≥ 1 |

**P1 — Others worth naming:**
- **`Claims` / `ClaimEvidences` missing from `schema/spanner.ddl`.** Tables exist in `schema/migrations/20260728_logistics_claims.ddl` (and are used by the live claims package). Greenfield `spanner.ddl` applies and schema-drift CI that only diffs the monolith file will miss them — mirror the CREATE TABLEs into `spanner.ddl`.
- **Payment bypass violates the fiscal hard gate its own test asserts.** `order/supplier_ops.go:237` writes `Status: COMPLETED` from `AWAITING_PAYMENT` via a raw `UpdateMap`; `state_machine_test.go:40-44` explicitly asserts that transition must be blocked. The test passes; production skips fiscalization on a real money path. The validator has **4 call sites against 12 direct status assignments**, and `ARRIVED_SHOP_CLOSED` has no inbound edge in the table yet is written at `shop_closed.go:151`.
- **Plaintext `"4321"` written into the `PinHash` column.** `warehouse/ops_fleet_handlers.go:77` — either every ops-created driver is locked out, or some path does a plaintext compare and they all share PIN 4321.
- **Monotonic outbox and audit primary keys guarantee a Spanner hotspot.** `outbox/outbox.go:211-213` returns `evt_<UnixNano>` under a comment claiming "Production uses crypto/rand UUIDv7". That is the PK of `OutboxEvents` and `AuditLog`. Every insert in the fleet lands on one split. `uuid` is already a direct dependency.
- **Idempotency keys are globally scoped** — `"idem:" + rawHeader`, no principal, no route (`idempotency/redis_store.go:29`). Caller B with the same key and body gets caller A's cached response.
- **Nil Spanner client makes every write silently succeed** — `spannerutils/retry.go:20-22` returns `nil` when `client == nil`. Misconfiguration becomes silent data loss reported as HTTP 200.
- **32-bit FNV hash guards checkout idempotency.** `packages/api-client/idempotency.ts:1-8` — ~50% collision probability around 77k distinct values; a collision means a different cart produces the same key and the second checkout is silently rejected as a duplicate. Lost order, no error surfaced.
- **`AbortSignal` is accepted but never plumbed into `fetch`.** `usePolling.ts:27,70` hands callers a signal; `RequestOptions` has no `signal` field. All 8 call sites do one pre-flight `if (signal.aborted) return` then issue an unabortable request — so a slow response for filter A overwrites fresh data for filter B. The code *looks* like it handles the race.
- **`usePolling` swallows every error** — `usePolling.ts:72-75`, the catch branch body is empty. Dashboards show indefinitely stale data during an outage with no indication.
- **Four dead chart components render permanently empty in production analytics pages** across all four portals (`SpendAnalytics.tsx:22`, `ProductionForecastChart.tsx:19`, `RevenueHeatmap.tsx:16`, `VelocityGauge.tsx:14`). Each has `// Removed Mock Data` above an empty array fed straight into a chart, and each is mounted in a live route. They take no data props and make no fetch.
- **Tokens in JS-readable cookies plus localStorage.** `supplier-portal/lib/auth.ts:58-63` — no `HttpOnly`, no `Secure`, 7-day refresh token, mirrored into `localStorage`, in a codebase with no CSP.
- **Production Tauri configs ship the dev updater public key** — byte-identical to `contracts/desktop-updater/dev.pub` in all four apps, with `"installMode": "passive"` (silent install), and `apply_desktop_updater_pubkey.sh:7` **defaults to the dev key rather than failing**.
- **Desktop SQLite cache has no TTL and is never cleared on logout.** `updated_at` is written and never read; `cacheDelete` is called by no application code; retailer keys are unscoped (`hooks.ts:63` caches by raw URL). A second retailer on the same install is served the first one's profile, orders, and pricing.
- **An app update wipes unsynced deliveries.** `NetworkModule.kt:88-91` uses `.fallbackToDestructiveMigration()` on a DB already at version 4.
- **Push / Firebase OTP still ops-blocked.** Client configs are now committed (`google-services.json` / `GoogleService-Info.plist` under each mobile app; Android google-services plugin wired). Remaining blockers: missing/incomplete `.entitlements` / `aps-environment` for APNs, uneven `FirebaseMessagingService` manifests, and **owner** Firebase Phone SHA-1 + real SMS. Do not treat configs as “push works in release.”
- **`AIPredictions.AggregateId` overflow.** `predictivepush/audit.go:51` writes `retailerId + ":" + productId` (73 chars) into `STRING(36)`. Step 3 of the daily cron fails on every run with real UUIDs. The synthesis engine already hit this and worked around it; nobody fixed this path.
- **`DelinquencyCount` is now bumped** on first OVERDUE via dunning (`credit.BumpDelinquency`). **Credit risk scoring remains deliberately removed** (Phase A) — CREDIT_LEAVE / placement use status + available only.
- **HPA targets 7 millicores.** `hpa.yaml:21` sets 70% CPU against a `cpu: 10m` request. An idle pod exceeds it and pins to `maxReplicas: 12`, which a 1–3 node e2-medium cluster cannot schedule.
- **OSRM will crash-loop** — `osrm/deployment.yaml:26` runs `osrm-routed /data/region.osrm` with no volume, no PVC, no init container, 512Mi for a multi-GB extract, and no liveness probe. And the solver never calls OSRM anyway: distances are haversine (`contract_solver.py:34-43`), which underestimates by 20–40% in a dense street grid, so every route is over-committed on time.
- **No CI compiles the 167k lines of mobile code**, no linter runs anywhere (no `.golangci.yml`, no ESLint step, no SwiftLint/detekt/ktlint config in the repo), no `-race`, and no security scanning of any kind. This is *why* P0-2 and P0-3 are in `HEAD`.
- **Repo hygiene:** a 53 MB compiled `ai-worker` binary and an 8.7 MB `handoff-service` binary are **tracked in git**; an unrelated personal CV site with ~40 MB of PNGs is vendored in; 8 `patch_*.sh` source-mutation scripts sit at the root; 308 files are uncommitted; `cmd/gen-contracts` is a **symlink to a path outside the repository** that CI invokes twice.

---

## 4. Duplication: the structural tax

| Surface | Sharing | Measured duplication |
|---|---|---|
| Web portals | `packages/types` (3,427 lines) and `api-client` (2,811) are properly shared | **12–15%** of 59,527 portal lines is cross-app copy-paste; 26 byte-identical file pairs; `SupplierShell.tsx` ↔ `WarehouseShell.tsx` are **84.6% identical (527 shared lines)**. `packages/ui-kit` is only 1,738 lines serving 59,527 — undersized 3–4×. |
| iOS | **0.8%** — 560 shared lines against 73,874 | Enterprise updater 1,570 → 1,238 redundant; HTTP plumbing 2,303 → 1,620. `AutoUpdater.swift` diffs to **zero lines** between apps. Driver and retailer share nothing at all. |
| Android | **1.0%** — 929 shared lines against 93,403 | Updater 2,023 → 1,667; WS envelope 1,984 → 1,518. No `gradle/libs.versions.toml` — every version is a string literal in 6 separate files. |

**~12,000 lines across mobile and ~8,000 across web are pure copy-paste.** This is the direct cause of the two worst systemic gaps: every accessibility fix and every localization fix must currently be applied 4–6 times, which is why neither ever happened.

---

## 5. Can it replace the field sales agent?

Twenty-two steps from need-detection to cash in the bank. "Automatable" means a flag, policy, or sweeper already exists.

| # | Step | Human needed? | Status |
|---|---|---|---|
| 1 | Detect the reorder need | No | **Automated** — `AIPredictions`, predictive push |
| 2 | Retailer confirms the suggestion | Default yes | **Automatable** — 5 per-scope auto-order toggles (global/category/supplier/product/variant) |
| 3 | Supplier accepts the order | Default yes | **Automatable** — midnight-guard sweeper → `AUTO_ACCEPTED` |
| 4 | Credit decision | No | **Automated** at placement — **limit + status only** (scoring removed); `DelinquencyCount` bumped by dunning on first OVERDUE |
| 5 | Price determination | No | **Automated** — override → promotion → list |
| 6 | Stock allocation / backorder | No | **Automated** — per-SKU policy over warehouse default |
| 7 | Delivery date agreement | Sometimes | **Partial** — auto by default; the negotiation path needs a human on both sides |
| 8 | Transfer approval | Sometimes | **Automatable** — touchless with a daily unit budget; CRITICAL always escalates by design |
| 9 | Dispatch planning & driver assignment | Optional | **Automatable** — 60s auto-dispatch worker, real closed loop |
| 10 | **Physical picking** | **YES — hard blocker** | **Absent at warehouse.** No WH pick list/bin/lot. (Retailer on-hand **does** exist: `RetailerStockBalances` / receive sessions.) |
| 11 | **Truck loading** | **YES — hard blocker** | Absent by design — a 26-endpoint payload terminal exists so a human can drive it |
| 12 | Manifest sealing | Yes | Partial — `seal-all` batches the clicks |
| 13 | **Driving** | **YES** | — |
| 14 | QR handshake | **YES — by design** | Geofence auto-detects arrival; the handshake is an intentional two-party control |
| 15 | Offload & condition verification | **YES** | Still human-reported at dock; **post-delivery** damage/shortage/concealed now file as claims → QUARANTINE + reverse |
| 16 | **Cash collection** | **YES — structural** | Absent by nature. The most developed path in the platform *because* it's manual |
| 17 | Card capture | No | **Automated** for Global Pay; stubbed elsewhere |
| 18 | Fiscal receipt | No | **Automated shape; PEGASUS commercial receipts live** — legal Soliq OFD still open (§2.1) |
| 19 | Reconciliation | On exception | Partial — detection automated, resolution manual |
| 20 | Returns disposition | Yes | **Stronger** — FileClaim + approve/reject + stock hold + reverse open; human still confirms disposition |
| 21 | Exceptions (shop closed, delay, overflow, rescue) | **YES** | Absent — every path ends in a human decision |
| 22 | **Dunning / collections** | Partial | **In-app wired** (terms/due/aging/step machine/hold/FCM+inbox) behind `AR_DUNNING_ENABLED`; **off-app SMS/email still absent** |

**The honest read.** Steps 1–9 — the whole commercial decision loop — are genuinely automatable today, and that is more than most SFA vendors ship. But the automation is only as good as its inputs, and right now step 1 produces `last_order / 2` even though **retailer on-hand balances now exist** — the auto-order path still does not consult them. An agent walks in, looks at the shelf, knows what sells, knows what's near expiry, knows the promo calendar, and negotiates. This halves the last invoice.

Two blockers are structural rather than unbuilt:
- **Cash collection.** In a COD-dominant market the driver *is* the collections function. The platform doesn't remove that human, it instruments him — arguably the correct product decision, but it converts field headcount from sales to logistics rather than eliminating it.
- **Dunning.** In-app step machine + hold + FCM/inbox are wired; the remaining gap for field-agent replacement is **off-app SMS/email** so retailers without the app still get reached.

---

## 6. Market reality — the premise needs correcting

The premise that "there is no such system in the world" is **not accurate**, and building on it is strategically dangerous. Three established categories already occupy this space:

**B2B wholesale marketplaces connecting many suppliers to many retailers.** Udaan (India, FMCG/staples/pharma, ~$917M deployed, settled a $178M Singapore insolvency in 2026, now EBITDA-positive per city across ~16 clusters and heading for an India listing), MaxAB-Wasoko (Africa, all-stock merger, 450k+ merchants), TradeDepot (Nigeria), Ankorstore (Europe, moved to 0% commission on reorders in January 2026), Faire, Frubana, Sabi, MarketForce.

**Sales-force automation / distributor management, which is exactly the "replace the agent's order pad" product.** FieldAssist (32+ countries, offline-first, distributor management + retailer ordering app + route optimization + planogram audits), Botree, PepUpSales, Massist. Predictive order recommendations before the rep walks in are a *standard* feature in this category, not a differentiator.

**Agentic order entry.** Proton.ai shipped GA agentic order & quote entry in July 2026 — it reads emails, PDFs, handwritten lists, and screenshots, applies contract pricing and sourcing rules, picks the warehouse, surfaces substitutes, and hands the rep a sendable draft. In a head-to-head at a large industrial distributor it found the right part 3× more often than the in-house cross-reference system on description-only lists and cut list-prep time 75%.

**The critical lesson from that landscape, and it is directly relevant.** The pure marketplace/e-commerce thesis largely *failed*. MaxAB-Wasoko retreated from 8 markets to 5 and is betting the company on **embedded fintech**; in Egypt fintech transactions already outpace e-commerce. TradeDepot pivoted to **advertising and data**. MarketForce shut its e-commerce arm. Sabi went to commodity exports. Udaan survived by abandoning geographic breadth for **city-level density** and private label (now 15% of revenue). Ankorstore gave up reorder commission entirely and monetizes via **subscription + fintech**.

Translated for PegasusX: **distribution margin does not pay for this platform. Credit does.** Terms/AR/aging/dunning/`DelinquencyCount`/CREDIT_HOLD are now code-wired (flag-gated); **risk scoring stays removed** (Phase A). The remaining collections gap is **off-app reach** (SMS/email) for retailers without the app.

**What is genuinely defensible here.** Not "a platform connecting all suppliers and retailers" — that exists and has been expensively fought over. What is unusual is the **vertical depth of a single stack from factory floor to shop shelf**: factory manifests → inter-hub transfers → warehouse loading bay with a dedicated terminal role → volumetric truck packing in VU → geofenced driver handshake → COD/split-payment/credit-delivery with ledger and reconciliation → fiscal receipt. Most competitors are a marketplace bolted onto 3PL, or an SFA app bolted onto someone else's ERP. **Nobody has the whole physical chain in one transactional model with one event bus.** That is the asset. It happens to be worth more as *deep software for one large distributor* than as a thin marketplace for many.

---

## 7. Fit with what retailers and suppliers already run

This is the adoption wall, and it is higher than the technical gaps.

A mid-size Uzbek/CIS distributor runs **1C** for accounting and often stock. A retail chain runs 1C or SAP, plus a POS system, plus possibly a WMS. Their reality: they will not re-key orders into your app, they will not run their inventory in your database, and they will not accept a system that can't produce a journal entry their accountant recognizes.

What the code offers them today:

| They need | PegasusX has |
|---|---|
| An API their ERP can call | **Partner API keys + OAuth2 client_credentials + `/partner/v1` + OpenAPI** (Wave 1). Human JWT remains for portal. |
| Events pushed to their system | **Outbound HMAC webhooks** (Wave 1) + list/deactivate/replay (Wave 2A). Inbound payment gateway webhooks unchanged. |
| A file drop | **Bulk export API + optional SFTP** (Wave 2A). EDI/AS2 still open. |
| EDI (ORDERS/ORDRSP/DESADV/INVOIC) | **EDI-lite over SFTP** (Wave 2B) — UNA segment dialect; no AS2 / certified EDIFACT. See [`docs/PARTNER_EDI.md`](./docs/PARTNER_EDI.md). |
| Bulk load | **The one real primitive:** a 9-state import wizard (`supplier/import_sessions.go:177`) with signed-URL upload, column auto-discovery, mapping, staging with raw+cleaned JSON, and error summaries. Production-grade — but **inventory/product only, one-way, human-driven**. |
| GS1 identifiers | **GTIN + GLN + SSCC + ZPL (Wave 2C) + DESADV GIN+BJ.** Shared `gs1/` package; GLN on org profiles; SSCC-18 on `ManifestShipUnits` at seal; ZPL label API; DESADV emits CPS/PAC/GIN from ship units. See [`docs/GS1_LABELS.md`](./docs/GS1_LABELS.md) / [`docs/PARTNER_EDI.md`](./docs/PARTNER_EDI.md). |
| Label printing | **No ZPL.** Scanning in, nothing out. |
| Accounting export | `PaymentLedgerEntries` and `MasterInvoices` internally; **no journal export, no chart-of-accounts mapping**. |
| Legal receipt | `FISCAL_PROVIDER=PEGASUS` issues platform commercial receipts; **`MY_SOLIQ` OFD adapter exists but needs sandbox/prod creds.** You cannot legally close a Soliq-mandated sale until L5 lands. |
| Data warehouse | **Zero BigQuery references.** |

**Superseded (Wave 1+2A+2B+2C + journals + DESADV SSCC + CoA + OAuth + AS2):** partner keys, `/partner/v1`, **OAuth2 client_credentials**, webhooks, bulk export/SFTP, EDI-lite (incl. DESADV CPS/PAC/GIN+BJ), **AS2 transport** (not Drummond), GLN/SSCC/ZPL, **1C journals CSV/JSON/XML**, and **configurable CoA** — see [`docs/PARTNER_API.md`](./docs/PARTNER_API.md) / [`docs/PARTNER_AS2.md`](./docs/PARTNER_AS2.md). Residual: certified EDIFACT, certified 1C exchange package.

The fix is small relative to its impact, because the hard part is already built: `OutboxEvents` + Kafka + a typed event registry (`registry.json`, 107KB of event shapes) is exactly the substrate an outbound webhook system needs. Items 1–4 in §8.9 are perhaps **5% of the engineering already invested** and are the difference between selling to one distributor's field team and selling to a retail chain.

---

## 8. Build specification

Each item: why it's needed, the algorithm with its math, inputs/outputs, where it plugs into this codebase, and how to prove it works. Ordered P0 (broken) → P1 (unlocks autonomy) → P2 (unlocks enterprise) → P3 (unlocks marketplace).

### 8.0 P0 — Fix what is broken before building anything

These are one-to-few-line fixes with outsized consequences. Do them first; several are blocking the ability to observe whether anything else works.

| Fix | Where | Detail |
|---|---|---|
| Time dimension in minutes | `contract_solver.py:145-184` | Register a second callback `travel_minutes(d_ij, speed) + service_minutes[i]` and pass *that* index to `AddDimension`. Add `routing.AddDisjunction([idx], penalty)` per node so infeasible stops drop individually instead of collapsing the model. |
| Detect zero-route responses | `dispatch/plan/optimize.go:84-110` | Reject when `len(res.Routes)==0 && len(job.Orders)>0` so the bin-pack fallback actually engages. |
| Delete one line, twice | `driver-app-ios/.../APIClient.swift:76`, `payload-app-ios/.../APIClient.swift:72` | Remove `.convertFromSnakeCase`; the explicit `CodingKeys` are already correct and exhaustive. |
| Invert offline enqueue logic | `FleetServiceLive.swift:57-64` | Re-throw on `APIError.problemDetail/.forbidden/.httpError(4xx)/.decodingError`; enqueue only on genuine transport failure. Make `currentLocation()` return `CLLocationCoordinate2D?` and throw `.locationUnavailable` instead of `(0,0)`. |
| Restore `supplier-app-android` | `ui/orgfleet/` | `git checkout` the pre-refactor file, delete `components/`, split with an AST-aware tool. Delete `refactor_android.py`, `refactor_ios.py`, `fix_imports.py`, and the 8 root `patch_*.sh`. |
| Lease the outbox relay | `outbox/spanner_store.go:105` | Add `ClaimedBy`/`ClaimedUntil`; make `Fetch` a read-write transaction that stamps a lease. Dedupe on `Event.EventID` via a Kafka header, not offset (`kafka/event_dedup_middleware.go:16`). Short term: worker `replicas: 1`. |
| Spanner backups | `infra/terraform` | `google_spanner_backup_schedule` (daily, 7–30d) + `version_retention_period` for PITR. **Then execute a restore into a scratch DB and record the RTO.** |
| Migration integrity | `cmd/apply-migration/main.go:267` | Remove `FailedPrecondition` from the benign set; narrow `InvalidArgument` to exact matches. Add `SchemaMigrations(version, checksum, applied_at)` and refuse a version whose checksum changed. Remove `migrate-job.yaml` from `base/kustomization.yaml:15`; run it as a gated pipeline step with a `-<git-sha>` suffix. |
| Terraform state to GCS | `backend.gcs.tf.example` | Activate against a versioned bucket, `state push` serial 143, delete the local file, **rotate the 5 secret values that were in plaintext state**. |
| ~~Wire the auto-confirm sweeper~~ | `runtime_workers.go` | **Done (Gate-0)** — ticker behind `AUTO_CONFIRM_PREORDERS_ENABLED=1` (default off). |
| Enforce `MinConfidenceScore` | `replenishment/touchless.go:41-52, 242-253` | Thread the policy value in. Property test: `MinConfidenceScore = 1.0` must auto-approve nothing. |
| Fix `AggregateId` overflow | `predictivepush/audit.go:51` | Write a UUID; move the composite key into `PredictionData`, matching what `synthesis` already does. |
| LIFO loading order | `warehouse/dispatch_execute.go:240`, `supplier/dispatch_execute.go:392` | `LoadingOrder = stopCount - 1 - idx` for rear-loading vehicles; add `LoadingPattern` to `Vehicles` for side-loaders. Today the first delivery is loaded at the *back* of the box — the loader must unload the whole truck at stop 1. |
| Real idempotency hash | `packages/api-client/idempotency.ts:1-8` | SHA-256 truncated to ≥128 bits, or a server-minted token. Add a collision test over a large synthetic cart corpus. Drop the time bucket from `driverFiscalRetryKey:28` — it defeats its own purpose on a tax receipt. |
| UUID event IDs | `outbox/outbox.go:211` | `uuid.NewString()`. Already a direct dependency. |
| Scope idempotency keys | `idempotency/middleware.go` | `sha256(principalID + "|" + routePattern + "|" + rawKey)`. |
| Fail loudly on nil client | `spannerutils/retry.go:20`, `chunker.go:21` | Return an error. Production already blocks memory fallback. |
| Fix the driver PIN | `warehouse/ops_fleet_handlers.go:77` | Generate random, store bcrypt, return plaintext once — mirroring `supplier/onboarding_handlers.go:319-324`, which already does it correctly. |
| CI gates | `.github/workflows/` | Add: `xcodebuild` for 6 iOS apps (XcodeGen first), `./gradlew assemble+test` for 6 Android roots, `golangci-lint`, `-race`, ESLint with `jsx-a11y`, `govulncheck`, Trivy, CodeQL, secret scanning. Align CI's Go 1.22 with the Dockerfiles' 1.25. Fail on any rendered image that is a placeholder or `:latest`. **This one gate would have caught three of the P0s above.** |

Also in P0 hygiene: `git rm --cached` the 53 MB and 8.7 MB binaries; commit the missing `.env.ssmr.example` and `.env.k8s.example` (their absence breaks the first two documented onboarding commands); vendor `cmd/gen-contracts` in place of the external symlink; delete or wire the four dead chart components; add `securityContext` to all 7 workloads and a default-deny NetworkPolicy pair.

---

### 8.1 Demand forecasting — WIRED (Croston / SES / Holt–Winters)

**Shipped.** Package `planning/forecast` (classify + Croston SBA + SES + Holt–Winters m=7 + holdout fit). Nightly `cmd/planning-forecast` (CronJob 02:00 UTC) loads COMPLETED `LineItemsJson` via `LoadCompletedActuals`, writes `DemandForecastBaseline` through `WriteBaselineWithOutbox` with residual p10/p90 bands, seasonal `Multiplier`, and §8.4 WAPE confidence when available. Flag `FORECAST_ALGO_ENABLED`; ops `POST /v1/admin/planning/forecast/run-once`; backtest `-mode=backtest` (+ optional `FORECAST_ALGO_REQUIRE_GATE`). Predictive-push skips baseline overwrite when algo on; fake weather/POS qty stubs removed. Docs: [`docs/FORECAST_ALGO.md`](./docs/FORECAST_ALGO.md). Marker: `PX_E2E_FORECAST_ALGO_OK` / `_SKIPPED`.

**Residual.** HW-estimated seasonal template library; real weather/POS feeds; replenishment burn still uses 7-day mean until §8.2.

---

### 8.2 Safety stock with an explicit service level — WIRED

**Shipped.** Flag `SAFETY_STOCK_V2_ENABLED`: when on, `SS = z_α · √(L·σ_d² + d̄²·σ_L²)` and `ReorderPoint = d̄·L + SS` in `replenishment/safety_stock.go` + `engine.go` (MEIO/echelon share the helper). Flag off keeps legacy `burn·lead·1.15`.

- `FactoryInternalTransfers.ReceivedAt` written on warehouse receive + factory INTERNAL create-as-RECEIVED; observed L/σ_L when ≥10 samples
- `ReplenishmentPolicies`: `TargetServiceLevel` / `LeadTimeDays` / `LeadTimeSigmaDays`; GET/PATCH + supplier portal knobs
- `InTransitQty` from open transfers ⋈ insights; `DemandBreakdown` carries `safety_stock`, `z_alpha`, `sigma_*`, `*_assumed`
- Docs: [`docs/SAFETY_STOCK.md`](./docs/SAFETY_STOCK.md). Marker: `PX_E2E_SAFETY_STOCK_OK` / `_SKIPPED`

**Residual.** Fill-rate replay: `cmd/safety-stock-replay` + `POST /v1/admin/planning/safety-stock/replay` (gate via `SAFETY_STOCK_REPLAY_REQUIRE_GATE`). Retailer `ReorderSuggestions.SafetyStock` uses the same SS helper when `SAFETY_STOCK_V2_ENABLED` (else legacy `demand·0.15`). §8.3 inventory-grounded shadow auto-order — **wired**.

---

### 8.3 Ground the auto-order in inventory, not in the last invoice — WIRED

**Shipped.** Inventory-grounded `(R,s,S)` proposals from `RetailerStockBalances` + sell-through / OPEN `ReorderSuggestions` + §8.2 ROP; confidence decay from stock `UpdatedAt` (option c). Global `execution_mode`: `off|shadow|draft|place`. Scoped enable/disable (variant → product → category → supplier → global) with disable-under-global fixed. Shadow ledger `RetailerAutoOrderShadowProposals` + 30d WAPE / unmodified-accept stats. When `AUTO_ORDER_INVENTORY_GROUNDED=true`, synthesis skips `AI_PREORDER` `/2` insert (advisory `AIPredictions` only). Place still via `order.Service.Create` + `AUTO_ORDER_PLACE_ENABLED`. Clients: desktop / Android / iOS mode + scopes + shadow inbox. Docs: [`docs/AUTO_ORDER.md`](docs/AUTO_ORDER.md). Markers: `PX_E2E_AUTO_ORDER_SHADOW_OK` / `_SKIPPED` (draft marker retained).

**Residual.** POS feed (a) / full shelf-count UX (b); auto-flip place at ≥80% acceptance without human+env; per-scope execution mode; `SourcePredictionId` on place path.

---

### 8.4 Forecast accuracy monitoring — WIRED (measure before §8.1–8.3)

**Shipped.** Nightly `cmd/planning-accuracy` (CronJob 03:00 UTC) joins `DemandForecastBaseline` to completed `LineItemsJson` SUM qty at SKU-day grain → `ForecastAccuracyDaily` (WAPE7/28, bias, tracking signal; `|TS|>4` → supplier ADMIN inbox). Training export uses the same actuals helper (`actual_units`, not warehouse-day order counts). `AggregateDemandConfidence` prefers WAPE28 → `ConfidencePct` (magic 65/72/75 removed when accuracy path applies). Supplier portal KPIs from `GET /v1/supplier/analytics/demand/accuracy`. Flag: `FORECAST_ACCURACY_ENABLED`. Ops: `POST /v1/admin/planning/accuracy/run-once`. Marker: `PX_E2E_FORECAST_ACCURACY_OK` / `_SKIPPED`.

**Gate 2 intelligence (§8.1–8.4):** **wired** behind flags. Residual: soak acceptance ≥80% before touchless place auto-flip.

---

### 8.5 Routing and dispatch

**Why.** Beyond the P0 unit bug, the solver historically dropped handling constraints at the contract boundary and forced a single depot. §8.5 wires those paths in code; cloud live OR-Tools remains ops-gated on AR image + replicas.

**Constraint fidelity — WIRED (2026-08-06).** Contract `Stop` carries `handling_class`, `requires_cold_chain`, `is_hazardous`, `service_minutes`, `access_restriction`. Vehicles carry `has_refrigeration`, `hazmat_certified`, `shift_start`/`shift_end`, `max_route_minutes`, `start_lat`/`start_lng` (+ optional end). OR-Tools uses allowed-vehicle filters + Time span / StopCount dimensions (`contract_solver.py`). Dispatch hydrate ORs line-item / product handling into `GeoOrder`. Unit tests: cold stop never on non-reefer; max_stops orphans without truncating metrics.

**Multi-depot — WIRED.** Per-vehicle start/end nodes via `RoutingIndexManager(starts, ends)`; Go client no longer forces one depot coordinate onto every vehicle (driver lat/lng when present, else warehouse depot).

**Real road distances — WIRED (matrix path).** Go calls OSRM `/table/v1/driving/?annotations=distance` (`routing/osrm.go` `DistanceMatrix`), embeds `distance_matrix_m` on `SolveRequest`, falls back to haversine. Solver stays pure (no OSRM from Python). Residual: GPS-telemetry calibration of matrix vs actual legs; Redis matrix cache optional/not required for correctness.

**`max_stops_per_route` — WIRED.** Modeled as a StopCount capacity dimension (no post-hoc tail chop that left stale `distance_km`/`duration_min`).

**Time budget — WIRED.** Solver default `time_limit_ms=5000` (honors tunables; clamp 60s); Go HTTP soft timeout 8s; sidecar `OPTIMIZER_SOFT_TIMEOUT_SEC=8`. Numpy vectorization of haversine remains a residual scale optimization when OSRM matrix is absent.

**Cloud OR-Tools deploy path — WIRED (replicas ops-gated).** SSMR overlay includes optimizer-core Deployment+Service with AR image remap (`replicas: 0` until image exists). Prod keeps pin + `replicas: 0` — raise ≥ 1 after publish. Build recipe in [`docs/OPTIMIZER_AND_ROUTING_RUNTIME.md`](./docs/OPTIMIZER_AND_ROUTING_RUNTIME.md). Exit criterion: `"optimizer_source":"optimizer"` + `PX_E2E_OPTIMIZER_CONSTRAINT_OK` when sidecar up.

**Stop reporting OPTIMAL from a heuristic.** `vrp.rs` / `cpsat.rs` still mis-report Optimal; Rust sidecar remains undeployed. Options: delete it, or deploy Go Clarke-Wright as a real A/B arm. **Deferred (explicit residual).**

**Dispatch.** Bin-packer remains solid. Automated exception remediation + `PriorityBoostPenalty` wiring remain **deferred (explicit residual).**

---

### 8.6 Credit, collections, and the monetization engine

**Per §6, this is where the money is.** It is also the thinnest layer relative to its importance.

**Payment terms and aging — LIVE.** `RetailerPaymentTerms`, `ArInvoices.DueAt`/`AgingBucket`/`DunningStep`, aging pass, credit policy portals — [`docs/CREDIT_ECOSYSTEM_BEHAVIOR.md`](./docs/CREDIT_ECOSYSTEM_BEHAVIOR.md). No separate `ARAging` rollup (buckets on invoices).

**Compute `DelinquencyCount` — WIRED (collections substance).** Bumped once when dunning first enters OVERDUE (`credit.Service.BumpDelinquency`).

**Risk scoring (product decision).** Phase A removed the scoring desk / worker / `RiskTier` gates — CREDIT_LEAVE and placement are **status + available only**. Do **not** re-add a scorecard unless product explicitly reverses that decision.

**Dunning — WIRED behind `AR_DUNNING_ENABLED`.** Full step machine `DUE_SOON → … → COLLECTIONS`, auto-hold at CREDIT_HOLD via `HoldRelationship`, inbox + FCM notify on step advances. Ops: `POST /v1/admin/ar/dunning/run-once`. Residual: **no SMS/email/WhatsApp** transports for off-app reach.

**Notification transports.** FCM + inbox + `LogTransport`. **No SMS, no email, no WhatsApp** — still open for retailers without the app.

**Fiscalization, for real (Soliq).** `ProviderFromEnv` already selects `PEGASUS` / `FAKE` / `MY_SOLIQ` / `GLOBAL_PAY`. Finish L5: Soliq sandbox SUCCESS with real creds behind `FISCAL_PROVIDER=MY_SOLIQ`. PEGASUS remains the non-legal commercial path. Retry state machine + `OrderFiscalReceipts` + hard gate are already built.

**Refunds and settlement.** The only occurrence of "Refund" in non-test Go is reading `charge.AmountRefunded` off a Stripe webhook. Add a refund initiation path (full/partial, per gateway, with ledger entries and fiscal reversal). Add real supplier payout execution — `GET /v1/payment/settlement/authority` is a reporting view, and `warehouse/ops_portal.go:772` returns a hardcoded fake invoice (`"invoice_id": "inv-1"`) in the treasury endpoint.

**Then the billing meter** (§2.2) — **schema + event decode WIRED (2026-08-06):** Spanner columns match; `BillingTierWorker` reads `amount_minor` / nested `total.amount` from live `ORDER_FINALIZED`. Residual: define a fee schedule (per-order, per-GMV-bps, or subscription) and emit invoices / payouts.

---

### 8.7 Warehouse execution — the capability that caps the addressable market

**Wave 1A — WIRED (flag `WMS_LOTS_ENABLED`).** `WarehouseLocations` + `StockLots` + `OrderLotReservations`, putaway APIs, FEFO/FIFO allocate with shelf-life gating, `SupplierInventoryV2` roll-up — [`docs/WMS_LOTS_FEFO.md`](./docs/WMS_LOTS_FEFO.md). Portal `/bins` + warehouse Android/iOS putaway.

**Wave 1B — WIRED (flag `WMS_PICK_WAVES_ENABLED`).** `PickWaves` + `PickTasks` + `Manifest.PickWaveId`, create/confirm/waive, hard seal gate (`pick_wave_required` / `pick_wave_incomplete`) — [`docs/WMS_PICK_WAVES.md`](./docs/WMS_PICK_WAVES.md). Portal `/pick-waves` + warehouse Android/iOS confirm.

**Wave 1C / PR-4 — WIRED (flag `WMS_CYCLE_COUNTS_ENABLED`).** `CycleCounts` + `InventoryAdjustments`; create/submit; **approve applies lot QoH + V2 roll-up**; reject; ABC enqueue; inventory-accuracy KPI; portal + Android/iOS cycle UI — [`docs/WMS_CYCLE_COUNTS.md`](./docs/WMS_CYCLE_COUNTS.md).

**PR-5 — WIRED (flags `WMS_PICK_SSHAPE_ENABLED`, `WMS_SEAL_SOFT_WARN`).** Zone serpentine + LIFO stop-rank pick ordering; soft-warn seal attaches `pick_wave_warning` instead of hard 409.

**PR-6 — WIRED (flag `WMS_COLD_CHAIN_ENABLED`).** `TemperatureReadings` ingest + excursion quarantine — [`docs/WMS_COLD_CHAIN.md`](./docs/WMS_COLD_CHAIN.md).

**PR-7 — WIRED (partial).** Inventory reconcile endpoint; ops checklist [`docs/WMS_GATE4_OPS.md`](./docs/WMS_GATE4_OPS.md); scan throughput deferred to native follow-up [`docs/WMS_GATE4_HARDENING.md`](./docs/WMS_GATE4_HARDENING.md). **Still open:** forbid all non-rollup V2 writers; serial tracking; full scanner UX.

**Why (original gap).** Without lots/expiry, warehouse stock is a bag of SKUs — disqualifying for food/pharma. Retailer on-hand **is** modeled (`RetailerStockBalances`, receive sessions, counts). Wave 1A addresses warehouse expiry + location; Wave 1B adds manifest pick waves + seal gate; Wave 1C stubs cycle counts; ABC apply-on-approve / S-shape / cold-chain remain residual.

**Schema.**
```
Locations(WarehouseId, LocationId, Zone, Aisle, Rack, Level, Bin,
          LocationType, PickSequence, MaxVolumeVU, IsActive)
StockLots(LotId, SupplierId, WarehouseId, ProductId, LocationId,
          LotCode, ExpiryDate, ManufacturedDate, QuantityOnHand,
          QuantityReserved, ReceivedAt, Status)
PickWaves(WaveId, WarehouseId, Strategy, Status, CreatedAt, ReleasedAt)
PickTasks(WaveId, TaskId, OrderId, ProductId, LotId, LocationId,
          QuantityRequested, QuantityPicked, PickerId, Status, PickSequence)
CycleCounts(CountId, WarehouseId, LocationId, ProductId, ExpectedQty,
            CountedQty, VarianceQty, ReasonCode, CountedBy, CountedAt)
InventoryAdjustments(AdjustmentId, WarehouseId, ProductId, LotId,
                     DeltaQty, ReasonCode, ActorId, ApprovedBy, CreatedAt)
```

**Allocation: FEFO with shelf-life gating.** For perishables, allocate First-Expiry-First-Out subject to a minimum remaining shelf life at delivery — reject a lot where `ExpiryDate − expected_delivery_date < min_shelf_life_days`, configurable per retailer (chains typically demand 2/3 remaining life). This alone prevents a class of rejected deliveries that currently surfaces as a driver-reported condition report after the truck has already driven there. For non-perishables, FIFO by `ReceivedAt`.

**Picking: zone-partitioned batch waves with S-shape traversal.** Group orders into a wave (target ≈ one truckload), split tasks by `Zone`, sort within zone by `PickSequence`, and route the picker by serpentine traversal of aisles — the standard warehouse heuristic that beats naive nearest-location by 15–30% on walking distance. Sequence tasks so the pick order produces the LIFO load order from §8.0: **last delivery picked first**, so the truck loads correctly without restaging. This is where picking, loading, and routing become one problem instead of three.

**Cycle counting: ABC-cadence perpetual counts.** Classify SKUs by annual movement value (A: top 80% of value / ~20% of SKUs, B: next 15%, C: last 5%). Count A monthly, B quarterly, C annually; trigger an off-cycle count on any negative-availability event or a variance above threshold. Track `InventoryAccuracy = 1 − Σ|variance| / Σexpected` per warehouse as the operational KPI. Route variances above a value threshold to `InventoryAdjustments` with mandatory reason code and approval.

**Where.** New `apps/backend-go/warehouseops/{locations,lots,picking,counting}.go`; extend the existing warehouse portal and the warehouse mobile apps (which already have competent barcode scanning). `SupplierInventoryV2` becomes a materialized roll-up of `StockLots` rather than the source of truth — keep it for the hot read path, with a reconciliation job asserting they agree.

**Cold chain, made real.** Add `TemperatureReadings(ManifestId, SensorId, RecordedAt, TempC, Lat, Lng)` with ingestion from a Bluetooth logger or the driver device, plus excursion detection (cumulative minutes outside `[StorageTempMinC, StorageTempMaxC]` per lot) that auto-raises a `TEMPERATURE_BREACH` condition report and quarantines the affected lot instead of waiting for a human to notice.

---

### 8.8 Mobile — make the field layer trustworthy

**Shared module per platform, first.** **Partial→Wired (2026-08-06):** [`packages/mobile-android-kit`](./packages/mobile-android-kit) + [`packages/mobile-ios-kit`](./packages/mobile-ios-kit) (`PegasusKit`) hold offline queue contract, HTTP flush semantics (409 ACK / 5xx retry / 4xx DEAD), reconnect backoff, client-policy snapshot. Design/barcode packages remain separate. Root [`gradle/libs.versions.toml`](./gradle/libs.versions.toml) added. Residual: fold AutoUpdater/WS clients into kit across all 6 role apps; payload iOS still lacks `.xcodeproj` for SPM. See [`docs/MOBILE_SHARED_KIT.md`](./docs/MOBILE_SHARED_KIT.md).

**Generic offline queue.** **Wired (driver + kit promotion 2026-08-06):** multi-endpoint flush was already live; now rows carry **capture-time** `capturedLat`/`capturedLng`/`capturedAtMs`, Room migrations replace `.fallbackToDestructiveMigration()` on driver, flush replays stored coords (not live GPS). Warehouse/factory Android use kit `PrefsOfflineQueueStore`; payload Android Room v2. **Still open:** warehouse scan throughput ([`docs/WMS_GATE4_HARDENING.md`](./docs/WMS_GATE4_HARDENING.md) / PR-7).

**Proof of delivery.** There is **no photo and no signature capture in either driver app** — a repo-wide grep for every capture API returns only supplier product images. `photoProofUrl` is threaded through four layers and called with `nil` at both sites (`FleetMapView.swift:93`, `OffloadReviewScreen.kt:328`). For a **credit delivery** — goods handed over with no payment — there is no evidence artifact at all. Add `PhotosPicker`/`ImageCapture` plus a signature pad, store the image as a file reference in the offline queue, and make it mandatory for `markCreditDelivery` and `reportShopClosed`.

**Background location, iOS.** Add `location` to `UIBackgroundModes` (the plist declares only `fetch`/`processing` while the code sets `allowsBackgroundLocationUpdates = true`, which raises `NSInternalInconsistencyException`), remove the malformed conflicting `INFOPLIST_KEY_UIBackgroundModes`, set `pausesLocationUpdatesAutomatically = false` and `activityType = .automotiveNavigation` — without which iOS auto-pauses and never resumes, the classic silent tracking death. Either register the declared `BGTaskScheduler` identifier or remove it; it is currently inert.

**Reconnect and reboot.** Remove `MAX_RECONNECT_ATTEMPTS` (`TelemetrySocket.kt:103-107`) — ten tunnels currently ends telemetry for the shift, since `reconnectAttempt` only resets in `onOpen`; hold at max delay and reset on `NetworkCallback.onAvailable`. Write telemetry to Room **before** attempting the send and delete only on acknowledgement, because OkHttp's `send()` returns true on *enqueue* so the offline fallback rarely fires. Add a `BOOT_COMPLETED` receiver (none exists in any manifest) and re-cancel-safe scoping in `TelemetryService` — `serviceScope.cancel()` in `stopTracking()` permanently kills the scope, so a `START_STICKY` restart comes back half-functional.

**Scanning throughput.** Warehouse and payload scanning is unusable at picker volume: a network round-trip per scan with no local EAN→SKU cache from the already-downloaded manifest, re-scanning a SKU **un-checks** it (`toggleItem`), a 1.5s debounce caps repeated-SKU rate at ~40/min, ML Kit runs all 13 symbologies per frame with no `BarcodeScannerOptions`, no torch control anywhere, and **no hardware scanner support at all** (zero DataWedge/Zebra/keyboard-wedge hits) — so pickers on Zebra TC-series devices can't use the trigger. Fix all six: prefetch the map, increment a per-line scanned count instead of toggling, drop the debounce, restrict to EAN-8/13, add torch, add a hidden `BasicTextField` wedge path plus a DataWedge intent receiver.

**Push, deep links, localization.** Firebase client JSON/plists are **committed** under each mobile app (2026-08). Still required: `aps-environment` entitlements and `PrivacyInfo.xcprivacy` (App Store), declare `FirebaseMessagingService` where missing, refuse `demo-pegasus`/`demo-key` fallbacks, and owner SHA-1 + real SMS for Phone Auth. Register URL schemes and intent filters — `deepLink` is decoded into DTOs in all 12 apps and then discarded because no app declares a handler. Wire `packages/i18n/generated/*` into `project.yml` resources and Gradle `res.srcDirs`, then mechanically replace 1,125 Kotlin literals and every Swift string.

**Security hardening.** Set `kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly` on Keychain items (currently unset, so tokens are unreadable while the phone is locked — breaking background reconnect — and are included in device backups). Add `FLAG_SECURE`/`isCaptured` blur on driver PIN, cash, and card screens. Add SQLCipher to Room and encryption to SwiftData — both currently hold orders, addresses, and a GPS trail in plaintext. Add certificate pinning. Split `network_security_config.xml` into `src/debug` — it currently permits cleartext to a developer's `192.168.0.101` in release. Sign `updater.json` with a detached Ed25519 signature verified against a pinned key and allowlist the manifest host; today Android verifies a hash supplied by the same server as the APK, and **iOS OTA verifies nothing at all**.

---

### 8.9 The integration layer — what turns this into a platform

Per §7 this is the highest-leverage work in the report relative to its cost, because the hard substrate already exists.

**1. Machine identity — WIRED (Wave 1 + OAuth).** `PartnerApiKeys` + bcrypt `pxk_` keys + scopes + rate limit by KeyId; `LOAD_BOOTSTRAP_SECRET` does not exempt `/partner/*`. **OAuth2 `client_credentials`** at `POST /partner/v1/oauth/token` issues short-lived `partner_access` JWTs (dual-accept with `pxk_` on `/partner/v1/*`). Issue keys via `POST /v1/admin/partner-keys`. See [`docs/PARTNER_API.md`](./docs/PARTNER_API.md).

**2. OpenAPI 3 — WIRED (partner + JWT core).** Partner: [`contracts/partner.openapi.yaml`](./contracts/partner.openapi.yaml). Human JWT core (~45 high-traffic ops): [`contracts/jwt-core.openapi.yaml`](./contracts/jwt-core.openapi.yaml) — [`docs/JWT_CORE_OPENAPI.md`](./docs/JWT_CORE_OPENAPI.md); `make jwt-openapi-gate`. Residual: expand coverage to full ~411-path catalog + replace hand-written `@pegasusx/api-client` with generated SDK.

**3. Partner order API — WIRED (Wave 1).** `POST/GET /partner/v1/orders`, `GET /partner/v1/catalog`, `GET /partner/v1/inventory/availability` via `order.Service.Create` / catalog enrichers; tenant-scoped IDOR fail-closed.

**4. Outbound webhooks — WIRED (Wave 1 + 2A ops).** `WebhookSubscriptions` + Kafka enqueue + HMAC-SHA256 delivery worker + dead-letter list + ping + list/deactivate + dead-letter replay (partner key + supplier JWT) + supplier portal Integrations UI.

**5. Bidirectional bulk data — WIRED (Wave 2A, export side).** `PartnerExportJobs` + `POST/GET /partner/v1/exports` (`exports:read`); CSV/JSON for orders/invoices/inventory/ledger; GCS or `PARTNER_EXPORT_LOCAL_ROOT`; optional SFTP via `PartnerSftpConfigs` + `PARTNER_SFTP_ENABLED` (GSM `SecretRef`). Import/spreadsheet wizard already existed.

**6. Enterprise EDI + GS1 + 1C journals — partial WIRED.** ORDERS inbound + ORDRSP/DESADV/INVOIC outbound — [`docs/PARTNER_EDI.md`](./docs/PARTNER_EDI.md). **GLN / SSCC / ZPL WIRED** — [`docs/GS1_LABELS.md`](./docs/GS1_LABELS.md). **DESADV SSCC segments WIRED** (CPS/PAC/GIN+BJ from `ManifestShipUnits`). **1C journals CSV/JSON/XML WIRED** (`resource=journals`) — [`docs/PARTNER_JOURNALS_1C.md`](./docs/PARTNER_JOURNALS_1C.md). **Configurable CoA WIRED** (`PartnerCoaMaps` + GET/PUT `/partner/v1/coa`). **AS2 transport WIRED** (sync MDN, sign-then-encrypt; **not Drummond-certified**) — [`docs/PARTNER_AS2.md`](./docs/PARTNER_AS2.md). **Still open:** certified EDIFACT/X12, certified 1C exchange package.

---

### 8.10 Multi-tenancy — the honest migration path

**Phase 1 — request-scoped tenancy. This is the whole game and it cannot be done incrementally.** Add `SupplierID` to the auth claim set and to a request-scoped tenant context; delete the `supplierID` field from every service struct; change the ~15 constructor sites in `bootstrap.go`; thread tenant context through every repository method that currently takes a constructor-bound ID; add middleware that **fails closed** when tenant context is absent. Tables changed: none — the schema is already correctly shaped, which is a genuine asset. Code changed: **150–250 files** (`order/` alone is 74 files, `warehouse/` 67, `supplier/` 61, `dispatch/` 26, `payment/` 28).

The reason it can't be incremental: today isolation is safe *only because* the ID is a startup constant. The moment it becomes request-derived, all 411 endpoints are potential IDOR vectors and there is no central enforcement point to lean on. Either every path is tenant-aware or none are safe. **Until Phase 1 lands, disable multi-supplier registration** (`supplier/service.go:433-447` currently mints up to 10 tenants the runtime cannot serve, and their orders would be attributed to `seed-supplier-1` — worse than refusing).

Also in Phase 1: per-tenant rate limits and quotas (the limiter keys on JWT `sub`, so a tenant with 500 users gets 500× the quota of a tenant with one), and outbox partitioning by tenant (`OutboxEvents` is a single global queue — one noisy tenant delays every tenant's events).

**Phase 2 — multi-supplier cart and order splitting.** New `ParentOrders`; add `ParentOrderId` to `Orders`; drop the supplier filter from cart reads (`retailer/repository_cart.go:44`); build a split engine fanning out per-supplier child orders each with its own credit check, inventory plan, pricing resolution, and warehouse assignment; roll status up for the retailer UI. Note this is *not* a cart change — one `SupplierId` is in the `Orders` primary key. 1 new table, 2 altered, 30–50 files.

**Phase 3 — global product master.** `GlobalProducts` keyed by GTIN with brand/manufacturer/pack-size; `SupplierProductOffers` mapping `(SupplierId, ProductId) → GlobalProductId` with price, MOQ, lead time; `ProductMatchQueue` for human review; `UnitsOfMeasure` with a real pack hierarchy (each/inner/case/pallet — currently a single nullable `UnitsPerPack`). Matching pipeline: exact GTIN, then fuzzy on brand + normalized pack size + unit measure, with conflicts queued. 4 new tables, 20–30 files, plus a worker. This is the prerequisite for cross-supplier comparison, and it reuses the GTIN checksum validation that already exists in `returns/barcode.go`.

**Phase 4 — marketplace commerce.** Billing meter schema + amount decode wired (§2.2); still need fee schedule/invoices. Then: supplier ratings and scorecards; RFQ / competing quotes (note `NegotiationProposals` is delivery-date negotiation, not price bidding); split payments and escrow — which likely **forces a second gateway integration**, since Global Pay probably lacks sub-merchant support; supplier payout execution.

**Phase 5 — tenant operations.** There is currently **no platform admin console at all** — no supplier management, no approval queue, no suspension, no offboarding. A supplier can self-register and nobody can approve or remove them. Add the console, an approval workflow with document collection and KYB, tenant-scoped audit, and per-tenant observability.

**Honest total: 250–400 files touched.** Multi-tenancy was designed into the schema and designed out of the runtime.

---

### 8.11 Per-role and per-app gap summary

| Role | App completeness | Top gaps |
|---|---|---|
| **Retailer** | iOS 75% / Android 80% / desktop good | Receive+stock+FileClaim/eligibility countdown shipped (G1/G2); still weak on inferred shelf for auto-order (§8.3), KYC, i18n, cross-supplier cart (§8.10 P2), tenant-scoped desktop cache |
| **Supplier** | Portal strong / iOS 60% / Android compiles (Gate-0 OrgFleet) | Payout execution, refunds, real pricing engine, billing/commission, forecast accuracy view |
| **Warehouse** | Portal good / iOS 60% / Android 55% | **§8.7 Gate 4 Waves 1A–1C + PR-4–7 coded**; residual forbid non-rollup V2 + scan UX; ViewModels |
| **Factory** | Portal 9k lines / iOS 55% / Android 55% (0 ViewModels) | Production scheduling, capacity/MRP (`GetSAndOP` is `factories × 700 × 7`), real transfer lead-time capture |
| **Driver** | iOS decoder Gate-0 fixed / Android 70% (best offline story) | background location, photo/signature POD, boot receiver; generic offline queue + capture-time coords **wired** (§8.8); scan UX residual is warehouse |
| **Payload terminal** | iOS 45% (no `.xcodeproj`) / Android 50% | Generate the Xcode project, hardware scanner, per-line quantities, split the 1,700-line god view |
| **Platform admin** | **Effectively absent** — 3 endpoints and a redirect stub | The entire console: tenant management, approval, suspension, fee schedule, global observability, support tooling |

Cross-cutting, all surfaces: localization (0% on web, 0% on iOS, ~1% on Android despite complete generated catalogs), accessibility (2 `htmlFor` against 90 `<label>`, 1 `onKeyDown`, 0 `tabIndex` across 51k lines of portal TSX), testing (frontend ~1 test file per 2,700 lines with zero e2e; mobile 2.2%/2.4% with zero UI tests), and server-state management (0 data-fetching libraries across 483 hand-rolled `useEffect` fetches).

---

## 9. Sequenced roadmap

**Gate 0 — Stop the bleeding (2–4 weeks).** All of §8.0. Non-negotiable end state: CI compiles and tests all 12 mobile apps and lints everything; Spanner backups exist and a restore has been *executed*; Terraform state is remote; the prod overlay renders real digest-pinned images; the optimizer either solves or provably falls back; `supplier-app-android` and `driver-app-ios` work.

**Gate 1 — Make it legal and reachable (4–6 weeks).** Soliq OFD SUCCESS behind `FISCAL_PROVIDER=MY_SOLIQ` (§8.6; PEGASUS commercial path already live). SMS + email transports (§8.6) — **still open**. Payment terms / due / aging / `DelinquencyCount` / dunning state machine (§8.6) — **wired** behind `AR_*` flags (no credit scoring re-add). Finish push/OTP ops (APNs entitlements, SHA-1, real SMS) on top of committed Firebase configs (§8.8). Without legal OFD + off-app notify you cannot fully collect outside the app.

**Gate 2 — Make the intelligence real (6–10 weeks).** §8.4 accuracy + §8.1 Croston/SES/HW + §8.2 safety stock + §8.3 inventory-grounded shadow auto-order — **wired** (flags; soak `ReceivedAt` / shadow acceptance). Ship touchless place only when shadow-mode acceptance exceeds 80% + human + env.

**Gate 3 — Make it integrable (4–6 weeks).** §8.9 items 1–5. This is the gate that changes who you can sell to.

**Gate 4 — Make the warehouse executable (8–12 weeks).** §8.7. This is the gate that changes *what* you can sell — food and pharma become addressable, and pick/load/route becomes one optimized problem.

**Gate 5 — Multi-tenancy (10–16 weeks).** §8.10 Phase 1, then 2. Do not start before Gate 3, because a partner API and outbound webhooks designed single-tenant will have to be rebuilt.

**Gate 6 — Marketplace, if the economics justify it.** §8.10 Phases 3–5. Per §6, decide this on evidence from Gates 1–4, not on the premise that no competitor exists. The survivors in this category monetize credit and data, not distribution margin — and Gate 1 is what makes credit monetizable.

**Parallel track, continuously:** collapse the duplication (§4) — a real shared module per mobile platform and a `<PortalShell>` in `ui-kit`. It is ~12,000 lines of deletion on mobile and ~4,000 on web, and it is the prerequisite for accessibility and localization ever getting fixed, since both currently require 4–6 edits per change.

---

## 10. Kill list

Delete or explicitly rename. Every one of these currently misleads the next engineer about what the system does.

| Item | Why |
|---|---|
| ~~`ledger/` package~~ | **Deleted (Gate-0).** |
| `services/optimizer-core/server-rust/` | 447 lines, index bug, deployed nowhere. Delete, or replace it with the Go Clarke-Wright as a real A/B arm. |
| `optimizationjobs/` package + `OptimizationJobs` table | `EnqueueJob` has zero callers; nothing publishes or consumes `TopicOptimizerJobs`. |
| `kafka.AnalyticsStreamProcessor` | Package still exists (`kafka/stream_processor.go`) but **is no longer started** from `runtime_workers.go` (dummy channel removed). Delete the orphan type or wire a real Kafka stream — do not reintroduce a dummy. |
| `enterprise/` (auth0, datadog, vault) | 225 lines of vendor SDKs bolted onto a system that already has Firebase Auth, Prometheus, and external-secrets. `InitVault` failure is logged and ignored. |
| The four dead chart components | Ship empty axis-labelled charts to four production analytics pages. Wire or delete, and add a CI grep failing on `MOCK_` outside tests. |
| `apps/admin-portal`, `apps/supplier-app-desktop` | 3-file redirect stubs that inflate the app count from 6 real web surfaces to 8. |
| `packages/validation` zod schemas | Both schemas and both inferred types imported nowhere. Only `normalizeEanBarcode` is live. |
| `cypress` in `retailer-app-desktop` | Declared dependency, no config, no directory, no spec. |
| `patch_*.sh` ×8, `refactor_ios.py`, `refactor_android.py`, `fix_imports.py`, `patch_spy*.py` ×5 | Regex source surgery. One of them broke an entire app. `refactor_ios.py:141-268` overwrites `OrgFleetView.swift` with a hardcoded re-authored body. |
| Tracked binaries: `apps/ai-worker/ai-worker` (53 MB), `apps/handoff-service/handoff-service` (8.7 MB) | In git. |
| `softwareengineercv-main/` | An unrelated personal CV site with ~40 MB of PNGs vendored into the platform repo. |
| Root `kustomize` binary (12 MB) + generated `kustomize.yaml` (23k lines) | Build artifact and a vendored tool. Pin the toolchain instead. |
| `infra/k8s/kafka.yaml` + `kafka-topics.yaml` | A third, non-reconciling Kafka topology (`apiVersion: kafka.strimzi.io/v1` doesn't exist) competing with the Managed Kafka actually in use. Pick one. |
| `create_cluster.sh` | Imperatively creates the cluster that carries traffic, invisible to Terraform, under a different name than the one Terraform manages. |
| `infra/redis.conf` | Documents AOF persistence settings that Memorystore BASIC cannot support. Either move to STANDARD_HA and implement it, or rewrite the file to state the truth. |
| `SourceFallback = "KMEANS_BINPACK"` | No k-means exists anywhere. |
| `RunMEIONetwork` naming | A two-node greedy swap is not multi-echelon inventory optimization. |
| `planning.projectStockouts` | Returns the literal strings `"sku-projection-1"`, `"sku-projection-2"`. |
| `warehouse/ops_portal.go:772` | Returns a hardcoded fake invoice row in the treasury endpoint. |
| `docs/CLOUD_BUDGET_MODEL.md` | Prices Spanner at $650–900/mo for 100 PU (actual ≈ $65–90 — off by 10×), assumes GKE Autopilot when the cluster is Standard, and budgets Cloud Run services that don't exist. |

**On the ~50 markdown runbooks in `docs/`:** two still carry a banner saying the environment is mid-migration and blocked on quota. Against two GitHub workflow files and no CD pipeline, the doc-to-automation ratio is the clearest signal of where effort has been going. Prefer executable gates over prose: a runbook that isn't a script is a runbook that drifts.

---

## Closing

The engineering is not the problem. The transactional outbox, the retry-safe closures, the out-of-band webhook verification, the idempotency key discipline mirrored across three platforms, the Kahan summation in the bin-packer, the deterministic tie-breaking in Clarke-Wright, the honest doc comment admitting the chunker breaks ACID — that is careful work by people who know what they are doing, and it is genuinely rare.

The problem is threefold and each part is fixable:

**First, labels outran substance.** `MinConfidenceScore` is stored and ignored. Seasonal multipliers are defined and never multiplied. A weather signal returns the integer 2 in summer. A confidence gate cannot mathematically reject. The accuracy metric compares products to order counts. Billing workers run against a schema they cannot write. Fiscal now has a real `PEGASUS` path, but Soliq OFD is still unfinished. Each of these shipped the *interface* of a capability, and the deferral was never tracked — which is why a long theatre list accumulated (claims have since been wired for real). The fix is cultural before it is technical: nothing customer-visible ships until the thing behind it works, and anything deferred gets an issue, not a comment.

**Second, there was no gate.** No CI compiling 167k lines of mobile code, no linter, no `-race`, no security scan, no CD, no backup, no restore rehearsal. That single absence explains an app that doesn't compile in `HEAD`, a decoder bug that breaks the driver's primary flow, a prod overlay that renders placeholder image names, and a database with no recovery path. Gate 0 is not cleanup — it is the precondition for knowing whether any subsequent work helped.

**Third, the strategy rests on a premise that isn't true.** Platforms connecting many suppliers to many retailers exist, are well funded, and have mostly discovered that distribution margin doesn't pay for them — the survivors monetize credit and data. The genuinely rare asset here is not the marketplace idea; it is the **vertical depth from factory floor to shop shelf in one transactional model with one event bus**, including COD, credit delivery, split payment, reconciliation, and a fiscal gate. That is worth more as deep software for large distributors — sold on integration and credit — than as a thin marketplace for many. Which reorders the roadmap: integration (Gate 3) and credit/collections (Gate 1) before marketplace (Gate 6), and the shadow-mode test in §8.3 before anyone bets the company on autonomy.

Fix the P0s. Build the accuracy harness before the algorithms it grades. Then decide the marketplace question on evidence.
