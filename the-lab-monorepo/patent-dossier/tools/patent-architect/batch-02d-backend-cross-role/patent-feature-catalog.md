# Batch 02D - Backend and Cross-Role Feature Catalog

## A. Factory and Warehouse Surface Catalog

| Role/Platform | Surface Families | Patent-Relevant Capability |
|---|---|---|
| Factory Portal (web/desktop shell) | fleet, staff, transfers, loading-bay, payload-override, supply-requests, insights | Factory-side execution control and supply orchestration |
| Factory Android/iOS | dashboard, fleet, staff, transfer list/detail, loading bay, insights | Mobile supervision and field-operational confirmation |
| Warehouse Portal (web/desktop shell) | dispatch-locks, analytics, treasury, manifests, orders, returns, inventory, demand-forecast, dispatch, products, drivers, payment-config | Warehouse command center for inventory, dispatch, and finance |
| Warehouse Android/iOS | dashboard, orders/detail, manifests, returns, inventory, vehicles, drivers, analytics, treasury, dispatch, CRM | On-device warehouse operations and exception management |

## B. Cross-Role Backend Domain Inventory (Discovered)

| Domain Package | Representative Files | Core Function in Ecosystem |
|---|---|---|
| auth | auth/middleware.go, auth/home_node.go, auth/factory_scope.go, auth/warehouse_scope.go | Role and scope enforcement across all mutating paths |
| dispatch | dispatch/service.go, dispatch/binpack.go, dispatch/split.go, dispatch/persist.go | Route assignment and manifest split algorithms |
| proximity | proximity/h3.go, proximity/engine.go, proximity/recommendation.go, proximity/read_router.go | Geospatial indexing and recommendation infrastructure |
| routing | routing/optimizer.go, routing/distance_matrix.go, routing/eta.go | Route optimization and ETA generation |
| warehouse | warehouse/dispatch.go, warehouse/orders.go, warehouse/manifests.go, warehouse/treasury.go | Warehouse operational APIs |
| factory | factory/transfers.go, factory/manifests.go, factory/look_ahead.go, factory/recommend.go | Factory operational APIs and predictive planning |
| supplier | supplier/manifest.go, supplier/fleet.go, supplier/warehouses.go, supplier/dispatcher.go | Supplier-scoped orchestration and shared control-plane paths |
| payment | payment/webhooks.go, payment/global_pay.go, payment/reconciler.go, payment/refund.go | Payment capture, webhook processing, and reconciliation |
| treasury | treasury/service.go, treasury/settlement.go, treasury/reversal.go | Treasury settlement and reversal controls |
| notifications | notifications/dispatcher.go, notifications/formatter.go, notifications/inbox.go | Multi-channel notification fanout |
| ws | ws/hub.go, ws/warehouse_hub.go, ws/driver_hub.go, ws/payloader_hub.go | Real-time delivery channels and room fanout |
| kafka | kafka/events.go, kafka/notification_dispatcher.go, kafka/dlq.go, kafka/headers.go | Event contract, relay, and consumer workflows |
| outbox | outbox/emit.go, outbox/relay.go | Atomic event persistence and asynchronous publish |
| cache | cache/invalidate.go, cache/pubsub.go, cache/ratelimit.go, cache/circuitbreaker.go | Consistency, backpressure, and protection middleware |
| idempotency | idempotency/middleware.go | Duplicate suppression for high-consequence requests |
| analytics | analytics/supplier.go, analytics/factory.go, analytics/demand.go, analytics/intelligence.go | Cross-role KPI and intelligence vectors |

## C. Cross-Role Feature Expansion Table

| Feature Expansion Area | Inputs | Outputs | Integrity Constraints |
|---|---|---|---|
| Inter-node restock orchestration | warehouse stock signals, factory capacity, vehicle availability | replenishment recommendations and manifests | node-scope auth + event ordering by aggregate key |
| Dispatch override governance | manual override requests, lock state, route context | deterministic lock/unlock lifecycle | freeze lock TTL + auditable lock events |
| Settlement-aware completion | offload proof, payment state, order status | completed order and balanced ledger entries | idempotency + double-entry + geofence gates |
| Predictive planning surfaces | historical demand, seasonality, regional patterns | advisory preload/procurement recommendations | assistive-only recommendation acceptance required |
| Multi-client parity rollout | backend DTO changes, event schema updates | synchronized web/mobile feature behavior | additive contracts and feature-flag consistency |
