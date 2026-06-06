# PX-12 Manual QA Runbook

Operational checklist for Boss sign-off on production v1 role rows. Pair with [`PX12_ROLE_ROW_QA.md`](./PX12_ROLE_ROW_QA.md).

Capability reference: [`ROLE_ROW_PARITY_MATRIX.md`](../ROLE_ROW_PARITY_MATRIX.md).

---

## 0. Environment bootstrap

### 0.1 Automated preflight (required)

```bash
cd pegasusX
bash scripts/qa/px12_preflight.sh
# or: make px12-preflight
```

Expected terminal line: `px12-preflight-ok`.

| Gate | Pass marker |
|------|-------------|
| `parity-contract-full` | `role-row-contract-full-ok` |
| `gap-hunter-gate` | `gap-hunter-gate-ok` |
| `validate-launch-readiness` | `launch-readiness-ok` |
| `test-ssmr-infra` (optional locally) | `__SSMR_OK__` + all `PX_E2E_*_OK` lines |

### 0.2 Local SSMR stack (manual UI testing)

```bash
cd pegasusX
cp -n .env.ssmr.example .env.ssmr   # first run only
make ssmr-infra-up
# wait for health: curl -fsS http://localhost:8180/v1/health
```

| Service | URL / port |
|---------|------------|
| Backend API | `http://localhost:8180` |
| Supplier portal | `http://localhost:3000` (`make supplier-portal-dev`) |
| Warehouse portal | `http://localhost:3002` (`cd apps/warehouse-portal && npm run dev`) |
| Factory portal | `http://localhost:3003` (`cd apps/factory-portal && npm run dev`) |
| Retailer desktop | `cd apps/retailer-app-desktop && npm run dev` (Tauri or web) |

**Physical device:** set `PEGASUS_DEV_HOST` / `LAB_DEV_HOST` to your Mac LAN IP; Android emulator uses `10.0.2.2:8180`.

### 0.3 SSMR demo credentials (from `.env.ssmr.example`)

| Role | Phone | Secret | Notes |
|------|-------|--------|-------|
| SUPPLIER (portal) | `+998901000001` | `SmokeTest!234` | Register on first run if login 401 |
| RETAILER | `+998901000099` | (register flow) | Or demo `+998901000077` / PIN `1234` when seeded |
| DRIVER | `+998901000066` | PIN `1234` | `ssmr-driver-1` |
| WAREHOUSE | `+998901000088` | PIN `1234` | `ssmr-warehouse-1` |
| FACTORY | `+998901000099` | PIN `1234` | Factory admin JWT |
| PAYLOAD | `+998901110022` | PIN `33333333` | Terminal + tablet native |

Record any deviation (staging secrets, Boss GCP) in the sign-off sheet.

---

## PX12-F — DRIVER

**Clients:** `driver-app-android`, `driverappios`  
**Automated proof:** `PX_E2E_DELIVERY_OK`, `PX_E2E_TELEMETRY_OK`, `PX_E2E_DRIVER_EDGES_OK`, `PX_E2E_SHOP_CLOSED_OK`

### Manual checks

| # | Action | Pass criteria |
|---|--------|---------------|
| F1 | Login with demo phone/PIN | Lands on active route or empty-state (not auth error) |
| F2 | Open map / live ops | Map loads; no perpetual spinner |
| F3 | Post telemetry (drive simulator or manual ping) | Backend accepts; no 401/503 loop |
| F4 | Advance stop / complete delivery (if order assigned) | State transitions or clear geofence message |
| F5 | Shop-closed flow (if surfaced) | Wait-state visible; no silent failure |
| F6 | Manual route reorder (if UI exposes) | Not `501`; error is actionable if no route |
| F7 | Kill network → restore | Reconnect without force-quit; session intact |
| F8 | Cold start after login | Cached session restores |

**Known v1 delta:** Maps + WS primary; FCM is fallback path (device token registered in SSMR e2e).

---

## PX12-G — RETAILER

**Clients:** `retailer-app-desktop`, `retailer-app-android`, `retailer-app-ios`  
**Automated proof:** `PX_E2E_ORDER_OK`, `PX_E2E_PAYMENT_OK`, `PX_E2E_CATALOG_OK`, `PX_E2E_DEVICE_TOKEN_OK`

### Manual checks

| # | Action | Pass criteria |
|---|--------|---------------|
| G1 | Register or login retailer | JWT/session valid |
| G2 | Browse catalog / categories | Products load; empty state labeled if none |
| G3 | Category → suppliers drill-down | List returns (P0-06 path) |
| G4 | Add to cart → checkout (card or cash) | Unified checkout succeeds or shows gateway error |
| G5 | Order tracking | Status updates after checkout webhook path |
| G6 | Profile view/edit (mobile) | GET/PUT profile; `company` field round-trips |
| G7 | Auto-order settings (if exposed) | Scaffold loads without 404 |
| G8 | Desktop vs mobile | Same order visible on both after login (same retailer) |

**Known v1 delta:** Desktop richest UX; mobile tracking-first (intentional).

---

## PX12-H — SUPPLIER

**Clients:** `supplier-portal` (primary), `supplier-app-android`, `supplier-app-ios`  
**Automated proof:** supplier portal API sweep in SSMR e2e; `PX_E2E_PAYMENT_OK`

### Portal checks

| # | Route / action | Pass criteria |
|---|----------------|---------------|
| H1 | `/auth/login` → dashboard | KPI tiles load |
| H2 | `/(portal)/orders` | Order list; filters work |
| H3 | `/(portal)/fleet`, `/org-fleet` | Drivers/vehicles list |
| H4 | `/(portal)/inventory`, `/pricing` | Read + patch path |
| H5 | `/(portal)/treasury`, `/payments` | Financial surfaces load |
| H6 | `/(portal)/manifests` | Manifest list (may be empty) |
| H7 | `/exceptions/negotiations` | Pending negotiations list |
| H8 | WebSocket (dashboard live) | Connection indicator or live refresh |

### Native checks (P1-01 integration layer)

| # | Screen | Pass criteria |
|---|--------|---------------|
| H9 | Dashboard | Loads; ops prefetch does not error toast |
| H10 | Orders | Uses ops repository (`limit=100`) |
| H11 | Fleet | Fleet orders prefetch |
| H12 | Earnings | Ledger fallback if earnings empty |
| H13 | Billing gate | Unconfigured supplier → billing setup |

**Known v1 delta:** Native exceptions/manifests/dispatch **panels** deferred (UI freeze); data layer wired.

---

## PX12-I — WAREHOUSE

**Clients:** `warehouse-portal`, `warehouse-app-android`, `warehouse-app-ios`  
**Automated proof:** `PX_E2E_WAREHOUSE_OK` (dispatch preview + lock acquire/release)

### Portal checks (P1-03)

| # | Route | Pass criteria |
|---|-------|---------------|
| I1 | `/` dashboard | `getWarehouseOpsDashboard` KPIs |
| I2 | `/orders` | Order list with state filter |
| I3 | `/dispatch` | Dispatch preview returns `preview_ready` or labeled empty |
| I4 | `/dispatch-locks` | List; acquire + release on test order |
| I5 | `/supply-requests` | Supply request feed |
| I6 | `/treasury` | Financials or fallback ops financials |
| I7 | `/demand-forecast` | Replenishment insights → product rows |

### Native checks

| # | Screen | Pass criteria |
|---|--------|---------------|
| I8 | Login `+998901000088` / `1234` | Session + home node scoped |
| I9 | Dispatch tab | Orders/drivers segments load |
| I10 | Treasury (Android/iOS) | Ops financials fallback path |
| I11 | Demand forecast | Insights fetch |

**Known v1 delta:** Transfer/order-mutation dedicated UI panels deferred (UI freeze).

---

## PX12-J — FACTORY

**Clients:** `factory-portal`, `factory-app-android`, `factory-app-ios`  
**Automated proof:** `PX_E2E_FACTORY_OK`

### Checks

| # | Action | Pass criteria |
|---|--------|---------------|
| J1 | Factory login (portal or native) | JWT with factory scope |
| J2 | Manifest dashboard | Manifest list loads |
| J3 | Supply requests | List + accept path (if request exists) |
| J4 | Dispatch / loading bay | Operational view loads |
| J5 | Inter-hub transfer transition | State change persists (Spanner-backed P1-04) |
| J6 | Native factory app | Same manifest count as portal (stale labeled if polling) |

---

## PX12-K — PAYLOAD

**Clients:** `payload-terminal` (Expo), `payload-app-ios`, `payload-app-android`  
**Automated proof:** `PX_E2E_PAYLOAD_OK` (umbrella); sub-markers: `PX_E2E_PAYLOAD_MANIFEST_LIFECYCLE_OK`, `PX_E2E_PAYLOAD_REASSIGN_OK`, `PX_E2E_PAYLOAD_DRIVER_GATE_OK`, `PX_E2E_PAYLOAD_DEVICE_TOKEN_OK`

### Checks

| # | Action | Pass criteria |
|---|--------|---------------|
| K1 | Payloader login `+998901110022` / `33333333` | Terminal session |
| K2 | Manifest list (`GET /v1/payloader/manifests`) | Active manifest visible |
| K3 | Loading / seal lifecycle | `PX_E2E_PAYLOAD_MANIFEST_LIFECYCLE_OK`; state progression or permission message |
| K4 | Driver gate handshake | `PX_E2E_PAYLOAD_DRIVER_GATE_OK`; scan or ID bind path responds |
| K5 | Manifest reassign | `PX_E2E_PAYLOAD_REASSIGN_OK`; recommend + apply not 404/501 |
| K6 | Device token registration | `PX_E2E_PAYLOAD_DEVICE_TOKEN_OK`; `POST /v1/user/device-token` 200 as payloader |
| K7 | Tablet native (iPad/Android) | Same `/v1/payloader/manifests/*` contract as Expo terminal |

---

## Failure logging

For any FAIL, capture:

1. `trace_id` from backend JSON log or response header `X-Trace-Id`
2. Role, client, route/screen, HTTP status + body snippet
3. SSMR marker missing (if automated gate failed)

File under `pegasusX/artifacts/qa/px12-<date>/` or release notes.

---

## Sign-off

Return to [`PX12_ROLE_ROW_QA.md`](./PX12_ROLE_ROW_QA.md) and check each phase box after manual + automated evidence is recorded.
