# 02 Per Role Client Apps

> **HISTORICAL / FROZEN — do not plan from this file alone.**
> Current SoT: [`PROD_READINESS_SEQUENCE.md`](../../PROD_READINESS_SEQUENCE.md) · [`ECOSYSTEM_GAP_REGISTER_2026-08-12.md`](../ECOSYSTEM_GAP_REGISTER_2026-08-12.md) · [`FEATURES_BY_APP_ROLE.md`](../../FEATURES_BY_APP_ROLE.md).
> Body is a point-in-time snapshot; re-verify any claim against code before acting.


_Source: subagent `bd693ff3-85c7-447f-97b6-3ab4479a9319` from End-Product Reality Report session (2026-08-07)._

# PegasusX / ATOMOS — End-Product Reality Report (Client Applications)

**Method:** static code audit only. Every claim below is backed by file path + line. Base: `/Users/shakhzod/Desktop/V.O.I.D/pegasusX`. Line counts via `find`/`grep` on 2026-08-07.

---

## 0. Platform-level ground truth

### Shared API client (`packages/api-client`)
- **Hand-written, not generated.** Single class `ApiClient` in `packages/api-client/index.ts` (2,991 lines; no "generated/DO NOT EDIT" markers anywhere). 164 exported methods (`api_client_methods.txt`), referencing **255 unique `/v1/*` endpoint paths** (`grep -oE '"/v1/[^"]+"' packages/api-client/index.ts | sort -u | wc -l`).
- Fetch-based; JWT Bearer injected via `getAuthToken` (`packages/api-client/index.ts:2924-2926`); idempotency-key support (`packages/api-client/idempotency.ts`, 797 lines); WS reconnect + session-reconcile helpers (`reconnect.ts`, `session-reconcile.ts`, `usePolling.ts`).
- Coverage spans all roles: retailer, supplier, driver, payloader, factory, warehouse, admin (`/v1/admin/fx-rates`, `/v1/admin/partner-keys`), payments, control-tower, planning.

### App sizes (real source files, excluding build checkouts/node_modules)
| App | Files | Note |
|---|---|---|
| retailer-app-ios | 1,533 total Swift but **only 144 real** (`retailerapp/retailerapp/**`); ~1,380 are checked-in SPM build artifacts under `retailerapp/build/SourcePackages/checkouts/` (firebase-ios-sdk, swift-protobuf) | size inflated; also dir name typo `retailerapp` |
| supplier-portal | 882 | Next.js + Tauri (`src-tauri/` present) |
| warehouse-portal | 597 | Next.js + Tauri |
| retailer-app-desktop | 578 (136 ts/tsx) | Next.js + Tauri |
| factory-portal | 365 | Next.js + Tauri |
| marketing-site | 239 | Next.js |
| retailer-app-android | 187 kt | |
| driver-app-android | 178 kt | |
| supplier-app-android | 143 kt | |
| supplier-app-ios | 138 swift | |
| driver-app-ios | 129 swift | |
| warehouse-app-android | 110 kt | |
| warehouse-app-ios | 96 swift | |
| factory-app-android | 77 kt | |
| factory-app-ios | 67 swift | |
| payload-app-android | 61 kt | |
| payload-app-ios | 41 swift | |
| payload-terminal | 33 | Expo/React Native |
| ai-worker | 23 Go files | real service (optimizer, predictivepush, synthesis, import worker) |
| handoff-service | 1 Go file (97 lines) | QR handoff-token validator (`apps/handoff-service/main.go`) |
| **admin-portal** | **0 source files** | retired stub |
| **supplier-app-desktop** | **0 source files** | retired stub |

**Retired apps:** `apps/admin-portal/README.md` ("Deprecated — use supplier-portal"; `redirect.mjs` exits non-zero on dev/build) and `apps/supplier-app-desktop/README.md` ("Canonical desktop surface: ../supplier-portal"). Both contain only `README.md`, `package.json`, `redirect.mjs`. Admin functionality lives in supplier-portal + warehouse-portal under ADMIN JWT.

**backend-go** (context, not a client): 1,077 Go files — the server side is real and deep.

---

## 1. RETAILER

Variants: `retailer-app-android` (187 kt), `retailer-app-ios` (144 swift real), `retailer-app-desktop` (27 dashboard routes).

### 1.1 Feature inventory & wiring

| Feature | Status | Evidence |
|---|---|---|
| Auth (login/register/org-select/memberships) | WIRED-LIVE | `retailer-app-android/.../data/api/PegasusApi.kt:64-77` (`/v1/auth/retailer/login`, `/register`, `/memberships`, `/select-org`, `/switch-org`); iOS `Services/AuthManager.swift:74,143`; desktop `lib/auth.ts:13` (OS keyring via Tauri, no localStorage) |
| Catalog (categories/products/search) | WIRED-LIVE | `PegasusApi.kt:133-154` (`/v1/catalog/*`); iOS `Screens/CatalogView.swift`; desktop `app/(dashboard)/catalog/page.tsx` |
| Cart + checkout (quoted pricing) | WIRED-LIVE | `PegasusApi.kt:93,145` (`/v1/order/create`, `/v1/retailer/checkout/quote`); `CartViewModel.kt:90-101`; desktop `components/CheckoutModal.tsx` |
| Orders list/detail/lifecycle | WIRED-LIVE | `PegasusApi.kt:90,99,105`; `OrdersViewModel.kt`; desktop `app/(dashboard)/orders/page.tsx` |
| AI predictions / AI preorder | WIRED-LIVE | `PegasusApi.kt:174-180` (`/v1/ai/preorder`, `/v1/ai/predictions`, correction PATCH); iOS `Screens/InsightsView.swift`; desktop `lib/api.ts:15-29` (confirm/reject AI order with idempotency keys) |
| Claims (eligibility, file, media upload) | WIRED-LIVE | `PegasusApi.kt:112-125` (`/v1/orders/{id}/claims*`), `:125` (`/v1/media/upload-ticket`); `FileClaimSheet.kt`; iOS `FileClaimView.swift`; desktop `FileClaimPanel.tsx` |
| Suppliers discovery/connect | WIRED-LIVE | `PegasusApi.kt:151-167`; `MySuppliersViewModel.kt`, `ConnectSupplierSheet.kt` |
| Auto-order rules | WIRED-LIVE | `AutoOrderScreen.kt`, `AutoOrderViewModel.kt`; iOS `AutoOrderView.swift`; desktop `app/(dashboard)/auto-order/page.tsx` |
| POS (in-store sales) | WIRED-LIVE + OFFLINE | `PosScreen.kt` + `PendingPosSaleDao.kt`, `PendingPosSaleSync.kt` (Room queue + replay); iOS `Services/PendingPosStore.swift` (file/UserDefaults persistence); desktop `app/(dashboard)/pos/page.tsx` |
| Stock counts / store stock / local SKUs | WIRED-LIVE | `StoreStockScreen.kt`, `LocalSkusScreen.kt`; api-client `RetailerStockCountCommitRequest` (`packages/api-client/index.ts`); desktop `app/(dashboard)/stock/page.tsx`, `stock/local-skus/page.tsx` |
| Shifts | WIRED-LIVE | `ShiftsScreen.kt`; iOS `ShiftsView.swift`; desktop `app/(dashboard)/shifts/page.tsx` |
| Reports / analytics | WIRED-LIVE | `ReportsScreen.kt`, `AnalyticsViewModel.kt`; iOS `ReportsProView.swift`; desktop `reports/page.tsx` |
| HQ multi-location analytics | WIRED-LIVE | desktop `app/(dashboard)/hq/page.tsx:72-120` (`/v1/retailer/hq/summary`, `/sales-by-location`, `/sales-by-sku`, `/export`) |
| Tracking / delivery map | WIRED-LIVE | `DeliveryTrackingViewModel.kt`, `TrackingMap.kt`, `RetailerWebSocket.kt`; iOS `DeliveryMapView.swift` |
| Payments / saved cards / credit profile | WIRED-LIVE | `SavedCardsViewModel.kt`, `CreditProfileViewModel.kt`; `PegasusApi.kt:200` (`/v1/retailer/credit-profile`); desktop `settings/cards/page.tsx`, `credit/page.tsx` |
| Team/family members, locations | WIRED-LIVE | `PegasusApi.kt:209-224`; `FamilyMembersViewModel.kt` |
| Setup wizard, capabilities packs | WIRED-LIVE | `PegasusApi.kt:188,234-244`; `SetupWizardScreen.kt` |
| Control tower (hex map, EKG graph) | WIRED-LIVE | `ui/controltower/ControlTowerScreen.kt`; iOS `Screens/ControlTower/ControlTowerView.swift:16` explicitly renders live counts ("never demo charts") |
| Notifications inbox + FCM push | WIRED-LIVE | `NotificationInboxViewModel.kt`, `PegasusFirebaseMessagingService.kt`, `PegasusApi.kt:86` (`/v1/user/device-token`) |
| Auto-updater (enterprise) | WIRED-LIVE | `service/AutoUpdater.kt:97,147` (manifest-driven) |

**Android API surface:** 120 Retrofit endpoints in `PegasusApi.kt` (`grep -c '@GET|@POST...' = 120`). **iOS:** 90 `/v1/` path literals in `Services/APIClient.swift`. **Desktop:** apiFetch wrappers with per-action idempotency keys (`lib/api.ts:15-80`).

### 1.2 Broken/incomplete
- `MySuppliersScreen.kt:291` — auto-order indicator is a placeholder icon ("always show icon if supplier has orders"). Cosmetic only.
- No other TODO/mock/hardcoded-data hits in any of the three variants (all "placeholder" grep hits are Compose `TextField` hint labels, e.g. `AuthScreen.kt:260,445`).
- iOS repo hygiene: checked-in `build/SourcePackages` artifacts; typo dir `retailerapp`.

### 1.3 Auth / Offline
- Auth: **JWT primary + Firebase phone/custom-token hybrid** with graceful degradation (`data/auth/FirebaseAuthHelper.kt:23-25,58,120`; iOS `Services/FirebaseAuthHelper.swift:32-33` emulator config `demo-pegasus` is emulator-only). Token in `TokenManager.kt:65` (`KEY_JWT`) android; Keychain iOS (`AuthManager.swift:222-233`); OS keyring desktop.
- Offline: **real.** Room DB with 4 entities (`AppDatabase.kt:8`, `CatalogDao.kt:6`, `PendingOrderEntity.kt:6`, `PendingPosSaleEntity.kt:6`, `PredictionDao.kt:6`) + `PendingOrderSyncWorker.kt`, `PendingPosSaleSync.kt`. iOS: `PendingOrderReplayer.swift`, `PendingPosStore.swift`. Desktop: `@pegasusx/desktop-cache` + `lib/retailer-offline-tray.tsx`.

**Retailer maturity: ~92%** — all three variants live, offline-capable, zero mock data. Weak/missing top-5: (1) auto-order indicator faked on android; (2) iOS repo polluted with build artifacts (build risk); (3) desktop has no independent catalog barcode scanner; (4) no E2E-visible refund/return flow from retailer side (claims only); (5) ControlTower parity differs (android+iOS rich, desktop simpler `control-tower/page.tsx`).

---

## 2. SUPPLIER

Variants: `supplier-app-android` (143 kt), `supplier-app-ios` (138 swift), `supplier-portal` (74 pages; also the desktop + admin surface via Tauri).

| Feature | Status | Evidence |
|---|---|---|
| Auth (register/login/billing/business setup) | WIRED-LIVE | `packages/api-client/index.ts:434-462` (`/v1/auth/supplier/register|login`, configure, billing); android `LoginScreen.kt`, `RegisterScreen.kt` |
| Dashboard/pulse | WIRED-LIVE | `DashboardScreen.kt`, `SupplierPulseStrip.kt`; portal `app/(portal)/dashboard/page.tsx` |
| Orders hub + detail | WIRED-LIVE | `OrdersHubScreen.kt`, `OrderDetailScreen.kt`, `OrdersViewModel.kt`; portal `(portal)/orders` |
| Catalog CRUD + barcode + image upload | WIRED-LIVE | `CatalogScreen.kt`, `CreateProductDialog.kt`, `CatalogImageUploader.kt`; portal `(portal)/catalog/[productId]/page.tsx` |
| Inventory + CSV import sessions | WIRED-LIVE | `InventoryImportScreen.kt`; api-client `createSupplierImportSession/applySupplierImportSession` (`api_client_methods.txt:17,18`); portal `(portal)/inventory/import/page.tsx` |
| Manifests + exceptions | WIRED-LIVE | `ManifestsScreen.kt`, `ManifestDetailScreen.kt`, `ManifestExceptionsScreen.kt`; portal `(portal)/manifest-exceptions/page.tsx` |
| Dispatch preview/execute (MapLibre) | WIRED-LIVE | `DispatchPreviewScreen.kt`, `DispatchPreviewMapLibre.kt`; api-client `createSupplierDispatchPreview`, `executeSupplierDispatch`; portal `(portal)/dispatch/page.tsx` |
| Fleet live map, org fleet (drivers/vehicles/members) | WIRED-LIVE | `FleetLiveMapScreen.kt`, `FleetLiveMapLibre.kt`, `OrgFleetScreen.kt`; portal `app/org-fleet/page.tsx`, `(portal)/fleet` |
| Claims/chargebacks/credit notes | WIRED-LIVE | `ClaimsScreen.kt`, `ChargebacksScreen.kt`, `ClaimChargebacksScreen.kt`, `CreditNotesScreen.kt`; portal `(portal)/chargebacks`, `exceptions/claims/page.tsx` |
| Exceptions: negotiations, shop-closed, early-complete | WIRED-LIVE | `NegotiationsScreen.kt`, `ShopClosedScreen.kt`, `EarlyCompleteScreen.kt`; portal `exceptions/negotiations|shop-closed|early-complete` |
| Finance: payments/ledger/reconciliation/treasury/earnings | WIRED-LIVE | `PaymentsScreen.kt`, `LedgerScreen.kt`, `ReconciliationScreen.kt`, `TreasuryHubScreen.kt`; portal `app/payments/page.tsx`, `(portal)/reconciliation`, `finance` |
| Promotions + performance | WIRED-LIVE | `PromotionsScreen.kt`; api-client `createSupplierPromotion`, `getPromotionPerformance`; portal `(portal)/promotions/page.tsx` |
| Delivery zones, topology, supply lanes | WIRED-LIVE | `DeliveryZonesScreen.kt`, `TopologyScreen.kt` + `TopologyMutations.kt`, `SupplyLanesScreen.kt`; portal `delivery-zones`, `topology` |
| Planning (S&OP, brain, policies, seasonal overrides) | WIRED-LIVE | `PlanningBrainScreen.kt`, `ReplenishmentPoliciesScreen.kt`; api-client `getPlanningSAndOP`, `checkPlanningSparsity`; portal `settings/planning`, `operations/replenishment-policies` |
| Retailer price overrides | WIRED-LIVE | `RetailerOverridesScreen.kt`, `CreateOverrideForm.kt`; api-client `createRetailerPriceOverride` |
| Geo report / route performance / knowledge graph | WIRED-LIVE | `GeoReportScreen.kt`, `RoutePerformanceScreen.kt`, `KnowledgeGraphScreen.kt`; portal `geo-report` |
| Admin-capable ops (assign driver, status patch, FX rates, partner keys) | WIRED-LIVE (portal) | `apps/admin-portal/README.md` routes table; api-client `/v1/admin/fx-rates`, `/v1/admin/partner-keys` (`packages/api-client/index.ts`) |
| Compliance, notification prefs, return policy | WIRED-LIVE | `ComplianceScreen.kt`, `NotificationPreferencesScreen.kt`, `ReturnPolicySettingsScreen.kt` |

**API depth:** android `SupplierApi.kt` = 123 Retrofit endpoints; iOS references 108 unique `/v1/` paths (via generic client `SupplierApp/Services/APIClient.swift:19`, base `https://api.pegasus.uz/`); portal uses 115 unique api-client methods.

### Broken/incomplete
- **Zero TODO/mock/demo hits** in android main source and portal (excluding test files). iOS clean too.
- iOS has duplicated component file names across targets (`CreateDriverSheet.swift` ×2, `CreateVehicleSheet.swift` ×2) — dual-target layout, not a defect, but confusing.

### Auth / Offline
- Auth: JWT (`/v1/auth/supplier/login`), portal middleware-gated; Firebase helper present on mobile.
- Offline: **none on android** (no Room/`@Entity` anywhere in `supplier-app-android/app/src/main`), **none on iOS** (no SwiftData/CoreData hit). Portals have `portal-offline-tray`-style read-only caches only in retailer/factory; supplier is effectively **online-only**.

**Supplier maturity: ~90%.** Top-5 gaps: (1) no offline queue on either mobile app (field reps on bad networks lose work); (2) admin-portal retirement means admin UX is scattered across supplier/warehouse portals; (3) iOS dual-target file duplication; (4) no native barcode-scan receiving flow in supplier mobile (field exists in catalog only); (5) desktop is Tauri-wrapped web, no native integrations.

---

## 3. DRIVER

Variants: `driver-app-android` (178 kt), `driver-app-ios` (129 swift).

| Feature | Status | Evidence |
|---|---|---|
| Auth | WIRED-LIVE | `DriverApi.kt:69` (`POST v1/auth/driver/login`); iOS `AuthService.swift`, `LoginViewModel.swift` |
| Manifest (route) load + lifecycle | WIRED-LIVE | `DriverApi.kt:80` (`/v1/driver/manifest`), `ManifestViewModel.kt`, `RouteManifest.swift` (iOS) |
| Delivery execution (deliver/complete/arrive) | WIRED-LIVE | `DriverApi.kt:109,141,170` (`/v1/order/deliver`, `/complete`, `/v1/delivery/arrive`) |
| POD: QR validate + signature pad | WIRED-LIVE | `DriverApi.kt:123,134` (`/validate-qr`, `/delivery/scan-qr`); `SignaturePad.kt`; iOS `SignaturePadView.swift`, `QRScannerView.swift` |
| Cash collection | WIRED-LIVE | `DriverApi.kt:148` (`/v1/order/collect-cash`), `CashCollectionViewModel.kt`; iOS `CashCollectionView.swift` |
| Delivery correction / amend / offload review | WIRED-LIVE | `DriverApi.kt:112,127` (`/order/amend`, `/confirm-offload`); `CorrectionViewModel.kt`, `OffloadReviewViewModel.kt` |
| Fiscalization (retry) | WIRED-LIVE | `DriverApi.kt:155` (`/v1/order/{orderId}/fiscal/retry`), `FiscalizingView.kt`, `FiscalFailedView.kt` |
| Payment waiting / shop-closed waiting | WIRED-LIVE | `PaymentWaitingViewModel.kt`, `ShopClosedWaitingViewModel.kt` |
| Live telemetry (WS + batched sync) | WIRED-LIVE | `TelemetryService.kt`, `TelemetrySocket.kt`, `TelemetrySyncWorker.kt`, `TelemetryLocationEntity` in `PegasusDriverDatabase.kt:13` |
| Geofencing + route deviation + navigation cues | WIRED-LIVE | `DriverGeofence.kt`, `RouteDeviation.kt`, `NavigationCueAnnouncer.kt`, `RouteNavigation.kt`; `DriverApi.kt:99` (route geometry) |
| Offline action queue + verifier + sync queue UI | WIRED-LIVE | `DriverOfflineQueue.kt`, `OfflineSyncWorker.kt`, `OfflineVerifierViewModel.kt`, `SyncQueueScreen.kt`; Room `PendingMutationEntity` (`PegasusDriverDatabase.kt:13`); iOS `Database/OfflineDeliveryStore.swift:7-9` (**SwiftData**-backed), `SyncService.swift` |
| Earnings/history/profile | WIRED-LIVE | `DriverApi.kt:84,209` (`/driver/earnings`, `/driver/history`) |
| Availability toggle | WIRED-LIVE | `DriverApi.kt:191-202` |
| Rescue requests + reassign handshake | WIRED-LIVE | `RequestRescueSheet.kt`; `DriverApi.kt:177` (`/v1/fleet/orders/{id}/reassign-handshake`) |
| Supply transfers | WIRED-LIVE | `SupplyTransfersViewModel.kt`; iOS `SupplyTransfersView.swift` |
| Early complete, handoff inbox | WIRED-LIVE | `EarlyCompleteDialog.kt`, `HandoffInboxCard.kt`, `HandoffPathResolver.kt` |
| Scanner | WIRED-LIVE | `ScannerViewModel.kt` (android, real — unlike warehouse) + `ScannerScreen.kt` |
| Notifications + push | WIRED-LIVE | `DriverNotificationInboxViewModel.kt`, `DriverFirebaseMessagingService.kt` |

**API depth:** android 56 Retrofit endpoints (`DriverApi.kt`); iOS 56 unique `/v1/` paths. **Zero mock/TODO hits on either platform.**

### Broken/incomplete
- Nothing material found. iOS `AutoUpdater.swift`/`EnterpriseUpdateConfig.swift` duplicated across targets (same pattern as supplier).

### Auth / Offline
- JWT (`/v1/auth/driver/login`) + Firebase helper; Keychain/TokenStore on iOS.
- Offline: **best-in-platform.** Room (4 entities incl. telemetry buffer, `PegasusDriverDatabase.kt:11-14`) + workers android; SwiftData offline delivery store + offline verifier + sync queue iOS.

**Driver maturity: ~95%** (the most production-hardened role). Top-5 gaps: (1) no turn-by-turn navigation engine (cues/banners only, geometry from backend); (2) scanner on android limited to QR flows; (3) no offline maps; (4) earnings breakdown thin (single endpoint); (5) duplicated iOS target files.

---

## 4. PAYLOAD / LOADING

Variants: `payload-app-android` (61 kt), `payload-app-ios` (41 swift), `payload-terminal` (33 files, Expo RN loading-bay terminal, `apps/payload-terminal/README.md`).

| Feature | Status | Evidence |
|---|---|---|
| Auth (payloader login/refresh) | WIRED-LIVE | `PayloadApi.kt:49,52` (`/v1/auth/payloader/login|refresh`); `TokenRefreshAuthenticator.kt`, `SecureStore.kt` |
| Trucks sidebar / fleet view | WIRED-LIVE | `PayloadApi.kt:56` (`/v1/payloader/trucks`); `TruckSidebar.kt`; iOS `TruckSidebar.swift` |
| Manifest list/detail | WIRED-LIVE | `PayloadApi.kt:81,87` (`/v1/payloader/manifests*`); `ManifestDetailPane.kt` |
| Start loading / seal / seal-completed / seal-all | WIRED-LIVE | `PayloadApi.kt:90-108`, `:143` (`/v1/payload/seal`); api-client `/v1/payloader/manifests/seal-all` |
| Inject order into manifest | WIRED-LIVE | `PayloadApi.kt:108,135` |
| Supplier-side manifests (dual scope) | WIRED-LIVE | `PayloadApi.kt:115-135` (`/v1/supplier/manifests*`) |
| Order checklist (loading verification) | WIRED-LIVE | `OrderChecklist.kt`; iOS `OrderChecklistSection.swift` |
| Manifest exceptions | WIRED-LIVE | `PayloadApi.kt:149,155` (`/v1/payload/manifest-exception`, list) |
| Recommend/execute reassign | WIRED-LIVE | `PayloadApi.kt:68,74` (`recommend-reassign`, `reassign-order`) |
| Inbound returns (sessions/scan) | WIRED-LIVE | `PayloadApi.kt:201-212` (`/v1/returns/inbound*`); `InboundReturnsViewModel.kt`; terminal `inboundReturns.tsx` |
| Missing-items report | WIRED-LIVE | `PayloadApi.kt:161` (`/v1/delivery/missing-items`) |
| Notifications + push | WIRED-LIVE | `PayloadApi.kt:175-185`; `PayloadFirebaseMessagingService.kt` |
| Offline queue | WIRED-LIVE | Room `PayloadDatabase.kt`, `QueuedActionDao.kt`, `OfflineSyncWorker.kt`; iOS `OfflineQueue.swift` |
| Terminal (shared Expo app) | WIRED-LIVE (smaller) | `payload-terminal/api.ts`, 15 unique `/v1/` paths; `App.tsx`, `firebaseAuth.ts` |

**API depth:** android 38 Retrofit endpoints; iOS 34 unique paths; terminal 15. **Zero mock/TODO hits anywhere.**

### Broken/incomplete
- Terminal is a strict subset (no seal-all, no reassign in the 15 paths) — expected for a shared-bay device, but seal workflows beyond basic are android/iOS-only.

**Payload maturity: ~85%.** Top-5 gaps: (1) terminal feature-subset (no reassign/seal-all); (2) no barcode-scan item-level verification screen (checklist is manual tap); (3) no weight/cold-chain capture UI; (4) iOS has minimal test coverage (`ExampleTests.swift` only); (5) no offline in terminal.

---

## 5. FACTORY

Variants: `factory-app-android` (77 kt), `factory-app-ios` (67 swift), `factory-portal` (20 routes, Tauri).

| Feature | Status | Evidence |
|---|---|---|
| Auth (login/register/refresh) | WIRED-LIVE | portal paths `/v1/auth/factory/login|register|refresh` (`lib/auth.ts:132` apiFetch); android `LoginScreen.kt`; iOS `LoginView.swift` |
| Dashboard + pulse | WIRED-LIVE | `/v1/factory/dashboard`, `/v1/factory/pulse`; `DashboardScreen.kt`, `DashboardHeroCard.kt` |
| Transfers (list/create/detail/move/driver) | WIRED-LIVE | `/v1/factory/transfers*` (`factory-portal/app/transfers/page.tsx:39`); android `CreateTransferScreen.kt`, `TransferDetailScreen.kt`, `TransferFilters.kt`; iOS `MoveTransferSheet.swift` |
| Loading bay (grid/controls) | WIRED-LIVE | `LoadingBayScreen.kt`, `LoadingBayGrid.kt`, `LoadingBayControls.kt`; iOS `BaySection.swift`; portal `loading-bay/page.tsx` |
| Manifests + lifecycle + rebalance/cancel | WIRED-LIVE | `/v1/factory/manifests`, `/cancel`, `/rebalance`, `/cancel-transfer`; `ManifestLifecycle.kt`, `ManifestDetailScreen.kt`; portal `manifests/[id]/page.tsx` |
| Manifest exceptions | WIRED-LIVE | `ManifestExceptionsScreen.kt`; portal `manifest-exceptions/page.tsx` |
| Supply requests + fulfill options | WIRED-LIVE | `/v1/factory/supply-requests*` incl. `/fulfill-options`; `SupplyRequestsScreen.kt`, `SupplyRequestCard.kt`; iOS `SupplyRequestsHubView.swift`; portal `supply-requests/page.tsx` |
| Fleet + live map + drivers/vehicles | WIRED-LIVE | `/v1/factory/fleet`, `/fleet/live-map`, `/fleet/drivers`, `/fleet/vehicles`; `FleetScreen.kt`; portal `fleet/page.tsx` + `lib/use-factory-fleet-live-map.ts` |
| Staff management | WIRED-LIVE | `/v1/factory/staff*`; `StaffScreen.kt`, `StaffDetailScreen.kt`; portal `staff/[id]/page.tsx` |
| Payload overrides | WIRED-LIVE | `PayloadOverrideScreen.kt`, `PayloadOverrideForm.kt`; iOS `PayloadOverrideView.swift`; portal `payload-override/page.tsx` |
| Dispatch | WIRED-LIVE | `/v1/factory/dispatch` (portal) |
| Analytics/insights | WIRED-LIVE | `/v1/factory/analytics/overview`; `AnalyticsScreen.kt`, `InsightsScreen.kt`; portal `analytics`, `insights` |
| Location setup/settings + ops location | WIRED-LIVE | `/v1/factory/ops/location`; `LocationSetupScreen.kt`, `GeocodeApi.kt` |
| Handoff timeline | WIRED-LIVE | `HandoffTimelineSection.kt`; iOS `FactoryHandoffTimelineSection.swift` |
| Notifications | WIRED-LIVE | `NotificationInboxScreen.kt`; portal `notifications/page.tsx` |
| Realtime (WS) | WIRED-LIVE | `FactoryRealtimeClient.kt`, `subscribeFactoryWS` (`portal/app/transfers/page.tsx:8`) |

**API depth:** android `FactoryApi.kt` = 42 Retrofit endpoints; iOS 44 unique paths; portal 43 unique `/v1/` paths. **Zero mock/TODO** in production code (only test-file mocks in `factory-portal/lib/__tests__/csv.test.ts:7-15`).

### Broken/incomplete
- Nothing broken found. Offline: android has `FactoryOfflineQueue.kt`; iOS/portal online-only.

**Factory maturity: ~85%.** Top-5 gaps: (1) no offline on iOS/portal; (2) no production-line/manufacturing execution (this is a dispatch-hub, not MES — no BOM/work-orders anywhere); (3) no bay-level barcode scanning; (4) analytics shallow (single overview endpoint); (5) portal transfers UI drives most power-user flows; mobile is read/act-light.

---

## 6. WAREHOUSE

Variants: `warehouse-app-android` (110 kt), `warehouse-app-ios` (96 swift), `warehouse-portal` (42 routes — **the deepest role portal**).

| Feature | Status | Evidence |
|---|---|---|
| Auth | WIRED-LIVE | `warehouse-portal/lib/auth.ts:90,109` (`/v1/auth/warehouse/refresh`); android `LoginScreen.kt` |
| Dashboard/pulse | WIRED-LIVE | `DashboardScreen.kt`; portal `page.tsx` |
| Orders + detail + ops actions | WIRED-LIVE | `OrdersScreen.kt`, `OrderDetailScreen.kt`, `OrderOpsActions.kt`; portal `orders/[id]/page.tsx` |
| **Pick waves** | WIRED-LIVE **portal + PARTIAL android** | portal `app/pick-waves/page.tsx:34,79,103` (`/v1/warehouse/ops/pick-waves`, create, task confirm); android API exists (`WarehouseApi.kt:124-136`) consumed only inside `TransferActionsScreen.kt:340-366` — **no dedicated screen**; iOS none |
| **Cycle counts + inventory adjustments** | WIRED-LIVE **portal only** | portal `app/cycle-counts/page.tsx:30-31,80,109,237-238`; no mobile UI |
| **Bins + lots + putaway** | WIRED-LIVE **portal only** | portal `app/bins/page.tsx:27-28,54` (`/ops/bins`, `/ops/lots`); android API exists (`WarehouseApi.kt:109-118`) but no screen |
| Inventory | WIRED-LIVE | `InventoryScreen.kt`, `InventoryStockList.kt`; portal `inventory/page.tsx` |
| Dispatch + locks + rescues + settings | WIRED-LIVE | `DispatchScreen.kt`, `DispatchPreviewMapLibre.kt`; portal `dispatch/page.tsx`, `dispatch-locks`, `dispatch/rescues`; api-client `acquireWarehouseDispatchLock`, `executeWarehouseDispatch` |
| Fleet live map / drivers / vehicles | WIRED-LIVE | `FleetLiveMapScreen.kt`; portal `fleet-live-map`, `drivers`, `vehicles/[vehicleId]` |
| Demand forecast | WIRED-LIVE | `DemandForecastScreen.kt`, `ForecastChartPanel.kt`, `ForecastSkuTable.kt`; portal `demand-forecast` |
| Replenishment | WIRED-LIVE | `ReplenishmentScreen.kt`; portal `replenishment/page.tsx` |
| Returns | WIRED-LIVE | `ReturnsScreen.kt`; portal `returns/page.tsx` |
| Claims | WIRED-LIVE | `ClaimsScreen.kt`; portal `claims/page.tsx` |
| Stock commitments | WIRED-LIVE | `StockCommitmentsScreen.kt`; portal `stock-commitments` |
| Supply requests | WIRED-LIVE | `SupplyRequestsScreen.kt`, `CreateSupplyRequestDialog.kt`; portal `supply-requests/new` |
| Transfers + actions | WIRED-LIVE | `TransferActionsScreen.kt` (incl. pick-wave create at `:340`); portal `transfers` |
| Treasury / payment config | WIRED-LIVE | `TreasuryScreen.kt`, `PaymentConfigScreen.kt`; portal `treasury`, `payment-config` |
| Preorders / tomorrow board | WIRED-LIVE | `PreordersScreen.kt`, `TomorrowBoardScreen.kt`; portal equivalents |
| CRM | WIRED-LIVE | `CRMScreen.kt`; portal `crm/page.tsx` |
| Operations (broadcast/pricing preview) | WIRED-LIVE | `OperationsScreen.kt`, `OperationsBroadcastForm.kt`; portal `operations` |
| Staff | WIRED-LIVE | `StaffScreen.kt`; portal `staff` |
| Control tower | WIRED-LIVE (portal) | portal `control-tower/page.tsx` |
| **Barcode scanner (android)** | **DECORATIVE / ORPHANED** | `ScannerViewModel.kt:22` (`// private val warehouseApi: WarehouseApi // TODO: Inject API when available`), `:47` (`// TODO: Dispatch telemetry event to backend` — marks every scan SUCCESS without any call); `BarcodeScannerScreen.kt` has **zero references** from `ui/navigation/WarehouseNavigation.kt` or anywhere (`grep` confirms no usage outside its own files) |
| Manifests | WIRED-LIVE | `ManifestsScreen.kt`; portal `manifests` |

**API depth:** android `WarehouseApi.kt` = 96 Retrofit endpoints; iOS 83 unique paths; portal 55 unique paths. No mocks anywhere else.

### Auth / Offline
- JWT `/v1/auth/warehouse/*`; Firebase helper on mobile.
- Offline: android `WarehouseOfflineQueue.kt:64` — **SharedPreferences-based queue** (not Room); iOS **no offline store at all**.

**Warehouse maturity: ~82%** (portal ~90%, mobile ~70%). Top-5 gaps: (1) barcode scanner is a dead stub with no nav entry and no API call (android); (2) pick-waves/cycle-counts/bins/lots are portal-only — floor workers on mobile can't execute them; (3) no FEFO/cold-chain UI anywhere (no hits for `FEFO`/`cold.?chain` in mobile or portal screens); (4) iOS online-only; (5) android offline queue is SharedPreferences, fragile vs Room peers.

---

## 7. ADMIN

- **admin-portal app: ABSENT (retired).** `apps/admin-portal/` contains only `redirect.mjs`, `README.md`, `package.json`. README: "retired discoverability stub… Canonical supplier/admin surface: `../supplier-portal`". `redirect.mjs` prints the redirect and exits 1.
- **Admin capability is real but lives elsewhere:** admin order ops (`POST /v1/orders/{id}/assign`, `PATCH /v1/order/{id}/status`), payment/chargeback/ledger routes, FX rates (`/v1/admin/fx-rates`), partner keys (`/v1/admin/partner-keys`) — all in api-client (`packages/api-client/index.ts`, methods `createSupplierOrgMember` etc.) and exercised through supplier-portal (`app/payments/page.tsx`, `(portal)/chargebacks`, `(portal)/credit/*`, `(portal)/reconciliation`) with ADMIN/WAREHOUSE_ADMIN/FACTORY_ADMIN JWT gating per `apps/admin-portal/README.md` table.
- **No dedicated admin UI** for: tenant/org management, user/role administration, feature flags, system health, audit logs. None found in any client.

**Admin maturity: ~40%** — finance/order admin is wired inside supplier-portal; a true admin console does not exist.

---

## 8. Non-role apps

- **marketing-site** (239 files): public Next.js site — about/capabilities/contact/customers/platform/roles/solutions routes + `app/playground/page.tsx` (52-line motion-debug page). No product data; appropriate.
- **ai-worker** (23 Go files): real backend worker, not a client. Spanner-backed predictive-push agent (`predictivepush/worker.go:14-23`), route optimizer with Clarke-Wright + 2-opt + tests (`optimizer/clarke_wright.go`, `two_opt.go`, `clarke_wright_test.go`), Kafka import worker (`import_worker.go`), synthesis engine (`synthesis/engine.go`).
- **handoff-service** (1 file, 97 lines): standalone QR handoff-token validation boundary wrapping `packages/handoff` (`apps/handoff-service/main.go:1-4` comment: backend-go embeds same engine in-process for local sims).

---

## 9. Cross-role comparison

| Role | Variants live | Endpoint depth (A/iOS/Portal) | Mocks/TODOs | Offline | Maturity |
|---|---|---|---|---|---|
| Retailer | 3/3 | 120 / 90 / full api-client | 1 cosmetic placeholder | Room+Workers / file-based / desktop-cache | ~92% |
| Supplier | 3/3 (desktop=portal Tauri) | 123 / 108 / 115 methods | 0 | **none** | ~90% |
| Driver | 2/2 | 56 / 56 / n/a | 0 | Room+telemetry buffer / SwiftData | ~95% |
| Payload | 3/3 | 38 / 34 / 15 (terminal) | 0 | Room / OfflineQueue / none | ~85% |
| Factory | 3/3 | 42 / 44 / 43 paths | 0 (test mocks only) | android queue only | ~85% |
| Warehouse | 3/3 | 96 / 83 / 55 paths | **1 dead scanner stub** | SharedPrefs queue / none | ~82% (portal 90 / mobile 70) |
| Admin | **0 dedicated app** | via api-client admin routes | n/a | n/a | ~40% |

Variant parity notes: iOS matches or exceeds Android on Retailer/Supplier/Driver/Factory. Warehouse is the only role where portal ≫ mobile. Retailer-iOS file count is misleading (1,533 incl. ~1,380 vendored SPM checkouts; real app = 144 files).

---

## 10. TOP 15 cross-app gaps (blunt)

1. **No admin console.** `admin-portal` is a redirect stub; org/tenant/user/role/flag management has zero UI anywhere.
2. **Warehouse android barcode scanner is fake** — `ScannerViewModel.kt:22,47` TODOs, reports SUCCESS with no API call, and the screen is orphaned (no nav reference).
3. **WMS execution is desktop-only:** pick-waves, cycle-counts, bins/lots/putaway exist only in warehouse-portal; warehouse mobile apps can't run floor operations.
4. **No FEFO / cold-chain / lot-tracking UI in any client** despite supply-chain positioning (zero hits across all apps).
5. **Supplier mobile has no offline mode at all** (no Room, no SwiftData) — worst offline story among role apps.
6. **Warehouse iOS has no offline queue**; warehouse android's queue is SharedPreferences-based (`WarehouseOfflineQueue.kt:64`), not Room like peers.
7. **Factory apps are dispatch hubs, not factory MES** — no production orders, BOM, or line management in any factory client.
8. **Payload terminal is a strict subset** (15 endpoints vs 38 android) — no reassign, no seal-all on shared bay devices.
9. **No item-level scan verification in payload apps** — loading checklist (`OrderChecklist.kt`) is manual taps.
10. **supplier-app-desktop retired** — suppliers on desktop get a Tauri shell of the web portal; fine functionally, but it's not a distinct app.
11. **Retailer android auto-order indicator is decorative** (`MySuppliersScreen.kt:291`).
12. **Driver navigation is cue-overlays, not turn-by-turn** (`NavigationCueBanner.kt` + backend geometry only).
13. **retailer-app-ios repo hygiene**: ~1,380 vendored `build/SourcePackages` files checked into the app tree + typo'd source dir `retailerapp`.
14. **Duplicated iOS target files** (supplier: `CreateDriverSheet.swift` ×2 etc.; driver: `AutoUpdater.swift` ×2) — drift risk between targets.
15. **No audit-log / system-health surface in any client** — compliance/observability UIs are backend-only.

**Overall platform read:** this is not a mock-up. Every role app is genuinely wired to a real backend (1,077-file Go server, 164-method hand-written client, JWT+Firebase hybrid auth, idempotency keys, WS realtime, offline queues on Android almost everywhere). The weakest links are concentrated and specific: warehouse mobile execution, the dead scanner, missing admin console, and supplier offline.