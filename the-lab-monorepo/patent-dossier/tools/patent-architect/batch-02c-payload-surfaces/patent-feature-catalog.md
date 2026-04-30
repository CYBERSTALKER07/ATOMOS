# Batch 02C - Payload Surfaces Feature Catalog

## A. Payload Surface Features

| Surface | Capability | Inputs | Outputs | Operational Value |
|---|---|---|---|---|
| Auth Loading | Session restoration at startup | secure token store | route to login or workspace | Fast recovery from tablet restarts |
| Login | Worker authentication | phone, PIN | authenticated worker context | Controlled role-scoped access |
| Truck Selection | Select target truck before loading | available truck list | selected truck context | Prevents manifest-to-truck mismatch |
| Manifest Workspace | Load composition and scan validation | order list, item scans | manifest draft, checklist status | Aligns physical load with digital manifest |
| Post-Seal Countdown | Confirmation hold before dispatch success | seal action, timer | confirmed or canceled seal | Reduces accidental premature dispatch |
| Dispatch Success | Completion confirmation and dispatch codes | sealed manifest | dispatch code, reset action | Clear handoff to driver and operations |
| Native Root/Home Equivalents | Platform-local execution for iOS/Android payload apps | auth state, manifest state | native workflow continuity | Maintains parity across payload clients |

## B. Backend Mechanism Inventory Supporting Payload Flows

| Mechanism | Primary Files | Patent-Relevant Function |
|---|---|---|
| Manifest lifecycle orchestration | backend-go/supplier/manifest.go, backend-go/supplier/manifests.go, backend-go/warehouse/manifests.go | Governs draft->sealed->dispatched transitions |
| Dispatch routing and split logic | backend-go/dispatch/service.go, backend-go/dispatch/split.go, backend-go/dispatch/persist.go | Converts loaded manifests into executable route artifacts |
| Outbox event durability | backend-go/outbox/emit.go, backend-go/outbox/relay.go | Atomic persistence of lifecycle events |
| Real-time payload and driver broadcast | backend-go/ws/payloader_hub.go, backend-go/ws/driver_hub.go | Synchronizes dispatch state to payload and driver surfaces |
| Idempotency and replay guard | backend-go/idempotency/middleware.go | Prevents duplicate seal/finalize side effects |
| Notification dispatch | backend-go/kafka/notification_dispatcher.go, backend-go/notifications/formatter.go | Operational awareness for downstream actors |

## C. Cross-Client Sync Constraints (Payload Role)

| Contract Area | Terminal (Expo) | iOS Payload | Android Payload | Shared Constraint |
|---|---|---|---|---|
| Auth/session DTO | secure store token and worker claims | TokenStore and RootView gating | SecureStore and PayloadRoot gating | Claim payload shape remains additive |
| Manifest DTOs | terminal workspace model | Swift models | Kotlin DTOs | JSON key names stay stable |
| Seal success contract | countdown->success state | home workflow completion state | home workflow completion state | Idempotency key semantics identical |
| Event handling | websocket + notification bus | websocket + push manager | websocket + firebase service | Event type/version compatibility across all clients |
