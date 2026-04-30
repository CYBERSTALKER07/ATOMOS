package com.pegasus.driver.data.remote

import com.pegasus.driver.data.model.AmendOrderRequest
import com.pegasus.driver.data.model.AmendOrderResponse
import com.pegasus.driver.data.model.AuthResponse
import com.pegasus.driver.data.model.AvailabilityRequest
import com.pegasus.driver.data.model.CollectCashRequest
import com.pegasus.driver.data.model.CollectCashResponse
import com.pegasus.driver.data.model.CompleteOrderRequest
import com.pegasus.driver.data.model.ConfirmOffloadRequest
import com.pegasus.driver.data.model.ConfirmOffloadResponse
import com.pegasus.driver.data.model.DepartRequest
import com.pegasus.driver.data.model.DeliverySubmitRequest
import com.pegasus.driver.data.model.DeliverySubmitResponse
import com.pegasus.driver.data.model.DriverProfileResponse
import com.pegasus.driver.data.model.LoginRequest
import com.pegasus.driver.data.model.Order
import com.pegasus.driver.data.model.ReorderStopsRequest
import com.pegasus.driver.data.model.ReturnCompleteRequest
import com.pegasus.driver.data.model.RouteManifest
import com.pegasus.driver.data.model.ValidateQRRequest
import com.pegasus.driver.data.model.ValidateQRResponse
import com.pegasus.driver.ui.screens.notifications.DriverNotificationsResponse
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

    // Fleet manifest
    @GET("v1/fleet/manifest")
    suspend fun getManifest(@Query("date") date: String): RouteManifest

    // Order details
    @GET("v1/orders/{id}")
    suspend fun getOrder(@Path("id") orderId: String): Order

    // Driver's assigned orders
    @GET("v1/fleet/orders")
    suspend fun getAssignedOrders(): List<Order>

    // Delivery confirmation (QR verified)
    @POST("v1/order/deliver")
    suspend fun submitDelivery(
        @Body request: DeliverySubmitRequest,
        @Header("Idempotency-Key") idempotencyKey: String? = null
    ): DeliverySubmitResponse

    // Amend order — batch line-item reconciliation at delivery
    @POST("v1/order/amend")
    suspend fun amendOrder(@Body request: AmendOrderRequest): AmendOrderResponse

    // Validate QR token — returns order info for review
    @POST("v1/order/validate-qr")
    suspend fun validateQR(@Body request: ValidateQRRequest): ValidateQRResponse

    // Confirm offload — ARRIVED → AWAITING_PAYMENT, triggers retailer payment
    @POST("v1/order/confirm-offload")
    suspend fun confirmOffload(@Body request: ConfirmOffloadRequest): ConfirmOffloadResponse

    // Complete order — AWAITING_PAYMENT → COMPLETED after payment settled
    @POST("v1/order/complete")
    suspend fun completeOrder(@Body request: CompleteOrderRequest): Order

    // Collect cash — PENDING_CASH_COLLECTION → COMPLETED with geofence validation
    @POST("v1/order/collect-cash")
    suspend fun collectCash(
        @Body request: CollectCashRequest,
        @Header("Idempotency-Key") idempotencyKey: String? = null
    ): CollectCashResponse

    // Transition order state
    @PATCH("v1/orders/{id}/state")
    suspend fun transitionState(
        @Path("id") orderId: String,
        @Body body: Map<String, String>
    ): Order

    // Mark arrived — driver enters 100m geofence (IN_TRANSIT → ARRIVED)
    @POST("v1/delivery/arrive")
    suspend fun markArrived(@Body body: Map<String, String>): Map<String, String>

    // Driver depart — starts route, transitions truck to IN_TRANSIT, triggers live ETA
    @POST("v1/fleet/driver/depart")
    suspend fun depart(@Body request: DepartRequest): Map<String, String>

    // End session — go offline with reason code
    @POST("v1/driver/availability")
    suspend fun setAvailability(@Body request: AvailabilityRequest): Map<String, String>

    // Return complete — RETURNING → AVAILABLE after arriving at warehouse
    @POST("v1/fleet/driver/return-complete")
    suspend fun returnComplete(@Body request: ReturnCompleteRequest): Map<String, String>

    // Reorder stops — driver reorders their active route stops
    @POST("v1/fleet/route/reorder")
    suspend fun reorderStops(@Body request: ReorderStopsRequest): Map<String, String>

    // ── Notifications ──
    @GET("v1/user/notifications")
    suspend fun getNotifications(@Query("limit") limit: Int = 50): DriverNotificationsResponse

    @POST("v1/user/notifications/read")
    suspend fun markNotificationsRead(@Body body: Map<String, @JvmSuppressWildcards Any>): Map<String, String>

    // ── Shop-Closed Protocol ──

    // Driver reports shop is closed (ARRIVED → ARRIVED_SHOP_CLOSED)
    @POST("v1/delivery/shop-closed")
    suspend fun reportShopClosed(@Body body: Map<String, String>): Map<String, String>

    // Driver uses bypass token to complete offload without retailer QR
    @POST("v1/delivery/bypass-offload")
    suspend fun bypassOffload(@Body body: Map<String, String>): Map<String, String>

    // Driver uses payment bypass token to complete when payment gateway is down
    @POST("v1/delivery/confirm-payment-bypass")
    suspend fun confirmPaymentBypass(@Body body: Map<String, String>): Map<String, String>

    // ── v3.1 Human-Centric Edges ──

    // Edge 27: Request early route completion (fatigue/issue)
    @POST("v1/fleet/route/request-early-complete")
    suspend fun requestEarlyComplete(@Body body: Map<String, String>): Map<String, @JvmSuppressWildcards Any>

    // Edge 28: Propose quantity negotiation to supplier
    @POST("v1/delivery/negotiate")
    suspend fun proposeNegotiation(@Body body: Map<String, @JvmSuppressWildcards Any>): Map<String, @JvmSuppressWildcards Any>

    // Edge 32: Mark order as delivered on credit
    @POST("v1/delivery/credit-delivery")
    suspend fun markCreditDelivery(@Body body: Map<String, String>): Map<String, String>

    // Edge 33: Report missing items after seal
    @POST("v1/delivery/missing-items")
    suspend fun reportMissingItems(@Body body: Map<String, @JvmSuppressWildcards Any>): Map<String, @JvmSuppressWildcards Any>

    // ── LEO: Ghost Stop Prevention ──

    // Check if manifest is sealed before allowing route start
    @GET("v1/driver/manifest-gate")
    suspend fun checkManifestGate(@Query("manifest_id") manifestId: String): Map<String, @JvmSuppressWildcards Any>

    // Edge 35: Create split payment
    @POST("v1/delivery/split-payment")
    suspend fun splitPayment(@Body body: Map<String, @JvmSuppressWildcards Any>): Map<String, @JvmSuppressWildcards Any>
}
