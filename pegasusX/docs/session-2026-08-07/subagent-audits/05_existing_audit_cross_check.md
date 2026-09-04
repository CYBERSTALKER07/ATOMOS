# 05 Existing Audit Cross Check

> **HISTORICAL / FROZEN — do not plan from this file alone.**
> Current SoT: [`PROD_READINESS_SEQUENCE.md`](../../PROD_READINESS_SEQUENCE.md) · [`ECOSYSTEM_GAP_REGISTER_2026-08-12.md`](../ECOSYSTEM_GAP_REGISTER_2026-08-12.md) · [`FEATURES_BY_APP_ROLE.md`](../../FEATURES_BY_APP_ROLE.md).
> Body is a point-in-time snapshot; re-verify any claim against code before acting.


_Source: subagent `b3099a18-b9cd-46ab-b680-9e39ed998d55` from End-Product Reality Report session (2026-08-07)._

# PegasusX / ATOMOS — Documentation-vs-Code Audit Grounding Report

**Audited tree:** `/Users/shakhzod/Desktop/V.O.I.D/pegasusX` (git clean; HEAD `7ee59ba9`, 2026-08-07). **Spot-check method:** ~45 claims verified by direct code reads/greps against `apps/backend-go`, `apps/*-app-*`, `services/`, `contracts/`, `infra/`, and the true monorepo root.

**Critical framing fact first:** the git toplevel is `/Users/shakhzod/Desktop/V.O.I.D` (not `pegasusX`). `PLATFORM_AUDIT.md:3` and `docs/SUBSTANCE_GATE.md:4` name `/Users/shakhzod/ATOMOS/pegasusX` as the evidence base; `docs/PEGASUSX_MASTER_ROADMAP.md:5` says ATOMOS is "canonical working tree; Desktop V.O.I.D is not the execution root." Both trees exist; **they diverge in 202 backend files**, and the Desktop tree is *newer* (it alone has `order/currency_picker.go`, `order/condition_system.go`, i.e. the Aug-7 FX Wave 2+ work). The docs' own canonical-root claim is stale — the live tree is the one they say is *not* the execution root.

---

## 1. CLAIM MAP — what the docs assert, by domain

Status labels below are the docs' own vocabulary: **Wired / Live / Done**, **Partial**, **Theatre**, **Deferred / SKIPPED**, **Gap / open**.

### 1.1 Ordering / order core
- **Live:** "18-state order machine with centralized transition table" (PLATFORM_AUDIT §0, line 13). Cross-role spine "checkout → reserve → create: Wired" (ROLE_ROW_PARITY_MATRIX.md, "Cross-role spine"). Shop-closed/partial-offload/proximity "Wired" (context/parity-ledger.md, Feature 1 table). Claims spine live, GCS evidence fail-closed (current_status.md §1, "Phase A/B"; PLATFORM_AUDIT §1 #15).
- **Partial:** FSM validator coverage — audit's P1 note: "4 call sites against 12 direct status assignments" (PLATFORM_AUDIT §3, line 124). Quantity negotiations flag-off/SKIPPED (SUBSTANCE_GATE §7; parity-ledger "Product-deferred").
- **Doc-stated defect fixed (claimed):** payment bypass skipping fiscalization → "Gate-0: never write COMPLETED from bypass" (current_status §1, Gate-0 Track A).

### 1.2 Payments
- **Live:** Global Pay webhook out-of-band re-verification (PLATFORM_AUDIT §1 #4). Cash path green (`PX_E2E_PAYMENT_CASH_FALLBACK_OK`, current_status §2).
- **Blocked (owner):** card SUCCESS — "real password owner-only" (DEPLOYMENT_READINESS_GAP_LEDGER.md line 15; L1_FIELD_UNLOCK_RELEASE_CHECKLIST all unchecked; current_status §3 #1). No refunds path; payout execution absent (PLATFORM_AUDIT §8.6, line 351).

### 1.3 Credit / collections
- **Live (flag-gated):** terms/AR/aging/dunning step machine/`DelinquencyCount`/CREDIT_HOLD auto-freeze — "§8.6 … are live behind flags" (PLATFORM_AUDIT line 5, §8.6 line 339-345; ROLE_ROW_PARITY_MATRIX line 59).
- **Removed (deliberate):** credit risk scoring — "Phase A removed the scoring desk / worker / RiskTier gates" (PLATFORM_AUDIT §8.6 line 343; master roadmap Phase A "Done"; ROLE_CAPABILITIES_MATH_LOGIC §0.2).
- **Missing:** SMS/email/WhatsApp dunning transports (PLATFORM_AUDIT §8.6 line 347). Fee schedule + invoices for billing meter (§2 #2, §8.6 line 353).

### 1.4 Fiscal
- **Live:** env-selected provider, PEGASUS commercial receipts default on SSMR (current_status §2 ConfigMap row; PLATFORM_AUDIT §2 #1).
- **Not production-ready:** legal Soliq OFD — adapter exists, sandbox creds owner-pending, `PX_E2E_SOLIQ_SANDBOX_SKIPPED` accepted (SOLIQ_SANDBOX_READINESS.md; SUBSTANCE_GATE §7 row "PROOF skipped"; L1/L5 in master roadmap Phase E).

### 1.5 Logistics / dispatch / driver
- **Live:** auto-dispatch worker (60s) + real closed loop (PLATFORM_AUDIT §5 row 9); optimizer constraint fidelity/multi-depot/OSRM matrix "WIRED" in code but **cloud runs heuristic** until AR image + replicas ≥1 (PLATFORM_AUDIT §8.5 line 327; ROLE_ROW_PARITY_MATRIX "Dispatch optimizer" rows — SSMR/prod `replicas: 0`).
- **Partial/open:** rescue capacity state machine (E8, ECOSYSTEM_HARDENING_GAP_PLAN), offline nonce single-use crypto (E9), cash-variance→recon seed (E10), condition→claim bridge (E11).

### 1.6 Warehouse (WMS)
- **Wired (all flag-gated):** lots/FEFO (Wave 1A, `WMS_LOTS_ENABLED`), pick waves + seal gate (1B), cycle counts apply-on-approve + ABC (1C/PR-4), S-shape/LIFO + soft-warn (PR-5), cold chain ingest + quarantine + TEMPERATURE_BREACH auto-raise (PR-6), reconcile + hardening (PR-7) — PLATFORM_AUDIT §8.7 lines 359-369; current_status §1 "WMS Gate 4".
- **Open:** serial tracking; native scan UX residual; Bluetooth sensor fleet / cumulative-minutes-outside-band.

### 1.7 Retailer OS
- **All packs "Wired" ×4 clients** (RETAILER_OS_PRODUCTION_GATE "Product surfaces" table lines 20-29; ROLE_ROW_PARITY_MATRIX lines 96-105: CORE/TEAM/LOCATIONS/STORE_STOCK/POS/SHIFTS/SECTIONS/REPORTS_PRO/CUSTOMER_ASSIST/CT-pulse).
- **Caveats in same docs:** automated gate checkboxes all `[ ]`; "SSMR image includes close-out routes (deploy pending)" unchecked (gate doc line 50); offline POS product-deferred; Reports inventory has no COGS (parity-ledger Retail OS "Partial").

### 1.8 Integration (Gate 3)
- **Wired:** partner keys + scopes, OAuth2 client_credentials, `/partner/v1`, HMAC outbound webhooks + dead-letter replay, bulk export + SFTP, EDI-lite (ORDERS in / ORDRSP/DESADV/INVOIC out, UNA dialect), AS2 (sync MDN, not Drummond), GS1 GLN/SSCC/ZPL, 1C journals CSV/JSON/XML + configurable CoA, partner + JWT-core OpenAPI (PLATFORM_AUDIT §8.9 lines 425-435; ROLE_ROW_PARITY_MATRIX lines 43-52; current_status §1 "Partner Integration").
- **Open:** certified EDIFACT/X12, certified 1C exchange package, Drummond, full ~411-path OpenAPI coverage + generated SDK.
- **Note:** PLATFORM_AUDIT §0 structural fact #3 (line 21) still says "no machine-to-machine integration surface at all" — flatly stale versus its own §8.9 (see §3 below).

### 1.9 Admin / platform
- **"Effectively absent"** (PLATFORM_AUDIT §8.11 line 471): no tenant approval/suspension/offboarding; admin-portal = redirect stub; E16 `PLATFORM_ADMIN` break-glass role open (ECOSYSTEM_HARDENING_GAP_PLAN §EH-IAM; master roadmap Phase F).

### 1.10 AI / optimization
- **Score 4/10** (PLATFORM_AUDIT scorecard line 37). "AI layer is arithmetic" (§0 fact #2, line 19). Forecast algo Croston/SES/Holt-Winters "WIRED" behind `FORECAST_ALGO_ENABLED`; accuracy WAPE/bias/TS "WIRED" behind `FORECAST_ACCURACY_ENABLED`; safety stock v2 "WIRED" behind `SAFETY_STOCK_V2_ENABLED`; inventory-grounded shadow auto-order "WIRED" (`off|shadow|draft|place`); promo elasticity "Partial … sandbox heuristic" (§2 #10); weather/POS signals return empty (§2 #9); MEIO = two-node greedy swap, naming-only (§2 line 86); Rust VRP sidecar undeployed, mis-reports Optimal (§8.5 line 329).

### 1.11 Multi-tenancy
- **1/10 runtime** (PLATFORM_AUDIT scorecard line 39): one supplier bound at startup; request plane serves 1 while data plane holds 10. Phase 1 ADR "Accepted; not yet coded" (§8.10 line 441; current_status §1 "No runtime code in this closure"). Per-supplier perimeter: "Design+helper ready — prod still global key" (master roadmap Phase C).

---

## 2. SPOT-CHECK — 20 consequential claims vs live code

**CONFIRMED** = code matches claim. **OVERSTATED** = code exists but weaker than claimed. **CONTRADICTED** = code disagrees.

1. **"Transactional outbox commits in the same Spanner txn"** — **CONFIRMED.** `apps/backend-go/order/repository_spanner.go:28-38` (`spannerTxnBuffer` collecting events inside closure), `:187`/`:540` single `txn.BufferWrite(mutations)`.
2. **"Optimistic concurrency CAS on Version"** — **CONFIRMED.** `order/repository_spanner.go:208-215` (`ReadRow` of `Version` inside RW txn; reject on mismatch). Doc cited :203-244 — line drift only.
3. **"18-state order machine"** — **CONFIRMED.** Exactly 18 `Status*` constants at `order/service.go:52-70`; centralized table `order/state_machine.go:14-81` (incl. ADR-009 fiscal gate comment at :33 and the "brick the order" comment at :54-55).
4. **"FSM enforced"** — **OVERSTATED (matches the doc's own P1 caveat).** `ValidateStatusTransition` has only 4 production call sites (`order/service.go:1523`, `:2153`, `order/preorder_sweeper.go:168`, `:241`); ~65 direct `Status:` writes exist across order/warehouse/supplier/driver packages, though the main driver funnel does route through `transitionDriverOrder` → validator (`service.go:1523`).
5. **"Payment bypass now enters FISCALIZING (Gate-0 fix)"** — **CONFIRMED.** `order/supplier_ops.go:199-219` — comment "Gate-0: never write COMPLETED from bypass", validates transition, responds `"status": "FISCALIZING"`.
6. **"ARRIVED_SHOP_CLOSED written with no inbound edge" (P1 defect)** — **FIXED in code.** Write path now uses `StatusShopClosedPending` (`order/shop_closed.go:263`), which has a valid inbound edge (`state_machine.go:35`); `ARRIVED_SHOP_CLOSED` survives only in a read query (`order/shop_closed_list.go:69`) and error strings (`order/driver_edges.go:434,973`).
7. **"Outbox relay: UUID ids + ClaimedBy/ClaimedUntil leases + Kafka event_id dedupe"** — **CONFIRMED.** `outbox/outbox.go:219` (`uuid.NewString()`); `outbox/spanner_store.go:89,112,162-163` (lease claim SQL); `kafka/event_dedup_middleware.go:17-18`.
8. **"Single-supplier runtime, 1/10"** — **CONFIRMED.** `bootstrap/bootstrap.go` injects `supplierSeed.SupplierID` into ~20 constructors (e.g. :518, :610, :744, :872-873, :897, :1059, :1183); `order/service.go:352` private `supplierID` field. Nuance the audit understates: create path honors `req.SupplierID` when present and falls back to seed only when empty (`order/service.go:1160-1163`) — but registration still mints new tenant UUIDs the request plane can't isolate (`supplier/service.go` `resolveRegistrationSupplierID`, :442-456 with `maxSuppliers` cap).
9. **"No ML anywhere; Python deps = one line ortools"** — **CONFIRMED.** `services/optimizer-core/requirements.txt` = `ortools==9.15.6755`; grep for tensorflow/pytorch/sklearn/xgboost/prophet/vertexai/gemini across Go modules, ai-worker, services, packages: zero hits.
10. **"Legacy auto-order synthesis `qty/2` diverted when AUTO_ORDER_INVENTORY_GROUNDED"** — **CONFIRMED, with residue.** Divert exists (`ai-worker/synthesis/engine.go:23` flag read, `:308` skip call; backend knob `retailer/auto_order_policy.go:47`), but the `/2` code itself still ships as the non-grounded fallback (`engine.go:331-343`, comment "Scale suggested qty: 50% of last order").
11. **"Fiscal: env-selected providers; MY_SOLIQ hard-fails on misconfig; PEGASUS default"** — **CONFIRMED.** `order/fiscal_provider.go:45-80` (`ProviderFromEnv`, `hardFailProvider` for misconfigured MY_SOLIQ/GLOBAL_PAY); SSMR ConfigMap `FISCAL_PROVIDER=PEGASUS` (current_status §2). **Legal OFD not live — CONFIRMED OPEN** (adapter + GSM slot list only, `docs/SOLIQ_SANDBOX_READINESS.md`; marker accepts `_SKIPPED`).
12. **"Credit: terms/AR/aging/dunning + DelinquencyCount + auto-hold wired behind AR flags"** — **CONFIRMED.** `credit/service.go:186-188` `BumpDelinquency`; `ar/dunning_worker.go:14,56,103` + `ar/dunning.go:76-77` (`ShouldBumpDelinquency` on first OVERDUE); wired in `bootstrap/bootstrap.go:1252` and started `runtime_workers.go` (hourly ticker). Tables `RetailerPaymentTerms`/`ArInvoices` at `schema/spanner.ddl:2374,2420`.
13. **"Credit scoring removed (Phase A)"** — **CONFIRMED in code.** `CREDIT_SCORE_ENFORCEMENT_ENABLED` has **zero** Go references; `RetailerCreditScores` has readers only (`segment/repository_spanner.go:268`), no writers. ⚠️ But see contradiction C2 — three gap-closure docs still instruct operators to enable that dead flag.
14. **"Auto-dispatch worker live, 60s closed loop"** — **CONFIRMED.** Started `runtime_workers.go:41`; default 60s in `warehouse/auto_dispatch*.go:120-131`.
15. **"EDI-lite implemented"** — **CONFIRMED (as 'lite').** `partner/edi/` package with UNA-dialect codec (`codec_test.go:45` asserts `UNA:+.? '`), DESADV with UNB/CPS/PAC/GIN (`desadv.go:18,37`); inbound+outbound workers started (`runtime_workers.go:113-119`); routes `partner/routes.go:38-40`.
16. **"AS2 transport wired, sync MDN, not Drummond"** — **CONFIRMED.** `partner/as2/` (client.go:43-85 MDN request/result; headers.go:28-43); receive route `POST /partner/v1/as2` (`partner/routes.go:21`); config GET/PUT routes.
17. **"1C journals CSV/JSON/XML + configurable CoA"** — **CONFIRMED.** `partner/export_journals.go` (stable CSV/XML column order, :13), `partner/coa.go:14-16` defaults `62.01`/`90.01`/`51.01`, GET/PUT `/partner/v1/coa` (`routes.go:41-42`).
18. **"OAuth2 client_credentials + partner keys + HMAC webhooks"** — **CONFIRMED.** `partner/oauth.go:108-120` (grant check), `oauth_jwt.go:18,135` (`partner_access` token_use); scoped route group `partner/routes.go:24-45`; webhook delivery worker signs `X-Pegasus-Signature: sha256=…` (`partner/delivery.go:42`) and runs on 15s loop (`runtime_workers.go:105-107`); `PartnerApiKeys`/`WebhookSubscriptions` tables (`spanner.ddl:2461,2479`).
19. **"WMS Gate 4 waves 1A–1C + PR-4–7 coded, flag-gated"** — **CONFIRMED.** Real implementation lives in `stocklots/` (`fefo.go`, `picking.go`, `counting.go`, `coldchain.go`, `seal_gate.go`, `rollup.go`) — *not* in `warehouseops/` as §8.7's "Where" said (that dir holds only `doc.go`+`facade.go`); flags wired `bootstrap/bootstrap.go:1148-1153`; tables `spanner.ddl:1134` (StockLots), `:1168` (PickWaves), `:1207` (CycleCounts); e2e files `cmd/ssmr-smokecheck/e2e_wms_pick_waves.go`, `e2e_wms_cycle_counts.go`.
20. **"Retailer POS shipped (packs 0–6)"** — **CONFIRMED backend+desktop; client "Wired" cells UNVERIFIED per-screen.** Routes `retailerroutes/routes.go:87-101` (sessions open/close, sales, void, refund, holds, catalog search); impl `retailer/pos.go`, `pos_holds.go`; tables `spanner.ddl:2166-2211`; desktop UI `retailer-app-desktop/app/(dashboard)/pos/page.tsx`. The gate doc's all-"Wired" matrix has no test/marker citations per client cell.
21. **"Multi-currency FX Wave 1+2+ wired"** — **CONFIRMED.** `fxrates/` package (`convert.go`, `spanner.go`, handlers); mismatch gates `payment/service.go:490`, `payment/global_pay_webhook.go:103`, `payment/retailer_checkout.go:22,151`; picker `order/currency_picker.go:22,99` + `spanner.ddl:2609` FxRates.
22. **"Platform admin effectively absent (3 endpoints)"** — **CONFIRMED in substance, STALE in count.** `apps/admin-portal/` = 3 files, package.json self-describes "Deprecated stub"; zero `PLATFORM_ADMIN` in `auth/`; zero tenant approve/suspend/offboard endpoints. But `/v1/admin/*` now has **17** distinct paths (partner-keys, fx-rates, planning run-once, AR dunning run-once, credit disables) — "3 endpoints" is outdated.
23. **"E1 sup-demo-1 demo desks removed"** — **CONFIRMED.** Zero `sup-demo-1` matches across all `apps/**` client sources (was the P0 in ECOSYSTEM_HARDENING_GAP_PLAN §E1).
24. **"E2 single global delivery perimeter still open"** — **CONFIRMED OPEN (docs agree).** `retailer/proximity_service.go:24` `DeliveryPerimeterKey = "ssmr:delivery_perimeter"` with comment "Production reads stay"; `PerimeterKeyForSupplier` (:35) exists as design-only helper.
25. **"E6 payload/factory in-memory overlays"** — **PARTIALLY STILL TRUE + comment-theatre.** `payload/service.go:41-48` in-memory overlay persists; the claimed gate `PAYLOAD_DEV_OVERLAY` is **only a comment** — no Go code reads that env var. `factory/service.go:63-65,271-273` still holds manifest state in in-memory maps.
26. **"Gate-0 CI: workflows compile all 12 native apps + race/golangci/gitleaks"** — **CONFIRMED — but at the parent repo root, not `pegasusX/`.** `pegasusX/.github/workflows/` holds only `ci.yml` + `desktop-windows-build.yml`; the real workflows live at `/Users/shakhzod/Desktop/V.O.I.D/.github/workflows/pegasusx-ci.yml` and `pegasusx-native-mobile-build.yml` (Android ×6 + iOS matrix confirmed). `docs/GATE0_CI.md` does disclose this nesting — but PLATFORM_AUDIT §3 line 143 links `../.github/workflows/pegasusx-ci.yml`, a path that **does not exist** relative to the audit file.
27. **"Dead chart components unmounted"** — **CONFIRMED.** `warehouse-portal/app/analytics/page.tsx:11` and `supplier-portal/app/(portal)/analytics/page.tsx:27,254` carry "unmounted — no SoT" comments; components no longer render. Files remain on disk.
28. **"HPA 7-milllicore bug fixed; OSRM crash-loop fixed"** — **CONFIRMED.** CPU request now `250m` (`infra/k8s/backend-go/deployment.yaml:123`) vs HPA 70% utilization; OSRM has `pvc.yaml` + volumeMount `claimName: osrm-data` (`infra/k8s/osrm/deployment.yaml:29,66-69`).
29. **"Idempotency keys principal/route-scoped + SHA-256"** — **CONFIRMED.** `idempotency/middleware.go:61-76` (`ScopeKey(principal, routePattern, rawKey)`, SHA-256 body hash).
30. **"Nil Spanner client fails loud; driver PIN bcrypt"** — **CONFIRMED.** `spannerutils/retry.go:26-28` returns `ErrNilSpannerClient`; `warehouse/ops_fleet_handlers.go:71-80` generates random PIN + `bcrypt.GenerateFromPassword` (the `"4321"` plaintext is gone).
31. **"iOS: snake_case decoder removed; background location wired"** — **CONFIRMED.** `driver-app-ios/.../APIClient.swift:76` and payload `:72` carry only the "do not convertFromSnakeCase" comment; `Custom-Info.plist` `UIBackgroundModes` includes `location`.
32. **Kill list executed?** — **MIXED.** `ledger/` package **deleted** (CONFIRMED absent); `SourceFallback="KMEANS_BINPACK"` renamed to `fallback_phase1`/`fallback_validation_rejected` (`dispatch/plan/optimize.go:25-26`). **NOT executed:** `enterprise/` (auth0.go/datadog.go/vault.go) still present; `optimizationjobs/` still present with zero `EnqueueJob` callers; `planning/service.go:212` still emits literal `sku-projection-%d` strings and `GetSAndOP` still returns `factoryCount * 700 * 7` (:252); `docs/adr/` is **empty** despite Phase G exit "docs/adr/ non-empty".

**Unverified (no strong evidence either way):** 409-as-ACK semantics in `OfflineSyncWorker.kt` (file exists; cited lines 106-124 not confirmed in excerpt read); payload-iOS missing `.xcodeproj`; per-client "Wired" cells in the role matrices (no executable evidence attached to any client cell).

---

## 3. INTERNAL CONTRADICTIONS

**C1 — PLATFORM_AUDIT.md contradicts itself on integration.** Line 21 (§0 fact #3): "There is no machine-to-machine integration surface at all… Zero matches for openapi, oauth2, EDIFACT, X12, SSCC, GLN, ZPL, SFTP… A retailer with an ERP cannot integrate today by any path." Line 38 (scorecard, same file): "Integration surface **9.5/10** (OAuth + GS1/ZPL + DESADV SSCC + CoA journals + EDI-lite + AS2 transport)". §7 table (lines 228-234) says "EDI… still open", "**No ZPL**", "**no journal export, no chart-of-accounts mapping**" — then line 236 ("Superseded") reverses all three. The banner/§8.9/scorecard were updated (Aug 6-7); §0/§7 body text was not. Code sides with the updates (spot-checks 15-18).

**C2 — Gap-closure docs instruct enabling a flag that no longer exists.** `docs/gap-closure/STAGING_FLAGS.md` step 3, `PRODUCTION_CUTOVER.md` flag step 3, and `STAGING_FOUNDATION.md` preconditions all turn on `CREDIT_SCORE_ENFORCEMENT_ENABLED`; `MANUAL_CRITICAL_WALKTHROUGHS.md` §4c expects "/credit/collections — score columns visible". Code: zero references (spot-check 13); scoring deliberately removed per `PLATFORM_AUDIT.md:343`, `current_status.md:63`, and master roadmap ("Not re-adding credit risk scoring", final section). Migration `schema/migrations/20260804_phase_c_credit_risk.ddl` still **creates** the orphaned `RetailerCreditScores` table.

**C3 — Canonical repo root disagreement + real divergence.** PLATFORM_AUDIT:3 / SUBSTANCE_GATE:4 cite `/Users/shakhzod/ATOMOS/pegasusX`; master roadmap:5 says Desktop V.O.I.D "is not the execution root"; ECOSYSTEM_HARDENING_GAP_PLAN:4 and the audit file itself live in the Desktop tree. Measured: 202 differing backend files; Desktop-only files are the newest work (Aug 7 FX picker). The ATOMOS copy is stale; docs pointing there are pointing at the older tree.

**C4 — Scale metrics disagree across docs and with code.** Endpoints: "411 distinct" (audit:23) vs "612 route registrations scanned" (FEATURES_BY_APP_ROLE:7) vs "~418 /v1 routes" (ECOSYSTEM_FEATURES_BY_ROLE, Part 0) vs **731 measured** `.Get/.Post/.Put/.Patch/.Delete` registrations today. Tables: "73 Spanner tables" (audit:23) vs **155 CREATE TABLE** in current `spanner.ddl`. Go LOC: "131,670" (audit:23) vs **213,719 measured**. All docs understate a fast-growing tree.

**C5 — CREDIT_COLLECTIONS_ENGINE_PLAN.md is stale as 'current state'.** Its "Verdict (current state)" (lines 5-8) says "no payment terms/due dates/invoices, DelinquencyCount never increments, no aging/dunning" and scopes "wire `ledger/`" — while audit §8.6 says terms/aging/dunning/DelinquencyCount are **live**, and `ledger/` was **deleted** in Gate-0. The plan's verdict predates its own implementation; nobody flipped the header.

**C6 — Deployment ledger vs later deploys.** DEPLOYMENT_READINESS_GAP_LEDGER (Aug 2): "Retail OS API image: **Gap** (pulse 404 on nomock4)". current_status (Aug 4-7) + SUBSTANCE_GATE signoff: SSMR runs `backend-go:ssmr-substance-gate-a66868b8-084112` with marker gate PASS. The ledger was never updated past Aug 2; RETAILER_OS_PRODUCTION_GATE's "SSMR image includes close-out routes" box is still `[ ]` despite the newer image.

**C7 — Missing evidence artifact.** SUBSTANCE_GATE.md:352 and `artifacts/SUBSTANCE_GATE_API_SIGNOFF_2026-08-04.md` cite `artifacts/ssmr-e2e-substance-gate-2026-08-04.log` — **file does not exist** (only `ssmr-e2e-2026-08-04[-final|-retry].log`). The load-bearing "Marker gate: PASS" evidence is a dangling reference.

**C8 — Hardening plan E7/E14 superseded but not marked.** ECOSYSTEM_HARDENING_GAP_PLAN E7 says "no pick-session/wave handlers" (P1) — code now has pick waves + seal gate (spot-check 19; doc's own Appendix B still routes E7 to Phase EH2 as future work). E14 "DR restore unproven" — current_status §1 claims "restore rehearsal RTO ~30 min" executed Aug 5 with artifacts. Both hardening entries are stale relative to code/status.

**C9 — Dated-in-the-future migration.** RETAILER_OS_E2E_MATRIX references `20260809_retailer_auto_order_runs.ddl` — Aug 9 filename while today is Aug 7 — flagged "Apply … DDL", i.e. planned-not-applied; harmless but sloppy provenance.

**C10 — Route/parity doc counts client features nav-only.** FEATURES_BY_APP_ROLE is route/nav-derived and honest about it, but ROLE_ROW_PARITY_MATRIX "Wired" rows for every role×client contradict SUBSTANCE_GATE §10's own signoff where **every** WEB/AND/IOS cell is `DEFERRED` (only API PASS). "Wired" is doing promotional work the signoff doesn't support.

---

## 4. DOC-STATED GAPS — consolidated P0/P1 (source doc → section)

**Owner/ops blockers (all docs agree):**
- Global Pay real merchant password + webhook registration → card SUCCESS marker (GAP_LEDGER "Owner blockers" 1; L1 checklist; current_status §3.1; master roadmap prod-flip policy 1).
- Firebase Phone SMS / SHA-1 / APNs → OTP on device (GAP_LEDGER 2; L1 checklist; current_status §3.2).
- optimizer-core AR image + replicas ≥1 (audit §8.5 line 327; parity matrix "Dispatch optimizer"; current_status §1 Gate-0 residual).
- Real GSM versions + prod overlay apply + ManagedCert DNS (audit P0-8 line 120; current_status "P0-8 prod overlay … Residual ops").
- Soliq sandbox creds → `PX_E2E_SOLIQ_SANDBOX_OK` (SOLIQ_SANDBOX_READINESS; audit Gate 1 line 481).

**P0 (hardening plan E-tier):** E1 done (verified); E2 per-supplier perimeter — **open in code**; E3 shop-closed DDL — done per Phase A.

**P1 (hardening plan):** E4 moot/stale (scoring removed); E5 billing meter minor-units + milestones + fee schedule/invoices (partial: schema+decode wired); E6 payload overlay gate not in code + factory in-memory maps (verified open); E7 superseded by WMS 1B (doc stale); E8 rescue state machine with VU residual; E9 single-use offline nonces; E10 cash-variance→recon seed same-txn; E11 condition→claim/reverse bridge (partially superseded by PR-6 TEMPERATURE_BREACH auto-raise for WMS side; driver condition path open); E12 control-tower playbooks default-off; E13 observability TF `enable_observability_resources` default false; E14 DR drill (claimed done Aug 5 — doc stale); E16 PLATFORM_ADMIN break-glass (verified absent).

**P1 (audit §8 residuals):** SMS/email/WhatsApp dunning transports; refunds + payout execution; billing fee schedule/invoices; certified EDIFACT/1C/Drummond; multi-currency AR ledger + Airwallex live FX; HW annual seasonality library + weather/POS feeds; auto-order place auto-flip at ≥80% shadow acceptance; serial tracking; multi-tenancy Phase 1 (150-250 files, ADR accepted, uncoded); platform admin console (Phase 5).

**Audit §8.8 security residuals (all admitted, none verified fixed in this pass):** Keychain accessibility flags, FLAG_SECURE, SQLCipher/SwiftData encryption, cert pinning, cleartext debug `network_security_config`, unsigned Android updater manifest / iOS OTA verifies nothing.

**CI residuals (GATE0_CI.md "Residual"):** ESLint/jsx-a11y hard gate, SwiftLint/detekt, govulncheck triage.

---

## 5. READINESS VERDICTS — what the gates themselves say

| Gate doc | Claimed status | My read |
|---|---|---|
| `docs/P0_LAUNCH_CHECKLIST.md` | 5 boxes, **all `[ ]`** — never executed as a checklist | Targets exist in Makefile (`wire-ready`, `qa-gate`, `p0-preflight`, marker gate — verified). Doc is a stub; no evidence of a completed run. |
| `docs/L1_FIELD_UNLOCK_RELEASE_CHECKLIST.md` | **All `[ ]`**; GP + OTP owner-blocked | Consistent with current_status §2 ("e2e uses cash fallback") — honest, still failing. |
| `docs/V1_STAGING_CLOSURE_CHECKLIST.md` | **3-line stub** pointing at gap-closure/ | Not a gate; content-free. |
| `docs/RETAILER_OS_PRODUCTION_GATE.md` | Automated boxes `[ ]`; pack table all "Wired"; close-out 2/3 checked; **"SSMR image … deploy pending" unchecked** | Partially superseded (Aug-4 substance-gate image deployed per current_status) but never re-checked; pack "Wired" claims carry no per-client evidence (C10). |
| `docs/SUBSTANCE_GATE.md` | **API PASS all roles 2026-08-04; every client cell DEFERRED**; marker gate PASS; theatre exceptions: touchless confidence (since marked WIRED in §7), SHOP_CLOSED inbox soft-fail, claim-window portal UX | The most honest gate in the repo — but its cited e2e log is missing (C7), and the signoff is 3 days stale relative to Aug 6-7 waves (FX picker, AS2, CoA shipped after the PASS). |
| Production flip policy (master roadmap) | "**Do not** set `PEGASUSX_ENV=production` until Phase A green (GP SUCCESS, Firebase OTP **still open**)" | Self-declared **not launch-ready** for full production; CORE-only retailer pilots permitted early (gate doc line 3). |

---

## 6. FINAL TABLE — Claim | Doc source | Code verdict | Evidence

| Claim | Doc source | Verdict | Evidence |
|---|---|---|---|
| Transactional outbox, same-txn | PLATFORM_AUDIT §1 #1 | CONFIRMED | `order/repository_spanner.go:28-38,187` |
| Optimistic concurrency on Version | PLATFORM_AUDIT §1 #3 | CONFIRMED | `order/repository_spanner.go:208-215` |
| 18-state FSM, central table | PLATFORM_AUDIT §0 line 13 | CONFIRMED | `order/service.go:52-70`; `order/state_machine.go:14-81` |
| FSM enforced everywhere | implicit | OVERSTATED (doc admits) | 4 validator call sites: `service.go:1523,2153`, `preorder_sweeper.go:168,241`; ~65 direct writes |
| Payment bypass → FISCALIZING fix | current_status Gate-0 Track A | CONFIRMED | `order/supplier_ops.go:199-219` |
| Outbox UUID + leases + Kafka dedupe | audit P0-5/P0-7 closures | CONFIRMED | `outbox/outbox.go:219`; `outbox/spanner_store.go:89-163`; `kafka/event_dedup_middleware.go:17` |
| Single-supplier runtime (1/10) | audit §0 fact 1, §8.10 | CONFIRMED | `bootstrap/bootstrap.go` ~20 seed injections; `order/service.go:352`; register mints UUIDs `supplier/service.go:442-456` |
| No ML; ortools-only | audit §0 fact 2 | CONFIRMED | `services/optimizer-core/requirements.txt` (1 dep); zero ML-dep grep hits |
| Auto-order `/2` diverted when grounded | audit §0/§8.3 | CONFIRMED (fallback retained) | `ai-worker/synthesis/engine.go:23,308` (divert), `:334` (legacy `/2`) |
| Fiscal env providers; Soliq not ready | audit §2 #1; SOLIQ_SANDBOX_READINESS | CONFIRMED | `order/fiscal_provider.go:45-80`; no sandbox creds; marker SKIPPED |
| Credit terms/AR/dunning/DelinquencyCount live behind flags | audit line 5, §8.6 | CONFIRMED | `credit/service.go:186-188`; `ar/dunning*.go`; `bootstrap.go:1252` |
| Credit scoring removed | audit §8.6; roadmap Phase A | CONFIRMED in code | 0 refs to `CREDIT_SCORE_ENFORCEMENT_ENABLED`; no `RetailerCreditScores` writers |
| Gap-closure docs: enable scoring flag | STAGING_FLAGS step 3 etc. | CONTRADICTED (stale docs) | flag absent from all Go code |
| Auto-dispatch 60s worker live | audit §5 row 9 | CONFIRMED | `runtime_workers.go:41`; `warehouse/auto_dispatch*.go:120-131` |
| EDI-lite implemented | audit §8.9 #6; PARTNER_EDI | CONFIRMED (lite) | `partner/edi/`; UNA codec; workers `runtime_workers.go:113-119` |
| AS2 wired (not Drummond) | current_status §1; PARTNER_AS2 | CONFIRMED | `partner/as2/*`; `partner/routes.go:21` |
| 1C journals + configurable CoA | audit §8.9; PARTNER_JOURNALS_1C | CONFIRMED | `partner/export_journals.go:13`; `partner/coa.go:14-16` |
| OAuth2 client_credentials | audit §8.9 #1 | CONFIRMED | `partner/oauth.go:108-120`; `oauth_jwt.go:18,135` |
| HMAC outbound webhooks + replay | audit §8.9 #4 | CONFIRMED | `partner/delivery.go:42`; routes `partner/routes.go:29-35`; worker 15s |
| GS1 GLN/SSCC/ZPL | audit §8.9 #6; GS1_LABELS | CONFIRMED | `gs1/checkdigit.go`, `gs1/zpl.go:19-30` |
| OpenAPI (partner + JWT core) | audit §8.9 #2 | CONFIRMED | `contracts/partner.openapi.yaml`; `contracts/jwt-core.openapi.yaml` |
| "No M2M integration surface at all" | audit §0 line 21 / §7 table | CONTRADICTED (by code + doc's own §8.9) | all of the above |
| WMS lots/FEFO/pick-waves/counts/cold-chain | audit §8.7; parity matrix 53-58 | CONFIRMED (flag-gated) | `stocklots/*.go`; `bootstrap.go:1148-1153`; `spanner.ddl:1134,1168,1207` |
| Retailer POS pack live | RETAILER_OS_PRODUCTION_GATE table | CONFIRMED backend+desktop; client parity UNVERIFIED | `retailerroutes/routes.go:87-101`; `retailer/pos.go`; desktop `pos/page.tsx` |
| FX multi-currency Wave 1+2+ | audit §2 #13; FX_RATES | CONFIRMED | `fxrates/`; `currency_mismatch` gates ×3; `order/currency_picker.go` |
| Forecast Croston/SES/HW wired | audit §8.1 | CONFIRMED (package) | `planning/forecast/{classify,croston,ses,holtwinters,fit}.go` |
| Safety stock v2 formula | audit §8.2 | CONFIRMED | `replenishment/safety_stock.go:25-47` |
| Admin console absent | audit §8.11/§8.10 P5 | CONFIRMED ("3 endpoints" stale → 17) | `apps/admin-portal` stub; no PLATFORM_ADMIN in `auth/`; 17 `/v1/admin/*` |
| E1 sup-demo-1 killed | hardening plan E1; roadmap Phase A | CONFIRMED | 0 matches across client sources |
| E2 per-supplier perimeter | hardening plan E2 (P0) | CONFIRMED OPEN | `retailer/proximity_service.go:24` global key; helper design-only :35 |
| E6 payload/factory overlays | hardening plan E6 (P1) | CONFIRMED STILL OPEN + comment-theatre | `payload/service.go:41-48` (env gate comment only, no code); `factory/service.go:63-65` |
| Gate-0 CI workflows (12 apps) | audit §3 line 143; GATE0_CI.md | CONFIRMED at parent root; doc link path wrong | `V.O.I.D/.github/workflows/pegasusx-ci.yml`, `pegasusx-native-mobile-build.yml`; absent under `pegasusX/.github/` |
| Dead charts removed | audit §3; parity-ledger | CONFIRMED (unmounted, files kept) | `warehouse-portal/app/analytics/page.tsx:11`; `supplier-portal/(portal)/analytics/page.tsx:27,254` |
| HPA/OSRM infra fixes | current_status Gate-0 | CONFIRMED | `deployment.yaml:123` cpu 250m; `osrm/pvc.yaml` + `deployment.yaml:66-69` |
| Idempotency scoped + SHA-256 | audit §8.0 | CONFIRMED | `idempotency/middleware.go:61-76` |
| Nil-Spanner fail-loud; PIN bcrypt | audit §8.0 | CONFIRMED | `spannerutils/retry.go:26-28`; `warehouse/ops_fleet_handlers.go:71-80` |
| iOS decoder + background location | audit P0-3/§8.8 | CONFIRMED | `APIClient.swift:76`; `Custom-Info.plist` UIBackgroundModes location |
| Kill list (ledger/, KMEANS rename) | audit §10 | PARTIALLY EXECUTED | `ledger/` gone; `fallback_phase1` rename; **enterprise/, optimizationjobs/, sku-projection literals, empty docs/adr/ remain** |
| Marker gate PASS 2026-08-04 | SUBSTANCE_GATE §10 | PLAUSIBLE but EVIDENCE FILE MISSING | `artifacts/ssmr-e2e-substance-gate-2026-08-04.log` does not exist |
| Role-row "Wired" all clients | ROLE_ROW_PARITY_MATRIX summary | OVERSTATED vs own signoff | SUBSTANCE_GATE §10: all WEB/AND/IOS cells DEFERRED |
| Scale: 411 endpoints / 73 tables / 132k Go LOC | audit §0 line 23 | STALE | measured: 731 route registrations / 155 CREATE TABLE / 213,719 Go LOC |

---

## Bottom line for the End-Product Reality Report

The documentation set is **unusually self-critical and mostly accurate at the moment each entry was written** — the PLATFORM_AUDIT'sGate-0/§8.x "WIRED" claims largely check out in code (I confirmed ~30 of them, including every marquee one: outbox leases, FSM fiscal gate repair, claims spine, dunning, partner stack, WMS waves, POS backend). The failures are of three kinds: **(1) temporal layering** — banners and status files were updated Aug 5-7 while the audit's own §0/§7 body, the gap-closure runbooks, the deployment ledger, and the credit-engine plan were not, so the docset argues with itself (C1, C2, C5, C6, C8); **(2) evidence integrity** — a dangling signoff log (C7), wrong relative workflow links (finding 26), and doc-level "Wired" labels for client cells that the project's own Substance Gate records as DEFERRED (C10); **(3) genuinely open substance** — single-supplier runtime, no platform admin, E2 global perimeter, E6 in-memory overlays, owner-blocked GP/Firebase/Soliq, and the un-executed kill-list leftovers. Launch-readiness gates are honest about being unpassed: production flip is self-blocked on payment/OTP secrets, and every client-side verification cell remains DEFERRED.

Now writing the comprehensive report.

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
