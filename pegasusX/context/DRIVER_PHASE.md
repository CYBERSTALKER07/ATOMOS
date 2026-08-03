# DRIVER PHASE - Codebase Status

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.


*Auto-generated directly from real codebase logic and algorithms.*

## 1. Domain Entities & Interfaces
- `AvailabilityUpdate`
- `AvailabilityWriter (Interface)`
- `DailyEarning`
- `DepartResult`
- `Deps`
- `Driver`
- `DriverEarningsResponse`
- `DriverNotificationReader (Interface)`
- `DriverOrderLineView`
- `DriverOrderView`
- `DriverProfileSnapshot`
- `HistoryRow`
- `ManifestGateResult`
- `PendingCollection`
- `Repository (Interface)`
- `RescueRespondRequest`
- `ReturnCompleteResult`
- `RouteGeometryOptions`
- `Service`
- `ServiceConfig`
- `SpannerRepository`
- `Vehicle`
- `availabilityPatchRequest`
- `inMemoryRepository`
- `inMemoryTxnBuffer`
- `spannerTxnBuffer`

## 2. Core Business Logic (Exported Functions)
- `Apply()`
- `ApplyAvailability()`
- `BufferAudit()`
- `BufferOutbox()`
- `CreateDriver()`
- `CreateVehicle()`
- `FindSiblingDriversForOrder()`
- `GenerateOfflineNonce()`
- `GetDriver()`
- `GetVehicle()`
- `HandleAvailability()`
- `HandleCreateDriver()`
- `HandleCreateVehicle()`
- `HandleDeliveryArrive()`
- `HandleDeliveryBypass()`
- `HandleDeliveryCompatOK()`
- `HandleDeliveryShopClosed()`
- `HandleDriverDepart()`
- `HandleDriverLogin()`
- `HandleDriverReturnComplete()`
- `HandleEarnings()`
- `HandleFleetEarlyComplete()`
- `HandleFleetOrders()`
- `HandleFleetRouteReorder()`
- `HandleGetDriver()`
- `HandleGetVehicle()`
- `HandleHistory()`
- `HandleListDrivers()`
- `HandleListVehicles()`
- `HandleManifest()`
- `HandleManifestGate()`
- `HandleMarkNotificationsRead()`
- `HandleOrderAmend()`
- `HandleOrderCollectCash()`
- `HandleOrderComplete()`
- `HandleOrderConfirmOffload()`
- `HandleOrderDeliver()`
- `HandleOrderGet()`
- `HandleOrderStatePatch()`
- `HandleOrderValidateQR()`
- `HandlePendingCollections()`
- `HandleProfile()`
- `HandleRescueRequest()`
- `HandleRescueRespond()`
- `HandleRouteGeometry()`
- `HandleUpdateDriver()`
- `HandleUpdateVehicle()`
- `HandleUserNotifications()`
- `HandleWSAck()`
- `ListDrivers()`
- `ListVehicles()`
- `NewInMemoryRepository()`
- `NewService()`
- `NewSpannerRepository()`
- `RegisterRoutes()`
- `SetDispatchPlanInvalidate()`
- `SetFleetAvailabilityBroadcaster()`
- `UpdateDriver()`
- `UpdateVehicle()`

## 3. Registered API Routes
