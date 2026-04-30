# Batch 02A - Driver Mobile Feature Catalog

## A. Driver Mobile Surface Features

| Surface | Capability | Inputs | Outputs | Operational Value |
|---|---|---|---|---|
| Login (Android/iOS) | Driver authentication and session bootstrap | phone, pin | signed token, role claims | Establishes scoped execution identity |
| Home Dashboard | Mission readiness and quick actions | route state, mission stats | action selection | Reduces dispatch-to-drive latency |
| Map | Live route execution and geofence context | telemetry, route polyline, stop set | next-stop action, mission context | Maintains route adherence visibility |
| Rides | Manifest and stop sequence inspection | assigned manifests | ride detail selection | Improves stop-level awareness |
| QR Scanner | Dock/retailer scan verification | camera frame, token | scanned proof payload | Adds physical proof link to digital flow |
| Offload Review | Accepted/rejected unit verification | manifest lines, received counts | correction payload, settlement path | Prevents silent quantity drift |
| Payment Waiting/Cash Collection | Settlement state machine progression | gateway state or cash input | completion authorization | Protects financial closure integrity |
| Delivery Correction | Exception quantity and refund handling | corrected lines, reason codes | amendment events | Enables controlled exception closure |
| Offline Verifier | Low-connectivity completion support | local proof bundle | delayed sync payload | Preserves continuity in weak networks |
| Notification Inbox | Driver event queue | broadcast payloads | read-state ack | Supports operational awareness |

## B. Backend Mechanism Inventory Supporting Driver Surfaces

| Mechanism | Primary Files | Patent-Relevant Function |
|---|---|---|
| H3 Geospatial Routing | backend-go/proximity/h3.go, backend-go/dispatch/geo.go, backend-go/dispatch/service.go | Spatially clusters workload and computes route candidates |
| Freeze Lock Protocol | backend-go/warehouse/dispatch_lock.go, backend-go/factory/replenishment_lock.go | Prevents AI-human override races during manual intervention |
| Transactional Outbox | backend-go/outbox/emit.go, backend-go/outbox/relay.go | Ensures atomic state + event durability |
| Idempotency Guard | backend-go/idempotency/middleware.go | Suppresses duplicate completion side effects |
| WebSocket Fanout | backend-go/ws/driver_hub.go, backend-go/ws/hub.go, backend-go/ws/keepalive.go | Real-time event propagation to mobile clients |
| Settlement and Reconciliation | backend-go/payment/webhooks.go, backend-go/payment/reconciler.go, backend-go/treasury/service.go | Guarantees payment-linked order closure integrity |
| Notification Dispatch | backend-go/kafka/notification_dispatcher.go, backend-go/notifications/formatter.go | Delivers role-scoped mission and exception signals |

## C. Cross-Surface Sync Requirements (Driver Role)

| Contract Area | Android Driver | iOS Driver | Shared Constraint |
|---|---|---|---|
| Auth Claims | Consumed by DriverNavigation and network interceptors | Consumed by RootView and APIClient | Role and scope claims must remain additive |
| Mission DTO Shape | FactoryModels/Warehouse models in Kotlin | FactoryModels/Warehouse models in Swift | Field names align with backend JSON tags |
| Real-Time Events | WebSocket + notification bridge | WebSocket + notification bridge | Event `type` discriminator must be stable |
| Completion Workflow | Scanner -> Offload -> Payment/Cash -> Complete | Scanner -> Offload -> Payment/Cash -> Complete | Geofence and idempotency gates are mandatory |
