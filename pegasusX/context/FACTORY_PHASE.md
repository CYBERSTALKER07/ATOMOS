# FACTORY PHASE - Codebase Status

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.


*Auto-generated directly from real codebase logic and algorithms.*

## 1. Domain Entities & Interfaces
- `Deps`
- `Factory`
- `FactoryTx (Interface)`
- `FleetDriver`
- `FleetVehicle`
- `ManifestDetailSnapshot`
- `ManifestException`
- `ManifestReassignment`
- `ManifestRow`
- `ManifestTransition`
- `PersistenceSnapshot`
- `Repository (Interface)`
- `Service`
- `ServiceConfig`
- `SpannerRepository`
- `StaffRow`
- `SupplyFulfillOptions`
- `SupplyRequest`
- `SupplyRequestItem`
- `TransferRow`
- `acceptSupplyRequest`
- `analyticsOverview`
- `crossManifestRebalanceResult`
- `dispatchRequest`
- `factoryLocationPatch`
- `factoryLocationResponse`
- `factorySetupRequest`
- `factoryStaffRecord`
- `factorySupplyItem`
- `fulfillLineInput`
- `inMemoryRepository`
- `manifestCancelRequest`
- `manifestCancelTransferRequest`
- `manifestRebalanceRequest`
- `spannerFactoryTx`
- `spannerSupplyRow`
- `spannerTxnBuffer`
- `transferCreateRequest`
- `transitionRequest`
- `wsEnvelope`

## 2. Core Business Logic (Exported Functions)
- `BufferOutbox()`
- `CreateFactory()`
- `GetFactory()`
- `HandleAcceptSupplyRequest()`
- `HandleAnalyticsOverview()`
- `HandleCreateFactory()`
- `HandleDashboard()`
- `HandleDispatch()`
- `HandleFactoryLogin()`
- `HandleFactoryRefresh()`
- `HandleFactoryRegister()`
- `HandleFactorySetup()`
- `HandleFleet()`
- `HandleFleetDrivers()`
- `HandleFleetVehicles()`
- `HandleGetFactory()`
- `HandleListFactories()`
- `HandleManifestCancel()`
- `HandleManifestCancelTransfer()`
- `HandleManifestComplete()`
- `HandleManifestDetail()`
- `HandleManifestDispatch()`
- `HandleManifestExceptions()`
- `HandleManifestRebalance()`
- `HandleManifestSeal()`
- `HandleManifestStartLoading()`
- `HandleManifests()`
- `HandleOpsLocation()`
- `HandleProfile()`
- `HandleReplenishmentInsights()`
- `HandleStaff()`
- `HandleStaffDetail()`
- `HandleSupplyRequestFulfillOptions()`
- `HandleSupplyRequestTransition()`
- `HandleSupplyRequests()`
- `HandleTransferByID()`
- `HandleTransferDriverUpdate()`
- `HandleTransferTransition()`
- `HandleTransfers()`
- `HandleUpdateFactory()`
- `Hydrate()`
- `ListFactories()`
- `ListManifests()`
- `ListTransfers()`
- `ManifestDetailSnapshotForDriver()`
- `ManifestGateSnapshot()`
- `NewInMemoryRepository()`
- `NewService()`
- `NewSpannerRepository()`
- `RegisterRoutes()`
- `RunTx()`
- `SaveManifest()`
- `SaveTransfer()`
- `UpdateFactory()`
- `UpdateSupplyRequestState()`
- `WarmManifestCache()`

## 3. Registered API Routes
