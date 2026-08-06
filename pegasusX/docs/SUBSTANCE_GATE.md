# Substance Gate — E2E verification for every role, app, and platform

**Status:** Operating instruction (anti-theatre)  
**Repo:** `/Users/shakhzod/ATOMOS/pegasusX`  
**Date:** 2026-08-04  
**Companion SoTs:**
- Marker catalog: [`contracts/ssmr_ecosystem_markers.json`](../contracts/ssmr_ecosystem_markers.json)
- Role features: [`ECOSYSTEM_FEATURES_BY_ROLE.md`](./ECOSYSTEM_FEATURES_BY_ROLE.md)
- Role-row parity: [`ROLE_ROW_PARITY_MATRIX.md`](./ROLE_ROW_PARITY_MATRIX.md)
- Retail OS packs: [`RETAILER_OS_E2E_MATRIX.md`](./RETAILER_OS_E2E_MATRIX.md)
- Theatre inventory: [`../PLATFORM_AUDIT.md`](../PLATFORM_AUDIT.md) §2
- Living status: [`../context/current_status.md`](../context/current_status.md)

**Hard rule:** A feature may be labeled **Done** only if it passes the Substance Gate (SG) below. UI, DDL, workers, or dashboards that imply capability without SG proof are **Theatre** — wire, rename/hide, or delete.

---

## 1. Substance Gate algorithm (SG)

For every capability **C**:

| # | Check | Pass criterion |
|---|--------|----------------|
| 1 | **SOURCES** | Real SoT write exists (Spanner row / outbox event / Redis key with documented semantics) |
| 2 | **READERS** | At least one **decision path** reads that SoT (not only a GET/settings screen) |
| 3 | **CONTROL** | A knob (env, policy field, role action) changes an observable outcome; unit/property test proves rejection is possible |
| 4 | **PROOF** | Automated proof: required `PX_E2E_*` marker **or** package test named in this doc **or** explicit “backend-only / API-only” qualifier with curl/e2e |
| 5 | **LABEL** | Customer-facing name matches behavior (no MEIO/confidence/seasonality theatre) |

```
Done      ⇔ 1 ∧ 2 ∧ 3 ∧ 4 ∧ 5
Theatre   ⇔ interface exists ∧ ¬Done
Deferred  ⇔ tracked as Theatre/Partial with owner + exit marker/test — UI must not claim Done
```

### Confidence layers

| Layer | Meaning | Tool |
|-------|---------|------|
| L0 | Compiles | CI / local build per app |
| L1 | Control changes outcome | Go/unit / property test |
| L2 | SoT round-trip | Spanner emulator package tests |
| L3 | Multi-role lifecycle | `ssmr-smokecheck` + marker gate |
| L4 | Cloud matches code | Schema-drift, ConfigMap, SSMR smoke |

**P0 product paths require L3 green on SSMR.** Claiming Done on L0 UI alone is Theatre.

---

## 2. Role × platform matrix (what exists)

| Role | Web / desktop | Android | iOS | Notes |
|------|---------------|---------|-----|-------|
| **Retailer** | `retailer-app-desktop` (Tauri) | `retailer-app-android` | `retailer-app-ios` | Retail OS packs; stock + claims |
| **Supplier** | `supplier-portal` (+ stub `supplier-app-desktop`) | `supplier-app-android` | `supplier-app-ios` | Org/fleet, treasury, ops |
| **Warehouse** | `warehouse-portal` | `warehouse-app-android` | `warehouse-app-ios` | Dispatch, returns dock, claims approve |
| **Factory** | `factory-portal` | `factory-app-android` | `factory-app-ios` | Manifests, supply requests |
| **Driver** | — | `driver-app-android` | `driver-app-ios` | Delivery, cash, shop-closed |
| **Payload** | terminal via `payload-app-*` | `payload-app-android` | `payload-app-ios` | Seal / load / reassign |
| **Admin** | `admin-portal` (redirect stub) | — | — | Not a full product surface |

**Platform versions to verify (each client row):**

| Code | Meaning |
|------|---------|
| `API` | Backend + `ssmr-smokecheck` markers (authoritative for money/stock/claims) |
| `WEB` | Portal or Tauri desktop against live/SSMR API |
| `AND` | Android flavor(s) — supplier: `Enterprise` + `Store`; others: debug/store as shipped |
| `IOS` | iOS target against same API base URL |

**Parity rule:** Role-row feature is Done only when **API proof exists** and **each shipped client** for that role either (a) has a manual/UI checklist pass on the same build train, or (b) is explicitly deferred in [`ROLE_ROW_PARITY_MATRIX.md`](./ROLE_ROW_PARITY_MATRIX.md).

---

## 3. How to run automated E2E (API / L3)

### 3.1 Prerequisites

- SSMR (or staging) API healthy: `https://api-ssmr.pegasusx.app/healthz` → `status:ok`
- Spanner migrations applied (incl. claims + claim-window policies)
- ConfigMap: `REQUIRE_INFRA_ADAPTERS=true`, `FISCAL_PROVIDER=PEGASUS`, claim window keys set
- Worker `replicas: 1` until outbox leases land
- For GCS claim media: WI SA can `signBlob` (TokenCreator)

### 3.2 Commands

```bash
# From repo root — build + run full e2e against SSMR (port-forward or public URL)
cd apps/backend-go
go build -o /tmp/ssmr-smokecheck ./cmd/ssmr-smokecheck
# See scripts / owner runbooks for env (PUBLIC_BASE_URL, tokens, seeds)
# Example pattern used in ops:
#   /tmp/ssmr-smokecheck e2e 2>&1 | tee /tmp/ssmr-e2e.log

# Marker gate (required list must all appear in the log)
make ssmr-ecosystem-marker-gate LOG=/tmp/ssmr-e2e.log
# or equivalent script that reads contracts/ssmr_ecosystem_markers.json "required"
```

**SoT for required markers:** `contracts/ssmr_ecosystem_markers.json` → `"required"`.  
Do not invent Done without adding a marker there when the capability is customer-visible.

### 3.3 Interpreting SKIPPED alternatives

Some capabilities are flag-gated. The gate accepts **OK or SKIPPED** pairs listed under `"alternatives"` (e.g. negotiation, Soliq, auto-order).  
**SKIPPED is not Done** — it is Deferred. Do not market the feature until OK is required.

---

## 4. Cross-role spine (must stay green)

Verify this path end-to-end before role deep-dives. Markers are API-level; then spot-check one client per hop.

| Hop | Substance | Required markers (subset) | Manual UI spot-check |
|-----|-----------|---------------------------|----------------------|
| Retailer checkout → order | Reserve + order SoT | `PX_E2E_ORDER_OK`, `PX_E2E_CHECKOUT_PREVIEW_OK` | Retailer WEB/AND/IOS place order |
| Payment | Capture / cash path | `PX_E2E_PAYMENT_OK` (+ card SUCCESS or cash fallback alt) | Payment sheet clears |
| Warehouse dispatch | Manifest LOADED | `PX_E2E_WAREHOUSE_DISPATCH_EXECUTE_OK`, `PX_E2E_WAREHOUSE_OK` | WH portal/Android dispatch board |
| Factory supply | Supply → factory manifest | `PX_E2E_FACTORY_SUPPLY_REQUEST_OK`, `PX_E2E_FACTORY_MANIFEST_LIFECYCLE_OK` | Factory portal/app |
| Payload seal | Seal / driver gate | `PX_E2E_PAYLOAD_MANIFEST_LIFECYCLE_OK`, `PX_E2E_PAYLOAD_DRIVER_GATE_OK`, `PX_E2E_PAYLOAD_SEAL_FLOWS_OK` | Payload terminal/app |
| Driver delivery | Geofence + complete | `PX_E2E_DELIVERY_OK`, `PX_E2E_DRIVER_EDGES_OK` | Driver AND/IOS route → complete |
| Realtime | WS + inbox | `PX_E2E_CROSS_ROLE_WS_OK`, `PX_E2E_NOTIFICATION_INBOX_OK` | Two roles see update without refresh spam |
| Telemetry | Live map feed | `PX_E2E_TELEMETRY_OK`, `PX_E2E_WAREHOUSE_FLEET_LIVE_MAP_OK` | WH live map moves |

---

## 5. Per-role E2E verification

For each role: **API markers first**, then **WEB / AND / IOS** checklist. Mark each cell `PASS` / `FAIL` / `N/A` / `DEFERRED`.

### 5.1 Retailer

**Apps:** desktop · Android · iOS  

| Capability | API proof | WEB | AND | IOS | Notes |
|------------|-----------|-----|-----|-----|-------|
| Login / session | (auth in e2e setup) | ☐ | ☐ | ☐ | Firebase OTP ops-owned |
| Catalog browse + cart | `PX_E2E_ORDER_OK` path | ☐ | ☐ | ☐ | |
| Checkout + fee preview | `PX_E2E_CHECKOUT_PREVIEW_OK`, `PX_E2E_DELIVERY_FEE_PREVIEW_OK` | ☐ | ☐ | ☐ | |
| Payment (cash/card) | `PX_E2E_PAYMENT_OK` + card/cash alt | ☐ | ☐ | ☐ | Card SUCCESS often ops-blocked |
| Manual preorder | `PX_E2E_MANUAL_PREORDER_OK` | ☐ | ☐ | ☐ | |
| Delivery proposal accept/reject | `PX_E2E_DELIVERY_PROPOSAL_*` | ☐ | ☐ | ☐ | |
| Credit profiles (status/limit) | `PX_E2E_CREDIT_PROFILES_OK` | ☐ | ☐ | ☐ | Scoring removed — no risk desk |
| Store stock receive | `PX_E2E_STORE_STOCK_CLAIM_HOLD_OK` (with claims) | ☐ | ☐ | ☐ | Pack STORE_STOCK |
| File claim + photo (GCS) | `PX_E2E_CLAIMS_FILE_OK`, `PX_E2E_CLAIM_MEDIA_GCS_OK`, `PX_E2E_CLAIMS_MEDIA_TICKET_OK` | ☐ | ☐ | ☐ | Fail if placehold.co |
| Claim eligibility countdown | `PX_E2E_CLAIM_ELIGIBILITY_OK`, `PX_E2E_CLAIM_WINDOW_SNAPSHOT_OK` | ☐ | ☐ | ☐ | CTA hides when expired |
| Concealed → quarantine | `PX_E2E_CLAIMS_CONCEALED_OK` | ☐ | ☐ | ☐ | |
| Claim idempotency | `PX_E2E_CLAIMS_IDEMPOTENCY_OK` | ☐ | ☐ | ☐ | Same `claim-file:` key |
| Stock “Request return” UX | (uses file-claim markers) | ☐ | ☐ | ☐ | G1 — picker → FileClaim |
| Multi-org auth | `PX_E2E_MULTI_ORG_AUTH_OK` + alts | ☐ | ☐ | ☐ | Flag-gated |
| Retail OS packs (POS/shifts/…) | Package tests + [`RETAILER_OS_E2E_MATRIX.md`](./RETAILER_OS_E2E_MATRIX.md) | ☐ | ☐ | ☐ | Not all in marker `required` yet |
| WS / inbox | `PX_E2E_NOTIFICATION_INBOX_OK`, `PX_E2E_DESKTOP_WS_OK` | ☐ | ☐ | ☐ | |

**Retailer UI smoke (all three clients):**

1. Login → see real empty or real data (no demo injection).  
2. Place order → pay → see status progress.  
3. After COMPLETED: Stock → Request return → eligibility countdown visible → file claim with photo.  
4. Claim appears in history; stock hold/quarantine if damage types.

---

### 5.2 Supplier

**Apps:** portal · Android · iOS  

| Capability | API proof | WEB | AND | IOS | Notes |
|------------|-----------|-----|-----|-----|-------|
| Topology edit | `PX_E2E_TOPOLOGY_EDIT_OK` | ☐ | ☐ | ☐ | |
| Org & fleet CRUD | `PX_E2E_ORG_FLEET_OK` | ☐ | ☐ | ☐ | Android compile Gate-0 |
| Import wizard | `PX_E2E_SUPPLIER_IMPORT_WIZARD_OK` | ☐ | ☐ | ☐ | |
| Catalog | `PX_E2E_CATALOG_OK` | ☐ | ☐ | ☐ | |
| Replenishment / colocate | `PX_E2E_REPLENISH_OK`, `PX_E2E_REPLENISH_COLOCATE_OK` | ☐ | ☐ | ☐ | Touchless policy still Theatre if unused |
| Supplier ops umbrella | `PX_E2E_SUPPLIER_OPERATIONS_OK` | ☐ | ☐ | ☐ | |
| Pricing override | `PX_E2E_RETAILER_PRICING_OVERRIDE_OK` | ☐ | ☐ | ☐ | |
| Claim approve / chargeback | `PX_E2E_CLAIMS_FILE_OK` path + chargeback list | ☐ | ☐ | ☐ | |
| Return policy API | curl `GET/PUT /v1/supplier/return-policy` | ☐ | N/A | N/A | G3 backend; portal UX open |
| Credit notes | `PX_E2E_CREDIT_NOTE_*` (optional/gap) | ☐ | ☐ | ☐ | |
| Cash recon | `PX_E2E_CASH_RECON_OK` (optional) | ☐ | ☐ | ☐ | |
| WS fanout | `PX_E2E_CROSS_ROLE_WS_OK` | ☐ | ☐ | ☐ | |

**Supplier UI smoke:**

1. Portal: topology + org fleet + import one SKU.  
2. Android/iOS: same org fleet list loads (no crash on Org & fleet).  
3. After retailer claim: approve → ledger/chargeback visible.  
4. Settings that claim “auto-approve above X%” — backend **WIRED** (`TouchlessEligible` reads `MinConfidenceScore`); still verify portal/settings UX shows the live threshold.

---

### 5.3 Warehouse

**Apps:** portal · Android · iOS  

| Capability | API proof | WEB | AND | IOS | Notes |
|------------|-----------|-----|-----|-----|-------|
| Ops umbrella | `PX_E2E_WAREHOUSE_OK` | ☐ | ☐ | ☐ | |
| Dispatch settings | `PX_E2E_WAREHOUSE_DISPATCH_SETTINGS_OK` | ☐ | ☐ | ☐ | |
| Dispatch execute | `PX_E2E_WAREHOUSE_DISPATCH_EXECUTE_OK` | ☐ | ☐ | ☐ | |
| Fleet mgmt | `PX_E2E_WAREHOUSE_FLEET_MGMT_OK` | ☐ | ☐ | ☐ | |
| Live map | `PX_E2E_WAREHOUSE_FLEET_LIVE_MAP_OK` | ☐ | ☐ | ☐ | |
| Order mutation | `PX_E2E_WAREHOUSE_ORDER_MUTATION_OK` | ☐ | ☐ | ☐ | |
| Stock / ops policy | `PX_E2E_WAREHOUSE_STOCK_POLICY_OK`, `PX_E2E_WAREHOUSE_OPS_POLICY_OK` | ☐ | ☐ | ☐ | |
| Broadcast | `PX_E2E_WAREHOUSE_BROADCAST_OPS_OK` | ☐ | ☐ | ☐ | |
| Capacity | `PX_E2E_DISPATCH_CAPACITY_OK` | ☐ | ☐ | ☐ | |
| Return gate receive | `PX_E2E_RETURN_GATE_RECEIVE_OK` | ☐ | ☐ | ☐ | |
| Reverse logistics | `PX_E2E_CLAIMS_REVERSE_OK`, `PX_E2E_REVERSE_LOGISTICS_OK` | ☐ | ☐ | ☐ | |
| Claim approve (admin) | claims approve routes | ☐ | ☐ | ☐ | |
| Return policy API | `GET/PUT /v1/warehouse/return-policy` | ☐ | N/A | N/A | WH Admin; no mobile settings yet |

**Warehouse UI smoke:** dispatch → execute → live map; inbound reverse after concealed claim.

---

### 5.4 Factory

**Apps:** portal · Android · iOS  

| Capability | API proof | WEB | AND | IOS | Notes |
|------------|-----------|-----|-----|-----|-------|
| Factory umbrella | `PX_E2E_FACTORY_OK` | ☐ | ☐ | ☐ | |
| Supply request | `PX_E2E_FACTORY_SUPPLY_REQUEST_OK` | ☐ | ☐ | ☐ | |
| Manifest lifecycle | `PX_E2E_FACTORY_MANIFEST_LIFECYCLE_OK` | ☐ | ☐ | ☐ | |
| Exception resolve | `PX_E2E_EXCEPTION_RESOLVE_OK` (optional) | ☐ | ☐ | ☐ | |
| Firebase OTP | `PX_E2E_FACTORY_FIREBASE_OTP_OK` (optional) | ☐ | ☐ | ☐ | Ops-owned SMS |

**Factory UI smoke:** create/fulfill supply request → manifest progresses; staff can resolve exception.

---

### 5.5 Driver

**Apps:** Android · iOS  

| Capability | API proof | AND | IOS | Notes |
|------------|-----------|-----|-----|-------|
| Assign / detect | `PX_E2E_DRIVER_ASSIGN_DETECTION_OK` | ☐ | ☐ | |
| Delivery complete | `PX_E2E_DELIVERY_OK` | ☐ | ☐ | iOS: no convertFromSnakeCase (Gate-0) |
| Driver edges | `PX_E2E_DRIVER_EDGES_OK` | ☐ | ☐ | |
| Shop closed | `PX_E2E_SHOP_CLOSED_OK`, `PX_E2E_SHOP_CLOSED_CANCEL_RELEASE_OK` | ☐ | ☐ | Inventory release |
| Telemetry | `PX_E2E_TELEMETRY_OK` | ☐ | ☐ | |
| Firebase OTP | `PX_E2E_DRIVER_FIREBASE_OTP_OK` (optional) | ☐ | ☐ | |
| Offline queue | Manual + unit | ☐ | ☐ | P0-4 iOS enqueue still Theatre risk |

**Driver UI smoke:**

1. Login → assigned orders decode (iOS critical).  
2. Navigate → arrive → QR/handshake → cash/card → complete.  
3. Shop-closed path releases stock (API marker).  
4. Kill network mid-action: Android offline queue; iOS must **not** fake success (audit P0-4).

---

### 5.6 Payload

**Apps:** Android · iOS (terminal)  

| Capability | API proof | AND | IOS | Notes |
|------------|-----------|-----|-----|-------|
| Payload umbrella | `PX_E2E_PAYLOAD_OK` | ☐ | ☐ | |
| Manifest lifecycle | `PX_E2E_PAYLOAD_MANIFEST_LIFECYCLE_OK` | ☐ | ☐ | |
| Driver gate | `PX_E2E_PAYLOAD_DRIVER_GATE_OK` | ☐ | ☐ | |
| Seal flows | `PX_E2E_PAYLOAD_SEAL_FLOWS_OK` | ☐ | ☐ | |
| Reassign | `PX_E2E_REASSIGN_FLOWS_OK` | ☐ | ☐ | |
| Firebase OTP | `PX_E2E_PAYLOAD_FIREBASE_OTP_OK` (optional) | ☐ | ☐ | |
| Decode / API client | — | ☐ | ☐ | iOS Gate-0 snake_case fix |

**Payload UI smoke:** load → seal → handoff to driver; reassign when needed.

---

## 6. Platform-version checklist (every client build)

Run before calling a mobile/desktop release “verified”:

| # | Check | WEB | AND | IOS |
|---|--------|-----|-----|-----|
| P1 | Points at intended API (`PUBLIC_BASE_URL` / build flavor) | ☐ | ☐ | ☐ |
| P2 | Auth: login + refresh; no demo-key Firebase in release | ☐ | ☐ | ☐ |
| P3 | Empty states honest (no fake lists) | ☐ | ☐ | ☐ |
| P4 | Idempotent mutation: retry does not double-create | ☐ | ☐ | ☐ |
| P5 | Realtime: WS reconnect or polling recovers | ☐ | ☐ | ☐ |
| P6 | Role-critical path from §5 for that role | ☐ | ☐ | ☐ |
| P7 | No customer copy claiming Theatre features (confidence, seasonality, Soliq “live”, etc.) | ☐ | ☐ | ☐ |

Android flavors: where `Enterprise` + `Store` exist (supplier), both must compile; smoke the flavor you ship.

---

## 7. Theatre — do **not** mark Done

Until SG passes, these stay **Theatre / Partial** (see `PLATFORM_AUDIT.md` §2):

| Item | Why it fails SG | Exit |
|------|-----------------|------|
| ~~Touchless `MinConfidenceScore`~~ | **WIRED** — `TouchlessEligible` + tests | — |
| ~~AI confidence gate always-pass~~ | **WIRED** — base 0.15 / rejectable + tests | — |
| Promo elasticity / closed-loop | **Partial** — elasticity input + line `promotion_id` attribution | Demand-model curves still open; keep `sandbox_only` |
| Seasonality multipliers | READERS missing | Multiply in forecast qty or remove UI |
| Weather/POS “signals” | SOURCES fake constants | Real feed or delete |
| Forecast MAPE in portal | **WIRED** server WAPE/bias/TS (`ForecastAccuracyDaily`) | Enable `FORECAST_ACCURACY_ENABLED` + migration |
| Billing meter | WIRED (schema + amount_minor decode) | Residual: fee schedule / invoices; e2e meter event |
| Soliq legal OFD | PROOF skipped | `PX_E2E_SOLIQ_SANDBOX_OK` required |
| Cold chain | **Partial** — solver flags + WMS ingest | Always-on prod sensor fleet / full excursion automation |
| i18n catalogs | **Desktop portals UI: Wired (draft translations)**; mobile still Partial / unwired | Do not claim full-platform or certified-linguistic Done |
| Mobile shared kit / offline (§8.8) | **Partial→Wired** — kit packages + capture-time coords + Room migrations; scan UX residual | See [`MOBILE_SHARED_KIT.md`](./MOBILE_SHARED_KIT.md); PR-7 scan still open |
| Supplier/WH claim-window **portal UX** | UI missing (API Done) | Portal/settings screens |
| Quantity negotiations | Flag off / SKIPPED | Product enable + OK marker |

---

## 8. Shipping a new feature (Substance Card)

Paste into the PR description:

```text
### Substance Card
- Capability:
- Customer-visible? Y/N
- SoT write: (table / event)
- Decision reader: (package.func)
- Control knob → expected change:
- Proof: PX_E2E_… | go test ./… | backend-only (curl)
- Clients: WEB / AND / IOS / N/A — parity deferred? (link)
- If not Done: Theatre/Deferred issue ID — UI copy does not claim Done
```

**Merge rules:**

1. Customer-visible Done ⇒ marker in `ssmr_ecosystem_markers.json` `required` (or documented alternative pair).  
2. New Spanner objects ⇒ migration **and** `schema/spanner.ddl`.  
3. Policy fields ⇒ unit test that extreme setting changes decisions.  
4. Roadmap/status “Done” must match SG; prefer “Backend Done / UX open” when accurate (G3 pattern).

---

## 9. Suggested verification cadence

| Cadence | What |
|---------|------|
| Every PR (touched role) | Substance Card + L1 tests |
| Nightly / pre-release | Full `ssmr-smokecheck e2e` + marker gate |
| Weekly | One role deep UI pass (rotate Retailer → Driver → WH → …) |
| After cloud apply | Schema-drift + claim/GCS markers |
| Before production flip | All §4 spine + `ValidateProductionProfile` + GP card SUCCESS |

---

## 10. Sign-off template

**Latest API fill (2026-08-04):** [`artifacts/SUBSTANCE_GATE_API_SIGNOFF_2026-08-04.md`](../artifacts/SUBSTANCE_GATE_API_SIGNOFF_2026-08-04.md)

```text
Environment: SSMR
API build / image: asia-south1-docker.pkg.dev/pegasus-503013/pegasusx-ssmr-images/backend-go:ssmr-substance-gate-a66868b8-084112
E2E log: artifacts/ssmr-e2e-substance-gate-2026-08-04.log  Marker gate: PASS
Date: 2026-08-04  Operator: agent (backend-first)

Role          API   WEB        AND        IOS
Retailer      PASS  DEFERRED   DEFERRED   DEFERRED
Supplier      PASS  DEFERRED   DEFERRED   DEFERRED
Warehouse     PASS  DEFERRED   DEFERRED   DEFERRED
Factory       PASS  DEFERRED   DEFERRED   DEFERRED
Driver        PASS  N/A        DEFERRED   DEFERRED
Payload       PASS  N/A        DEFERRED   DEFERRED

Theatre exceptions still open: touchless confidence; SHOP_CLOSED inbox soft-fail; claim-window portal UX
Blockers (ops): GP card SUCCESS (password); Firebase SMS/device OTP; TokenCreator OK this pass (claim GCS green)
```

Client-policy smoke (all roles × web/android/ios): HTTP 200. Interactive UI walks remain DEFERRED until a dedicated Phase 2 session.

---

## 11. Quick links

| Need | Path |
|------|------|
| Required markers | [`contracts/ssmr_ecosystem_markers.json`](../contracts/ssmr_ecosystem_markers.json) |
| Claims e2e source | [`apps/backend-go/cmd/ssmr-smokecheck/e2e_claims.go`](../apps/backend-go/cmd/ssmr-smokecheck/e2e_claims.go) |
| Claims role-row | [`CLAIM_ROLE_ROW.md`](./CLAIM_ROLE_ROW.md) |
| Master sequence | [`PEGASUSX_MASTER_ROADMAP.md`](./PEGASUSX_MASTER_ROADMAP.md) |
| Audit theatre | [`../PLATFORM_AUDIT.md`](../PLATFORM_AUDIT.md) §2 |

---

*End of Substance Gate. Prefer executable markers over prose; prefer honest Partial over fake Done.*
