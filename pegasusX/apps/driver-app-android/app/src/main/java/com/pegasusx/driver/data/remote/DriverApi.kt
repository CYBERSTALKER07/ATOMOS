package com.pegasusx.driver.data.remote

import com.pegasusx.driver.data.model.ArriveSupplyTransferRequest
import com.pegasusx.driver.data.model.ArriveSupplyTransferResponse
import com.pegasusx.driver.data.model.SupplyTransfersResponse
import com.pegasusx.driver.data.model.AmendOrderRequest
import com.pegasusx.driver.data.model.AmendOrderResponse
import com.pegasusx.driver.data.model.AuthResponse
import com.pegasusx.driver.data.model.AvailabilityRequest
import com.pegasusx.driver.data.model.ClientPolicyResponse
import com.pegasusx.driver.data.model.CashReconciliationRow
import com.pegasusx.driver.data.model.CashReconciliationsResponse
import com.pegasusx.driver.data.model.SubmitCashReconciliationRequest
import com.pegasusx.driver.data.model.CollectCashResponse
import com.pegasusx.driver.data.model.CompleteOrderRequest
import com.pegasusx.driver.data.model.ConfirmOffloadRequest
import com.pegasusx.driver.data.model.ConfirmOffloadResponse
import com.pegasusx.driver.data.model.DeliveryScanQRRequest
import com.pegasusx.driver.data.model.DeliveryScanQRResponse
import com.pegasusx.driver.data.model.DepartRequest
import com.pegasusx.driver.data.model.DeliverySubmitRequest
import com.pegasusx.driver.data.model.DeliverySubmitResponse
import com.pegasusx.driver.data.model.DriverEarningsResponse
import com.pegasusx.driver.data.model.DriverHistoryResponse
import com.pegasusx.driver.data.model.DriverProfileResponse
import com.pegasusx.driver.data.model.EarlyCompletePayload
import com.pegasusx.driver.data.model.EarlyCompleteRequestResponse
import com.pegasusx.driver.data.model.LoginRequest
import com.pegasusx.driver.data.model.ManifestGateResponse
import com.pegasusx.driver.data.model.MissingItemsPayload
import com.pegasusx.driver.data.model.MissingItemsResponse
// Quantity negotiation disabled ecosystem-wide — backend returns 410 feature_disabled.
import com.pegasusx.driver.data.model.OpenFiscalResponse
import com.pegasusx.driver.data.model.Order
import com.pegasusx.driver.data.model.PendingCollection
import com.pegasusx.driver.data.model.PulseResponse
import com.pegasusx.driver.data.model.RouteGeometryResponse
import com.pegasusx.driver.data.model.ReorderStopsRequest
import com.pegasusx.driver.data.model.ReturnCompleteRequest
import com.pegasusx.driver.data.model.ReturnGoodsResponse
import com.pegasusx.driver.data.model.RouteManifest
import com.pegasusx.driver.data.model.RouteReorderResponse
import com.pegasusx.driver.data.model.SplitPaymentPayload
import com.pegasusx.driver.data.model.SplitPaymentResponse
import com.pegasusx.driver.data.model.SyncBatchRequest
import com.pegasusx.driver.data.model.SyncBatchResponse
import com.pegasusx.driver.data.model.UpdateOrderDuringDeliveryRequest
import com.pegasusx.driver.data.model.UpdateOrderDuringDeliveryResponse
import com.pegasusx.driver.data.model.ValidateQRRequest
import com.pegasusx.driver.data.model.ValidateQRResponse
import com.pegasusx.driver.data.model.VerifyHandshakeRequest
import com.pegasusx.driver.data.model.VerifyHandshakeResponse

import com.pegasusx.driver.ui.screens.notifications.DriverNotificationsResponse
import retrofit2.Response
import retrofit2.http.Body
import retrofit2.http.GET
import retrofit2.http.Header
import retrofit2.http.PATCH
import retrofit2.http.POST
import retrofit2.http.Path
import retrofit2.http.Query

interface DriverApi {

    // Auth — Driver PIN login
    @POST("v1/auth/driver/login")
    suspend fun login(@Body request: LoginRequest): AuthResponse

    // Driver profile (polled every 60s for vehicle reassignment)
    @GET("v1/driver/profile")
    suspend fun getProfile(): DriverProfileResponse

    @GET("v1/driver/pulse")
    suspend fun getPulse(): PulseResponse

    // Driver hash manifest
    @GET("v1/driver/manifest")
    suspend fun getManifest(@Query("date") date: String): RouteManifest

    // Per-driver earnings report (lifetime totals + last 30 days)
    @GET("v1/driver/earnings")
    suspend fun getEarnings(): DriverEarningsResponse

    // Outstanding cash collections (PENDING_CASH_COLLECTION orders)
    @GET("v1/driver/pending-collections")
    suspend fun getPendingCollections(): List<PendingCollection>

    // Order details
    @GET("v1/orders/{id}")
    suspend fun getOrder(@Path("id") orderId: String): Order

    // Driver's assigned orders
    @GET("v1/fleet/orders")
    suspend fun getAssignedOrders(): List<Order>

    @GET("v1/fleet/route/{routeId}/geometry")
    suspend fun getRouteGeometry(
        @Path("routeId") routeId: String,
        @Query("include_steps") includeSteps: Boolean = true,
        @Query("from_lat") fromLat: Double? = null,
        @Query("from_lng") fromLng: Double? = null,
        @Query("reroute") reroute: Boolean? = null,
    ): RouteGeometryResponse

    // Delivery confirmation (QR verified)
    @POST("v1/order/deliver")
    suspend fun submitDelivery(
        @Body request: DeliverySubmitRequest,
        @Header("Idempotency-Key") idempotencyKey: String? = null
    ): DeliverySubmitResponse

    // Amend order — batch line-item reconciliation at delivery
    @POST("v1/order/amend")
    suspend fun amendOrder(
        @Body request: AmendOrderRequest,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): AmendOrderResponse

    // Validate QR token — returns order info for review
    @POST("v1/order/validate-qr")
    suspend fun validateQR(@Body request: ValidateQRRequest): ValidateQRResponse

    // Confirm offload — ARRIVED → AWAITING_PAYMENT (legacy; prefer scanDeliveryQR)
    @POST("v1/order/confirm-offload")
    suspend fun confirmOffload(
        @Body request: ConfirmOffloadRequest,
        @Header("Idempotency-Key") idempotencyKey: String? = null
    ): ConfirmOffloadResponse

    // Doorstep QR scan — ARRIVED → AWAITING_PAYMENT (canonical transition)
    @POST("v1/delivery/scan-qr")
    suspend fun scanDeliveryQR(
        @Body request: DeliveryScanQRRequest,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): DeliveryScanQRResponse

    // Complete order — capture → FISCALIZING (ADR-009); COMPLETED after fiscal SUCCESS
    @POST("v1/order/complete")
    suspend fun completeOrder(
        @Body request: CompleteOrderRequest,
        @Header("Idempotency-Key") idempotencyKey: String? = null
    ): Order

    // Collect cash — geofence capture → FISCALIZING (ADR-009)
    @POST("v1/order/collect-cash")
    suspend fun collectCash(
        @Body request: CollectCashRequest,
        @Header("Idempotency-Key") idempotencyKey: String? = null
    ): CollectCashResponse

    // Fiscal retry when FISCAL_FAILED
    @POST("v1/order/{orderId}/fiscal/retry")
    suspend fun retryFiscal(
        @Path("orderId") orderId: String,
        @Header("Idempotency-Key") idempotencyKey: String? = null
    ): CollectCashResponse

    // Transition order state
    @PATCH("v1/orders/{id}/state")
    suspend fun transitionState(
        @Path("id") orderId: String,
        @Body body: Map<String, String>,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): Order

    // Mark arrived — driver enters 500m geofence (IN_TRANSIT → ARRIVED)
    @POST("v1/delivery/arrive")
    suspend fun markArrived(
        @Body body: Map<String, String>,
        @Header("Idempotency-Key") idempotencyKey: String? = null
    ): Map<String, String>

    // Start Handshake for reassigned (partial/complete) orders
    @POST("v1/fleet/orders/{id}/reassign-handshake")
    suspend fun reassignHandshake(
        @Path("id") orderId: String,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Map<String, String>

    // Driver depart — starts route, transitions truck to IN_TRANSIT, triggers live ETA
    @POST("v1/fleet/driver/depart")
    suspend fun depart(
        @Body request: DepartRequest,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Map<String, String>

    // End session — go offline with reason code
    @POST("v1/driver/availability")
    suspend fun setAvailability(
        @Body request: AvailabilityRequest,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Map<String, String>

    // Read current availability
    @GET("v1/driver/availability")
    suspend fun getAvailability(): kotlinx.serialization.json.JsonObject

    // Partial update to availability
    @PATCH("v1/driver/availability")
    suspend fun updateAvailability(
        @Body body: Map<String, @JvmSuppressWildcards Any>,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Map<String, String>

    // Fetch driver history
    @GET("v1/driver/history")
    suspend fun getHistory(): DriverHistoryResponse

    // Fetch fleet manifest
    @GET("v1/fleet/manifest")
    suspend fun getFleetManifest(): kotlinx.serialization.json.JsonObject

    // Return complete — RETURNING → AVAILABLE after arriving at warehouse
    @GET("v1/driver/return-goods")
    suspend fun getReturnGoods(): ReturnGoodsResponse

    /** Phase 6: open fiscal count soft-freezes cash bag / shift-end. */
    @GET("v1/driver/open-fiscal")
    suspend fun getOpenFiscal(): OpenFiscalResponse

    @POST("v1/fleet/driver/return-complete")
    suspend fun returnComplete(
        @Body request: ReturnCompleteRequest,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Map<String, String>

    @GET("v1/driver/cash-reconciliations")
    suspend fun listCashReconciliations(): CashReconciliationsResponse

    @POST("v1/driver/cash-reconciliations")
    suspend fun submitCashReconciliation(
        @Body request: SubmitCashReconciliationRequest,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): CashReconciliationRow

    // Reorder stops — driver reorders their active route stops
    @POST("v1/fleet/route/reorder")
    suspend fun reorderStops(
        @Body request: ReorderStopsRequest,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Map<String, String>

    @POST("v1/user/device-token")
    suspend fun registerDeviceToken(@Body body: Map<String, String>): Map<String, String>

    // ── Notifications ──
    @GET("v1/user/notifications")
    suspend fun getNotifications(
        @Query("limit") limit: Int = 100,
        @Query("offset") offset: Int = 0,
    ): DriverNotificationsResponse

    @POST("v1/user/notifications/read")
    suspend fun markNotificationsRead(@Body body: Map<String, @JvmSuppressWildcards Any>): Map<String, String>

    @GET("v1/platform/client-policy")
    suspend fun getClientPolicy(
        @Query("role") role: String = "DRIVER",
        @Query("platform") platform: String,
        @Query("version") version: String,
        @Query("channel") channel: String = "production",
    ): ClientPolicyResponse

    @POST("v1/ws/ack")
    suspend fun ackWebSocketCommand(@Body body: Map<String, @JvmSuppressWildcards Any>): Map<String, String>

    // ── Shop-Closed Protocol ──

    // Driver reports shop is closed (ARRIVED → ARRIVED_SHOP_CLOSED ≡ SHOP_CLOSED_PENDING)
    @POST("v1/delivery/shop-closed")
    suspend fun reportShopClosed(
        @Body body: Map<String, String>,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Map<String, String>

    // Unlock cash/credit/split when physically at stop (H3 or ≤100m)
    @POST("v1/delivery/proximity-unlock")
    suspend fun proximityUnlock(
        @Body body: Map<String, @JvmSuppressWildcards Any>,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Map<String, @JvmSuppressWildcards Any>

    // Line-level partial offload (delivered_qty + remaining_qty == qty)
    @POST("v1/delivery/partial-offload")
    suspend fun partialOffload(
        @Body body: Map<String, @JvmSuppressWildcards Any>,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Map<String, @JvmSuppressWildcards Any>

    // Driver uses bypass token to complete offload without retailer QR
    @POST("v1/delivery/bypass-offload")
    suspend fun bypassOffload(
        @Body body: Map<String, String>,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Map<String, String>

    // Driver uses payment bypass token to complete when payment gateway is down
    @POST("v1/delivery/confirm-payment-bypass")
    suspend fun confirmPaymentBypass(
        @Body body: Map<String, String>,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Map<String, String>

    // ── v3.1 Human-Centric Edges ──

    // Edge 27: Request early route completion (fatigue/issue)
    @POST("v1/fleet/route/request-early-complete")
    suspend fun requestEarlyComplete(
        @Body body: EarlyCompletePayload,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): EarlyCompleteRequestResponse

    // Quantity negotiation disabled ecosystem-wide — backend returns 410 feature_disabled.
    // @POST("v1/delivery/negotiate")
    // suspend fun proposeNegotiation(@Body body: NegotiationPayload): NegotiationProposalResponse

    // Edge 32: Mark order as delivered on credit
    @POST("v1/delivery/credit-delivery")
    suspend fun markCreditDelivery(
        @Body body: Map<String, String>,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Map<String, String>

    // ── Rescue Operations ──

    @POST("v1/driver/ops/rescue/request")
    suspend fun requestRescue(
        @Body body: Map<String, String>,
    ): Map<String, String>

    @POST("v1/driver/ops/rescue/respond")
    suspend fun respondRescue(
        @Body body: Map<String, @JvmSuppressWildcards Any>,
    ): Map<String, String>

    // Edge 33: Report missing items after seal (exception-report alias; DAMAGED needs photo_url)
    @POST("v1/delivery/missing-items")
    suspend fun reportMissingItems(
        @Body body: MissingItemsPayload,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): MissingItemsResponse

    /** Signed GCS PUT ticket for claim/exception evidence photos. */
    @GET("v1/media/upload-ticket")
    suspend fun getMediaUploadTicket(
        @Query("purpose") purpose: String = "driver_exception",
        @Query("ext") ext: String = "jpg",
        @Query("order_id") orderId: String? = null,
    ): MediaUploadTicket

    // ── LEO: Ghost Stop Prevention ──

    // Check if manifest is sealed before allowing route start
    @GET("v1/driver/manifest-gate")
    suspend fun checkManifestGate(@Query("manifest_id") manifestId: String): Response<ManifestGateResponse>

    // Edge 35: Create split payment
    @POST("v1/delivery/split-payment")
    suspend fun splitPayment(
        @Body body: SplitPaymentPayload,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): SplitPaymentResponse

    // ── Dynamic Delivery Handshake ──

    // Dynamic Delivery Handshake Verification
    @POST("v1/delivery/verify-handshake")
    suspend fun verifyHandshake(@Body body: VerifyHandshakeRequest): VerifyHandshakeResponse

    // Dynamic Delivery Edge Handling (in-delivery updates)
    @POST("v1/delivery/update-order-during-delivery")
    suspend fun updateOrderDuringDelivery(@Body body: UpdateOrderDuringDeliveryRequest): UpdateOrderDuringDeliveryResponse

    /** Offline delivery batch upload — preferred path for verifier-queued deliveries. */
    @POST("v1/sync/batch")
    suspend fun syncBatch(
        @Body request: SyncBatchRequest,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): SyncBatchResponse

    // Factory-driver supply legs (factory → warehouse)
    @GET("v1/driver/supply-transfers")
    suspend fun getSupplyTransfers(): SupplyTransfersResponse

    @POST("v1/driver/supply-transfers/{id}/arrive")
    suspend fun arriveSupplyTransfer(
        @Path("id") transferId: String,
        @Header("Idempotency-Key") idempotencyKey: String,
        @Body body: ArriveSupplyTransferRequest,
    ): ArriveSupplyTransferResponse
}
