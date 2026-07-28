package com.pegasusx.retailer.data.api

import com.pegasusx.retailer.data.model.ApiResponse
import com.pegasusx.retailer.data.model.ActiveFulfillmentsResponse
import com.pegasusx.retailer.data.model.AuthResponse
import com.pegasusx.retailer.data.model.AutoOrderSettings
import com.pegasusx.retailer.data.model.CardCheckoutRequest
import com.pegasusx.retailer.data.model.CheckoutQuoteRequest
import com.pegasusx.retailer.data.model.CheckoutQuoteResponse
import com.pegasusx.retailer.data.model.CheckoutPreviewResponse
import com.pegasusx.retailer.data.model.CardCheckoutResponse
import com.pegasusx.retailer.data.model.ConfirmCashRequest
import com.pegasusx.retailer.data.model.ConfirmCashResponse
import com.pegasusx.retailer.data.model.CashCheckoutRequest
import com.pegasusx.retailer.data.model.CashCheckoutResponse
import com.pegasusx.retailer.data.model.DemandForecast
import com.pegasusx.retailer.data.model.FileClaimRequestBody
import com.pegasusx.retailer.data.model.LoginRequest
import com.pegasusx.retailer.data.model.MediaUploadTicket
import com.pegasusx.retailer.data.model.Order
import com.pegasusx.retailer.data.model.OrderTimelineResponse
import com.pegasusx.retailer.data.model.PendingPaymentsResponse
import com.pegasusx.retailer.data.model.Product
import com.pegasusx.retailer.data.model.ProductCategory
import com.pegasusx.retailer.data.model.ProcurementOrderRequest
import com.pegasusx.retailer.data.model.ProcurementOrderResponse
import com.pegasusx.retailer.data.model.RegisterRequest
import com.pegasusx.retailer.data.model.ResolvedLocationResponse
import com.pegasusx.retailer.data.model.RetailerAnalytics
import com.pegasusx.retailer.data.model.RetailerClaim
import com.pegasusx.retailer.data.model.RetailerClaimsListResponse
import com.pegasusx.retailer.data.model.RetailerDetailedAnalytics
import com.pegasusx.retailer.data.model.Supplier
import com.pegasusx.retailer.data.model.TrackingResponse
import com.pegasusx.retailer.data.model.UnifiedCheckoutRequest
import com.pegasusx.retailer.data.model.UnifiedCheckoutResponse
import com.pegasusx.retailer.data.model.UpdateGlobalSettingsRequest
import com.pegasusx.retailer.data.model.UpdateSettingsRequest
import com.pegasusx.retailer.ui.screens.notifications.NotificationsResponse
import kotlinx.serialization.json.JsonElement
import retrofit2.http.Body
import retrofit2.http.DELETE
import retrofit2.http.GET
import retrofit2.http.Header
import retrofit2.http.PATCH
import retrofit2.http.POST
import retrofit2.http.PUT
import retrofit2.http.Path
import retrofit2.http.Query

/**
 * Retrofit interface for the Pegasus backend.
 * Mirrors the iOS APIClient endpoint surface exactly.
 */
interface PegasusApi {

    // ── Auth ──
    @POST("/v1/auth/retailer/login")
    suspend fun login(@Body body: LoginRequest): AuthResponse

    @POST("/v1/auth/retailer/register")
    suspend fun register(@Body body: RegisterRequest): AuthResponse

    @GET("/v1/platform/geocode/reverse")
    suspend fun reverseGeocode(
        @Query("lat") lat: Double,
        @Query("lng") lng: Double,
    ): ResolvedLocationResponse

    @POST("/v1/user/device-token")
    suspend fun registerDeviceToken(@Body body: Map<String, String>): ApiResponse

    // ── Orders ──
    @GET("/v1/retailers/{id}/orders")
    suspend fun getOrders(@Path("id") retailerId: String): List<Order>

    @POST("/v1/order/create")
    suspend fun createOrder(
        @Body body: ProcurementOrderRequest,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): ProcurementOrderResponse

    @POST("/v1/order/cancel")
    suspend fun cancelOrder(
        @Body body: Map<String, String>,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): ApiResponse

    @POST("/v1/orders/request-cancel")
    suspend fun requestCancelOrder(
        @Body body: Map<String, @JvmSuppressWildcards Any>,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): JsonElement

    // ── Claims (post-delivery, 48h window) ──
    @GET("/v1/orders/{orderId}/claims")
    suspend fun listOrderClaims(@Path("orderId") orderId: String): RetailerClaimsListResponse

    @POST("/v1/orders/{orderId}/claims")
    suspend fun fileOrderClaim(
        @Path("orderId") orderId: String,
        @Body body: FileClaimRequestBody,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): RetailerClaim

    @GET("/v1/media/upload-ticket")
    suspend fun getMediaUploadTicket(
        @Query("purpose") purpose: String = "claim_evidence",
        @Query("ext") ext: String = "jpg",
        @Query("order_id") orderId: String? = null,
    ): MediaUploadTicket

    // ── Catalog ──
    @GET("/v1/catalog/categories")
    suspend fun getCategories(): List<ProductCategory>

    @GET("/v1/catalog/products")
    suspend fun getCatalogProducts(
        @Query("category_id") categoryId: String? = null,
        @Query("supplier_id") supplierId: String? = null,
        @Query("retailer_id") retailerId: String? = null,
        @Query("limit") limit: Int? = null,
        @Query("offset") offset: Int? = null,
    ): List<Product>

    @POST("/v1/retailer/checkout/quote")
    suspend fun checkoutQuote(@Body body: CheckoutQuoteRequest): CheckoutQuoteResponse

    @POST("/v1/retailer/promotions/watch")
    suspend fun watchSupplierPromotions(@Body body: Map<String, String>)

    @GET("/v1/catalog/categories/{id}/suppliers")
    suspend fun getCategorySuppliers(@Path("id") categoryId: String): List<Supplier>

    @GET("/v1/catalog/suppliers/search")
    suspend fun searchSuppliers(@Query("q") query: String): List<Supplier>

    // ── Retailer Suppliers ──
    @GET("/v1/retailer/suppliers")
    suspend fun getMySuppliers(): List<Supplier>

    @POST("/v1/retailer/suppliers/{id}/add")
    suspend fun addSupplier(
        @Path("id") supplierId: String,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): ApiResponse

    @POST("/v1/retailer/suppliers/{id}/remove")
    suspend fun removeSupplier(
        @Path("id") supplierId: String,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): ApiResponse

    // ── AI / Predictions ──
    @POST("/v1/ai/preorder")
    suspend fun aiPreorder(@Body body: Map<String, @JvmSuppressWildcards Any>): ApiResponse

    @GET("/v1/ai/predictions")
    suspend fun getPredictions(@Query("retailer_id") retailerId: String): List<DemandForecast>

    @PATCH("/v1/ai/predictions/correct")
    suspend fun correctPrediction(
        @Query("prediction_id") predictionId: String,
        @Body body: Map<String, @JvmSuppressWildcards Any>,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): ApiResponse

    // ── Retailer Profile & Setup ──
    @POST("/v1/retailer/setup")
    suspend fun setupRetailer(
        @Body body: Map<String, @JvmSuppressWildcards Any>,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): JsonElement

    @GET("/v1/retailer/pricing/rules")
    suspend fun getPricingRules(): JsonElement

    @GET("/v1/retailer/profile")
    suspend fun getRetailerProfile(): Map<String, String>

    @PUT("/v1/retailer/profile")
    suspend fun updateRetailerProfile(
        @Body body: Map<String, String>,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): ApiResponse

    @GET("/v1/retailer/family-members")
    suspend fun getFamilyMembers(): JsonElement

    @POST("/v1/retailer/family-members")
    suspend fun createFamilyMember(
        @Body body: Map<String, @JvmSuppressWildcards Any>,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): JsonElement

    @DELETE("/v1/retailer/family-members/{memberID}")
    suspend fun deleteFamilyMember(
        @Path("memberID") memberId: String,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): JsonElement

    // ── Analytics ──
    @GET("/v1/retailer/analytics/expenses")
    suspend fun getRetailerExpenses(): RetailerAnalytics

    @GET("/v1/retailer/analytics/detailed")
    suspend fun getRetailerDetailedAnalytics(
        @Query("from") from: String? = null,
        @Query("to") to: String? = null,
    ): RetailerDetailedAnalytics

    // ── Checkout ──
    @POST("/v1/checkout/preview")
    suspend fun checkoutPreview(@Body body: UnifiedCheckoutRequest): CheckoutPreviewResponse

    @POST("/v1/checkout/unified")
    suspend fun unifiedCheckout(
        @Body body: UnifiedCheckoutRequest,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): UnifiedCheckoutResponse

    // ── Post-Offload Payment ──
    @POST("/v1/order/cash-checkout")
    suspend fun cashCheckout(
        @Body body: CashCheckoutRequest,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): CashCheckoutResponse

    @POST("/v1/delivery/confirm-cash")
    suspend fun confirmCash(
        @Body body: ConfirmCashRequest,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): ConfirmCashResponse

    @POST("/v1/order/card-checkout")
    suspend fun cardCheckout(
        @Body body: CardCheckoutRequest,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): CardCheckoutResponse

    @POST("/v1/retailer/shop-closed-response")
    suspend fun shopClosedResponse(
        @Body body: Map<String, @JvmSuppressWildcards Any>,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): JsonElement

    @POST("/v1/retailer/orders/confirm-ai")
    suspend fun confirmAiOrder(
        @Body body: Map<String, @JvmSuppressWildcards Any>,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): JsonElement

    @POST("/v1/retailer/orders/reject-ai")
    suspend fun rejectAiOrder(
        @Body body: Map<String, @JvmSuppressWildcards Any>,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): JsonElement

    @POST("/v1/orders/edit-preorder")
    suspend fun editPreorder(
        @Body body: Map<String, @JvmSuppressWildcards Any>,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): JsonElement

    @POST("/v1/orders/confirm-preorder")
    suspend fun confirmPreorder(
        @Body body: Map<String, @JvmSuppressWildcards Any>,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): JsonElement

    @POST("/v1/orders/accept-delivery-proposal")
    suspend fun acceptDeliveryProposal(
        @Body body: Map<String, String>,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): JsonElement

    @POST("/v1/orders/reject-delivery-proposal")
    suspend fun rejectDeliveryProposal(
        @Body body: Map<String, @JvmSuppressWildcards Any>,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): JsonElement

    @POST("/v1/orders/reject-preorder")
    suspend fun rejectPreorder(
        @Body body: Map<String, @JvmSuppressWildcards Any>,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): JsonElement

    // ── Empathy Engine Settings ──
    // ── Active Fulfillment ──
    @GET("/v1/retailer/active-fulfillment")
    suspend fun getActiveFulfillments(): ActiveFulfillmentsResponse

    @GET("/v1/retailer/pending-payments")
    suspend fun getPendingPayments(): PendingPaymentsResponse

    @GET("/v1/retailer/cart/sync")
    suspend fun getCartSync(): JsonElement

    @POST("/v1/retailer/cart/sync")
    suspend fun postCartSync(
        @Body body: Map<String, @JvmSuppressWildcards Any>,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): JsonElement

    @GET("/v1/retailer/settings/auto-order")
    suspend fun getAutoOrderSettings(): AutoOrderSettings

    @PATCH("/v1/retailer/settings/auto-order/global")
    suspend fun updateGlobalAutoOrder(@Body body: UpdateGlobalSettingsRequest): ApiResponse

    @PATCH("/v1/retailer/settings/auto-order/supplier/{id}")
    suspend fun updateSupplierAutoOrder(
        @Path("id") supplierId: String,
        @Body body: UpdateSettingsRequest,
    ): ApiResponse

    @PATCH("/v1/retailer/settings/auto-order/category/{id}")
    suspend fun updateCategoryAutoOrder(
        @Path("id") categoryId: String,
        @Body body: UpdateSettingsRequest,
    ): ApiResponse

    @PATCH("/v1/retailer/settings/auto-order/product/{id}")
    suspend fun updateProductAutoOrder(
        @Path("id") productId: String,
        @Body body: UpdateSettingsRequest,
    ): ApiResponse

    @PATCH("/v1/retailer/settings/auto-order/variant/{id}")
    suspend fun updateVariantAutoOrder(
        @Path("id") skuId: String,
        @Body body: UpdateSettingsRequest,
    ): ApiResponse

    @GET("/v1/order/{orderId}/timeline")
    suspend fun getOrderTimeline(@Path("orderId") orderId: String): OrderTimelineResponse

    // ── Delivery Tracking ──
    @GET("/v1/retailer/tracking")
    suspend fun getTrackingOrders(): TrackingResponse

    @POST("/v1/retailer/card/initiate")
    suspend fun initiateCard(
        @Body body: Map<String, @JvmSuppressWildcards Any>,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): JsonElement

    @POST("/v1/retailer/card/confirm")
    suspend fun confirmCard(
        @Body body: Map<String, @JvmSuppressWildcards Any>,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): JsonElement

    @GET("/v1/retailer/cards")
    suspend fun getCards(): JsonElement

    @POST("/v1/retailer/card/deactivate")
    suspend fun deactivateCard(
        @Body body: Map<String, @JvmSuppressWildcards Any>,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): JsonElement

    @POST("/v1/retailer/card/default")
    suspend fun setDefaultCard(
        @Body body: Map<String, @JvmSuppressWildcards Any>,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): JsonElement

    // ── Notifications ──
    @GET("/v1/user/notifications")
    suspend fun getNotifications(
        @Query("limit") limit: Int = 100,
        @Query("offset") offset: Int = 0,
    ): NotificationsResponse

    @POST("/v1/user/notifications/read")
    suspend fun markNotificationsRead(@Body body: Map<String, @JvmSuppressWildcards Any>): ApiResponse

    @GET("/v1/platform/client-policy")
    suspend fun getClientPolicy(
        @Query("role") role: String = "RETAILER",
        @Query("platform") platform: String,
        @Query("version") version: String,
        @Query("channel") channel: String = "production",
    ): JsonElement
}
