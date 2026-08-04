# PegasusX — Platform Audit & Build Specification

Evidence base: the source tree at `/Users/shakhzod/ATOMOS/pegasusX` as of 2026-08-04. No repo markdown was trusted; every claim below traces to code, schema, or config. File:line references are given so each finding can be re-checked.

**Re-aligned 2026-08-04 (post Phase A/B G1–G3 backend):** claims/receive/stock/eligibility/window snapshot are **live**; fiscal is env-selected (`PEGASUS` default, not unconditional Fake); billing workers are constructed but still schema-broken; credit **scoring product removed** (limits + status only); Firebase client configs are committed (OTP/SMS/SHA still ops-owned). Structural findings (single-supplier runtime, no ML, no partner API) remain.

---

## 0. Executive verdict

**Is it "just a stupid CRUD system"? No. Emphatically no.** A CRUD app has handlers calling an ORM. This has a transactional outbox whose event rows commit inside the same Spanner transaction as the state mutation, optimistic concurrency with real compare-and-swap on `Version`, an 18-state order machine with a centralized transition table, money as integer minor units with zero float money anywhere, bcrypt throughout, payment webhook verification that re-verifies settlement out-of-band against the gateway, Redis pub/sub WebSocket fan-out for horizontal scale, deterministic content-fingerprinted idempotency keys mirrored across web/iOS/Android, and a production config validator that fails closed on dev secrets. That is the work of people who have operated real transactional systems.

**But it is not the system the docs and naming claim it is.** Three structural facts dominate everything else:

1. **It is a single-supplier system, at runtime, by construction.** The schema is multi-tenant-shaped (`SupplierId` leads most keys), and supplier registration will mint up to 10 tenants — but `bootstrap/bootstrap.go` injects one `supplierSeed.SupplierID` into ~15 service constructors at process start, and `order.Service` holds it as a private `supplierID` field (`order/service.go` ~351) used as a constant on create/list paths. **The supplier the retailer picks in the UI is discarded during order creation.** Registering a second supplier today produces a tenant whose orders are attributed to the seed supplier. The data plane can hold 10 suppliers; the request plane can serve exactly 1.

2. **There is zero machine learning, and the "AI" layer is arithmetic.** The Python service's entire dependency list is one line: `ortools==9.15.6755`. No Vertex, Gemini, OpenAI, TensorFlow, PyTorch, ONNX, sklearn, XGBoost, Prophet, statsmodels, embeddings, or vector search anywhere in Go modules, requirements, Cargo, or package.json. The code says so itself: `planning/baseline_sources.go:14` — *"Never returns 'ml' — training inference is deferred"*; `cmd/planning-training-export/main.go:24` — *"collect-only; no training"*. The auto-order quantity is `line.Quantity / 2` (`ai-worker/synthesis/engine.go:310`).

3. **There is no machine-to-machine integration surface at all.** Zero matches across the backend for `openapi`, `swagger`, `oauth2`, `client_credentials`, EDIFACT, X12, SSCC, GLN, ZPL, SAP, 1C, Odoo, NetSuite, BigQuery, SFTP. No outbound webhooks (the 5 webhook routes are inbound gateway receivers). No export endpoint of any kind. **A retailer with an ERP cannot integrate today by any path.** The only automated inbound channel is a human uploading a spreadsheet through a browser wizard and clicking approve.

**Scale, for calibration:** ~410k lines of real code — 131,670 Go / 81,116 TSX / 94,332 Kotlin / 74,434 Swift / 22,858 TS / 1,826 Python / 988 Rust / 32,118 YAML / 1,507 Terraform. 411 distinct HTTP endpoints, 73 Spanner tables, 12 native apps, 6 web surfaces. This is 18–30 months of competent engineering, not a prototype.

**Can it replace the field sales agent?** It replaces the agent's **order pad**, not the agent. The commercial loop (demand signal → proposal → confirmation → credit → pricing → allocation → dispatch) is genuinely automatable with flags that already exist. But picking, loading, driving, the delivery handshake, cash collection, and every exception path terminate in a human — and **dunning happens entirely outside the system** because there are no payment terms, no due dates, and no SMS or email transport. Honest number: **~35–40% of the agent's job is automatable with what exists, ~65% with the P1 work in §8, and the cash-collection half is structural, not a gap.**

**Scorecard:**

| Layer | Score | One-line justification |
|---|---|---|
| Go backend | **8/10** | Same transactional primitives; claims/receive/stock liability spine + claim-window snapshot now wired. Still loses points on duplicate event publishing and state-machine bypasses. |
| Domain modelling | **8.5/10** | 18-state order machine, two-sided delivery negotiation, volumetric dispatch, COD/credit/split payment, post-delivery claims ↔ quarantine ↔ reverse. Genuinely deep. |
| Web frontend | **6/10** | Excellent type hygiene and idempotency; no server-state library across 60k lines, 0% localized, accessibility unusable. |
| iOS | **4/10** | Modern SwiftUI/`@Observable`, but a decoder bug breaks the driver's primary flow and background location is misconfigured three ways. |
| Android | **5/10** | Best offline queue in the repo; `supplier-app-android` does not compile. |
| Infra / DevOps | **3/10** | Well-authored manifests, zero operationalization: no Spanner backup, local Terraform state, prod overlay renders placeholder images, no CD. |
| AI / optimization | **2.5/10** | Real OR-Tools and a correct Clarke-Wright exist; the deployed solver has a fatal unit bug and forecasting is a 7-day mean. |
| Integration surface | **0.5/10** | A spreadsheet wizard. |
| Multi-tenancy (runtime) | **1/10** | One supplier, bound at startup. |

---

## 1. What is genuinely strong

Worth stating precisely, because the rest of this report is critical and the good work deserves to be identified so it isn't refactored away.

1. **Outbox events commit inside the state transaction.** `order/repository_spanner.go:247-291`, `ledger/repository_spanner.go:37-73` — a `spannerTxnBuffer` collects events *inside* the closure, then base + outbox mutations are written in one `txn.BufferWrite`. The event cannot commit without the state change.
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

The single most important pattern in this codebase: **features ship the *interface* of a capability with none of the substance, and the deferral was never tracked.** This is more dangerous than missing features, because dashboards, policy toggles, and state machines all imply the thing works. (Count refreshed 2026-08-04 — claims removed from theatre; fiscal/billing downgraded to partial.)

| # | Feature | What exists | What's actually there | Evidence |
|---|---|---|---|---|
| 1 | **Fiscalization (legal OFD)** | Full FISCALIZING → FISCAL_FAILED retry state machine, `OrderFiscalReceipts`, env-selected providers (`PEGASUS` / `FAKE` / `MY_SOLIQ` / `GLOBAL_PAY`) | **Not unconditional Fake anymore** — `defaultFiscalProvider()` → `ProviderFromEnv()`; SSMR/staging use `FISCAL_PROVIDER=PEGASUS` (platform commercial receipts). **Legal Soliq OFD still not production-ready** — `MY_SOLIQ` needs sandbox creds; misconfig returns hard-fail rather than silent Fake. | `order/fiscal.go` (`defaultFiscalProvider`); `order/fiscal_provider.go` (`ProviderFromEnv`) |
| 2 | **Marketplace commission** | 3 billing tables, meter + tier workers, consumer started in `runtime_workers.go` | Workers **are constructed** (`bootstrap` + `BillingTierConsumer.Start`). Writes still **don't match schema** (`AmountDelta` vs `CurrentValue`; insert omits `EventId`/`MeterType` NOT NULL columns) — first live meter event fails. **Nothing is successfully metered.** | `internal/services/billing/meter_worker.go:47-60`; `schema/spanner.ddl` `BillingMeterEvents` / `BillingSupplierMeters` |
| ~~3~~ | ~~**Claims / disputes**~~ | — | **Retired from theatre (2026-08-04).** Live service + routes + e2e. Residual hygiene: `Claims`/`ClaimEvidences` live in migration `20260728_logistics_claims.ddl` but are **missing from `schema/spanner.ddl`** (schema-drift risk for greenfield applies). Portal claim-window **settings UX** still open (API only). | `bootstrap/bootstrap.go` claims wiring; `orderroutes/routes.go`; `claims/*`; `order/claim_window.go` |
| 4 | **Double-entry ledger** | Correct balanced-transfer implementation with normal-balance semantics | Nothing imports `ledger/` outside its own package, and `LedgerAccounts`/`LedgerEntries`/`LedgerTransactions` **are not in the DDL** — it would fail with table-not-found. Live money path is `PaymentLedgerEntries` / `ArLedgerEntries` (event log), not the textbook package. | `ledger/service.go` vs `schema/spanner.ddl` |
| 5 | **AI confidence gate** | `MinConfidence` knob, debug log for rejections | Minimum possible score is `0.4 + 0.15 + 0 + 0 = 0.55`; the gate is `score < 0.55`. **`0.55 < 0.55` is false — every event passes.** | `synthesis/engine.go:181` vs `:99, 451-455` |
| 6 | **Touchless replenishment policy** | `MinConfidenceScore FLOAT64 DEFAULT 0.85` | Loaded, written, and **never used in any decision**. A supplier setting "only auto-approve above 95%" is silently ignored. | `replenishment/policies.go:52,83` vs `touchless.go:41-52,242-253` |
| 7 | **Auto-confirm of AI preorders** | `AutoConfirmAt` set to +24h; `AutoConfirmDueOrders` implemented | **Zero call sites.** Not in `runtime_workers.go`. AI preorders sit at `PENDING` forever, and dispatch eligibility requires `CONFIRMED`/`AUTO_CONFIRMED` — so they are invisible to dispatch until a human taps confirm. | `order/preorder_service.go:295`; `dispatch/eligibility.go:5-8` |
| 8 | **Seasonality** | Two templates with `Multiplier` ×1.35 / ×1.15 | The `Multiplier` field is **never read by any quantity calculation** — only serialized and cached. Seasonality is decorative. | `planning/seasonal_templates.go:46-49` |
| 9 | **Weather & POS demand signals** | A `CompositeSignalProvider` "demand sensing stack" | `externalWeatherSignals` returns hardcoded `Qty: 2` if the month is June–August. `externalPOSSignals` returns `Qty: 3` on the 1st, 15th, and last of the month. No API calls. | `predictivepush/signals.go:96-130` |
| 10 | **Price elasticity / promo simulation** | Promo simulator with projected volume, margin, and a "closed-loop score" | Fixed 0.5 elasticity for every product and season. The actual-vs-projected scorer does `_ = promotionID` — it **counts all completed orders in 30 days** and calls that the promotion's result, comparing an order *count* against a projected unit *volume*. | `planning/promo_eval.go:59`, `:156-163` |
| 11 | **Forecast accuracy** | A MAPE figure shown to suppliers | Computed client-side in React, never persisted, mislabeled (it's WAPE), and joins a **per-product baseline against a per-warehouse order count** — it measures nothing coherent. | `supplier-portal/app/(portal)/analytics/demand/page.tsx:82-85` |
| 12 | **Cold chain** | `RequiresColdChain`, `StorageTempMinC/MaxC`, a `TEMPERATURE_BREACH` condition type | **No temperature ingestion, no sensor integration, no reading table, no excursion detection.** A human reports a breach after the fact. The flags never reach the route solver. | ddl:625-633; `order/condition.go:25` |
| 13 | **Multi-currency** | 17 `Currency STRING(3)` columns threaded through orders and payments | No FX rate table, no conversion, no multi-currency arithmetic. | `schema/spanner.ddl` |
| 14 | **i18n** | 738 keys × en/ru/uz, generated to web JSON + iOS `.strings` + Android `strings.xml` | **~1,582 hardcoded English strings and 0 translation calls in the portals**; 0 `NSLocalizedString` in 453 Swift files; 1,125 hardcoded `Text("...")` in Kotlin. The generated catalogs are referenced by nothing. | `packages/i18n/generated/*`; portal `.tsx` scan |

Two more that are naming rather than function: `RunMEIONetwork` ("Multi-Echelon Inventory Optimization") is a two-node greedy donor/receiver swap per SKU (`replenishment/mei_engine.go:168`); `SourceFallback = "KMEANS_BINPACK"` labels an algorithm that contains **no k-means anywhere in the repository**.

**Recommendation:** every item above is either wired, deleted, or explicitly renamed to what it is. Shipping `MinConfidenceScore` to a customer-facing settings screen when the field is ignored is a trust liability, not a feature.

---

## 3. Correctness defects that will bite in production

Ranked. These are bugs, not gaps.

**P0-1 — The deployed route optimizer has a unit error that silently produces zero manifests.**
`services/optimizer-core/server/contract_solver.py:145-151` registers a transit callback returning **meters**. The same callback index is then registered as the **Time** dimension with a horizon of 1440 (intended as 24×60 minutes), and cumulative variables are constrained to HH:MM-derived **minutes** (`:171-184`). **Any route totalling more than 1.44 km is infeasible.** `AddDisjunction` is never called, so no node can be dropped — the whole model goes infeasible and every order is orphaned (`:264-268`). The backend's validator iterates `res.Routes`, so an empty route list **passes validation** and returns `SourceOptimizer` as a success (`dispatch/plan/optimize.go:92-108`) — the well-built bin-packing fallback never engages. Receiving windows are populated and inherited from the retailer record (`dispatch/repository.go:58-59`), so this triggers in any real deployment.

**P0-2 — `supplier-app-android` does not compile.** `refactor_android.py` used `re.search` with `DOTALL` and a `^\}` terminator, which swallowed whole files into each extracted "component". `OrgFleetPickers.kt` is 2,132 lines containing **four verbatim copies** of the original screen; `OrgFleetScreen`, `DriverRoster`, and five other declarations each appear 6 times in the same package. The app with the widest endpoint surface (90) is unbuildable.

**P0-3 — Driver iOS cannot decode any response.** `APIClient.swift:76` sets `keyDecodingStrategy = .convertFromSnakeCase`, which rewrites incoming keys to camelCase *before* they are matched against 154 explicit snake_case `CodingKeys`. `Order.retailerId` is non-optional, so `getAssignedOrders()` throws — **the driver cannot load a route.** The `ClientPolicy` force-update gate fails the same way and its error is swallowed by a bare `catch`. `payload-app-ios/.../APIClient.swift:72` has the identical bug. Fix: delete one line in each file.

**P0-4 — Server rejections become "successful" offline deliveries on iOS.** `FleetServiceLive.swift:57-64` treats only `FleetServiceError` as a business rejection, but `APIClient` never throws that type — it throws `APIError`. A 403 "you are 4 km from the customer" is caught by the `else` branch, queued offline, and **reported to the driver as a completed delivery**, then replayed forever with coordinates `(0, 0)` (`:266-272`, directly beneath a comment claiming it refuses to fabricate coordinates).

**P0-5 — Every domain event is published to Kafka roughly twice.** `infra/k8s/backend-go-worker/deployment.yaml:10` sets `replicas: 2`; `runtime_workers.go:15-18` starts `OutboxRelay.Start` unconditionally on every pod; `outbox/relay.go:124-153` fetches unpublished rows with a plain `Single()` read and no lease or claim. The relay's own doc comment warns against exactly this. And the dedupe layer **cannot catch it** — `kafka/event_dedup_middleware.go:16` keys on `(group, topic, partition, offset)`, and two pods publishing the same logical event produce two distinct offsets. No leader election exists anywhere in the repo.

**P0-6 — No Spanner backup exists.** No `google_spanner_backup_schedule`, no `version_retention_period`, zero backup resources in state. No documented RPO/RTO, no restore script, no restore rehearsal. Combined with P0-7, there is an unbounded-loss path with no recovery mechanism.

**P0-7 — Failed migrations report success.** `cmd/apply-migration/main.go:258-277` treats `AlreadyExists`, **all of `FailedPrecondition`**, and any `InvalidArgument` mentioning "already exists" as benign. `FailedPrecondition` is what Spanner returns for genuine failures like adding a `NOT NULL` column to a populated table. The Job exits 0 and the rollout proceeds against a schema the code doesn't expect. There is **no migration version tracking table** — for an existing database, `cmd/setup` reapplies all 1,292 DDL lines on every invocation.

**P0-8 — The prod overlay is undeployable.** `kustomize build infra/k8s/overlays/prod` emits literal `PEGASUSX_BACKEND_GO_IMAGE_PLACEHOLDER` for 3 workloads, `pegasusx-optimizer-core:local`, and two `:latest` images from a **different GCP project**. The `backend-go-secrets` ExternalSecret requires 12 keys; Terraform provisions 5 of those names and only 5 of all 19 secrets have any version — none of them JWT, GlobalPay, Adyen, Stripe, or Maps. ESO sync is atomic, so no pod starts. The Ingress references TLS secret `pegasusx-api-tls` that nothing creates.

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
- **`DelinquencyCount` is never computed.** It is read and persisted verbatim; nothing increments it. **Credit risk scoring was deliberately removed** (Phase A) — CREDIT_LEAVE / placement use status + available only — so this is no longer “the risk engine’s primary input,” but aging/dunning still cannot start without it.
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
| 4 | Credit decision | No | **Automated** at placement — **limit + status only** (scoring removed); `DelinquencyCount` still never incremented |
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
| 22 | **Dunning / collections** | **YES, entirely offline** | **Absent.** No `PaymentTerms`, no `DueDate`, no aging, and no SMS/email transport to send a reminder through |

**The honest read.** Steps 1–9 — the whole commercial decision loop — are genuinely automatable today, and that is more than most SFA vendors ship. But the automation is only as good as its inputs, and right now step 1 produces `last_order / 2` even though **retailer on-hand balances now exist** — the auto-order path still does not consult them. An agent walks in, looks at the shelf, knows what sells, knows what's near expiry, knows the promo calendar, and negotiates. This halves the last invoice.

Two blockers are structural rather than unbuilt:
- **Cash collection.** In a COD-dominant market the driver *is* the collections function. The platform doesn't remove that human, it instruments him — arguably the correct product decision, but it converts field headcount from sales to logistics rather than eliminating it.
- **Dunning.** The single most valuable field-agent activity the platform does nothing about, and it's blocked by two small absences (a due-date field and an SMS provider), not by anything architectural. **This is the highest ROI-per-line-of-code item in the entire report.**

---

## 6. Market reality — the premise needs correcting

The premise that "there is no such system in the world" is **not accurate**, and building on it is strategically dangerous. Three established categories already occupy this space:

**B2B wholesale marketplaces connecting many suppliers to many retailers.** Udaan (India, FMCG/staples/pharma, ~$917M deployed, settled a $178M Singapore insolvency in 2026, now EBITDA-positive per city across ~16 clusters and heading for an India listing), MaxAB-Wasoko (Africa, all-stock merger, 450k+ merchants), TradeDepot (Nigeria), Ankorstore (Europe, moved to 0% commission on reorders in January 2026), Faire, Frubana, Sabi, MarketForce.

**Sales-force automation / distributor management, which is exactly the "replace the agent's order pad" product.** FieldAssist (32+ countries, offline-first, distributor management + retailer ordering app + route optimization + planogram audits), Botree, PepUpSales, Massist. Predictive order recommendations before the rep walks in are a *standard* feature in this category, not a differentiator.

**Agentic order entry.** Proton.ai shipped GA agentic order & quote entry in July 2026 — it reads emails, PDFs, handwritten lists, and screenshots, applies contract pricing and sourcing rules, picks the warehouse, surfaces substitutes, and hands the rep a sendable draft. In a head-to-head at a large industrial distributor it found the right part 3× more often than the in-house cross-reference system on description-only lists and cut list-prep time 75%.

**The critical lesson from that landscape, and it is directly relevant.** The pure marketplace/e-commerce thesis largely *failed*. MaxAB-Wasoko retreated from 8 markets to 5 and is betting the company on **embedded fintech**; in Egypt fintech transactions already outpace e-commerce. TradeDepot pivoted to **advertising and data**. MarketForce shut its e-commerce arm. Sabi went to commodity exports. Udaan survived by abandoning geographic breadth for **city-level density** and private label (now 15% of revenue). Ankorstore gave up reorder commission entirely and monetizes via **subscription + fintech**.

Translated for PegasusX: **distribution margin does not pay for this platform. Credit does.** And credit is precisely where the code is weakest — `RetailerCreditProfiles` exists with limits and status, but **risk scoring was deliberately removed** (Phase A; `RiskTier` cleared / ignored), `DelinquencyCount` is never computed, there are no payment terms, no due dates, no aging, no dunning, and no channel to reach a retailer who hasn't installed the app.

**What is genuinely defensible here.** Not "a platform connecting all suppliers and retailers" — that exists and has been expensively fought over. What is unusual is the **vertical depth of a single stack from factory floor to shop shelf**: factory manifests → inter-hub transfers → warehouse loading bay with a dedicated terminal role → volumetric truck packing in VU → geofenced driver handshake → COD/split-payment/credit-delivery with ledger and reconciliation → fiscal receipt. Most competitors are a marketplace bolted onto 3PL, or an SFA app bolted onto someone else's ERP. **Nobody has the whole physical chain in one transactional model with one event bus.** That is the asset. It happens to be worth more as *deep software for one large distributor* than as a thin marketplace for many.

---

## 7. Fit with what retailers and suppliers already run

This is the adoption wall, and it is higher than the technical gaps.

A mid-size Uzbek/CIS distributor runs **1C** for accounting and often stock. A retail chain runs 1C or SAP, plus a POS system, plus possibly a WMS. Their reality: they will not re-key orders into your app, they will not run their inventory in your database, and they will not accept a system that can't produce a journal entry their accountant recognizes.

What the code offers them today:

| They need | PegasusX has |
|---|---|
| An API their ERP can call | **Nothing.** No OpenAPI, no API keys, no OAuth client credentials. Auth is phone/PIN JWT — human sessions only. |
| Events pushed to their system | **Nothing outbound.** The 5 webhook routes are inbound gateway receivers. |
| A file drop | **No SFTP, no export endpoint of any kind.** |
| EDI (ORDERS/ORDRSP/DESADV/INVOIC) | **Zero matches** for EDIFACT, X12, or any EDI message type. |
| Bulk load | **The one real primitive:** a 9-state import wizard (`supplier/import_sessions.go:177`) with signed-URL upload, column auto-discovery, mapping, staging with raw+cleaned JSON, and error summaries. Production-grade — but **inventory/product only, one-way, human-driven**. |
| GS1 identifiers | **GTIN only.** `returns/barcode.go:10-55` validates EAN-8/12/13 and GTIN-14 checksums correctly. **SSCC: 0 matches. GLN: 0 matches.** No ASN, no pallet labels. |
| Label printing | **No ZPL.** Scanning in, nothing out. |
| Accounting export | `PaymentLedgerEntries` and `MasterInvoices` internally; **no journal export, no chart-of-accounts mapping**. |
| Legal receipt | `FISCAL_PROVIDER=PEGASUS` issues platform commercial receipts; **`MY_SOLIQ` OFD adapter exists but needs sandbox/prod creds.** You cannot legally close a Soliq-mandated sale until L5 lands. |
| Data warehouse | **Zero BigQuery references.** |

**Direct answer: no retailer or supplier with existing systems can integrate today, by any path, with any amount of effort on their side.** Every order requires a human typing into a mobile app.

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
| Wire the auto-confirm sweeper | `runtime_workers.go:44` | `go app.OrderService.AutoConfirmDueOrders(ctx, 200)` on a ticker, behind a per-supplier flag. |
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

### 8.1 Demand forecasting — replace the 7-day mean

**Why.** Every downstream decision — reorder point, suggested quantity, auto-order, transfer, dispatch volume — inherits the forecast's error. Today it is an unweighted trailing mean divided by a hardcoded `7` regardless of how many days had data (`replenishment/engine.go:471-475`), so a SKU with one order 6 days ago gets `qty/7` instead of `qty/1`. Intermittent and new SKUs are systematically under-forecast, which in FMCG distribution is *most* SKU-store pairs.

**Algorithm.** Classify each `(warehouse, product)` pair by ADI (average inter-demand interval) and CV² of non-zero demand — the Syntetos–Boylan–Croston quadrants:

- **Smooth** (ADI < 1.32, CV² < 0.49) → Holt–Winters triple exponential smoothing with weekly seasonality:
  `L_t = α(y_t / S_{t-m}) + (1-α)(L_{t-1} + T_{t-1})`, `T_t = β(L_t - L_{t-1}) + (1-β)T_{t-1}`, `S_t = γ(y_t / L_t) + (1-γ)S_{t-m}`, forecast `ŷ_{t+h} = (L_t + hT_t)·S_{t-m+h}`
- **Erratic** (ADI < 1.32, CV² ≥ 0.49) → simple exponential smoothing on level only, wider intervals
- **Intermittent / lumpy** (ADI ≥ 1.32) → **Croston with the Syntetos–Boylan bias correction**: maintain separate smoothed estimates of demand size `z_t` and inter-arrival interval `p_t`, updated only on non-zero periods, then `ŷ = (1 - α/2)·z_t / p_t`. The `(1 - α/2)` factor removes the positive bias in classic Croston — this matters because a biased-high forecast silently inflates every retailer's working capital.

Fit `α, β, γ` per series by minimizing one-step-ahead squared error on a holdout, with sane bounds (0.05–0.4) and a global fallback for short series.

**Inputs.** Daily units per `(WarehouseId, ProductId)` from `Orders.LineItemsJson` where `Status='COMPLETED'`, ≥60 days. Exogenous flags from `SupplierPromotions` (active window) and `SeasonalTemplateOverrides`.
**Outputs.** `DemandForecastBaseline.BaselineQty` = point forecast; `LowUnits`/`HighUnits` from the **empirical residual quantiles**, not the current ±10% constant; `Confidence` from measured historical accuracy for that series, not `orderCount/8`.
**Where.** New `apps/backend-go/planning/forecast/{classify,croston,holtwinters}.go`. Replace the `AVG(ci.Quantity)` SQL in `ai-worker/predictivepush/analyzer.go:52-66`. Keep `WriteBaselineWithOutbox` as the sink; delete the ±10% defaults at `planning/baseline_write.go:37-44` and `planning/forecast_confidence.go:153`, and the magic-number table at `:225-237`.
**Validation.** Rolling-origin backtest: for each of the last 90 days, fit on `[0,t)`, predict `t+lead`, score. Report **WAPE and bias** (`Σ(forecast−actual)/Σactual`) per demand class. Gate deployment on beating the current 7-day mean by >15% WAPE on ≥80% of series. Bias is the metric that matters most — a consistently +15% forecast bankrupts a retailer regardless of WAPE.

**Then make seasonality real:** multiply the baseline by `SeasonalTemplateOverrides.Multiplier` (currently defined and never read) and replace the two hardcoded templates with multipliers *estimated* from the Holt–Winters `S_t` indices. Replace the fake weather/POS signals (`predictivepush/signals.go:96-130`) with either a real weather API and real POS feed, or nothing — a hardcoded `Qty: 2` in summer is worse than no signal because it looks like one.

---

### 8.2 Safety stock with an explicit service level

**Why.** The current reorder point is `burn·lead·1.15` — a flat 15% buffer with no notion of variability or target service level (`replenishment/engine.go:259`). Two SKUs with identical mean demand but wildly different volatility get identical protection. Suppliers cannot answer "what fill rate am I buying?"

**Algorithm.** `SS = z_α · √(L·σ_d² + d̄²·σ_L²)` and `ReorderPoint = d̄·L + SS`, where `z_α` is the normal quantile for the target cycle service level (98% → 2.054), **`σ_d` is the standard deviation of forecast residuals from §8.1's backtest** — not of raw demand, which is the mistake most implementations make — `d̄` is mean daily demand, `L` is mean lead time, `σ_L` is lead-time standard deviation.

**Prerequisite that does not exist.** `FactoryLeadDays` is hardcoded to `2` (`engine.go:207`) because **there is no observed lead-time data**. `FactoryInternalTransfers` has no `ReceivedAt` column, so actual transfer duration is unmeasurable. Add `ReceivedAt TIMESTAMP` and start collecting before this formula can be honest; until then run with a conservative `σ_L` and label it as assumed.

**Inputs.** Residual σ from §8.1; observed lead times from the new column; `TargetServiceLevel FLOAT64`, `LeadTimeDays`, `LeadTimeSigmaDays` added to `ReplenishmentPolicies`.
**Outputs.** Replaces `reorderPoint` at `engine.go:259`, feeding `computeSuggestedQty`. Also fix `InTransitQty`, declared at `engine.go:148` and **never populated by any code path** — it should sum open transfer quantities, and its permanent zero currently inflates every suggestion.
**Validation.** Replay the last 90 days of inventory positions under both policies; report achieved fill rate and average on-hand. The new policy must hit target ±2pp at equal or lower inventory.

---

### 8.3 Ground the auto-order in inventory, not in the last invoice

**This is the single change that moves autonomous replenishment from toy to credible.**

**Why.** `qty := line.Quantity / 2` (`synthesis/engine.go:310`) reads no inventory, no burn rate, no forecast, no lead time. The `DemandForecastBaseline` table the system already computes is **never consulted by the order-creation path**. And because the write is a raw `spanner.InsertMap` (`:336`) rather than `order.Service.Create`, an order the system generates for itself gets **no stock reservation, no price re-quote, no promotion application, and no credit check**.

**Algorithm.** Periodic-review `(R, s, S)`: at review interval `R`, if inventory position `IP = on_hand + on_order − backorders ≤ s` (the reorder point from §8.2), order up to `S = s + D(R)` where `D(R)` is forecast demand over the review period, then round up to case/pallet quantity using `UnitsPerPack`.

**The missing input, and how to get it.** The platform does not know the retailer's shelf. Three options in preference order:
- **(a) POS integration** — the correct answer, and `externalPOSSignals` (`signals.go:114`) is exactly where it belongs.
- **(b) Periodic shelf counts** from the retailer app — cheap, low-friction, gives ground truth.
- **(c) Inferred position** — cumulative deliveries minus forecast consumption since the last known count, with **confidence decaying by days-since-count**. Buildable now with zero external dependency.

Option (c) is what should gate touchless: high inferred confidence → auto-confirm; low → propose-and-ask. That makes the confidence score *mean something* instead of being the always-passing arithmetic at `engine.go:181`.

**Outputs.** Replaces `synthesis/engine.go:306-326`, and critically routes through `order.Service.Create` instead of the raw insert at `:336`. Link the order to its rationale — the `AI_PREORDER` row carries `DerivedFromOrderId` but **no reference to the `AIPredictions.PredictionId` that justified it**, even though both are written in the same `Apply` batch (`:125`). Add `SourcePredictionId` so a retailer can ask "why?".
**Validation.** **Shadow mode for 60 days**: generate proposals, don't send them, record what the retailer actually ordered. Report proposal-vs-actual WAPE per retailer and the fraction a human would have accepted unmodified. **Do not enable touchless below 80% acceptance.** This is also the mechanism that tells you whether the whole autonomy thesis is real.

---

### 8.4 Forecast accuracy monitoring — build this before anything else in §8.1–8.3

**Why.** Right now nothing in the backend measures whether a prediction was right. The only accuracy figure in the product is computed in React, never persisted, mislabeled, and joins per-product baselines against per-warehouse order counts (§2.11). **Without this you cannot know whether §8.1–8.3 helped.** It is the cheapest item here and it gates the value of the three most expensive ones.

**Algorithm.** Nightly job joining `DemandForecastBaseline(supplier, warehouse, product, date)` to actual units shipped **at the same grain**. Compute per-SKU and per-warehouse rolling 7/28-day WAPE, bias, and the **tracking signal** `TS = cumulative_error / MAD` — alert when `|TS| > 4`, the classic out-of-control threshold that catches a series whose model has stopped fitting.

**First, fix the join.** `cmd/planning-training-export/main.go:146-155` aggregates `COUNT(*)` of orders grouped by warehouse-day, so **every product row for a warehouse-day gets the same label** — the number of orders that warehouse completed. It must be `SUM(quantity)` from `LineItemsJson` grouped by `ProductId`. As it stands the export is unusable as training data, which makes the `-min-rows` flag on the cronjob purely decorative.
**Outputs.** New `ForecastAccuracyDaily` table. Feed measured reliability into `AggregateDemandConfidence` so `ConfidencePct` becomes an empirical number instead of `{75, 72, 65}`.
**Where.** New `apps/backend-go/planning/accuracy.go`; replace the client-side calc at `supplier-portal/.../analytics/demand/page.tsx:67-86`.

---

### 8.5 Routing and dispatch

**Why.** Beyond the P0 unit bug, the solver is blind to the constraints that make a delivery legal and physically possible. `Products.HandlingClass`, `RequiresColdChain`, and `IsHazardous` are loaded into line items (`order/volume.go:80`) and carried on events (`events/types.go:493-495`) — then **dropped at the contract boundary**, because `contract.Stop` (`packages/optimizer-contract/types.go:68-90`) has no such field. A frozen order and a hazmat order are indistinguishable to the router.

**Constraint fidelity.** Add to `contract.Stop`: `handling_class`, `requires_cold_chain`, `is_hazardous`, `service_minutes`, `access_restriction`. Add per vehicle: `has_refrigeration`, `hazmat_certified`, `shift_start`/`shift_end`, `max_route_minutes`, `start_lat`/`start_lng`. In OR-Tools these become `AddAllowedVehiclesForIndex` (incompatibility) and a `RouteDuration` dimension (shift/HOS). Validate: assert no cold-chain stop is ever assigned to a non-refrigerated vehicle.

**Multi-depot.** `contract_solver.py:98-99` forces all vehicles to `vehicles_in[0]`'s origin, and the Go client makes it structural by passing one `depotLat/depotLng` to every vehicle (`optimizerclient/client.go:172-184`). Pass per-vehicle start/end nodes and use OR-Tools' native multi-depot form.

**Real road distances.** Call OSRM `/table/v1/driving/` for the matrix instead of haversine. The OSRM deployment exists (needs the P0 volume fix) and the HTTP client with circuit breaker exists (`routing/osrm.go`) — it is only ever used for polyline geometry, never for `/table/`. Cache matrices in Redis keyed by the sorted stop-ID set; retailer locations are static, so hit rates will be high. Validate against GPS-actual leg durations from driver telemetry — expect to find today's haversine estimates 20–40% short.

**Fix `max_stops_per_route`.** Currently post-hoc truncation (`:228-234`): the solver builds a full route, the tail is chopped to orphans, and **`distance_km`/`duration_min` are not recalculated** — so persisted route metrics are wrong. The tail is where guided local search puts the geographically awkward stops, so you systematically orphan the hardest deliveries. Model it as a real count dimension.

**Fix the time budget.** `OPTIMIZER_SOFT_TIMEOUT_SEC: "2"` equals the solver's internal `DEFAULT_TIME_LIMIT_MS = 2000`, so any solve that uses its full budget returns HTTP 504. Make the HTTP timeout strictly greater (internal 5s, HTTP 8s), honor `tunables.time_limit_ms` (ignored at `:195`, making `MAX_TIME_LIMIT_MS` unreachable), and vectorize the O(n²) pure-Python haversine matrix with numpy — 500 stops is 250k `atan2` calls *inside* the HTTP budget, which is the real ceiling (~100–150 stops today).

**Stop reporting OPTIMAL from a heuristic.** `vrp.rs:183-187` and `cpsat.rs:133-137` return `SolverStatus::Optimal` whenever nothing is unassigned. Also fix the Rust index bug — indices are computed from the *shrinking* `unassigned` vector (`vrp.rs:51-62`), so after the first `remove()` every matrix lookup reads the wrong cell, while the 2-opt block a few lines down indexes correctly. Since the Rust service is deployed nowhere, the honest options are: delete it, or deploy the **Go Clarke-Wright** (`ai-worker/optimizer/clarke_wright.go`) as a genuine A/B arm — it is correct, deterministic, has proper window feasibility with wait-at-open semantics, and deserves better than being orphaned on port 8081.

**Dispatch.** The bin-packer is good (balanced first-fit over H3 groups, smallest-fit vehicle with tetris buffer, Kahan summation) — just rename `KMEANS_BINPACK`, which contains no k-means. Then add **automated exception remediation**: `ManifestExceptions` is currently read-and-display with manual reassignment. On exception, re-run the solver for the affected subset with the failed stop penalized, and escalate only if no feasible reassignment exists. And wire `PriorityBoostPenalty` (`dispatch/zone_override.go:81-92`, never called) so `PRIORITY_BOOST` and `REROUTE` zones stop being advisory metadata that changes nothing.

---

### 8.6 Credit, collections, and the monetization engine

**Per §6, this is where the money is.** It is also the thinnest layer relative to its importance.

**Payment terms and aging.** Add to `Orders`: `PaymentTermsDays INT64`, `DueDate TIMESTAMP`. Add `RetailerPaymentTerms(RetailerId, SupplierId, TermsDays, GracePeriodDays, EarlyPayDiscountBps)`. Compute a nightly aging bucket per retailer (current / 1-30 / 31-60 / 61-90 / 90+) into a new `ARAging` table. Without a due date there is no such thing as overdue, which is why collections currently happens entirely off-platform.

**Compute `DelinquencyCount`.** Nothing increments it today (`credit` persists the field verbatim). On the nightly aging job: increment when an invoice crosses its grace period, decrement or decay on sustained on-time payment. Required for dunning even though **score-based `RiskTier` gating was removed**.

**Risk scoring (product decision).** Phase A removed the scoring desk / worker / `RiskTier` gates — CREDIT_LEAVE and placement are **status + available only**. Do **not** re-add a scorecard unless product explicitly reverses that decision. Prefer aging + dunning + hard status holds over resurrecting ML-ish tiers.

**Dunning.** A state machine per overdue invoice: `DUE_SOON` (T−3) → `OVERDUE` (T+1) → `ESCALATED_1` (T+7) → `ESCALATED_2` (T+14) → `CREDIT_HOLD` (T+21) → `COLLECTIONS`. Each transition emits a notification and, at `CREDIT_HOLD`, sets `RetailerCreditProfiles.Status` so `credit.CheckOrder` blocks new orders automatically. This is a small amount of code with a direct revenue effect.

**Which requires notification transports that do not exist.** `notifications/` has exactly two: real FCM and a `LogTransport` (`transport.go:15`). **No SMS, no email, no WhatsApp** — zero references to Twilio, SendGrid, SMTP, or WhatsApp Business API in the repo. A retailer who hasn't installed the app is unreachable. Add an `SMSTransport` (local aggregator for UZ), an `EmailTransport`, and — given WhatsApp's dominance in this market segment — a WhatsApp Business API transport. Route by channel preference with fallback. **This is a prerequisite for dunning, for OTP reliability, and for onboarding retailers who don't have the app yet.**

**Fiscalization, for real (Soliq).** `ProviderFromEnv` already selects `PEGASUS` / `FAKE` / `MY_SOLIQ` / `GLOBAL_PAY`. Finish L5: Soliq sandbox SUCCESS with real creds behind `FISCAL_PROVIDER=MY_SOLIQ`. PEGASUS remains the non-legal commercial path. Retry state machine + `OrderFiscalReceipts` + hard gate are already built.

**Refunds and settlement.** The only occurrence of "Refund" in non-test Go is reading `charge.AmountRefunded` off a Stripe webhook. Add a refund initiation path (full/partial, per gateway, with ledger entries and fiscal reversal). Add real supplier payout execution — `GET /v1/payment/settlement/authority` is a reporting view, and `warehouse/ops_portal.go:772` returns a hardcoded fake invoice (`"invoice_id": "inv-1"`) in the treasury endpoint.

**Then fix the billing meter** (§2.2) so the platform can actually charge: workers are already constructed and the consumer starts — **fix the column mismatch** (`EventId`/`MeterType`/`CurrentValue` vs `AmountDelta`), define a fee schedule (per-order, per-GMV-bps, or subscription), and emit invoices. Today nothing is *successfully* metered.

---

### 8.7 Warehouse execution — the capability that caps the addressable market

**Why.** Warehouse bins, lots, expiry, serials, pick lists, cycle counts, and stocktake are **absent** — not unimplemented, absent. (Retailer on-hand **is** modeled: `RetailerStockBalances`, `RetailerReceiveSessions`, movements, counts — Phase B.) `Products` has `IsPerishable` and `StorageTempMinC/MaxC` but nothing tracks an actual expiry date on **warehouse** stock. The current model is "a warehouse is a bag of SKUs; a human finds the goods from memory." That works for a small distributor and is **disqualifying for food or pharma**, and it is why step 10 in §5 is a hard blocker.

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

**Shared module per platform, first.** iOS shares 0.8% and Android 1.0% of their code; ~12,000 lines are byte-identical copy-paste concentrated in the enterprise updater, HTTP plumbing, WS envelope, and theme. Create a real Swift Package (`PegasusKit`) and an Android library module, move those four categories in, and add `gradle/libs.versions.toml`. Nothing else on this list is affordable to maintain 6× per platform.

**Generic offline queue.** Today Android supports **one endpoint** and silently deletes everything else (`OfflineSyncWorker.kt:45-50`), while iOS's SwiftData store holds 4 fields with no endpoint, no payload, no retry count, and no coordinates (`OfflineDeliveryStore.swift:21-30`). Persist `(endpoint, method, payloadJson, idempotencyKey, capturedLat, capturedLng, capturedAt, attemptCount)` and replay by endpoint. **Send the coordinates captured at the time of the action, not the current ones** — the server's 500m geofence check currently has nothing to validate against on replay. Remove `.fallbackToDestructiveMigration()` and write real Room migrations, or an app update will keep destroying offline deliveries.

**Proof of delivery.** There is **no photo and no signature capture in either driver app** — a repo-wide grep for every capture API returns only supplier product images. `photoProofUrl` is threaded through four layers and called with `nil` at both sites (`FleetMapView.swift:93`, `OffloadReviewScreen.kt:328`). For a **credit delivery** — goods handed over with no payment — there is no evidence artifact at all. Add `PhotosPicker`/`ImageCapture` plus a signature pad, store the image as a file reference in the offline queue, and make it mandatory for `markCreditDelivery` and `reportShopClosed`.

**Background location, iOS.** Add `location` to `UIBackgroundModes` (the plist declares only `fetch`/`processing` while the code sets `allowsBackgroundLocationUpdates = true`, which raises `NSInternalInconsistencyException`), remove the malformed conflicting `INFOPLIST_KEY_UIBackgroundModes`, set `pausesLocationUpdatesAutomatically = false` and `activityType = .automotiveNavigation` — without which iOS auto-pauses and never resumes, the classic silent tracking death. Either register the declared `BGTaskScheduler` identifier or remove it; it is currently inert.

**Reconnect and reboot.** Remove `MAX_RECONNECT_ATTEMPTS` (`TelemetrySocket.kt:103-107`) — ten tunnels currently ends telemetry for the shift, since `reconnectAttempt` only resets in `onOpen`; hold at max delay and reset on `NetworkCallback.onAvailable`. Write telemetry to Room **before** attempting the send and delete only on acknowledgement, because OkHttp's `send()` returns true on *enqueue* so the offline fallback rarely fires. Add a `BOOT_COMPLETED` receiver (none exists in any manifest) and re-cancel-safe scoping in `TelemetryService` — `serviceScope.cancel()` in `stopTracking()` permanently kills the scope, so a `START_STICKY` restart comes back half-functional.

**Scanning throughput.** Warehouse and payload scanning is unusable at picker volume: a network round-trip per scan with no local EAN→SKU cache from the already-downloaded manifest, re-scanning a SKU **un-checks** it (`toggleItem`), a 1.5s debounce caps repeated-SKU rate at ~40/min, ML Kit runs all 13 symbologies per frame with no `BarcodeScannerOptions`, no torch control anywhere, and **no hardware scanner support at all** (zero DataWedge/Zebra/keyboard-wedge hits) — so pickers on Zebra TC-series devices can't use the trigger. Fix all six: prefetch the map, increment a per-line scanned count instead of toggling, drop the debounce, restrict to EAN-8/13, add torch, add a hidden `BasicTextField` wedge path plus a DataWedge intent receiver.

**Push, deep links, localization.** Firebase client JSON/plists are **committed** under each mobile app (2026-08). Still required: `aps-environment` entitlements and `PrivacyInfo.xcprivacy` (App Store), declare `FirebaseMessagingService` where missing, refuse `demo-pegasus`/`demo-key` fallbacks, and owner SHA-1 + real SMS for Phone Auth. Register URL schemes and intent filters — `deepLink` is decoded into DTOs in all 12 apps and then discarded because no app declares a handler. Wire `packages/i18n/generated/*` into `project.yml` resources and Gradle `res.srcDirs`, then mechanically replace 1,125 Kotlin literals and every Swift string.

**Security hardening.** Set `kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly` on Keychain items (currently unset, so tokens are unreadable while the phone is locked — breaking background reconnect — and are included in device backups). Add `FLAG_SECURE`/`isCaptured` blur on driver PIN, cash, and card screens. Add SQLCipher to Room and encryption to SwiftData — both currently hold orders, addresses, and a GPS trail in plaintext. Add certificate pinning. Split `network_security_config.xml` into `src/debug` — it currently permits cleartext to a developer's `192.168.0.101` in release. Sign `updater.json` with a detached Ed25519 signature verified against a pinned key and allowlist the manifest host; today Android verifies a hash supplied by the same server as the APK, and **iOS OTA verifies nothing at all**.

---

### 8.9 The integration layer — what turns this into a platform

Per §7 this is the highest-leverage work in the report relative to its cost, because the hard substrate already exists.

**1. Machine identity.** Per-tenant API keys or OAuth2 client credentials with scopes and per-key rate limits. This blocks everything else. The Redis sliding-window limiter (`bootstrap/redis_rate_limiter.go`) and request-class differentiation already exist; this is mostly issuance, storage, and a middleware. Also remove or tightly gate `LOAD_BOOTSTRAP_SECRET`, which currently exempts requests from rate limiting entirely (`reliability_middleware.go:261-267`).

**2. OpenAPI 3 spec + generated SDKs, published.** Zero `openapi` references exist today. No integration team will start without a contract. Generate from the Go handlers so it cannot drift — and note this also fixes the frontend's 374 hand-written TS types against 774 Go `json:` tags reconciled by nothing.

**3. Partner order API.** `POST /partner/v1/orders` (idempotency-keyed — the infrastructure exists), `GET /partner/v1/orders/{id}`, `GET /partner/v1/catalog` with the authenticated retailer's resolved prices, `GET /partner/v1/inventory/availability`. **This alone eliminates most manual entry** and is the single feature that makes a chain retailer able to buy.

**4. Outbound webhooks.** The hard part is done: `OutboxEvents` + Kafka + a typed event registry. What's needed is `WebhookSubscriptions(TenantId, Url, EventTypes, Secret, IsActive)`, an HTTP delivery worker with HMAC-SHA256 signing over the raw body, exponential backoff, and a dead-letter view with replay. **Realistically 1–2 weeks on top of existing plumbing**, and it is what lets a customer's ERP react without polling.

**5. Bidirectional bulk data.** Reuse the import wizard's staging machinery for the missing **export** side — orders, invoices, inventory snapshots, ledger — over both API and SFTP. There is currently no export endpoint of any kind.

**6. Then, for enterprise accounts:** EDI mapping (ORDERS / ORDRSP / DESADV / INVOIC) over AS2 or SFTP; **GLN** party identifiers and **SSCC** logistic-unit identifiers (both entirely absent) to enable ASNs and pallet labels; ZPL label printing; and a 1C connector (journal export with chart-of-accounts mapping), which in this market matters more than SAP.

---

### 8.10 Multi-tenancy — the honest migration path

**Phase 1 — request-scoped tenancy. This is the whole game and it cannot be done incrementally.** Add `SupplierID` to the auth claim set and to a request-scoped tenant context; delete the `supplierID` field from every service struct; change the ~15 constructor sites in `bootstrap.go`; thread tenant context through every repository method that currently takes a constructor-bound ID; add middleware that **fails closed** when tenant context is absent. Tables changed: none — the schema is already correctly shaped, which is a genuine asset. Code changed: **150–250 files** (`order/` alone is 74 files, `warehouse/` 67, `supplier/` 61, `dispatch/` 26, `payment/` 28).

The reason it can't be incremental: today isolation is safe *only because* the ID is a startup constant. The moment it becomes request-derived, all 411 endpoints are potential IDOR vectors and there is no central enforcement point to lean on. Either every path is tenant-aware or none are safe. **Until Phase 1 lands, disable multi-supplier registration** (`supplier/service.go:433-447` currently mints up to 10 tenants the runtime cannot serve, and their orders would be attributed to `seed-supplier-1` — worse than refusing).

Also in Phase 1: per-tenant rate limits and quotas (the limiter keys on JWT `sub`, so a tenant with 500 users gets 500× the quota of a tenant with one), and outbox partitioning by tenant (`OutboxEvents` is a single global queue — one noisy tenant delays every tenant's events).

**Phase 2 — multi-supplier cart and order splitting.** New `ParentOrders`; add `ParentOrderId` to `Orders`; drop the supplier filter from cart reads (`retailer/repository_cart.go:44`); build a split engine fanning out per-supplier child orders each with its own credit check, inventory plan, pricing resolution, and warehouse assignment; roll status up for the retailer UI. Note this is *not* a cart change — one `SupplierId` is in the `Orders` primary key. 1 new table, 2 altered, 30–50 files.

**Phase 3 — global product master.** `GlobalProducts` keyed by GTIN with brand/manufacturer/pack-size; `SupplierProductOffers` mapping `(SupplierId, ProductId) → GlobalProductId` with price, MOQ, lead time; `ProductMatchQueue` for human review; `UnitsOfMeasure` with a real pack hierarchy (each/inner/case/pallet — currently a single nullable `UnitsPerPack`). Matching pipeline: exact GTIN, then fuzzy on brand + normalized pack size + unit measure, with conflicts queued. 4 new tables, 20–30 files, plus a worker. This is the prerequisite for cross-supplier comparison, and it reuses the GTIN checksum validation that already exists in `returns/barcode.go`.

**Phase 4 — marketplace commerce.** Fix the billing meter schema (§2.2 — workers already wired); supplier ratings and scorecards; RFQ / competing quotes (note `NegotiationProposals` is delivery-date negotiation, not price bidding); split payments and escrow — which likely **forces a second gateway integration**, since Global Pay probably lacks sub-merchant support; supplier payout execution.

**Phase 5 — tenant operations.** There is currently **no platform admin console at all** — no supplier management, no approval queue, no suspension, no offboarding. A supplier can self-register and nobody can approve or remove them. Add the console, an approval workflow with document collection and KYB, tenant-scoped audit, and per-tenant observability.

**Honest total: 250–400 files touched.** Multi-tenancy was designed into the schema and designed out of the runtime.

---

### 8.11 Per-role and per-app gap summary

| Role | App completeness | Top gaps |
|---|---|---|
| **Retailer** | iOS 75% / Android 80% / desktop good | Receive+stock+FileClaim/eligibility countdown shipped (G1/G2); still weak on inferred shelf for auto-order (§8.3), KYC, i18n, cross-supplier cart (§8.10 P2), tenant-scoped desktop cache |
| **Supplier** | Portal strong / iOS 60% / **Android 35%, doesn't compile** | Restore Android build, payout execution, refunds, real pricing engine, billing/commission, forecast accuracy view |
| **Warehouse** | Portal good / iOS 60% (0 ViewModels, 0 tests) / Android 55% (146 `!!`, 1 ViewModel for 35 screens) | **Bins, lots, expiry, pick waves, cycle counts (§8.7)** — the largest single capability gap in the product; ViewModels; scanning throughput |
| **Factory** | Portal 9k lines / iOS 55% / Android 55% (0 ViewModels) | Production scheduling, capacity/MRP (`GetSAndOP` is `factories × 700 × 7`), real transfer lead-time capture |
| **Driver** | **iOS 40% (decoder bug blocks route loading)** / Android 70% (best offline story) | P0-3, P0-4, background location, photo/signature POD, generic offline queue, boot receiver |
| **Payload terminal** | iOS 45% (no `.xcodeproj`) / Android 50% | Generate the Xcode project, hardware scanner, per-line quantities, split the 1,700-line god view |
| **Platform admin** | **Effectively absent** — 3 endpoints and a redirect stub | The entire console: tenant management, approval, suspension, fee schedule, global observability, support tooling |

Cross-cutting, all surfaces: localization (0% on web, 0% on iOS, ~1% on Android despite complete generated catalogs), accessibility (2 `htmlFor` against 90 `<label>`, 1 `onKeyDown`, 0 `tabIndex` across 51k lines of portal TSX), testing (frontend ~1 test file per 2,700 lines with zero e2e; mobile 2.2%/2.4% with zero UI tests), and server-state management (0 data-fetching libraries across 483 hand-rolled `useEffect` fetches).

---

## 9. Sequenced roadmap

**Gate 0 — Stop the bleeding (2–4 weeks).** All of §8.0. Non-negotiable end state: CI compiles and tests all 12 mobile apps and lints everything; Spanner backups exist and a restore has been *executed*; Terraform state is remote; the prod overlay renders real digest-pinned images; the optimizer either solves or provably falls back; `supplier-app-android` and `driver-app-ios` work.

**Gate 1 — Make it legal and reachable (4–6 weeks).** Soliq OFD SUCCESS behind `FISCAL_PROVIDER=MY_SOLIQ` (§8.6; PEGASUS commercial path already live). SMS + email transports (§8.6). Payment terms, due dates, aging, `DelinquencyCount`, dunning state machine (§8.6) — **without re-adding removed credit scoring**. Finish push/OTP ops (APNs entitlements, SHA-1, real SMS) on top of committed Firebase configs (§8.8). Without this gate you cannot legally OFD-transact and cannot collect.

**Gate 2 — Make the intelligence real (6–10 weeks).** §8.4 **first** — accuracy measurement, because it is how you evaluate everything after it. Then §8.1 forecasting, §8.2 safety stock (with lead-time capture started immediately, since it needs history), §8.3 inventory-grounded auto-order **in shadow mode**. Ship touchless only when shadow-mode acceptance exceeds 80%.

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
| `ledger/` package | Textbook double-entry, zero callers, tables not in the DDL. Either add the DDL and wire it into `payment` with a reconciliation invariant, or delete it. |
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
