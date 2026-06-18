# pegasusX DRIVER Role — Phased Execution Ledger

**Scope:** pegasusX only · **Parent plan:** `VEGETABLE_PLAN.md` §2.4  
**Last updated:** 2026-06-17 (DR-11 Firebase phone OTP).

## Status model

`TODO` → `IN_PROGRESS` → `WIRED` → `E2E_SSMR_GREEN` → `PROD_CANDIDATE`

---

## Phase DR-0 — Core delivery spine (pre-existing)

| ID | Feature | Android | iOS | Status |
|----|---------|---------|-----|--------|
| DR0-01 | Manifest + map + offload | ✓ | ✓ | **E2E_SSMR_GREEN** (`PX_E2E_DELIVERY_OK`) |
| DR0-02 | Telemetry | ✓ | ✓ | **E2E_SSMR_GREEN** (`PX_E2E_TELEMETRY_OK`) |
| DR0-03 | Delivery edges | ✓ | ✓ | **E2E_SSMR_GREEN** (`PX_E2E_DRIVER_EDGES_OK`) |
| DR0-04 | Assign detection | ✓ | ✓ | **E2E_SSMR_GREEN** (`PX_E2E_DRIVER_ASSIGN_DETECTION_OK`) |

---

## Phase DR-7 — Production blockers (auth refresh, mutations)

| ID | Feature | Android | iOS | Status |
|----|---------|---------|-----|--------|
| DR7-01 | Token refresh | `NetworkModule` `/v1/auth/refresh` | `APIClient.attemptRefresh` | **WIRED** (pre-existing) |
| DR7-02 | Shop-closed / bypass / credit edges | ✓ | ✓ | **WIRED** (pre-existing) |
| DR7-03 | Manifest gate + offline queue | ✓ | ✓ | **WIRED** (pre-existing) |

---

## Phase DR-8 — Parity wiring (notifications inbox)

| ID | Feature | Android | iOS | SSMR | Status |
|----|---------|---------|-----|------|--------|
| DR8-01 | Notification inbox | `DriverNotificationInboxScreen` | `DriverNotificationInboxView` | `PX_E2E_DRIVER_NOTIFICATION_INBOX_OK` | **WIRED** |
| DR8-02 | Mark all read | ✓ | ✓ | — | **WIRED** |

---

## Phase DR-9 — Client policy & platform gating

| ID | Feature | Android | iOS | SSMR | Status |
|----|---------|---------|-----|------|--------|
| DR9-01 | `GET /v1/platform/client-policy?role=DRIVER` | `HomeScreen` banner | `HomeView` banner | — | **WIRED** |
| DR9-02 | SSMR marker | — | — | `PX_E2E_DRIVER_CLIENT_POLICY_OK` (+ legacy `PX_E2E_CLIENT_POLICY_OK`) | **WIRED** |
| DR9-03 | Firebase OTP | `LoginScreen` OTP + PIN dev | `LoginView` OTP + `FirebaseAuthHelper` | `PX_E2E_DRIVER_FIREBASE_OTP_OK` (when `DRIVER_FIREBASE_TEST_ID_TOKEN` set) | **WIRED** |

---

## Phase DR-10 — Deep native UI/UX parity (iOS)

| ID | Feature | Android | iOS | Status |
|----|---------|---------|-----|--------|
| DR10-01 | Shared KPI / status / section primitives | — (iOS-only batch) | `KpiTile`, `DriverStatusBadge`, `DriverSectionHeader` | **WIRED** |
| DR10-02 | Theme tokens (`statusTint`, `readableMaxWidth`) | — | `LabTheme` + `labReadableWidth()` | **WIRED** |
| DR10-03 | Home KPI grid + section headers | pre-existing DR-9 | `HomeView` — `KpiTile` today summary, `DriverSectionHeader`, `DriverEmptyView` recent | **WIRED** |
| DR10-04 | Manifest list polish | — | `MissionListCard` — `DriverStatusBadge`, `DriverLoadingView` / `DriverEmptyView` | **WIRED** |
| DR10-05 | Rides manifest + seal gate banner | — | `RidesListView` — `DriverStatusBadge` awaiting-seal, shared state views | **WIRED** |
| DR10-06 | GPS error banner polish | — | `GPSErrorBanner` — tactical destructive chrome | **WIRED** |
| DR10-07 | Telemetry badge | — | `TelemetryBadge` (pre-existing, verified on `FleetMapView`) | **WIRED** |
| DR10-08 | PIN login tactical UI | `LoginScreen` Material | `LoginView` — PIN dots, AUTHENTICATE CTA, `DriverStateCard` error | **WIRED** |
| DR10-09 | Notification inbox polish | — | `DriverNotificationInboxView` — shared state views, `DriverStatusBadge` unread | **WIRED** |
| DR10-10 | Loading/error/empty panes | — | `DriverLoadingView` / `DriverErrorView` / `DriverEmptyView` | **WIRED** |

**UI audit vs pegasus reference:** pegasusX iOS matches pegasus `HomeView` / `FleetMapView` / `MissionListCard` workflow and is ahead on client-policy banner (DR-9), device-token registration on login, manifest seal gate, navigation cues, and live socket observers. DR-10 extracts inline KPI/section/status patterns into shared primitives aligned with payload PL-5 / warehouse WH-11 / supplier SP-7 discipline.

**Exit:** Primary driver ops screens use shared components; no new SSMR markers (UI-only parity).

---

## Phase DR-10A — Deep native UI/UX parity (Android)

| ID | Feature | Android | iOS | Status |
|----|---------|---------|-----|--------|
| DR10A-01 | Shared UI kit (`DriverUiComponents`, `DriverState`) | KPI tiles, status chips, section titles, GPS banner, connection strip | — | **WIRED** |
| DR10A-02 | Home KPI + state panes | `DriverTodayKpiCard`, `DriverLoadingState`, recent empty via `DriverStatePane` | pre-existing DR-10 | **WIRED** |
| DR10A-03 | Manifest workflow | shimmer loading, empty pane, `IconButton` refresh on header | — | **WIRED** |
| DR10A-04 | Map + delivery flow | `DriverGpsBanner` (permission + cash collection GPS errors), map empty pane | — | **WIRED** |
| DR10A-05 | Notifications + PIN auth | `DriverStatePane` / `DriverLoadingState` on inbox; compact auth failure on login | pre-existing DR-10 | **WIRED** |
| DR10A-06 | Correction screen | `DriverSectionTitle`, empathetic loading/error panes | — | **WIRED** |

**UI audit vs pegasus reference:** pegasus `driver-app-android` matches pegasusX on Home/manifest ride cards, transit control, and notification inbox scaffolding; pegasusX is ahead on `WsConnectionPill`, client-policy banner (DR-9), offline verify/cash-collection resume, and FCM registration. DR-10A aligns pegasusX Android with supplier SP-7 / payload PL-5A discipline (shared primitives, empathetic loading/empty copy, semantic status chips, GPS banner, manifest refresh).

**Exit:** Primary driver Android workflow surfaces share M3 discipline with cross-role native patterns. UI-only — no new SSMR markers.

---

## Phase DR-11 — Firebase phone OTP (deferred closure)

| ID | Feature | Android | iOS | SSMR | Status |
|----|---------|---------|-----|------|--------|
| DR11-01 | Phone OTP primary login | `FirebaseAuthHelper` + `LoginScreen` | `FirebaseAuthHelper` + `LoginViewModel` | — | **WIRED** |
| DR11-02 | PIN dev fallback | `LoginScreen` mode toggle | `LoginView` mode toggle | — | **WIRED** |
| DR11-03 | Backend `id_token` login | — (pre-existing `driver/auth_login.go`) | — | `PX_E2E_DRIVER_FIREBASE_OTP_OK` | **WIRED** |
| DR11-04 | Parity matrix DRIVER row | — | — | — | **WIRED** |

**Exit:** DR9-03 closed; driver auth matches payload PL-6A pattern (OTP-first, PIN dev fallback, emulator in DEBUG).

---

## Verification

```bash
cd pegasusX/apps/backend-go && go build ./cmd/ssmr-smokecheck/
cd pegasusX/apps/driver-app-android && ./gradlew :app:compileDebugKotlin
cd pegasusX/apps/driver-app-ios/driverappios && xcodebuild -scheme driverappios -destination 'platform=iOS Simulator,name=iPhone 17' CODE_SIGNING_ALLOWED=NO build
cd pegasusX && make test-ssmr-infra   # PX_E2E_DRIVER_* markers
```

---

## Next execution batch

1. ~~DR-7/8/9 cross-client parity batch~~ — **CLOSED** (2026-06-15)
2. ~~DR-10 iOS deep UI/UX parity~~ — **CLOSED** (2026-06-15): shared primitives, tactical PIN login, GPS banner polish — see DR-10 table
3. ~~DR-10A Android deep UI/UX parity~~ — **CLOSED** (2026-06-15): shared `DriverUiComponents` / `DriverState`, HomeScreen + manifest + map + delivery flow — see DR-10A table
4. ~~DR-11 Firebase phone OTP~~ — **CLOSED** (2026-06-17): OTP-first login on Android + iOS; DR9-03 → **WIRED**
5. **Cross-role next** — platform hardening or Boss-picked row per `VEGETABLE_PLAN.md` §3

---

## Enterprise production readiness (2026-06-18)

**Row status:** `PROD_CANDIDATE` — delivery edges idempotent, offline flush, telemetry → Redis → WS, Firebase OTP + FCM, Maps Android key required for production maps.
