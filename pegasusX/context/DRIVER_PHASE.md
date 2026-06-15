# pegasusX DRIVER Role — Phased Execution Ledger

**Scope:** pegasusX only · **Parent plan:** `VEGETABLE_PLAN.md` §2.4  
**Last updated:** 2026-06-15 (DR-7/8/9 driver cross-client parity batch).

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
| DR9-03 | Firebase OTP | login scaffold | `FirebaseAuthHelper` | — | **Open** (deferred) |

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
2. **Cross-role next** — platform hardening or Boss-picked row per `VEGETABLE_PLAN.md` §3
