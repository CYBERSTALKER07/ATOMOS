package com.pegasus.warehouse.data.remote

import com.pegasus.warehouse.data.model.*
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

    @PATCH("v1/warehouse/ops/drivers/{id}/assign-vehicle")
    suspend fun assignDriverVehicle(
        @Path("id") id: String,
        @Body body: AssignVehicleRequest,
    ): Response<AssignVehicleResponse>

    // ── Vehicles ──
    @GET("v1/warehouse/ops/vehicles")
    suspend fun getVehicles(): Response<VehicleListResponse>

    @POST("v1/warehouse/ops/vehicles")
    suspend fun createVehicle(@Body body: CreateVehicleRequest): Response<Vehicle>

    @PATCH("v1/warehouse/ops/vehicles/{id}")
    suspend fun updateVehicle(
        @Path("id") id: String,
        @Body body: UpdateVehicleRequest,
    ): Response<VehicleMutationResponse>

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

    @GET("v1/warehouse/supply-requests")
    suspend fun getSupplyRequests(
        @Query("state") state: String? = null,
    ): Response<List<WarehouseSupplyRequest>>

    @POST("v1/warehouse/supply-requests")
    suspend fun createSupplyRequest(
        @Body body: CreateWarehouseSupplyRequestRequest,
    ): Response<CreateWarehouseSupplyRequestResponse>

    @PATCH("v1/warehouse/supply-requests/{id}")
    suspend fun transitionSupplyRequest(
        @Path("id") id: String,
        @Body body: WarehouseSupplyRequestTransitionRequest,
    ): Response<WarehouseSupplyRequestTransitionResponse>

    @GET("v1/warehouse/dispatch-locks")
    suspend fun getDispatchLocks(): Response<List<WarehouseDispatchLock>>

    @POST("v1/warehouse/dispatch-lock")
    suspend fun createDispatchLock(
        @Body body: CreateWarehouseDispatchLockRequest,
    ): Response<CreateWarehouseDispatchLockResponse>

    @DELETE("v1/warehouse/dispatch-lock")
    suspend fun releaseDispatchLock(
        @Query("lock_id") lockId: String,
    ): Response<ReleaseWarehouseDispatchLockResponse>

    // ── Staff ──
    @GET("v1/warehouse/ops/staff")
    suspend fun getStaff(): Response<StaffListResponse>

    @POST("v1/warehouse/ops/staff")
    suspend fun createStaff(@Body body: CreateStaffRequest): Response<CreateStaffResponse>

    // ── Payment Config ──
    @GET("v1/warehouse/ops/payment-config")
    suspend fun getPaymentConfig(): Response<PaymentConfigResponse>
}
