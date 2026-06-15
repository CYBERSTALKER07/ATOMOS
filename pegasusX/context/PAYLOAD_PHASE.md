# pegasusX PAYLOAD Role — Phased Execution Ledger

**Scope:** pegasusX only · **Parent plan:** `VEGETABLE_PLAN.md` §2.6  
**Last updated:** 2026-06-15 (PL-5P terminal deep component parity).

## Status model

`TODO` → `IN_PROGRESS` → `WIRED` → `E2E_SSMR_GREEN` → `PROD_CANDIDATE`

---

## Phase PL-0 — Manifest lifecycle core (pre-existing)

| ID | Feature | Backend | Terminal | Android | iOS | Status |
|----|---------|---------|----------|---------|-----|--------|
| PL0-01 | Payloader login | `POST /v1/auth/payloader/login` | `App.tsx` | `LoginScreen` | `LoginView` | **WIRED** |
| PL0-02 | Trucks + orders | `GET /v1/payloader/trucks`, `.../orders` | wired | `PayloadApi` | `APIClient` | **WIRED** |
| PL0-03 | Manifest list/detail | `GET /v1/payloader/manifests*` | supplier + payloader paths | wired | wired | **WIRED** |
| PL0-04 | Start loading | `POST .../start-loading` | `PayloadTerminalApi` | wired | wired | **WIRED** |
| PL0-05 | Inject order | `POST .../inject-order` | wired | wired | wired | **WIRED** |
| PL0-06 | Per-order seal | `POST /v1/payload/seal` | wired | wired | wired | **WIRED** |
| PL0-07 | Manifest seal batch | `POST .../manifests/seal-completed` | wired | wired | wired | **WIRED** |
| PL0-08 | Exceptions | `POST /v1/payload/manifest-exception` + list | wired | wired | wired | **WIRED** |
| PL0-09 | Reassign recommend+apply | `POST .../recommend-reassign`, `.../reassign-order` | wired | wired | wired | **WIRED** |
| PL0-10 | Fleet reassign (UI path) | `POST /v1/fleet/reassign` | Expo re-dispatch modal | wired | wired | **WIRED** |
| PL0-11 | Driver gate + depart | manifest gate routes | — | — | — | **E2E_SSMR_GREEN** (`PX_E2E_PAYLOAD_DRIVER_GATE_OK`, `PX_E2E_PAYLOAD_DRIVER_DEPART_OK`) |
| PL0-12 | SSMR umbrella | smokecheck | — | — | — | **E2E_SSMR_GREEN** (`PX_E2E_PAYLOAD_OK`, `PX_E2E_PAYLOAD_SEAL_FLOWS_OK`, `PX_E2E_PAYLOAD_MANIFEST_LIFECYCLE_OK`) |

**Exit:** Full manifest lifecycle + seal flows green in SSMR; all three payload clients call verified backend paths.

---

## Phase PL-1 — Production blockers (auth, path correctness)

| ID | Feature | Backend | Terminal | Android | iOS | Status |
|----|---------|---------|----------|---------|-----|--------|
| PL1-01 | API path audit | `payloaderroutes/routes.go` | `/v1/payloader/*` + supplier manifest aliases | `PayloadApi.kt` | `APIClient.swift` | **WIRED** (no drift found) |
| PL1-02 | Token refresh | `POST /v1/auth/payloader/refresh` | `authSession.ts` 401 retry | `TokenRefreshAuthenticator` | `APIClient.attemptRefresh` | **WIRED** (`PX_E2E_PAYLOAD_AUTH_REFRESH_OK`) |
| PL1-03 | Mock/hardcoded data | — | live API only | live API | live API | **WIRED** |
| PL1-04 | Missing-items post-seal | `POST /v1/delivery/missing-items` | double-check screen | wired | wired | **WIRED** |
| PL1-05 | Device token | `POST /v1/user/device-token` | — (Expo OTA separate) | FCM service | APNs manager | **WIRED** (`PX_E2E_PAYLOAD_DEVICE_TOKEN_OK`) |

**Exit:** No P0 path bugs; honest session expiry via re-login; mutations idempotent with `Idempotency-Key`.

---

## Phase PL-2 — Parity wiring (notifications, exceptions inbox, offline queue)

| ID | Feature | Backend | Terminal | Android | iOS | Status |
|----|---------|---------|----------|---------|-----|--------|
| PL2-01 | Notification inbox | `GET /v1/user/notifications` | modal panel + WS live | `NotificationsSheet` | `NotificationsSheet` | **WIRED** |
| PL2-02 | Mark read / mark all | `POST /v1/user/notifications/read` | wired | wired | wired | **WIRED** |
| PL2-03 | Manifest exceptions panel | `GET /v1/payloader/manifest-exceptions` | modal panel | bottom sheet | sheet | **WIRED** |
| PL2-04 | WS PAYLOAD_SYNC refresh | PayloaderHub | WS handler | `PayloadWebSocket` | `WebSocketClient` | **WIRED** |
| PL2-05 | Offline action queue | idempotent replay | SecureStore queue | `SecureStore` + flush | `OfflineQueue` | **WIRED** |
| PL2-06 | Multi-truck batch seal | `seal-completed` aggregate | wired | `finalizeBatchSeal` | `finalizeBatchSeal` | **WIRED** |

**Exit:** Native tablet apps no longer hand off notifications; unified inbox API with graceful empty states.

---

## Phase PL-3 — Client policy & platform gating

| ID | Feature | Backend | Terminal | Android | iOS | Status |
|----|---------|---------|----------|---------|-----|--------|
| PL3-01 | Client version policy | `GET /v1/platform/client-policy?role=PAYLOAD` | banner in `App.tsx` | `ClientPolicyBanner` | `ClientPolicyBanner` | **WIRED** |
| PL3-02 | SSMR marker | smokecheck | — | — | — | **WIRED** (`PX_E2E_PAYLOAD_CLIENT_POLICY_OK`) |
| PL3-03 | Firebase OTP / custom token | login `firebase_token` when Admin Auth configured | stored; PIN-only UX | `FirebaseAuthHelper` exchange | stub + graceful | **SCAFFOLD** (PIN production path; phone OTP deferred) |
| PL3-04 | Expo OTA updates | — | `expo-updates` | APK `AutoUpdater` | `AutoUpdater` | **WIRED** (platform-specific; separate from policy API) |

**Exit:** Outdated/force-update surfaces show honest banners on all payload clients; SSMR asserts PAYLOAD role policy tuple.

---

## Phase PL-4 — Home UX parity (KPI headers, connection chrome, batch seal)

| ID | Feature | Backend | Terminal | Android | iOS | Status |
|----|---------|---------|----------|---------|-----|--------|
| PL4-01 | Manifest KPI header grid | — | `ManifestKpiGrid` tactical tiles (PL-5P) | `ManifestKpiGrid` tactical tiles | `ManifestWorkflow` tactical cards | **WIRED** |
| PL4-02 | Connection / queue status chrome | WS hub | sidebar live-sync dot + queue hint | `OnlineDot` in top bar | `OnlineDot` in sidebar toolbar | **WIRED** |
| PL4-03 | Multi-truck batch seal UI | `seal-completed` | sidebar batch action card | `TruckListPane` batch card | `TruckSidebar` batch section | **WIRED** |
| PL4-04 | Post-seal missing-items action | `POST /v1/delivery/missing-items` | countdown screen | `PostSealCountdownCard` | `PostSealCountdownView` | **WIRED** |
| PL4-05 | Manifest exceptions inbox | `GET /v1/payloader/manifest-exceptions` | modal panel | `ManifestExceptionsSheet` | `ManifestExceptionsSheet` | **WIRED** (PL-2; verified PL-4) |
| PL4-06 | Client-policy banner placement | `GET /v1/platform/client-policy` | top of manifest + truck picker | below top bar | above `NavigationSplitView` | **WIRED** (PL-3; verified PL-4) |
| PL4-07 | HomeScreen structure | — | monolithic `App.tsx` (pegasus pattern) | `ManifestKpiGrid` extracted | `ManifestWorkflow` sections | **WIRED** |

**Exit:** All three payload clients show consistent KPI/action/state panes for manifest workflow; batch seal + connection chrome synced; pegasusX additive features (exceptions, policy, missing-items) present on every surface.

---

## Phase PL-5P — Deep payload-terminal UI/UX (terminal)

| ID | Feature | pegasus ref | pegasusX terminal | Status |
|----|---------|-------------|-------------------|--------|
| PL5P-01 | Sidebar connection strip | — (pegasusX PL-4) | `ConnectionStrip.tsx` live-sync dot + queue hint | **WIRED** |
| PL5P-02 | Manifest KPI header grid | volume bar only | `ManifestKpiGrid.tsx` state pill + volume/stops/zone tiles | **WIRED** |
| PL5P-03 | Status badges (state/cleared/escalated) | plain text labels | `StatusBadge.tsx` on orders, exceptions, re-dispatch | **WIRED** |
| PL5P-04 | Workflow section headers | inline copy | `WorkflowSectionHeader.tsx` batch seal, checklist, seal/inject | **WIRED** |
| PL5P-05 | Skeleton loaders | `PayloadStatePanel` only | `SkeletonPulse.tsx` / `SkeletonList` exceptions inbox | **WIRED** |
| PL5P-06 | Right-pane empty state | plain text hint | `PayloadStatePanel` manifest variant | **WIRED** |
| PL5P-07 | Exceptions inbox polish | — (pegasusX PL-2) | badge count on icon, reason/escalated chips, fixed invalid variant | **WIRED** |
| PL5P-08 | KPI refresh after inject/exception | — | `fetchTruckManifest` after inject + exception | **WIRED** |

**PL-5P audit gaps (intentional / blocked):** Monolithic `App.tsx` retained (pegasus pattern); re-dispatch apply still uses `fleet/reassign` (durable `payloader/reassign-order` in `api.ts` for programmatic use); Firebase phone OTP UI deferred; barcode scanning ecosystem-wide deferred.

**Exit:** Component-level desk tokens, KPI tiles, section chrome, skeleton loaders, and status badges on terminal sidebar + manifest workflow panes. UI-only — no new SSMR.

---

## Phase PL-5A — Deep native UI/UX parity (Android)

| ID | Feature | Backend | Terminal | Android | iOS | Status |
|----|---------|---------|----------|---------|-----|--------|
| PL5A-01 | Shared UI kit (`PayloadUiComponents`, `PayloadState`) | — | — | KPI tiles, status chips, section titles, connection status | `PayloadStateView` (pre-existing) | **WIRED** |
| PL5A-02 | Manifest KPI grid refactor | — | — | `ManifestKpiGrid` → shared `PayloadKpiTile` / `PayloadStatusChip` | tactical cards | **WIRED** |
| PL5A-03 | Home workflow state panes | — | — | truck/manifest/order loading + empty via `PayloadStatePane` / `PayloadLoadingState` | `PayloadStateView` variants | **WIRED** |
| PL5A-04 | Exceptions + notifications sheets | `GET /v1/payloader/manifest-exceptions` | modal panel | status chips + empathetic empty/loading | sheet | **WIRED** |
| PL5A-05 | PIN auth error surface | `POST /v1/auth/payloader/login` | login form | compact `PayloadStatePane` AuthFailure | `PayloadStateView` warning | **WIRED** |
| PL5A-06 | Connection chrome | WS hub | sidebar live-sync dot | `PayloadConnectionStatus` in top bar | `OnlineDot` in sidebar | **WIRED** (PL-4; verified PL-5A) |

**UI audit vs pegasus reference:** pegasus `payload-app-android` matches pegasusX on master-detail scaffold and loading workflow; pegasusX is ahead on `ManifestKpiGrid`, batch seal, exceptions inbox, client-policy banner, post-seal missing-items, and notification inbox. PL-5A aligns pegasusX Android with supplier SP-7 / warehouse WH-11A discipline (shared primitives, empathetic loading/empty copy, semantic status chips).

**Exit:** Primary payload Android workflow surfaces share M3 discipline with cross-role native patterns. UI-only — no new SSMR markers.

---

## Phase PL-5 — Deep native UI/UX parity (iOS)

| ID | Feature | Android | iOS | Status |
|----|---------|---------|-----|--------|
| PL5-01 | Shared KPI / status / section primitives | `ManifestKpiGrid` (pre-existing PL-4) | `KpiTile`, `PayloadStatusBadge`, `PayloadSectionHeader` | **WIRED** |
| PL5-02 | Manifest KPI grid extraction | pre-existing PL-4 | `ManifestKpiGrid` — adaptive volume/stops/zone + state badge | **WIRED** |
| PL5-03 | Connection dot chrome | `OnlineDot` in top bar | `OnlineDot` extracted — monospaced LIVE/OFFLINE label | **WIRED** |
| PL5-04 | Order checklist + batch seal headers | pre-existing | `PayloadSectionHeader` on checklist + batch finalize | **WIRED** |
| PL5-05 | Exceptions inbox polish | `ManifestExceptionsSheet` | status badges, `PayloadLoadingView`, tactical row copy | **WIRED** |
| PL5-06 | PIN login tactical UI | `LoginScreen` Material | `LoginView` — TermTheme card, PIN dots, AUTHENTICATE CTA | **WIRED** |
| PL5-07 | Loading/error/empty panes | `SupplierStatePane` pattern | `PayloadLoadingView` / `PayloadErrorView` / `PayloadEmptyView` | **WIRED** |
| PL5-08 | Seal/inject/re-dispatch sheets | pre-existing | unchanged tactical sheets (pegasus parity + missing-items) | **WIRED** |

**UI audit vs pegasus reference:** pegasusX iOS matches pegasus `HomeView` workflow (master-detail, seal/inject/exception sheets, OnlineDot, notifications) and adds PL-2/4 features (exceptions panel, client-policy banner, batch seal, missing-items). PL-5 extracts inline KPI cards into shared primitives aligned with Android `ManifestKpiGrid` and warehouse WH-11 / supplier SP-7 discipline.

**Exit:** Primary payload iOS screens use shared components; no new SSMR markers (UI-only parity).

---

## Verification

```bash
cd pegasusX/apps/backend-go && go build ./cmd/ssmr-smokecheck/
cd pegasusX && make test-ssmr-infra   # PX_E2E_PAYLOAD_* markers
cd pegasusX/apps/payload-app-android && ./gradlew compileDebugKotlin
# iOS: xcodegen generate && xcodebuild -scheme payload-app-ios -destination 'generic/platform=iOS' build
```

---

## Known remaining gaps

- Firebase phone OTP UI not exposed — PIN login is the production path; custom-token exchange wired on Android when backend mints tokens (`FIREBASE_CREDENTIALS_PATH` or emulator).
- Re-dispatch UI uses `fleet/reassign` on Expo; durable apply path `payloader/reassign-order` wired on API surfaces for programmatic use.
- Barcode scanning deferred ecosystem-wide (see payload iOS `project.yml` comment).

---

## Next execution batch

1. ~~PL-0 manifest lifecycle~~ — pre-existing **E2E_SSMR_GREEN**
2. ~~PL-1 production blockers audit~~ — **CLOSED** (2026-06-15)
3. ~~PL-2 parity wiring~~ — **CLOSED** (notifications/exceptions/offline pre-existing; verified)
4. ~~PL-3 client policy~~ — **CLOSED** (2026-06-15)
5. ~~PL-4 home UX parity~~ — **CLOSED** (2026-06-15): KPI grid Android, terminal batch seal + connection chrome
6. ~~PL-5A Android deep UI/UX parity~~ — **CLOSED** (2026-06-15): shared `PayloadUiComponents` / `PayloadState`, HomeScreen + LoginScreen state panes
7. ~~PL-5P terminal deep UI/UX (component-level)~~ — **CLOSED** (2026-06-15): `ManifestKpiGrid`, `StatusBadge`, `ConnectionStrip`, workflow section chrome — see PL-5P table
8. ~~PL-5 iOS deep UI/UX parity~~ — **CLOSED** (2026-06-15): shared primitives, `ManifestKpiGrid`, tactical PIN login — see PL-5 table
9. **Cross-role next** — Boss-picked role row per `VEGETABLE_PLAN.md` §3
