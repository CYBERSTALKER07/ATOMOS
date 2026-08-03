# PAYLOAD PHASE - Codebase Status

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.


*Auto-generated directly from real codebase logic and algorithms.*

## 1. Domain Entities & Interfaces
- `Deps`
- `ManifestException`
- `ManifestOrder`
- `ManifestRow`
- `OrderExpectationReader (Interface)`
- `OrderRow`
- `PayloadTx (Interface)`
- `PersistenceSnapshot`
- `PortalManifestLister (Interface)`
- `ReassignRecommendation`
- `Reassignment`
- `Repository (Interface)`
- `Service`
- `ServiceConfig`
- `SpannerRepository`
- `TruckRow`
- `applyReassignRequest`
- `inMemoryRepository`
- `injectOrderRequest`
- `liveOrderLineWire`
- `liveOrderWire`
- `manifestExceptionRequest`
- `payloaderTruckWire`
- `recommendReassignRequest`
- `sealRequest`
- `spannerPayloadTx`
- `spannerTxnBuffer`
- `truckRecommendationWire`
- `wsEnvelope`

## 2. Core Business Logic (Exported Functions)
- `BufferOutbox()`
- `HandleApplyReassign()`
- `HandleDeviceTokenNoop()`
- `HandleFleetReassign()`
- `HandleInjectOrder()`
- `HandleManifestDetail()`
- `HandleManifestException()`
- `HandleManifestExceptions()`
- `HandleManifestsList()`
- `HandleMarkNotificationsRead()`
- `HandleOrders()`
- `HandlePayloaderLogin()`
- `HandlePayloaderRefresh()`
- `HandleRecommendReassign()`
- `HandleSeal()`
- `HandleSealAll()`
- `HandleSealCompletedManifests()`
- `HandleSealManifest()`
- `HandleStartLoading()`
- `HandleTrucks()`
- `HandleUserNotifications()`
- `Hydrate()`
- `ListExceptions()`
- `ListManifestOrders()`
- `ListManifests()`
- `ManifestDetailSnapshotForDriver()`
- `ManifestGateSnapshot()`
- `NewInMemoryRepository()`
- `NewService()`
- `NewSpannerRepository()`
- `RegisterRoutes()`
- `RunTx()`
- `SaveException()`
- `SaveManifest()`
- `SaveManifestOrder()`
- `SetOrderExpectationReader()`
- `SetPortalManifestLister()`
- `WarmManifestCache()`

## 3. Registered API Routes
