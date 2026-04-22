package com.thelab.warehouse.data.remote

import com.thelab.warehouse.data.model.*
import retrofit2.Response
import retrofit2.http.*

interface WarehouseApi {

    // ── Auth ──
    @POST("v1/auth/warehouse/login")
    suspend fun login(@Body body: LoginRequest): Response<AuthResponse>

    @POST("v1/auth/warehouse/refresh")
    suspend fun refreshToken(): Response<AuthResponse>

    // ── Dashboard ──
    @GET("v1/warehouse/ops/dashboard")
    suspend fun getDashboard(): Response<DashboardData>

    // ── Orders ──
    @GET("v1/warehouse/ops/orders")
    suspend fun getOrders(
        @Query("state") state: String? = null,
        @Query("date") date: String? = null,
    ): Response<OrderListResponse>

    @GET("v1/warehouse/ops/orders/{id}")
    suspend fun getOrder(@Path("id") id: String): Response<Order>

    // ── Drivers ──
    @GET("v1/warehouse/ops/drivers")
    suspend fun getDrivers(): Response<DriverListResponse>

    @POST("v1/warehouse/ops/drivers")
    suspend fun createDriver(@Body body: CreateDriverRequest): Response<CreateDriverResponse>

    // ── Vehicles ──
    @GET("v1/warehouse/ops/vehicles")
    suspend fun getVehicles(): Response<VehicleListResponse>

    @POST("v1/warehouse/ops/vehicles")
    suspend fun createVehicle(@Body body: CreateVehicleRequest): Response<Vehicle>

    // ── Inventory ──
    @GET("v1/warehouse/ops/inventory")
    suspend fun getInventory(
        @Query("search") search: String? = null,
        @Query("low_stock") lowStock: Boolean? = null,
    ): Response<InventoryListResponse>

    @PATCH("v1/warehouse/ops/inventory")
    suspend fun adjustInventory(@Body body: InventoryAdjustRequest): Response<Unit>

    // ── Products ──
    @GET("v1/warehouse/ops/products")
    suspend fun getProducts(
        @Query("search") search: String? = null,
    ): Response<ProductListResponse>

    // ── Manifests ──
    @GET("v1/warehouse/ops/manifests")
    suspend fun getManifests(
        @Query("date") date: String? = null,
    ): Response<ManifestListResponse>

    // ── Analytics ──
    @GET("v1/warehouse/ops/analytics")
    suspend fun getAnalytics(
        @Query("period") period: String = "30d",
    ): Response<AnalyticsData>

    // ── CRM ──
    @GET("v1/warehouse/ops/crm")
    suspend fun getRetailers(): Response<RetailerListResponse>

    // ── Returns ──
    @GET("v1/warehouse/ops/returns")
    suspend fun getReturns(): Response<ReturnListResponse>

    // ── Treasury ──
    @GET("v1/warehouse/ops/treasury")
    suspend fun getTreasuryOverview(
        @Query("view") view: String = "overview",
    ): Response<TreasuryOverview>

    @GET("v1/warehouse/ops/treasury")
    suspend fun getInvoices(
        @Query("view") view: String = "invoices",
    ): Response<InvoiceListResponse>

    // ── Dispatch ──
    @GET("v1/warehouse/ops/dispatch/preview")
    suspend fun getDispatchPreview(): Response<DispatchPreview>

    // ── Staff ──
    @GET("v1/warehouse/ops/staff")
    suspend fun getStaff(): Response<StaffListResponse>

    @POST("v1/warehouse/ops/staff")
    suspend fun createStaff(@Body body: CreateStaffRequest): Response<CreateStaffResponse>

    // ── Payment Config ──
    @GET("v1/warehouse/ops/payment-config")
    suspend fun getPaymentConfig(): Response<PaymentConfigResponse>
}
