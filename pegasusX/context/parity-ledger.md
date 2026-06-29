# pegasusX ⇄ Pegasus Parity Ledger

Maps each pegasusX surface to the Pegasus reference and tracks intentional divergence.

## Backend Route Families
| pegasusX | Pegasus reference | Divergence |
|---|---|---|
| `authroutes/` | `pegasus/apps/backend-go/authroutes` | None planned. |
| `supplierroutes/` | `pegasus/apps/backend-go/supplierroutes` | Single seeded supplier; same DTOs. |
| `supplierplanningroutes/` | `pegasus/apps/backend-go/supplierplanningroutes` | Same; topology bootstrap surfaces wired during onboarding. |
| `supplierinsightsroutes/` | `pegasus/apps/backend-go/supplierinsightsroutes` | Same. |
| `supplierlogisticsroutes/` | `pegasus/apps/backend-go/supplierlogisticsroutes` | Same. |
| `supplieroperationsroutes/` | `pegasus/apps/backend-go/supplieroperationsroutes` | Same. |
| `suppliercoreroutes/` | `pegasus/apps/backend-go/suppliercoreroutes` | Same. |
| `suppliercatalogroutes/` | `pegasus/apps/backend-go/suppliercatalogroutes` | Same. |
| `retailerroutes/` | `pegasus/apps/backend-go/retailerroutes` | Supplier discovery returns the seeded supplier. |
| `driverroutes/` | `pegasus/apps/backend-go/driverroutes` | Same. |
| `warehouseroutes/` | `pegasus/apps/backend-go/warehouseroutes` | Same. |
| `factoryroutes/` | `pegasus/apps/backend-go/factoryroutes` | Advanced lifecycle mounted additively (`start-loading`/`seal`/`dispatch`/`complete`, rebalance/cancel/cancel-transfer, exception queue) with scaffold in-memory state; outbox/event flow is active in scaffold runtime while Spanner-backed durability remains pending. |
| `payloaderroutes/` | `pegasus/apps/backend-go/payloaderroutes` | Advanced payload lifecycle/exception/reassignment mounted additively (manifests list/detail/start-loading/inject/seal, exception queue, recommendation + apply reassignment) with scaffold in-memory state; websocket relay path is active via typed fanout envelopes while production Kafka/Redis dependency hard guarantees remain pending. |
| `orderroutes/` | `pegasus/apps/backend-go/orderroutes` | Same. |
| `paymentroutes/` | `pegasus/apps/backend-go/paymentroutes` | Additive scaffold parity: checkout + chargeback/reversal + deprecated global-pay initiate are mounted with idempotency replay support and outbox events, backed by in-memory payment repository seams. |
| `webhookroutes/` | `pegasus/apps/backend-go/webhookroutes` | Signature-first HMAC scaffold handling with transaction-id idempotency and minimal provider payload contracts pending full provider SDK wiring. |
| `telemetryroutes/` | `pegasus/apps/backend-go/telemetryroutes` | Same. |

## Client Surfaces
| pegasusX | Pegasus reference | Divergence |
|---|---|---|
| `apps/supplier-portal` | `pegasus/apps/admin-portal` | Renamed for clarity; same role (SUPPLIER). **UI layout parity (2026-06):** `SupplierShell`, split auth, Bento dashboard, `PageChrome`/`desk-page` list surfaces — flat pegasusX URLs preserved; API wiring unchanged. |
| `apps/retailer-app-android` | `pegasus/apps/retailer-app-android` | Same role row. |
| `apps/retailer-app-ios` | `pegasus/apps/retailer-app-ios` | Same. |
| `apps/retailer-app-desktop` | `pegasus/apps/retailer-app-desktop` | Same. |
| `apps/driver-app-android` | `pegasus/apps/driver-app-android` | Same. |
| `apps/driver-app-ios` | `pegasus/apps/driverappios` | Folder renamed to `driver-app-ios` for consistency. |
| `apps/warehouse-portal` | `pegasus/apps/warehouse-portal` | Same role (WAREHOUSE_ADMIN). **UI layout parity (2026-06):** `WarehouseShell`, split auth, KPI dashboard grid, `PageChrome`/`desk-page` list surfaces — pegasusX-only `/transfers` + `/orders/[id]` wrapped in same chrome; `warehouseApi` wiring unchanged. |
| `apps/warehouse-app-android` | `pegasus/apps/warehouse-app-android` | Same. |
| `apps/warehouse-app-ios` | `pegasus/apps/warehouse-app-ios` | Same. |
| `apps/factory-portal` | `pegasus/apps/factory-portal` | Same. |
| `apps/factory-app-android` | `pegasus/apps/factory-app-android` | Same. |
| `apps/factory-app-ios` | `pegasus/apps/factory-app-ios` | Same. |
| `apps/payload-terminal` | `pegasus/apps/payload-terminal` | Same. |
| `apps/payload-app-ios` | `pegasus/apps/payload-app-ios` | Same. |
| `apps/payload-app-android` | `pegasus/apps/payload-app-android` | Same. |

## Onboarding
| Concept | Pegasus | pegasusX |
|---|---|---|
| Supplier signup | Open registration | Single-tenant company bootstrap (one supplier seeded) |
| Step 2 | Single warehouse address + lat/lng on supplier row | Topology builder creates real `Factories` + `Warehouses` |
| Employment / staffing questions | Inferred via `/supplier/org` | Out of scope (intentionally removed) |
| Billing gate | `/setup/billing` | `/setup/billing` (identical) |

## Supplier portal UI layout parity (2026-06)

| pegasusX route | Layout tier | Notes |
|---|---|---|
| `/auth/*`, `/setup/billing` | Parity | Split auth + circle stepper + centered billing |
| `/dashboard` | Parity | `BentoGrid` cells + per-cell skeletons |
| `/orders`, `/dispatch`, `/manifests`, `/fleet` | Parity | `PageChrome` + dense ops tables/kanban |
| Inventory/catalog/pricing/treasury/network/exceptions | Parity | `PortalSurface` / `desk-page` chrome |
| `/org-fleet`, `/payments`, `/earnings`, `/ai/recommendations` | Parity | Shell-wrapped `desk-page` headers |

Intentional divergence: flat URLs (no `/supplier/*` prefix); pegasusX-only topology step and multi-gateway billing content unchanged.

## Warehouse portal UI layout parity (2026-06)

| pegasusX route | Layout tier | Notes |
|---|---|---|
| `/auth/login` | Parity | Split auth panel + `auth-card` (phone + PIN unchanged) |
| `/` (dashboard) | Parity | KPI card grid + `desk-page-header` (not Bento — pegasus warehouse pattern) |
| `/orders`, `/dispatch`, `/manifests`, `/dispatch-locks`, `/supply-requests` | Parity | `PageChrome` + dense ops tables |
| `/orders/[id]` | pegasusX-only | `PageChrome` + mutation panel (delay/reject/overflow) |
| `/transfers` | pegasusX-only | `PageChrome` + transfer action panel; nav under Operations |
| Inventory/products/demand-forecast, fleet (drivers/vehicles/staff), CRM/returns/analytics/treasury/payment-config | Parity | `PageChrome` + `desk-table` where applicable |

Intentional divergence: pegasusX-only transfer controls and order mutation surface; HeroUI + framer-motion retained per pegasus reference.

## Role-row phase ledger (portal wiring)

Tracks coordinated portal phase delivery across pegasusX role rows. Native rows follow the same backend contracts; portal phases gate operator-facing wiring first.

| Phase | Role | pegasusX portal route(s) | Backend endpoint(s) | SSMR marker |
|---|---|---|---|---|
| P1–P4 | SUPPLIER | `supplier-portal` ops spine (dashboard, orders, dispatch, manifests, treasury) | `supplierroutes/*` | `PX_E2E_ORDER_OK`, supplier portal API probe |
| P1–P4 | RETAILER | `retailer-app-desktop` procurement + mobile catalog/tracking | `retailerroutes`, `orderroutes`, `catalogroutes` | `PX_E2E_CATALOG_OK`, register/order/tracking |
| P1–P4 | DRIVER | native Android/iOS execution surfaces | `driverroutes`, `telemetryroutes` | `PX_E2E_DRIVER_EDGES_OK`, `PX_E2E_TELEMETRY_OK` |
| P1–P4 | PAYLOAD | terminal + tablet manifest lifecycle | `payloaderroutes` | `PX_E2E_PAYLOAD_*`, `manifest-seal` subcheck |
| P5 | FACTORY | `/analytics` → GET `/v1/factory/analytics/overview`; `/insights` approve/dismiss → POST `/v1/warehouse/replenishment/insights/{id}/{action}`; supply-requests transitions | `factoryroutes`, shared replenishment insights | `PX_E2E_FACTORY_ANALYTICS_OK`, `PX_E2E_FACTORY_REPLENISHMENT_ACTION_OK` |
| P6 | WAREHOUSE | `/dispatch-settings` → GET/PATCH `/v1/warehouse/ops/dispatch/settings`; `/replenishment` approve/dismiss | `warehouseroutes` replenishment + dispatch settings | `PX_E2E_WAREHOUSE_DISPATCH_SETTINGS_OK`, `PX_E2E_WAREHOUSE_REPLENISHMENT_OK` |

Focused SSMR subchecks (CI-friendly, no full e2e): `payment`, `shop-closed`, `manifest-seal`.

## Divergence Log
_Add an entry whenever pegasusX intentionally drifts from Pegasus behavior._
- 2026-06-29: Supplier ecosystem gap closure — import wizard + returns resolve + topology PUT + replenishment trigger now use Redis idempotency guards; `POST /v1/supplier/broadcast` fans to role-scoped WS hubs (driver fleet list, warehouse/factory topology rooms, retailer promo room, payload room); factory nodes persist `H3Cell` on topology PUT (DDL `20250628_factories_h3_cell`); dispatch execute returns structured `dispatch_partial_commit` when a later chunk fails after partial commit (idempotency key released — retry warehouse-scoped execute); `POST /v1/supplier/orders/vet` emits `Deprecation: true` (orders auto-confirm; route kept for SSMR).
- 2026-06-29: Warehouse ecosystem gap closure — self-service `PATCH /v1/warehouse/ops/location` persists `H3Cell`, emits `WAREHOUSE_LOCATION_UPDATED` via outbox in the same RW txn, invalidates dispatch plan cache, and supports Redis idempotency replay; `warehouseOpsLocationKey` / `warehouseOpsSettingsKey` wired on portal settings, Android/iOS location screens; warehouse-portal root `Providers` wraps pages in `Suspense` for Next.js `useSearchParams` build compliance.
- 2026-06-12: Phase 5 factory-portal — wired `/analytics` to GET `/v1/factory/analytics/overview` and replenishment insight approve/dismiss on `/insights`. Phase 6 warehouse-portal — wired `/dispatch-settings` (GET/PATCH dispatch settings) and `/replenishment` (insight approve/dismiss). SSMR smokecheck extended with focused `payment`, `shop-closed`, `manifest-seal` subchecks plus factory analytics, warehouse dispatch settings, and replenishment action markers in full e2e.
- 2026-06-06: Driver iOS (`driverappios`) design audit — no graft needed; pegasusX ahead. Completes the full ecosystem sweep (all roles × all clients). `PegasusTheme.swift` byte-identical; `TelemetryBadge`, `StatusPill`, and every view (Home/Profile/Cash/Offload/PaymentWaiting/ShopClosed/MissionDetail/MissionList/QRScanner/Rides/MainTab) byte-identical except 2. pegasusX adds 1 infra file (`Services/DriverWsRefresh.swift`, mirrors Android); no pegasus-only files. UI diffs additive: `FleetMapView` (+13) adds a `MapPolyline` location trail (`LabTheme.fg.opacity(0.25)`, lineWidth 2) + live socket-state `onChange` observers — mirrors Android `MapScreen` trail; `LoginView` (+3) additive. Data: `APIClient`/`AuthService`/`TelemetryServiceLive`/`FleetViewModel`/`TelemetryViewModel` socket+telemetry wiring. NOTE: the iOS `TelemetryBadge` (present in both pegasus & pegasusX) is the reference the Android `WsConnectionPill` was built to mirror — cross-driver-client live-connection telemetry is in sync, expressed natively per platform. Driver role row closed; ENTIRE ecosystem audit complete.
- 2026-06-06: Driver Android (`driver-app-android`) design audit — no graft needed; pegasusX ahead. Theme (`Color/Theme/Tokens/Type.kt`) byte-identical; Home/Profile/offload suite (cash/offload/payment-waiting/shop-closed)/scanner/notifications/navigation/MainTab + `PegasusCard`/`Shimmer` all byte-identical. pegasusX strict superset (+4 infra files: `ConnectionState`, `DriverWsRefresh`, `LocationTrail`, generated `PegasusWSEventEnvelope`; no pegasus-only files). UI diffs are additive feature work: `StateBadge` (+80) adds a NEW `WsConnectionPill` (LIVE/SYNCING/OFFLINE) built on `LocalPegasusColors` + `MotionTokens`, mirroring the iOS driver `TelemetryBadge` command-socket semantics — pegasus driver Android lacks this; `ManifestScreen` wraps the (verbatim-preserved) "LOADING SEQUENCE/UPCOMING" header `Text` in a `Row(SpaceBetween)` to seat the pill; `MapScreen` (+10) adds location-trail polyline; `LoginScreen` (+8) additive. Data/telemetry: `DriverApi` (+3), `DriverWebSocket` (+17/-3 refresh+connection-state), `TelemetrySocket`, `TelemetryService`. Full design parity, pegasusX ahead on live-connection UX + telemetry trail.
- 2026-06-06: Factory Android (`factory-app-android`) design audit — no graft needed; 1:1 mirror of the factory-app-ios changes. Theme (`Color/Theme/Tokens/Type.kt`) byte-identical; Login/Fleet/Insights/LoadingBay/PayloadOverride/TransferDetail + `FactoryState` component identical. pegasusX strict superset (+7 files: `ManifestListScreen`, `ManifestDetailScreen`, `ManifestLifecycle`, `ManifestExceptionsScreen`, `StaffDetailScreen`, `CreateTransferScreen`, generated `PegasusWSEventEnvelope`; no pegasus-only files). UI diffs are feature-wiring: `FactoryNavigation` (+65) adds routes for the new screens; `DashboardScreen` relabels "Critical Insights" KpiCard → "Gate Exceptions" (same `criticalInsights` value/Warning icon, repointed to MANIFEST_EXCEPTIONS) + `WorkflowLaunchRow` rows on existing components; `StaffScreen` adds `.clickable` to the existing `ElevatedCard` → `StaffDetailScreen`; `TransferListScreen` adds create-transfer entry. Data layer: `FactoryModels` (+108), `FactoryApi` (+34 endpoints, payloader/factory scoping). Factory role row (portal + ios + android) fully closed.
- 2026-06-06: Factory iOS (`factory-app-ios`) design audit — no graft needed. `PegasusTheme.swift` byte-identical; LoginView/MainTabView/RootView/Fleet/Insights/LoadingBay/TransferDetail identical. pegasusX is a strict superset (+5 feature views: `ManifestsView`, `ManifestDetailView`, `ManifestExceptionsView`, `StaffDetailView`, `CreateTransferView`; no pegasus-only files). UI diffs are all feature-wiring/cosmetic: `FactoryStateViews` = `body`→`message` property rename (identical render); `DashboardView` wires new manifest/exception sheets via existing `WorkflowLaunchRow` + relabels "Critical insights" metric to "Gate exceptions" (same value/icon); `StaffView` wraps the visually-identical staff row in a `NavigationLink` → `StaffDetailView`; `TransferListView` adds a "Create" toolbar button → `CreateTransferView`. Data layer: `FactoryModels` (+213 fields for new features), `APIClient` (endpoint scoping), `FactoryService` (+23 methods). pegasusX at full design parity, ahead on features.
- 2026-06-06: Payload role (all 3 clients) design audit — no graft needed. Identical file sets, themes byte-identical, zero pegasusX-only or pegasus-only files. (1) `payload-terminal` (Expo): UI component `PayloadStatePanel.tsx` + `theme.ts` + `localization.ts` byte-identical; only `App.tsx` differs and purely data-layer — payloader-scoped endpoints (`/v1/payloader/manifests` vs pegasus `/v1/supplier/manifests`), dev port 8180, WS path `/v1/ws`. (2) `payload-app-ios` (iPad SwiftUI): `TermTheme` + all views identical except `HomeView` (+3) adding `NavigationSplitView` column-width tuning (`.balanced`, min/ideal/max) for iPad; APIClient/WebSocket/HomeViewModel are endpoint scoping. (3) `payload-app-android` (tablet Compose): `Theme.kt` + all UI identical except `HomeScreen` (+7) adding a `LaunchedEffect` that auto-opens the `ListDetailPaneScaffold` detail pane (M3 Adaptive two-pane polish); PayloadApi/WebSocket/HomeViewModel are endpoint scoping. pegasusX at full design parity, marginally ahead on tablet/iPad adaptive layout. Payload role row closed.
- 2026-06-06: Retailer iOS (`retailer-app-ios`) design audit — no graft needed. Theme (`Theme.swift` 271, `PegasusIcon.swift` 75, `AnimationConstants.swift` 33) byte-identical to pegasus; res/branding parity (same Xcode asset structure). pegasusX UI tree is a strict superset (70/70 shared views match modulo brand token; +5 pegasusX-only files: `AccountProfileView`, `ConnectSupplierSheet`, `RetailerCheckoutService`, `RetailerProfileService`, `RetailerSupplierDiscoveryService`). The 9 shared files differing are feature/data-layer: checkout POST extracted into `RetailerCheckoutService` (Orders/Checkout), `CatalogView` adds capsule browse-chips + all-products mode (bento grid + all 4 bento card styles preserved byte-identical), `ProfileView` wires "Company Info" → `AccountProfileView`, `MySuppliersView` adds connect/remove-vendor (supplierCard/emptyState/skeleton preserved), `AppModels` adds `supplierName`/`categoryName` for broader search. pegasusX at full design parity and ahead on features. Whole retailer role row (android/ios/desktop) closed.
- 2026-06-06: Retailer Android (`retailer-app-android`) design audit — no graft needed. Theme (`Color/Type/Theme/Shape/MotionTokens.kt`) byte-identical to pegasus; pegasusX UI tree is a strict superset of pegasus (59/59 shared screens+components match modulo package rename). The 9 shared files that differ and 4 pegasusX-only files (`AccountProfileScreen`, `ReceivingWindowValidator`, `ConnectSupplierSheet`, account VM) are all feature/data-layer additions built on the same M3 primitives: payment deep-link launch (Cart), functional catalog browse modes (Categories/All products/Suppliers), account-profile + receiving-window screen, connect/remove-vendor flows (`MySuppliers`). pegasusX is at full design parity and ahead on features.
- 2026-06-06: Supplier portal status-pill graft — ported pegasus admin-portal `StatusChip`/`StatusBadge` (self-contained, `--desk-*` tokens) into `supplier-portal/components` and applied to the six list surfaces that previously rendered raw status text (`orders`, `manifests`, `returns`, `exceptions`, `fleet/orders`, dashboard recent-manifests). Look-only; API client, DTOs, and routes unchanged. `ai/recommendations` intentionally left on `md-chip` (its status vocabulary is outside the canonical `STATUS_MAP`). `tsc --noEmit` clean.
- 2026-06-06: Warehouse portal UI layout graft landed in pegasusX (`PageChrome`, split auth, KPI dashboard header, `/transfers` shell nav). Visual parity with pegasus warehouse-portal chrome; pegasusX `warehouseApi` / `warehouse-ops` wiring unchanged.
- 2026-06-06: Supplier portal UI layout graft landed in pegasusX (`SupplierShell`, `BentoGrid`, auth split, `PageChrome`). Visual parity with pegasus admin-portal supplier surfaces; pegasusX API client and single-tenant routes unchanged.
- 2026-05-21: Checkout payment session + first attempt persistence now runs atomically in pegasusX through repository `CreateSessionWithAttempt`, closing the prior split-write gap where checkout called `CreateSession` then `SaveAttempt` in separate operations.
- 2026-05-21: Phase-1.2 payment durability is now additive in pegasusX: `schema/spanner.ddl` now includes durable payment write tables (`PaymentSessions`, `PaymentAttempts`, `PaymentChargebacks`, `PaymentReversals`, `PaymentWebhooks`) and `payment/repository_spanner.go` persists these aggregates with outbox events atomically in Spanner transactions. Bootstrap now prefers the Spanner payment repository when available with explicit in-memory fallback retained for degraded/local runtime paths.
- 2026-05-21: Phase-1 backend durability implementation started additively in pegasusX: Orders are now schema-backed in Spanner and `order/repository_spanner.go` persists order rows + outbox events atomically; bootstrap selects this path when Spanner is available but intentionally preserves in-memory fallback for degraded/local runs until broader repository migration lands.
- 2026-05-21: Phase-2 Option-1 spatial hub landed additively in pegasusX: supplier delivery perimeter coverage is precomputed into Redis (`ssmr:delivery_perimeter`, `ssmr:delivery_perimeter:compacted`) at bootstrap via `h3.PolygonToCells` + `h3.CompactCells`, and order creation now fail-closes with `zone_miss` using O(1) `SISMEMBER` checks on server-derived H3 indexes. Runtime gate now includes `cmd/ssmr-smokecheck spatial`; full smoke execution remains environment-dependent on local Docker daemon availability.
- 2026-05-21: Phase-2 SSMR hardening now keeps Go build/module caches warm across sandbox runs via named compose volumes and non-`-v` teardown, and the repo-root `.github/workflows/ssmr-infra.yml` permanently enforces `make test-ssmr-infra` for `pegasusX/**` changes before merge.
- 2026-05-21: Phase-2 SSMR smoke gate is now additive in pegasusX with `scripts/smoke_ssmr.sh`, `make test-ssmr-infra`, `npm run infra:ssmr:test`, and `apps/backend-go/cmd/ssmr-smokecheck` providing executable proof of isolated Spanner/Redis/Kafka/bootstrap wiring before any UI wiring. `infra/docker-compose.ssmr.yml` also pins `/usr/local/go/bin/go` for Go containers to avoid Docker shell PATH drift during bootstrap/runtime bring-up.
- 2026-05-21: Phase-1 SSMR physical sandbox baseline now lives in pegasusX with dedicated `docker-compose.ssmr.yml`, `.env.ssmr.example`, `apps/backend-go/cmd/setup`, tenant-scoped terraform resource/secret naming, and env-resolved `KAFKA_TOPIC_MAIN` for outbox topic isolation. Divergence remains explicit: the compose stack intentionally stops short of a Rust optimizer sidecar because pegasusX does not yet carry a concrete implementation.
- 2026-05-17: Firebase bearer verification is implemented as optional route-level middleware in pegasusX (`FIREBASE_AUTH_ENABLED`) while supplier onboarding keeps cookie JWT as canonical auth path.
- 2026-05-17: Supplier and retailer backend operational route coverage was expanded additively in pegasusX route composers. Retailer protected endpoints are Firebase-role-gated when enabled; local development fallback remains open when Firebase wiring is disabled.
- 2026-05-17: Driver and warehouse backend operational route coverage was expanded additively in pegasusX route composers. Warehouse local-scaffold mode currently uses cookie `ADMIN` fallback auth when Firebase verifier wiring is disabled.
- 2026-05-17: Factory and payload backend operational route coverage was expanded additively in pegasusX route composers with in-memory scaffold handlers. Factory (`FACTORY_ADMIN|ADMIN`) and payload (`PAYLOAD|ADMIN`) protected endpoints are Firebase-role-gated when enabled, with cookie `ADMIN` fallback auth in local scaffold mode.
- 2026-05-17: Payment and webhook backend operational route coverage was expanded additively in pegasusX route composers. Payment checkout/mutation endpoints now use idempotency replay guards and outbox emission, while webhook endpoints use signature-first HMAC verification and transaction-id idempotency with simplified provider payload contracts.
- 2026-05-17: Factory/payload advanced workflow parity moved beyond scaffold lists: manifest lifecycle transitions, exception handling, and reassignment depth are now available through additive endpoints. Current implementation persists in-memory only in pegasusX services and intentionally diverges from Pegasus transactional outbox + websocket fanout paths until persistence wiring is added.
- 2026-05-17: Shared event contracts are extended additively for advanced factory/payload workflow (`MANIFEST_ORDER_INJECTED`, `MANIFEST_ORDER_EXCEPTION`, `MANIFEST_DLQ_ESCALATION`, `MANIFEST_REBALANCED`, `MANIFEST_CANCELLED`) to preserve cross-client discriminator compatibility ahead of full outbox/websocket emission parity.
- 2026-05-17: P1 adapter bridge now attempts Redis cache backend and Kafka outbox publisher with fail-open fallback to in-memory/logging seams; websocket hubs now relay typed `ws:<hub>:fanout` envelopes with source-instance suppression. Additive upgrade: relay read/mark authority now binds to Spanner `OutboxEvents` through `outbox.SpannerStore` when Spanner is reachable, with in-memory fallback when unavailable.
- 2026-05-17: P1 strict reliability mode is additive in pegasusX: `REQUIRE_INFRA_ADAPTERS=true` enforces fail-fast startup if Redis or Kafka adapters fail to initialize, and bootstrap tests now cover strict fail-fast and healthy-adapter startup paths. Divergence remains: most domain repositories are still scaffold in-memory (no end-to-end Spanner ReadWriteTransaction domain persistence parity yet).
- 2026-05-17: Payment execution routing is now additive in pegasusX (`payment/execution.go`) with bounded retry/backoff+jitter and typed gateway policy errors, but provider SDK depth remains partial versus Pegasus; AIRWALLEX direct execution is feature-gated via `AIRWALLEX_DIRECT_EXECUTION_ENABLED` and defaults off in scaffold mode.
- 2026-05-18: Checkout attempt execution metadata persistence is now additive in pegasusX (`payment/service.go` + repository `SaveAttempt`) and checkout responses/events now expose `attempt_id`, `execution_action`, `execution_mode`, and `provider_reference`. Divergence remains: current scaffold persistence stores attempts in-memory; Spanner-backed `PaymentAttempts` durability is still pending.
- 2026-06-28: **Spanner ABORTED retry** — `spannerutils.RunReadWriteTransaction` wraps hot RW paths (order create, payment writes, dispatch chunks, factory setup/location). Factory multi-row `Apply` migrated to RW txn.
- 2026-06-28: **Quantity negotiation** — intentionally **DEFERRED v2**; `order/negotiation_disabled.go` returns HTTP 410 for all negotiation endpoints; SSMR prints `PX_E2E_NEGOTIATION_SKIPPED`.
- 2026-06-28: **Kafka domain topics** — notification dispatcher fans in `TopicMain` + domain topics when `KAFKA_TOPIC_CONSUME_DOMAIN=true`; dual-write via `KAFKA_TOPIC_DUAL_WRITE=true`.
- 2026-06-28: **H3 GridDisk** — `proximity.CellsInRadius` + retailer k-ring zone fallback; `Warehouses.H3Cell` index-backed resolver pre-filter.
- 2026-06-28: **Optimizer runtime** — v1 dispatch uses Go `ai-worker` HTTP (`OPTIMIZER_BASE_URL` → port 8081); Rust `optimizer-core` manifests retained for v2 only.
