package com.pegasus.retailer.data.api

import com.pegasus.retailer.data.model.ApiResponse
import com.pegasus.retailer.data.model.AuthResponse
import com.pegasus.retailer.data.model.UnifiedCheckoutRequest
import com.pegasus.retailer.data.model.UnifiedCheckoutResponse
import com.pegasus.retailer.data.model.DemandForecast
import com.pegasus.retailer.data.model.LoginRequest
import com.pegasus.retailer.data.model.Order
import com.pegasus.retailer.data.model.Product
import com.pegasus.retailer.data.model.ProductCategory
import com.pegasus.retailer.data.model.RegisterRequest
import com.pegasus.retailer.data.model.RetailerAnalytics
import com.pegasus.retailer.data.model.RetailerDetailedAnalytics
import com.pegasus.retailer.data.model.Supplier
import com.pegasus.retailer.data.model.TrackingResponse
import com.pegasus.retailer.ui.screens.notifications.NotificationsResponse
import com.pegasus.retailer.data.model.AutoOrderSettings
import com.pegasus.retailer.data.model.UpdateGlobalSettingsRequest
import com.pegasus.retailer.data.model.UpdateSettingsRequest
import retrofit2.http.Body
import retrofit2.http.DELETE
import retrofit2.http.GET
import retrofit2.http.PATCH
import retrofit2.http.POST
import retrofit2.http.Path
import retrofit2.http.Query

/**
 * Retrofit interface for the Lab backend.
 * Mirrors the iOS APIClient endpoint surface exactly.
 */
interface LabApi {

    // ── Auth ──
    @POST("/v1/auth/retailer/login")
    suspend fun login(@Body body: LoginRequest): AuthResponse

    @POST("/v1/auth/retailer/register")
    suspend fun register(@Body body: RegisterRequest): AuthResponse

    @POST("/v1/user/device-token")
    suspend fun registerDeviceToken(@Body body: Map<String, String>): ApiResponse

    // ── Orders ──
    @GET("/v1/retailers/{id}/orders")
    suspend fun getOrders(@Path("id") retailerId: String): List<Order>

    @POST("/v1/order/create")
    suspend fun createOrder(@Body body: Map<String, @JvmSuppressWildcards Any>): Order

    @POST("/v1/order/cancel")
    suspend fun cancelOrder(@Body body: Map<String, String>): ApiResponse

    // ── Catalog ──
    @GET("/v1/catalog/categories")
    suspend fun getCategories(): List<ProductCategory>

    @GET("/v1/catalog/products")
    suspend fun getCatalogProducts(
        @Query("category_id") categoryId: String? = null,
        @Query("supplier_id") supplierId: String? = null,
    ): List<Product>

    @GET("/v1/catalog/categories/{id}/suppliers")
    suspend fun getCategorySuppliers(@Path("id") categoryId: String): List<Supplier>

    // ── Retailer Suppliers ──
    @GET("/v1/retailer/suppliers")
    suspend fun getMySuppliers(): List<Supplier>

    @POST("/v1/retailer/suppliers/{id}/add")
    suspend fun addSupplier(@Path("id") supplierId: String): ApiResponse

    @POST("/v1/retailer/suppliers/{id}/remove")
    suspend fun removeSupplier(@Path("id") supplierId: String): ApiResponse

    // ── AI / Predictions ──
    @POST("/v1/ai/preorder")
    suspend fun aiPreorder(@Body body: Map<String, @JvmSuppressWildcards Any>): ApiResponse

    @GET("/v1/ai/predictions")
    suspend fun getPredictions(@Query("retailer_id") retailerId: String): List<DemandForecast>

    @PATCH("/v1/ai/predictions/correct")
    suspend fun correctPrediction(
        @Query("prediction_id") predictionId: String,
        @Body body: Map<String, @JvmSuppressWildcards Any>,
    ): ApiResponse

    // ── Retailer Profile ──
    @GET("/v1/retailer/profile")
    suspend fun getRetailerProfile(): Map<String, String>

    // ── Analytics ──
    @GET("/v1/retailer/analytics/expenses")
    suspend fun getRetailerExpenses(): RetailerAnalytics

    @GET("/v1/retailer/analytics/detailed")
    suspend fun getRetailerDetailedAnalytics(
        @Query("from") from: String? = null,
        @Query("to") to: String? = null,
    ): RetailerDetailedAnalytics

    // ── Checkout ──
    @POST("/v1/checkout/unified")
    suspend fun unifiedCheckout(@Body body: UnifiedCheckoutRequest): UnifiedCheckoutResponse

    // ── Post-Offload Payment ──
    @POST("/v1/order/cash-checkout")
    suspend fun cashCheckout(@Body body: Map<String, String>): Map<String, @JvmSuppressWildcards Any>

    @POST("/v1/order/card-checkout")
    suspend fun cardCheckout(@Body body: Map<String, String>): Map<String, @JvmSuppressWildcards Any>

    // ── Empathy Engine Settings ──
    // ── Active Fulfillment ──
    @GET("/v1/retailer/active-fulfillment")
    suspend fun getActiveFulfillments(): TrackingResponse

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

    // ── Delivery Tracking ──
    @GET("/v1/retailer/tracking")
    suspend fun getTrackingOrders(): TrackingResponse

    // ── Notifications ──
    @GET("/v1/user/notifications")
    suspend fun getNotifications(@Query("limit") limit: Int = 50): NotificationsResponse

    @POST("/v1/user/notifications/read")
    suspend fun markNotificationsRead(@Body body: Map<String, @JvmSuppressWildcards Any>): ApiResponse
}
