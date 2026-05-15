# Technology Inventory

Canonical inventory of tools, technologies, and external services used across Pegasus.

This file is the human-readable companion to `pegasus/context/technology-inventory.json` and must stay synchronized with it.

## Inventory Sources

- `pegasus/**/package.json`
- `pegasus/**/go.mod`
- `pegasus/**/build.gradle.kts`
- `pegasus/**/Cargo.toml`
- `pegasus/docker-compose.yml`
- `pegasus/infra/terraform/*.tf`
- keyword sweep across `pegasus/**/*.{go,ts,tsx,js,json,kts,tf,yml,yaml,md}`

## Languages And Runtimes

- Go
- TypeScript and JavaScript
- Kotlin
- Swift and SwiftUI
- Rust
- Python
- Shell

## Web And Desktop Stack

- Next.js + React (admin, factory, warehouse, retailer desktop)
- Tailwind CSS + HeroUI + Motion + Recharts + Mapbox/MapLibre
- Tauri 2 desktop shells with Rust backends
- Expo/React Native payload terminal

## Backend Core Stack

- Go services with `chi` HTTP routing and WebSockets
- Spanner data access
- Kafka eventing
- Redis cache and Pub/Sub invalidation
- Firebase integration
- OpenTelemetry and Prometheus metrics
- Structured logging via `log/slog` on runtime paths (including recent high-noise migration in `analytics/`, `sync/`, `replenishment/`, `treasury/`, `vault/`, and `kafka/emitter.go`, plus tranche-1 cache infra migration in `cache/redis.go`, `cache/pubsub.go`, `cache/invalidate.go`, `cache/middleware.go`, `cache/warehouse_load.go`, `cache/warehouse_geo.go`, `cache/priority.go`, and `cache/circuitbreaker.go`)

## Runtime Contract Surfaces

- Checkout fee snapshot substrate: `pegasus/apps/backend-go/{schema/spanner.ddl,migrations/migrations.go,order/unified_checkout.go,order/settlement_target.go}`
	- Schema now provisions additive `SupplierPayoutPolicies`, `InvoiceSettlementSlices`, and `MasterInvoices` fee summary fields (`FeePolicyVersion`, `FeeAmount`, `NetPayoutAmount`, `SettlementSliceCount`).
	- Unified checkout now writes immutable per-supplier settlement slices (gross, fee policy version, fee basis points, fee amount, net payout, payout owner metadata) in the same `ReadWriteTransaction` as invoice and order writes.
	- Downstream payment execution now resolves snapshots through `payment/settlement_snapshot.go`; `order/{unified_checkout.go,service.go}` compute Global Pay split recipients from persisted fee amounts via `ComputeSplitRecipientsWithFeeAmount` when snapshots are available.
	- Downstream settlement execution now applies snapshot authority: `payment/refund.go` computes reversals from snapshot fee ratios/payout owners, `kafka/treasurer.go` credits payout-owner ledger accounts from snapshot metadata, and `internal/services/billing/meter_worker.go` now keeps billing milestones without mutating live `platform_fee_*` authority keys.
	- Treasury read models now expose additive payout snapshot fields via `treasury/settlement.go` and `warehouse/treasury.go`, consumed in supplier/warehouse treasury web and native clients.
	- Missing supplier policy rows default to `HQ_SUPPLIER`, and `WAREHOUSE_LOCAL` policy mode fails closed when participating warehouses do not resolve active credentials for the selected gateway.

- Regional degressive checkout fee policy: `pegasus/apps/backend-go/{countrycfg/regions.go,schema/spanner.ddl,migrations/migrations.go,order/{fee_policy.go,settlement_target.go,unified_checkout.go}}`
	- `RegionalConfigs` now stores additive supplier-effective-currency defaults `DegressiveFeeGrowthThresholdAmount`, `DegressiveFeeScaleThresholdAmount`, and `DegressiveFeeCapAmount` keyed by `RegionId`.
	- `countrycfg/regions.go` seeds per-currency default thresholds and cap values and resolves supplier region defaults through `ResolveSupplierRegion` plus `GetRegionalConfig`.
	- `order/fee_policy.go` now centralizes `computeCheckoutFee` with explicit `REGIONAL_DEGRESSIVE_V1` versioning and tier keys `REGIONAL_DEGRESSIVE_BASE`, `REGIONAL_DEGRESSIVE_GROWTH`, and `REGIONAL_DEGRESSIVE_SCALE`.
	- Regional degressive evaluation activates only when the checkout currency matches the supplier-effective region currency; otherwise checkout falls back to legacy `LEGACY_PLATFORM_FEE_BPS_V1` flat basis-point math.
	- `order/unified_checkout.go` now persists immutable per-slice fee outputs (`SelectedTierKey`, `FeeBasisPoints`, `FeeCapApplied`, `FeeAmount`, `NetPayoutAmount`) into `InvoiceSettlementSlices` and rolls invoice fee summaries into `MasterInvoices`.

- Supplier payout-policy control plane and authority gate: `pegasus/apps/backend-go/{supplier/payout_policy.go,treasury/payout_policy_override.go,settings/platform_config.go,kafka/treasurer.go}`
	- Supplier self-service endpoint now exposes `GET|PATCH /v1/supplier/payout-policy`, and internal support override now exposes `PATCH /v1/internal/treasury/supplier-payout-policy` with INTERNAL-only access guard.
	- Both mutation paths persist audited before/after metadata into `AuditLog` and invalidate supplier profile cache keys after commit.
	- Runtime key `fee_snapshot_authoritative_read` now gates treasurer snapshot-authoritative fee math; when disabled or snapshot reads fail, treasurer logic falls back to legacy `platform_fee_basis_points`.
	- Supplier payment-config UI now includes payout-mode controls and writes policy changes through the same supplier endpoint contract.

- Warehouse import anomaly queue analytics parity: `pegasus/apps/backend-go/warehouse/analytics.go` + `pegasus/apps/warehouse-portal/app/analytics/page.tsx` + `pegasus/apps/{warehouse-app-android,warehouse-app-ios}`
	- `GET /v1/warehouse/ops/analytics` now projects additive `import_anomaly_queue` by scanning warehouse-scoped staged import validation errors from `SupplierImportStagedRows` over the selected period.
	- Warehouse portal analytics now renders Import Anomaly Queue beside Import Freshness for warehouse-scoped operational triage.
	- Warehouse Android and iOS analytics models/views now consume both `import_freshness` and `import_anomaly_queue` with additive top-product quantity alias compatibility (`total_qty` and `total_sold`).

- Task 1.3 native contract bridge: `pegasus/apps/backend-go/cmd/gen-contracts/main.go` + `pegasus/contracts/events.schema.json` + native build hooks in `pegasus/apps/{driver-app-android,retailer-app-android,factory-app-android,warehouse-app-android,payload-app-android}/app/build.gradle.kts`, `pegasus/apps/{driverappios,retailer-app-ios}/*.xcodeproj/project.pbxproj`, and `pegasus/apps/{payload-app-ios,warehouse-app-ios}/project.yml`
	- `gen-contracts` now supports mode-selectable JSON-Schema emission (`-mode json-schema|all`, `-schema-out`) so one backend-owned artifact can drive both TS and native generation workflows.
	- Android role-row modules now run preBuild `generateEventSchema` + `generateWsEventModels` tasks and emit Kotlin websocket envelope models via quicktype.
	- iOS role-row projects now run pre-build schema->Swift generation and emit `Generated/PegasusWSEventEnvelope.swift` via shell script phases (driver/retailer pbxproj) and XcodeGen preBuildScripts (payload/warehouse).

- Task 1.4 desktop-to-native handshake bridge: `pegasus/apps/backend-go/{ws/command_registry.go,userroutes/routes.go,ws/driver_hub.go}` + driver native ACK clients in `pegasus/apps/{driver-app-android,driverappios}`
	- Verified command lifecycle state (`INITIATED -> DISPATCHED -> RECEIVED -> SETTLED`) is now persisted on Redis keyspace `ws:cmdreg:*` with additive 24h TTL (`cache.TTLWSCommandRegistry`) and local fail-open fallback.
	- Shared command APIs now mount `POST /v1/ws/command/dispatch` (desktop initiation) and `POST /v1/ws/ack` (native ACK settlement) via `userroutes` with role-scoped guards.
	- Driver websocket payloads now include additive `command_id` and `command_state`; supplier realtime receives lifecycle frames `COMMAND_DISPATCHED`, `COMMAND_RECEIVED`, and `COMMAND_SETTLED` for desktop spinner/state closure.
	- Driver Android `DriverWebSocket.kt` and driver iOS waiting screens (`PaymentWaitingView.swift`, `ShopClosedWaitingView.swift`) now emit best-effort ACKs through `APIClient.ackWebSocketCommand`.
	- Contract artifacts now include command lifecycle payload definitions in `pegasus/contracts/events.schema.json` and `pegasus/packages/types/ws-events.ts`.

- Phase 2.1 websocket envelope guard: `pegasus/apps/backend-go/{ws/envelope_guard.go,ws/driver_hub.go,ws/retailer_hub.go,ws/supplier_hub.go,ws/payloader_hub.go,ws/warehouse_hub.go,ws/factory_hub.go,kafka/notification_dispatcher.go}` + driver native wait-state clients in `pegasus/apps/{driver-app-android,driverappios}`
	- Role websocket hubs now resolve per-connection schema dialect via `sv` query param and `X-Schema-Version` / `X-Client-Schema-Version` headers (browser-latest fallback, native-legacy fallback).
	- `ws/envelope_guard.go` now centralizes event minimum schema requirements, additive v2->v1 downgrade rules, and incompatible fallback substitution to `SYSTEM_APP_OUTDATED`.
	- Guarded delivery now runs on local hub writes before Redis fanout consumption, so mixed-version websocket clients receive safe payloads instead of crash-prone envelopes.
	- `kafka/notification_dispatcher.go` websocket frame projection now stamps additive `schema_version` metadata for v2 event payloads.
	- Driver Android (`DriverWebSocket.kt`, `PaymentWaitingViewModel.kt`, `ShopClosedWaitingViewModel.kt`) and driver iOS (`PaymentWaitingView.swift`, `ShopClosedWaitingView.swift`) now connect with `sv=2` and treat `SYSTEM_APP_OUTDATED` as a blocking upgrade-required state.

- S-level dynamic billing metering + fee milestones: `pegasus/apps/backend-go/{schema/spanner.ddl,migrations/migrations.go,kafka/billing_tier_worker.go,internal/services/billing/meter_worker.go,settings/platform_config.go,order/service.go,kafka/treasurer.go,treasury/service.go}` + `pegasus/apps/admin-portal/app/treasury/page.tsx`
	- `ORDER_FINALIZED` is now consumed by the billing worker to update `BillingMeterEvents` idempotently and increment sharded `BillingSupplierMeters`/`BillingGlobalMeters` counters.
	- Milestone crossings now atomically update `SystemConfig.platform_fee_basis_points` (with additive `billing_*` control keys) and emit `FEE_RATE_ADJUSTED` via transactional outbox on `kafka.TopicMain`.
	- Split/ledger fee math now resolves runtime basis points; treasury ledger now returns additive `billing_history` + `billing_milestone` telemetry rendered by the supplier treasury dashboard.

- Dynamic delivery handshake settlement surface: `pegasus/apps/backend-go/{schema/spanner.ddl,migrations/migrations.go,order/delivery_handshake.go,order/service.go,deliveryroutes/routes.go,payment/session.go,orderroutes/routes.go}`
	- Adds additive `DeliverySessions` and immutable `DeliverySessionAdjustments` audit rows with order, driver-state, retailer-state, and supplier-state indexes.
	- `POST /v1/delivery/verify-handshake` validates signed JWT or compatibility token handshakes with assignment and geofence checks; `POST /v1/delivery/update-order-during-delivery` applies reconciliation edits through existing amend-order logic and writes immutable adjustment audits.
	- `POST /v1/order/confirm-offload` now transitions session state to `SETTLEMENT_AWAIT` and emits outbox `SETTLEMENT_REQUIRED` + `DELIVERY_SESSION_UPDATED`; settlement notifications fan out over retailer websocket as additive `SETTLEMENT_REQUIRED` plus legacy `PAYMENT_REQUIRED`.
	- `PATCH /v1/treasury/invoice/status` now emits outbox `DELIVERY_DISPUTED` whenever invoice status transitions into `DISPUTED`, carrying additive session/order/supplier/driver/retailer context for downstream manual-review flows.
	- `kafka/notification_dispatcher.go` now consumes `DELIVERY_SESSION_UPDATED` and projects additive settlement/session fields (`amount`, `original_amount`, `adjusted_amount`, `fee_basis_points`, `fee_amount`, `currency`, `session_id`, `invoice_id`) into websocket notification frames so retailer payment consumers keep modal/state parity under Kafka fanout paths.
	- `kafka/notification_dispatcher.go` now consumes `DELIVERY_DISPUTED` and fans supplier-facing realtime/inbox notifications so dispute escalation is not producer-only.
	- `order/service.go#CompleteOrder` now enforces delivery-session settlement lock state and blocks completion while an active session remains uncleared.
	- `order/service.go#ActiveFulfillments` now includes `AWAITING_PAYMENT` with legacy `AWAITING_GLOBAL_PAYNT` compatibility to avoid active-payment visibility regressions during state convergence.
	- Settlement clear paths in `payment/session.go` and `order/service.go#CollectCash` now advance matching delivery sessions to `FINAL_SETTLEMENT`, set `PaymentClearedAt`, and preserve lock-release semantics for both card and cash settlement paths.
	- Retailer role-row consumers now handle settlement events additively: desktop `components/PaymentModal.tsx` consumes `SETTLEMENT_REQUIRED` and in-modal `DELIVERY_SESSION_UPDATED` amount updates, iOS `Services/RetailerWebSocket.swift` maps `SETTLEMENT_REQUIRED` into payment-required flow and maps `DELIVERY_SESSION_UPDATED` to refresh signaling, and Android (`data/api/RetailerWebSocket.kt`, `ui/navigation/NavigationViewModel.kt`, `ui/screens/orders/OrdersViewModel.kt`) consumes both events for payment and order refresh parity.

- Supplier inventory import staging surface: `pegasus/apps/backend-go/{schema/spanner.ddl,migrations/migrations.go,supplier/imports.go,suppliercoreroutes/imports.go}` + `pegasus/apps/admin-portal/app/{supplier/inventory/import/page.tsx,inventory/import/page.tsx}`
	- Adds additive `InventoryImportSessions` and interleaved `InventoryImportRows` schema substrate for supplier-scoped bulk import staging with explicit session and row lifecycle states.
	- `supplier/imports.go` now persists `GET|POST /v1/supplier/inventory/import`, `GET /v1/supplier/inventory/import/upload-ticket`, `GET /v1/supplier/inventory/import/{session}`, `GET /v1/supplier/inventory/import/{session}/rows`, `PATCH /v1/supplier/inventory/import/{session}/mapping`, `POST /v1/supplier/inventory/import/{session}/approve`, `POST /v1/supplier/inventory/import/{session}/apply`, and `GET /v1/supplier/inventory/import/{session}/status` with supplier and warehouse-scope enforcement.
	- Phase-2 compatibility adds `SupplierImportSessions`, interleaved `SupplierImportStagedRows`, and `SupplierImportMapping` (supplier_id/session_id keyed) in schema and migrations, plus `suppliercoreroutes/imports.go` repository methods (`CreateImportSession`, `UpdateSessionStatus`, `SaveStagedRows`) and route stubs on `/v1/supplier/inventory/imports` (`POST /`, `GET /{id}`, `POST /{id}/mapping`, `POST /{id}/approve`, `POST /{id}/apply`).
	- Phase-6 atomic apply now processes sandbox sessions through `ApplyImportSession` in `suppliercoreroutes/imports.go` with supplier ownership checks, `APPROVED|APPLYING` state gate, approved-row filtering, additive `is_new_product` catalog upsert (`SupplierProducts`), dual-write quantity mutations (`SupplierInventory` + `SupplierInventoryV2`), `InventoryAuditLog` journaling (`BULK_IMPORT`), idempotent `APPLIED` completion semantics, and failure fallback to `FAILED` + `error_summary`.
	- Phase-7 freshness wiring now emits websocket `INVENTORY_SYNC_COMPLETE` frames after apply commits from `suppliercoreroutes/imports.go` to both `ws.SupplierHub` and affected `ws.WarehouseHub` channels with session, rows, timestamp, and additive warehouse/product identifiers; supplier and warehouse inventory surfaces now consume the frame for immediate refresh, success toasts, and 3-second row pulse feedback.
	- Phase-3 signed-upload bridge now issues deterministic GCS PUT tickets on `/v1/supplier/inventory/imports/` with object path `imports/{supplier_id}/{session_id}/raw.xlsx`, writes `INITIALIZED` session status before ingress, enforces a 50MB upload-size ceiling, and exposes `POST /v1/supplier/inventory/imports/{id}/uploaded` to transition `UPLOADED` and emit outbox `INVENTORY_IMPORT_UPLOADED` on `inventory.import.events`.
	- Phase-4 AI mapping worker now consumes `inventory.import.events` in `apps/ai-worker/import_worker.go`, claims sessions into `DISCOVERING`, streams first-50-row samples from GCS, resolves schema through provider-agnostic `packages/ai-bridge` (`InventoryMapper` + Vertex `GeminiProvider` token counting) with deterministic exact-header skip and heuristic fallback on provider failures/429, persists `DISCOVERED` or `MAPPING_REQUIRED` plus anomaly summaries into `SupplierImportSessions` and `SupplierImportMapping`, and emits `INVENTORY_IMPORT_STATUS_UPDATE` for supplier websocket `IMPORT_STATUS` fanout.
	- Import sandbox route handlers initialize signed upload tickets without local file writes and enforce supplier scope exclusively from JWT claims.
	- Staged import mapping audit writes use `BULK_IMPORT_STAGED` in `InventoryAuditLog`; phase-6 atomic apply journals production quantity mutations with reason `BULK_IMPORT`.
	- Admin portal now exposes bulk-import ingress surfaces at `app/supplier/inventory/import/page.tsx` and `app/inventory/import/page.tsx`, linked from inventory entry points with drag-drop direct upload, uploaded-signal bridge, and processing scanner/skeleton states.

- Supplier import analytics attribution convergence: `pegasus/apps/backend-go/suppliercoreroutes/routes.go` + `pegasus/apps/admin-portal/app/{supplier/analytics/page.tsx,inventory/import/page.tsx}`
	- Legacy `/v1/supplier/inventory/import*` aliases now fail closed with structured `410 Gone` responses that point to canonical `/v1/supplier/inventory/imports` routes.
	- Supplier analytics now hydrates warehouses through `GET /v1/supplier/warehouses` and scopes `GET /v1/supplier/analytics/revenue` by optional `warehouse_id` for global-versus-warehouse attribution.
	- Import apply finalize now redirects to `/supplier/analytics` with `import_session` and optional single-warehouse focus so post-import attribution review opens in-context.
	- Supplier import fact attribution now persists additive warehouse/date/SKU facts in `SupplierImportAnalyticsFacts` (schema + migration), and `ApplyImportSession` now performs deterministic per-apply upserts (`applied_rows`, `quantity_delta`, `session_count`, `last_session_id`, `last_applied_at`) in the same transaction as inventory writes.
	- Warehouse analytics now projects additive `import_freshness` on `GET /v1/warehouse/ops/analytics`, and the warehouse portal analytics screen now renders the Import Freshness card for warehouse-scoped attribution visibility.

- InventoryV2 runtime activation: `pegasus/apps/backend-go/{supplier/inventory.go,warehouse/inventory.go,order/unified_checkout.go,supplier/{reconcile.go,returns.go,vetting.go},factory/{transfers.go,force_receive.go},replenishment/engine.go,factory/{look_ahead.go,predictive_push.go},warehouse/dashboard.go}`
	- Supplier and warehouse inventory reads now prefer warehouse-scoped `SupplierInventoryV2` quantities with legacy `SupplierInventory` fallback.
	- Unified checkout mirrors effective stock decrements into `SupplierInventoryV2` for warehouse-assigned plans while retaining existing `SupplierInventory` locking path.
	- Supplier/factory restock and receive flows now mirror additive stock restoration/increment mutations into `SupplierInventoryV2` when order or transfer warehouse context is present.
	- Replenishment/planning/dashboard reads now also resolve `SupplierInventoryV2` first with legacy fallback in `getWarehouseStock`, `fetchWarehouseInventory`, predictive breach scans, and warehouse low-stock KPI counting.

- Autonomous routing fallback: `pegasus/apps/backend-go/factory/network_optimizer.go`
	- `SelectOptimalFactoryWithTelemetry` now falls back to nearest active supplier factory when no active `SupplyLanes` candidate exists for the supplier plus warehouse pair.
	- Fallback selection applies additive product-aware `Factories.ProductTypes` preference when explicit product mappings exist, and uses warehouse `PrimaryFactoryId`/`SecondaryFactoryId` when warehouse coordinates are unavailable.
	- Planning loops in `factory/look_ahead.go` and `factory/pull_matrix.go` now log missing routing candidates instead of lane-only misses.

- Checkout warehouse tie-break determinism: `pegasus/apps/backend-go/proximity/warehouse_resolver.go`
	- Candidate ranking now uses H3 grid distance in both Redis grid-cell and Spanner fallback resolver paths.
	- Equal-ring candidates now use Redis-backed round-robin distribution via `wh:rr:<supplierId>:<retailerCell>` to avoid dense-cell hotspotting.
	- Resolver tie handling falls back to deterministic lexical warehouse ordering when Redis is unavailable or round-robin state cannot be advanced.

- Settlement-locality bootstrap: `pegasus/apps/backend-go/{schema/spanner.ddl,migrations/migrations.go,order/{unified_checkout.go,service.go},supplier/warehouses.go,warehouse/payment_config.go,vault/vault.go}`
	- Adds additive `MasterInvoices.SettlementTarget`, `Warehouses.PaymentConfigId`, and `SupplierInventoryV2` (`SupplierId,WarehouseId,ProductId` + `H3Cell`) schema surfaces.
	- Unified/card/cash checkout invoice writes now persist settlement target (`GLOBAL_SUPPLIER`/`LOCAL_WAREHOUSE`/`MIXED_WAREHOUSE`).
	- Warehouse CRUD now carries `payment_config_id`, warehouse ops payment-config reads prefer warehouse-scoped entries, and vault order credential resolution now applies precedence `Warehouses.PaymentConfigId -> SupplierPaymentConfigs.WarehouseId -> supplier default`.

- Replenishment event notification consumer coverage: `pegasus/apps/backend-go/kafka/notification_dispatcher.go`
	- Consumes `REPLENISHMENT_LOCK_ACQUIRED`, `REPLENISHMENT_LOCK_RELEASED`, `STOCK_THRESHOLD_BREACH`, and `LOOK_AHEAD_COMPLETED`
	- Producer coverage is complete: lock events in `pegasus/apps/backend-go/factory/replenishment_lock.go`, stock-threshold events in `pegasus/apps/backend-go/factory/pull_matrix.go`, and look-ahead completion events in `pegasus/apps/backend-go/factory/look_ahead.go`

- Shared order compatibility route composition: `pegasus/apps/backend-go/orderroutes/routes.go`
	- Owns `GET /v1/orders`, `GET /v1/orders/line-items/history`, `GET /v1/order/refunds`, `GET /v1/orders/{id}`, `GET /v1/orders/{id}/events`, `PATCH /v1/orders/{id}/status`, `PATCH /v1/orders/{id}/state`, `POST /v1/order/{deliver,validate-qr,confirm-offload,complete,collect-cash,refund,amend}`, `GET /v1/routes`, `POST /v1/prediction/create`, and `PATCH /v1/vehicle/*`
	- Serves an additive superset detail payload for driver iOS, driver Android, and retailer desktop order detail consumers, plus the supplier portal order timeline feed
	- Unified checkout emits `ORDER_VALIDATION_FAILED`, `PAYMENT_CLEARED`, and `ORDER_FINALIZED` to canonical `kafka.TopicMain` through the transactional outbox, replacing the stale `topic.orders.v1` path.
	- Unified checkout now resolves supplier-effective currency via `countrycfg.Service` and persists event-aligned currency across `MasterInvoices`, `Orders`, and `OrderLineItems`; current single-invoice semantics reject mixed-supplier-currency carts with `422` until invoice-split support lands.
	- Payment sessions now honor request currency in `payment.CreateSession` and keep prior session currency during `RetryPaymentSession` gateway switches.
	- Payment gateway client selection now routes through `payment.NewProviderClient` with normalized `GLOBAL_PAY`/`ADYEN`/`CASH` support; `ADYEN` now uses the official `github.com/adyen/adyen-go-api-library/v21` adapter in `payment/adyen.go` for hosted checkout Payment Links.
	- Payment execution now has a second seam in `payment/execution.go`: `ProviderExecutionRouter` normalizes stored-method charge, authorize, capture, void, and refund operations for order/refund callers while `payment/provider.go` continues to normalize gateway identity.
	- Payment execution paths enforce those seams end-to-end: `payment/session.go` create/attempt/bind/retry normalize gateway values through `NewProviderClient`, checkout URL generators fail closed for unsupported/unavailable providers, Global Pay webhook settlement validates session gateway via provider resolver, `order/service.go` now routes retailer card checkout plus fulfillment saved-card charge through `ProviderExecutionRouter`, `order/unified_checkout.go#authorizeAtCheckout` now uses it for Global Pay holds, and `payment/refund.go` now resolves provider credentials through `VaultResolver` and refunds against durable session/attempt provider references. Adyen direct/refund execution remains additive-but-unsupported via `ErrAdyenDirectOperationUnsupported`.
	- Checkout gateway authority now resolves in `order/{service.go,unified_checkout.go,checkout.go}` from region defaults plus supplier country override exceptions with vault-backed credential readiness filtering, so `payment_gateway` is only an optional client hint at checkout entry points; `retailerroutes/payments.go` accepts optional `gateway` hints on `/v1/order/card-checkout`, shared `packages/types/order.ts` now models additive `ADYEN` plus `available_card_gateways`, and the supplier portal payment-config page dedupes duplicate fallback capability cards by gateway.
	- Webhook route composition now mounts `POST /v1/webhooks/{global-pay,stripe,adyen}` through `webhookroutes/routes.go`; Adyen webhook ingress in `payment/adyen_webhook.go` enforces signature-first HMAC verification plus idempotency guard processing.
	- Refund and chargeback mutations now accept additive `amount`/`currency` compatibility input (while preserving `amount_uzs`), refund responses include additive `amount` + `currency`, and ledger-anomaly currency now resolves from runtime input/session currency instead of hardcoded UZS.
	- Treasury anomaly read models now expose additive `currency`: `/v1/admin/reconciliation` returns row currency with legacy `UZS` fallback, `/v1/treasury/cash-holdings` returns report + row currency (default `UZS`), and admin-portal reconciliation/treasury pages consume the additive field while preserving legacy `amount_uzs` request compatibility.
- Telemetry and legacy infra route split: `pegasus/apps/backend-go/telemetryroutes/routes.go`, `pegasus/apps/backend-go/catalogroutes/routes.go`, `pegasus/apps/backend-go/infraroutes/routes.go`, `pegasus/apps/backend-go/authroutes/routes.go`
	- `telemetryroutes` owns `GET /ws/telemetry` and `GET /ws/fleet`
	- `telemetry/hub.go` preserves the JSON websocket ingress/egress contract while deriving or forwarding `trace_id`; `telemetryaudit/{journal,sink}.go` add best-effort Kafka journaling on `pegasus-telemetry-raw` plus replay-safe Spanner persistence into `DriverTelemetry`
	- `catalogroutes` owns legacy `GET /v1/products`
	- `infraroutes` is infra-only with `GET /v1/health`
	- `authroutes` mounts development-only `GET|POST /debug/mint-token` when `EnableDebugMintToken` is true
- Retailer role-row route composition: `pegasus/apps/backend-go/retailerroutes/routes.go`
	- Owns `GET /v1/retailer/analytics/{expenses,detailed}`, `POST /v1/{orders/request-cancel,order/cash-checkout,order/card-checkout,retailer/shop-closed-response}`, `GET/POST/DELETE /v1/retailer/family-members*`, `POST /v1/retailer/orders/{confirm-ai,reject-ai}`, `POST /v1/orders/{edit-preorder,confirm-preorder}`, `GET/POST /v1/retailer/cart/sync`, `GET/POST /v1/retailer/suppliers*`, `GET/PUT /v1/retailer/profile`, `GET /v1/retailers/{retailerID}/orders`, `GET /v1/retailer/{tracking,cards,pending-payments,active-fulfillment}`, `POST /v1/retailer/card/{initiate,confirm,deactivate,default}`, `PATCH|GET /v1/retailer/settings/auto-order*`, and `GET /v1/ws/retailer`
	- Current role-row consumers span retailer desktop supplier/analytics/tracking/saved-card surfaces plus retailer iOS and Android order, fulfillment, payment, settings, and realtime flows; `POST /v1/order/create` and `POST /v1/order/cancel` now mount here with idempotency guards, while shared order list/detail/refund reads moved to `orderroutes`
	- Retailer desktop and iOS order-list consumers now converge on `GET /v1/retailers/{retailerID}/orders`; iOS and desktop websocket payment handlers also accept both `PAYMENT_*` and `GLOBAL_PAYNT_*` event aliases for settlement/failure continuity.
	- `POST /v1/retailer/cart/sync` now emits `CART_SYNC_UPDATED` over `ws/retailer` post-commit, and retailer desktop, iOS, and Android cart stores rehydrate from `GET /v1/retailer/cart/sync` on that signal for cross-device cart convergence.
	- `POST /v1/retailer/orders/{confirm-ai,reject-ai}` now emits `AI_ORDER_CONFIRMED`/`AI_ORDER_REJECTED` via transactional outbox in the same order mutation transaction, and `ORDER_REASSIGNED` fanout now resolves affected retailer recipients so desktop, iOS, and Android order/tracking clients refresh on reassignment.
	- Retailer desktop notification inbox refresh now includes `SETTLEMENT_REQUIRED` and `DELIVERY_SESSION_UPDATED`, iOS/Android inbox icon mapping now classifies `ORDER_REASSIGNED` plus settlement/session-update events, and shared `packages/types/ws-events.ts` now carries additive retailer ws-event contract entries for `ORDER_REASSIGNED`, `SETTLEMENT_REQUIRED`, `DELIVERY_SESSION_UPDATED`, and `CART_SYNC_UPDATED`.
	- Retailer Android push fallback mapping (`PegasusFirebaseMessagingService`) now explicitly maps `SETTLEMENT_REQUIRED` and `DELIVERY_SESSION_UPDATED` title/body variants so settlement/session-update alerts remain user-visible while backgrounded or after socket disconnect.
- Driver role-row route composition: `pegasus/apps/backend-go/driverroutes/routes.go`
	- Owns `GET /v1/driver/{earnings,history,availability,pending-collections,profile,manifest-gate,manifest}`, legacy `GET /v1/fleet/manifest`, and `GET /v1/ws/driver`
	- Current role-row consumers are `apps/driver-app-android` and `apps/driverappios`; both now target `GET /v1/driver/manifest` while backend keeps `/v1/fleet/manifest` as an additive compatibility alias
	- High-consequence driver mutations (`/v1/order/deliver`, `/v1/order/confirm-offload`, `/v1/order/complete`, `/v1/order/collect-cash`, `/v1/delivery/arrive`) now carry deterministic `Idempotency-Key` headers across Android and iOS clients for replay-safe retry behavior
	- Driver availability toggles (`PATCH|POST /v1/driver/availability`) now emit `DRIVER_AVAILABILITY_CHANGED` inside the same Spanner transaction via outbox; post-commit `kafka.EmitNotification` remains best-effort UX fanout.
	- Driver login/profile payloads now include additive dual-node metadata (`home_node_type`, `home_node_id`, `driver_mode`, `factory_*`) so one driver app role row can support both factory-transfer and warehouse-delivery assignments
	- `/v1/auth/driver/login` now supports tenant-safe same-phone disambiguation with optional `supplier_id` and rejects ambiguous multi-tenant matches instead of relying on implicit global phone uniqueness
- Factory role-row route composition: `pegasus/apps/backend-go/factoryroutes/routes.go`
	- Owns `GET /v1/factory/{dashboard,profile,analytics/overview,transfers*,manifests*,fleet*,staff*,supply-requests*}` plus `POST /v1/factory/{dispatch,transfers/create,manifests/rebalance,manifests/cancel-transfer,manifests/cancel}`
	- Current role-row consumers are `apps/factory-portal`, `apps/factory-app-android`, and `apps/factory-app-ios`; shell/header identity now hydrates from `GET /v1/factory/profile`
	- Android and iOS clients now expose additive advanced-factory endpoints (`transfers/create`, `manifests/{id}`, `manifests/{id}/{action}`, `fleet/{drivers,vehicles}`, `staff/{id}`) while preserving existing operator-loop routes
	- Factory live-update contract is additive: `GET /v1/ws/factory` now emits `FACTORY_SUPPLY_REQUEST_UPDATE`, `FACTORY_TRANSFER_UPDATE`, `FACTORY_MANIFEST_UPDATE`, and `FACTORY_OUTBOX_FAILED`; envelopes carry stable `trace_id` keys; portal + Android + iOS subscribe to websocket-triggered reloads, while polling/manual refresh remains the fallback path
	- `/v1/auth/factory/login` now supports tenant-safe same-phone disambiguation with optional `supplier_id`/`factory_id` and rejects ambiguous multi-tenant matches instead of relying on implicit global phone uniqueness
- Supplier geo-planning route composition: `pegasus/apps/backend-go/proximityroutes/routes.go`
	- Owns `GET /v1/supplier/serving-warehouse`, `GET /v1/supplier/geo-report`, `GET /v1/supplier/dispatch-audits`, `GET /v1/supplier/zone-preview`, `POST /v1/supplier/warehouses/validate-coverage`, and `GET /v1/supplier/warehouse-loads`
	- Current portal consumers are `app/supplier/geo-report/page.tsx`, `app/supplier/warehouses/CoverageEditor.tsx`, `components/warehouse/CoverageMap.tsx`, and `components/dashboard/OrphanAlertsCell.tsx`; `dispatch-audits` exposes the indexed supplier coverage-alert feed while the remaining endpoints stay supplier-facing support surfaces for coverage and load planning
- Supplier self-service route composition: `pegasus/apps/backend-go/supplierroutes/routes.go`
	- Owns `POST /v1/supplier/configure`, `POST /v1/supplier/billing/setup`, `GET/PUT /v1/supplier/profile`, `PATCH /v1/supplier/shift`, `GET/POST/DELETE /v1/supplier/payment-config`, `GET/POST/DELETE /v1/supplier/gateway-onboarding`, and `POST /v1/supplier/payment/recipient/register`
	- Current portal consumers span `app/setup/billing/page.tsx`, `app/supplier/profile/page.tsx`, `app/supplier/payment-config/page.tsx`, `hooks/useSupplierShift.tsx`, and supplier profile readers in product-management screens
	- Supplier gateway capability metadata now includes additive `ADYEN` manual-onboarding fields, and payment runtime gateway resolution is centralized in `payment.NewProviderClient`
- Regional config substrate: `pegasus/apps/backend-go/schema/spanner.ddl`, `pegasus/apps/backend-go/migrations/migrations.go`, `pegasus/apps/backend-go/countrycfg/regions.go`, `pegasus/apps/backend-go/bootstrap/new.go`
	- Adds `Regions` and `RegionalConfigs` tables plus additive nullable `RegionId` links/indexes on `Suppliers`, `Warehouses`, and `Retailers`
	- `countrycfg.Service` now provides `GetDefaultRegionByCountry`, `ResolveSupplierRegion`, and `GetRegionalConfig`, while bootstrap startup seeds `UZ-DEFAULT` fallback region/config rows
	- `auth/region_scope.go` now introduces `RequireRegionScopeWithClient` and `AppendRegionFilter*`; supplier and warehouse role-route composers now inject the middleware so `region_id` query filters are scope-validated for SUPPLIER/ADMIN/WAREHOUSE callers before handler execution
	- Region SQL helper adoption now includes supplier-scoped treasury reads (`/v1/treasury/cash-holdings`, `/v1/supplier/settlement-report`), supplier picking-manifest reads (`/v1/supplier/picking-manifests*`), warehouse ops reads (`/v1/warehouse/ops/{dashboard,orders,orders/{id},dispatch/preview,crm,returns,analytics,treasury,financials}`), and warehouse demand-forecast order-side reads (`/v1/warehouse/demand/forecast`) via `Retailers.RegionId` filters when scoped `region_id` is present
- Supplier warehouse-ops route composition: `pegasus/apps/backend-go/supplierroutes/routes.go`
	- Owns `GET /v1/supplier/org/members`, `POST /v1/supplier/org/members/invite`, `PUT/DELETE /v1/supplier/org/members/{id}`, `GET/POST /v1/supplier/staff/payloader`, `POST /v1/supplier/staff/payloader/{id}/rotate-pin`, `GET /v1/supplier/warehouse-staff`, `PATCH /v1/supplier/warehouse-staff/{id}`, `GET/POST /v1/supplier/warehouses`, `GET/PUT/DELETE /v1/supplier/warehouses/{id}`, `POST /v1/supplier/warehouses/{id}/coverage`, and `GET /v1/supplier/warehouse-inflight-vu`
	- Current portal consumers span `app/supplier/org/page.tsx`, `app/supplier/staff/page.tsx`, `app/supplier/warehouses/page.tsx`, `app/supplier/warehouses/WarehouseStaffPanel.tsx`, `app/supplier/warehouses/CoverageEditor.tsx`, and `components/factory/FactoryNetworkMap.tsx`
	- Coverage saves now update warehouse H3 coverage through the same warehouse spatial outbox + cache refresh path as coordinate edits, so portal coverage editing is no longer a dead-end mutation
	- Driver and factory staff provisioning now enforce duplicate-phone checks within supplier scope (`SupplierId`) on `POST /v1/supplier/fleet/drivers`, `POST /v1/warehouse/ops/drivers`, `POST /v1/factory/staff`, and `POST /v1/auth/factory/register`
	- Warehouse-scoped supplier routes now inject `auth.RequireWarehouseScopeWithClient(d.Spanner)` so `SupplierRole=FACTORY_ADMIN` is constrained to warehouses linked through `Warehouses.PrimaryFactoryId` / `SecondaryFactoryId`; out-of-scope `warehouse_id` returns `403`, and multi-link scope without explicit `warehouse_id` returns `400`
- Supplier catalog-pricing route composition: `pegasus/apps/backend-go/suppliercatalogroutes/routes.go`
	- Owns `GET /v1/supplier/products/upload-ticket`, `GET/POST /v1/supplier/products`, `GET/PUT/DELETE /v1/supplier/products/{sku_id}`, `GET/POST /v1/supplier/pricing/rules`, `DELETE /v1/supplier/pricing/rules/{tier_id}`, `GET/POST /v1/supplier/pricing/retailer-overrides`, and `DELETE /v1/supplier/pricing/retailer-overrides/{id}`
	- Current portal consumers span `components/SupplierProductForm.tsx`, `components/SupplierPromotionForm.tsx`, `app/supplier/products/page.tsx`, `app/supplier/products/[sku_id]/page.tsx`, `app/supplier/catalog/page.tsx`, `app/supplier/pricing/page.tsx`, and `app/supplier/pricing/retailer-overrides/page.tsx`
	- Pricing-rule GET support is preserved through the existing `PricingService.HandleUpsertPricingRule` method while route ownership moves out of `main.go`
- Supplier logistics route composition: `pegasus/apps/backend-go/supplierlogisticsroutes/routes.go`
	- Owns `GET /v1/supplier/picking-manifests`, `GET /v1/supplier/picking-manifests/orders`, `GET /v1/supplier/manifests`, `GET /v1/supplier/manifests/{id}`, `POST /v1/supplier/manifests/{id}/{start-loading|seal|inject-order}`, `POST /v1/payload/manifest-exception`, `GET /v1/supplier/manifest-exceptions`, `POST /v1/supplier/manifests/{auto-dispatch|dispatch-recommend|manual-dispatch}`, `GET /v1/supplier/manifests/waiting-room`, `GET /v1/supplier/fleet-volumetrics`, `POST /v1/supplier/dispatch-queue`, and `GET /v1/supplier/dispatch-preview`
	- Current portal consumers span `app/supplier/manifests/page.tsx`, `app/supplier/manifest-exceptions/page.tsx`, `app/supplier/dispatch/page.tsx`, and the supplier orders page auto-dispatch trigger
	- Includes the payload-facing manifest exception entrypoint so the supplier and payload roles keep one manifest exception contract while route ownership moves out of `main.go`
- Payload loading role-row surface: `pegasus/apps/backend-go/authroutes/routes.go`, `pegasus/apps/backend-go/payloaderroutes/routes.go`, `pegasus/apps/backend-go/supplierlogisticsroutes/routes.go`, `pegasus/apps/backend-go/fleetroutes/routes.go`, `pegasus/apps/backend-go/userroutes/routes.go`, `pegasus/apps/backend-go/ws/payloader_hub.go`
	- Owns `POST /v1/auth/payloader/login`, `GET /v1/payloader/trucks`, `GET /v1/payloader/orders`, `POST /v1/payloader/recommend-reassign`, the shared `GET /v1/supplier/manifests*` and `POST /v1/supplier/manifests/{id}/{start-loading|seal|inject-order}` lifecycle routes, `POST /v1/payload/{manifest-exception,seal}`, `POST /v1/delivery/missing-items`, `POST /v1/fleet/reassign`, `GET /v1/user/notifications`, `POST /v1/user/notifications/read`, and `/v1/ws/payloader`
	- `POST /v1/delivery/missing-items` includes additive request compatibility for payload clients: accepts `missing_items` and alias `items`; accepts empty-item review flags when `source=PAYLOAD_TERMINAL`
	- Current payload consumers span `apps/payload-terminal/App.tsx`, `apps/payload-app-ios/payload-app-ios/Services/APIClient.swift`, `apps/payload-app-ios/payload-app-ios/Services/WebSocketClient.swift`, `apps/payload-app-android/app/src/main/java/com/pegasus/payload/data/remote/PayloadApi.kt`, and `apps/payload-app-android/app/src/main/java/com/pegasus/payload/services/PayloadWebSocket.kt`
	- Shared supplier manifest routes and `/v1/ws/payloader` now admit `PAYLOADER`, keeping Expo, iOS, and Android payload clients aligned with the `SupplierTruckManifests` lifecycle contract
	- The payloader websocket now distinguishes `PUSH` notification frames from `PAYLOAD_SYNC` refresh frames so payload clients silently reload active manifest data on external overrides instead of surfacing empty notifications
	- `PAYLOAD_SYNC` is emitted atomically on draft creation plus the supplier manifest `start-loading`, `inject-order`, `seal`, and `manifest-exception` mutation paths so other payload surfaces stay coherent after cross-device or supplier-portal changes
- Simulation harness route composition: `pegasus/apps/backend-go/simroutes/routes.go`
	- Owns `POST /v1/internal/sim/start`, `POST /v1/internal/sim/stop`, and `GET /v1/internal/sim/status`
	- Handler ownership remains in `pegasus/apps/backend-go/simulation/handler.go`; the extracted composer keeps behavior identical while removing this ADMIN-only cluster from `main.go`
	- Route registration now flows through `chi.Router` with no new `http.DefaultServeMux` mounts; endpoint exposure remains gated by `app.Simulation` (armed only when `SIMULATION_ENABLED=true`)
- Supplier insights route composition: `pegasus/apps/backend-go/supplierinsightsroutes/routes.go`
	- Owns `GET/PUT /v1/supplier/country-overrides`, `GET/DELETE /v1/supplier/country-overrides/{code}`, `GET /v1/supplier/analytics/{velocity,demand/today,demand/history,transit-heatmap,throughput,load-distribution,node-efficiency,sla-health,revenue,top-retailers}`, `GET /v1/supplier/financials`, and `GET /v1/supplier/crm/retailers*`
	- Current portal consumers span `app/supplier/country-overrides/page.tsx`, `app/supplier/analytics/page.tsx`, `app/supplier/analytics/demand/page.tsx`, `app/supplier/dashboard/page.tsx`, `hooks/useAnalytics.ts`, `hooks/useAdvancedAnalytics.ts`, and `app/supplier/crm/page.tsx`
	- Supplier country overrides now validate requested `payment_gateways` against region/country policy and active supplier credential readiness before persistence, return additive `payment_gateway_policy` context from `/v1/supplier/country-overrides*`, and persist additive `OverrideReason`/`UpdatedBy`/`UpdatedByType` plus `AuditLog` rows on upsert/delete
	- Supplier CRM list/detail payloads now include additive retailer `email` for the supplier portal contact drawer
	- Groups supplier read-side settings, analytics, financials, and CRM under one extracted contract while preserving the existing handler ownership in `countrycfg`, `analytics`, and `supplier`
	- Warehouse-scoped analytics endpoints in this composer use `auth.RequireWarehouseScopeWithClient(d.Spanner)` so FACTORY_ADMIN reads are validated against factory-linked warehouses before handler execution
- Supplier operations route composition: `pegasus/apps/backend-go/supplieroperationsroutes/routes.go`
	- Owns `GET/POST /v1/supplier/fleet/drivers`, `GET/PATCH/POST /v1/supplier/fleet/drivers/{id}`, `GET/POST /v1/supplier/fleet/vehicles`, `GET/PATCH/DELETE /v1/supplier/fleet/vehicles/{id}`, `POST /v1/supplier/fulfillment/pay`, `GET /v1/supplier/returns`, `POST /v1/supplier/returns/resolve`, `GET /v1/supplier/quarantine-stock`, and `POST /v1/inventory/reconcile-returns`
	- Current portal consumers span `app/supplier/fleet/page.tsx`, the legacy supplier driver fetch on `app/page.tsx`, `app/supplier/returns/page.tsx`, and `app/supplier/depot-reconciliation/page.tsx`
	- Preserves the existing supplier fulfillment-pay error mapping while grouping fleet and reverse-logistics surfaces under one extracted route owner
- Supplier planning route composition: `pegasus/apps/backend-go/supplierplanningroutes/routes.go`
	- Owns `GET/POST /v1/supplier/delivery-zones`, `PUT/DELETE /v1/supplier/delivery-zones/{id}`, `GET/POST /v1/supplier/factories`, `GET/PATCH/DELETE /v1/supplier/factories/{id}`, `GET /v1/supplier/factories/{recommend-warehouses,optimal-assignments}`, `GET /v1/supplier/geocode/reverse`, `GET /v1/supplier/retailers/locations`, `GET/POST /v1/supplier/supply-lanes`, `GET/PATCH/DELETE /v1/supplier/supply-lanes/{id}`, `GET/PUT /v1/supplier/network-mode`, `GET /v1/supplier/network-analytics`, `POST /v1/supplier/replenishment/{kill-switch,pull-matrix,predictive-push}`, `GET /v1/supplier/replenishment/audit`, and `GET/POST /v1/supplier/warehouses/{territory-preview,apply-territory}`
	- Supports the supplier map/planning loop: delivery-zone administration, supplier→factory recommendation, retailer location overlays, supply-lane management, network optimization controls, replenishment audit/triggers, and territory reassignment
	- Supplier factory create/list/detail/profile payloads now persist and return additive `h3_index` and `product_types` metadata for the supplier factory-planning surfaces
	- Supply-lane mutate actions now resolve organisation supplier scope via `claims.ResolveSupplierID()` and honor `PATCH` for plain lane updates so planning clients stop hitting success-shaped no-ops
	- Keeps cron startup in `main.go` but removes the HTTP route composition for those planning surfaces from the monolith
- Warehouse ops compatibility layer: `pegasus/apps/backend-go/warehouse/inventory.go`, `pegasus/apps/backend-go/warehouse/staff.go`, `pegasus/apps/backend-go/warehouse/vehicles.go`
	- Keeps `GET/PATCH /v1/warehouse/ops/inventory`, `GET/POST /v1/warehouse/ops/staff`, `GET/POST /v1/warehouse/ops/drivers`, `PATCH /v1/warehouse/ops/drivers/{id}/assign-vehicle`, and `GET/POST/PATCH /v1/warehouse/ops/vehicles` additive across warehouse portal, warehouse iOS, and warehouse Android
	- Inventory accepts `q` and `search`, accepts `sku_id` or `product_id` on mutation, and returns both `inventory` and `items` with `sku_id`/`product_id` aliases
	- Staff create accepts an optional PIN and returns the effective one-time PIN; vehicle responses expose both `max_volume_vu` and `capacity_vu` plus a derived `status`
	- `GET /v1/warehouse/ops/treasury?view=invoices` now returns additive per-invoice `currency` with `amount` + `amount_uzs` compatibility so warehouse portal, iOS, and Android treasury invoice surfaces can render currency dynamically while preserving legacy amount contracts
	- Supplier warehouse aggregate stats now include explicit driver `SupplierId` predicates in addition to warehouse/home-node filters to preserve strict multitenant isolation on derived counts
	- Fleet controls now let warehouse admins assign or reset driver vehicles and toggle vehicle availability from portal, iOS, and Android against the same backend contract; dispatch preview excludes inactive vehicles from available-driver output
	- Vehicle availability is schema-backed with `Vehicles.UnavailableReason`, and portal plus native warehouse clients surface the persisted reason when a truck is unavailable
	- Driver list payloads now carry assigned vehicle availability metadata, and dispatch preview publishes both `available_drivers` and `unavailable_drivers` so warehouse clients can explain why an assigned driver is blocked by vehicle availability
	- Warehouse ops read models now apply `auth.AppendRegionFilter` via `Retailers.RegionId` joins across dashboard, orders/detail, dispatch preview, CRM, returns, analytics, treasury, and financials when scoped `region_id` is present.
	- Warehouse demand forecast order-side quantity and burn-rate reads also apply `auth.AppendRegionFilter` via `Retailers.RegionId` joins so replenishment recommendations stay aligned with region-scoped access.
- Warehouse live websocket surface: `/ws/warehouse`
	- Mounted by `pegasus/apps/backend-go/warehouseroutes/routes.go` and broadcast by `pegasus/apps/backend-go/ws/warehouse_hub.go` with post-commit emitters in `pegasus/apps/backend-go/warehouse/supply_requests.go` and `pegasus/apps/backend-go/warehouse/dispatch_lock.go`
	- Emits `SUPPLY_REQUEST_UPDATE` and `DISPATCH_LOCK_CHANGE` frames with `warehouse_id` and `timestamp`
	- Consumed by `pegasus/apps/warehouse-portal/app/supply-requests/page.tsx`, `pegasus/apps/warehouse-portal/app/supply-requests/[id]/page.tsx`, `pegasus/apps/warehouse-portal/app/dispatch-locks/page.tsx`, `pegasus/apps/warehouse-app-ios/WarehouseApp/Views/Dispatch/DispatchView.swift`, and `pegasus/apps/warehouse-app-android/app/src/main/java/com/pegasus/warehouse/ui/screens/dispatch/DispatchScreen.kt`
	- Client helpers in portal, iOS, and Android now auto-reconnect and surface reconnecting/offline state instead of requiring a manual screen reopen
- Warehouse dispatch mutation surface: `/v1/warehouse/supply-requests` and `/v1/warehouse/dispatch-lock*`
	- Owned by `pegasus/apps/backend-go/warehouse/supply_requests.go` and `pegasus/apps/backend-go/warehouse/dispatch_lock.go`
	- Mobile and portal dispatch surfaces create demand-forecast-backed supply requests, cancel warehouse-owned requests, acquire `MANUAL_DISPATCH` locks, and release active locks through the same additive contract
	- Lock release/list enforcement now resolves warehouse/factory scope from JWT claims and rejects out-of-scope lock actions with explicit `403`
	- Portal + Android + iOS dispatch surfaces map `403` to explicit restricted-state messaging for lock and dispatch controls

## Android Stack

- Jetpack Compose + Material 3
- Hilt DI + Retrofit/OkHttp networking
- Room + DataStore local state
- Firebase Auth/Messaging + Google Maps Compose
- CameraX + ML Kit barcode scanning

## iOS Stack

- SwiftUI native apps
- APNs push channel and Apple-native design patterns

## Data, Messaging, And Local Emulators

From `pegasus/docker-compose.yml`:

- Kafka (KRaft) + Kafka UI + topic bootstrap job
- Redis
- Spanner emulator
- Firebase Auth emulator
- WireMock Global Pay mock

## Cloud And Infrastructure Services

From Terraform under `pegasus/infra/terraform`:

- Cloud Run
- GKE + Workload Identity + KEDA via Helm
- Cloud Spanner + Memorystore Redis
- Cloud Armor + Cloud CDN + Cloud DNS + Cloud NAT + Private Service Connect
- Artifact Registry
- Google Cloud Monitoring alert policies and uptime checks
- Multi-region Spanner/GKE topology options

## Security And Reliability Patterns

- Transactional outbox for durable state-change events
- Redis Pub/Sub invalidation after commit
- Maglev-style consistent hash affinity
- Cloud Armor WAF + OWASP rules + per-IP throttling
- Circuit-breaker and priority-guard readiness patterns

## Engineering Guard Tooling

- Local Sequential Thinking MCP: `.agents/extensions/sequential-thinking/mcp-server.mjs`
	- Provides the canonical `sequential_thinking` tool without relying on registry-fetched `npx` startup.
- Workspace retrieval baseline: `file_search`, `grep_search`, `read_file`, `vscode_listCodeUsages`, `semantic_search`
	- Provides the default codebase-retrieval path for symbol discovery, usage checks, and local blast-radius analysis.
- Contract Guard MCP: `pegasus/scripts/contract_guard_mcp.py`
	- Enforces codebase-first context weighting on contract-triggered diffs (runtime code surfaces must dominate context-doc touches).
- Architecture Guard MCP: `pegasus/scripts/architecture_guard_mcp.py`
	- Enforces codebase-first context weighting on architecture-triggered diffs (runtime code surfaces must dominate context-doc touches).
- Design System Guard MCP: `pegasus/scripts/design_system_guard_mcp.py`
	- Enforces codebase-first context weighting on design-triggered diffs (runtime code surfaces must dominate context-doc touches).
- Production Safety Guard: `pegasus/scripts/production_safety_guard.py`
- Visual + Test Intelligence Guard: `pegasus/scripts/visual_test_intelligence_guard.py`
- Security Guard: `pegasus/scripts/security_guard.py`
- Aggregated PR workflow: `.github/workflows/one-eye-guards.yml`

## External Integrations And Providers

- Payme
- Click
- Global Pay
- Adyen
- Stripe
- Telegram
- Firebase
- Google Maps ecosystem

## Sync Contract

If any feature, dependency, service, or runtime changes, update all of:

1. `pegasus/context/technology-inventory.md`
2. `pegasus/context/technology-inventory.json`
3. `.github/ACT.md`
4. `.github/copilot-instructions.md`
5. `.github/gemini-instructions.md`
6. `pegasus/context/architecture.md`
7. `pegasus/context/architecture-graph.json`
- Supplier core route composition: `pegasus/apps/backend-go/suppliercoreroutes/routes.go`
	- Owns `GET /v1/supplier/dashboard`, `GET /v1/supplier/earnings`, `GET/PATCH /v1/supplier/inventory`, `GET /v1/supplier/inventory/audit`, `GET /v1/supplier/orders`, and `POST /v1/supplier/orders/vet`
	- Supports the supplier core portal loop: dashboard metrics, earnings analytics, inventory management, and supplier-side order approval
	- Supplier inventory now honors the mounted root `PATCH /v1/supplier/inventory` contract and returns additive `sku_id`/`product_name` aliases on inventory and audit rows for the supplier portal
	- `ORDER_REASSIGNED` now includes additive `supplier_id` at emission, dispatcher fanout resolves supplier recipients via payload plus order lookup fallback, and supplier dashboard/orders/fleet pages refresh through `ORDER_REASSIGNED` hybrid invalidation
	- Removes the final inline `/v1/supplier/*` registrations from `backend-go/main.go`
