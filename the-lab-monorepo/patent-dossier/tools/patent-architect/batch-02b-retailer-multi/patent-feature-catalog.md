# Batch 02B - Retailer Multi-Surface Feature Catalog

## A. Retailer Surface Inventory

| Platform | Key Surfaces | Core Function |
|---|---|---|
| Android | root shell, auth, home, catalog, cart, orders, active deliveries, checkout sheet, payment sheet, analytics, auto-order, profile, suppliers, location picker, QR overlay | Operational procurement and delivery confirmation on handheld devices |
| iOS | root shell, login, home, category suppliers, my suppliers, supplier products, category products, product detail, cart, checkout, orders, active deliveries, arrival, future demand, procurement, inbox, history, search, insights, profile, location picker | Native-flow retail operations and AI-assisted demand handling |
| Desktop | landing, dashboard, catalog, orders, tracking, procurement, dock, notifications, insights, settings | High-density planning, monitoring, and procurement execution |

## B. Functional Capability Table

| Capability | Inputs | Outputs | Business Effect |
|---|---|---|---|
| Supplier and Catalog Discovery | category, search text, supplier filters | supplier/product result sets | Accelerates sourcing path |
| Cart and Checkout | selected SKUs, quantities, payment preference | checkout request, payment initiation | Converts demand into confirmed order intent |
| Delivery Acceptance | active order state, QR handoff token, offload summary | payment-required transition or completion | Ensures controlled handoff and proof integrity |
| Auto-Order Governance | hierarchical toggles (global/supplier/category/product) | policy updates | Enables configurable replenishment automation |
| Forecast-Driven Procurement | historical orders, forecast confidence | draft procurement lines | Improves stocking accuracy and timing |
| Tracking and Inbox | order state events, ETA updates, notifications | user-visible status feed | Improves order transparency and responsiveness |

## C. Backend Mechanism Inventory Supporting Retailer Flows

| Mechanism | Primary Files | Patent-Relevant Function |
|---|---|---|
| Retailer-facing analytics and demand | backend-go/analytics/retailer.go, backend-go/analytics/demand.go | Supplies retailer KPIs and forecast surfaces |
| Dispatch and H3 clustering | backend-go/dispatch/geo.go, backend-go/proximity/h3.go | Spatial assignment and route candidate generation |
| Checkout/payment state transitions | backend-go/payment/webhooks.go, backend-go/payment/reconciler.go | Reliable settlement-linked order progression |
| Transactional outbox relay | backend-go/outbox/emit.go, backend-go/outbox/relay.go | Atomic persistence of state and downstream events |
| Real-time hubs and notifications | backend-go/ws/retailer_hub.go, backend-go/kafka/notification_dispatcher.go | Cross-platform update fanout |
| Cache, rate-limit, and idempotency controls | backend-go/cache/invalidate.go, backend-go/cache/ratelimit.go, backend-go/idempotency/middleware.go | Duplicate suppression and consistency under high concurrency |

## D. Cross-Platform Sync Constraints (Retailer Role)

| Contract Area | Android | iOS | Desktop | Shared Constraint |
|---|---|---|---|---|
| Auth/session contract | token-managed local session | token-managed root gate | cookie/token hybrid desktop session | Claim shape remains backward compatible |
| Order/tracking DTOs | Kotlin models | Swift Codable models | TS interfaces | JSON field names remain stable |
| Event stream consumption | notifications + polling fallback | notifications + polling fallback | websocket + dashboard refresh | Event `type` and payload version remain stable |
| Procurement assist payload | forecast cards and draft list | forecast cards and draft composer | procurement table and actions | AI suggestion payloads are additive only |
