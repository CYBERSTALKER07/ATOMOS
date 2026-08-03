# RETAILER PHASE - Codebase Status

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.


*Auto-generated directly from real codebase logic and algorithms.*

## 1. Domain Entities & Interfaces
- `AutoOrderSettings`
- `CartItem`
- `CartRepository (Interface)`
- `CategoryOverride`
- `Deps`
- `FamilyMember`
- `NotificationReader (Interface)`
- `OrderLifecycle (Interface)`
- `PerimeterSetStore (Interface)`
- `PerimeterSnapshot`
- `ProductOverride`
- `RegisterRequest`
- `RegisterResponse`
- `Repository (Interface)`
- `Retailer`
- `RetailerProximityConfig`
- `RetailerProximityService`
- `Service`
- `ServiceConfig`
- `SpannerCartRepository`
- `SpannerRepository`
- `SupplierOverride`
- `SupplierPricingRule`
- `TrackingEvent`
- `TrackingLineItem`
- `TrackingLocation`
- `TrackingOrder`
- `TrackingPaymentEvidence`
- `TrackingReceiptChargebackRecord`
- `TrackingReceiptDeliveryProof`
- `TrackingReceiptDossier`
- `TrackingReceiptGatewayWebhook`
- `TrackingReceiptPaymentRecord`
- `TrackingReceiptProofStatus`
- `TrackingReceiptReversalRecord`
- `VariantOverride`
- `fiscalSnap`
- `orderLineItemJSON`
- `retailerPricingRuleResponse`
- `retailerPricingSummary`
- `retailerProfileUpdateRequest`
- `spannerTxnBuffer`
- `supplierPreference`
- `trackingLocationLookup`
- `trackingReceiptPaymentRecordSnapshot`

## 2. Core Business Logic (Exported Functions)
- `BufferAudit()`
- `BufferOutbox()`
- `CellForCoordinate()`
- `ClearCart()`
- `CreateRetailer()`
- `Error()`
- `FindByPhone()`
- `GetRetailer()`
- `GetSupplierPricingRule()`
- `HandleAIPredictions()`
- `HandleAIPredictionsAlias()`
- `HandleAIPreorder()`
- `HandleAcceptDeliveryProposal()`
- `HandleActiveFulfillment()`
- `HandleAutoOrderPatch()`
- `HandleAutoOrderSettings()`
- `HandleCancelOrder()`
- `HandleCardCheckout()`
- `HandleCartSync()`
- `HandleCashCheckout()`
- `HandleConfirmAIOrder()`
- `HandleConfirmPreorder()`
- `HandleCorrectPrediction()`
- `HandleCreateOrder()`
- `HandleDeleteFamilyMember()`
- `HandleDetailedAnalytics()`
- `HandleDeviceToken()`
- `HandleEditPreorder()`
- `HandleExpensesAnalytics()`
- `HandleFamilyMembers()`
- `HandleMarkNotificationsRead()`
- `HandleMobileRegister()`
- `HandleOrders()`
- `HandleOrdersAlias()`
- `HandlePendingPayments()`
- `HandlePricingRule()`
- `HandleProfile()`
- `HandleRegister()`
- `HandleRejectAIOrder()`
- `HandleRejectDeliveryProposal()`
- `HandleRejectPreorder()`
- `HandleRequestCancel()`
- `HandleRetailerCardMutation()`
- `HandleRetailerCards()`
- `HandleRetailerLogin()`
- `HandleRetailerRefresh()`
- `HandleRetailerSetup()`
- `HandleShopClosedResponse()`
- `HandleSupplierAdd()`
- `HandleSupplierRemove()`
- `HandleSuppliers()`
- `HandleTracking()`
- `HandleUnifiedCheckout()`
- `HandleUserNotifications()`
- `IsRetailerInZone()`
- `ListByRetailer()`
- `ListRecentReceipts()`
- `ListRetailersBySupplier()`
- `ListTrackingOrders()`
- `NewRetailerProximityService()`
- `NewService()`
- `NewSpannerCartRepository()`
- `NewSpannerRepository()`
- `PerimeterReady()`
- `PrecomputeDeliveryZone()`
- `PrecomputeDeliveryZoneForCenter()`
- `Register()`
- `RegisterRoutes()`
- `SetOrderLifecycle()`
- `UpdateRetailer()`
- `UpsertItems()`
- `Validate()`

## 3. Registered API Routes
