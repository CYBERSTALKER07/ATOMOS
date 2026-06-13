package com.pegasusx.warehouse.data.remote

import com.pegasusx.warehouse.data.model.*
import retrofit2.Response
import retrofit2.http.*

interface WarehouseApi {

    // ── Auth ──
    @POST("v1/auth/warehouse/login")
    suspend fun login(@Body body: LoginRequest): Response<AuthResponse>

    @POST("v1/auth/warehouse/refresh")
    suspend fun refreshToken(@Body body: RefreshTokenRequest): Response<AuthResponse>

    @POST("v1/warehouse/setup")
    suspend fun setupWarehouse(@Body body: com.google.gson.JsonElement): Response<com.google.gson.JsonElement>


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

    @POST("v1/warehouse/ops/dispatch/preview")
    suspend fun createDispatchPreview(@Body body: com.google.gson.JsonElement): Response<DispatchPreview>

    @POST("v1/warehouse/ops/dispatch/execute")
    suspend fun executeDispatch(@Body body: com.google.gson.JsonElement): Response<DispatchExecuteResponse>

    @GET("v1/warehouse/demand/forecast")
    suspend fun getDemandForecast(
        @Query("days") days: Int = 7,
    ): Response<DemandForecastResponse>

    @GET("v1/warehouse/supply-requests")
    suspend fun getSupplyRequests(
        @Query("state") state: String? = null,
    ): Response<SupplyRequestListResponse>

    @GET("v1/warehouse/supply-requests/{id}")
    suspend fun getSupplyRequest(@Path("id") id: String): Response<WarehouseSupplyRequest>

    @GET("v1/warehouse/ops/dispatch/settings")
    suspend fun getDispatchSettings(): Response<DispatchSettingsResponse>

    @PATCH("v1/warehouse/ops/dispatch/settings")
    suspend fun patchDispatchSettings(
        @Body body: DispatchSettingsPatchRequest,
    ): Response<Map<String, String>>

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
    suspend fun getDispatchLocks(): Response<DispatchLockListResponse>

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

    // ── P1-03 ops depth ──
    @POST("v1/warehouse/transfers/emergency")
    suspend fun emergencyTransfer(@Body body: EmergencyTransferRequest): Response<TransferMutationResponse>

    @POST("v1/warehouse/transfers/force-receive")
    suspend fun forceReceive(@Body body: ForceReceiveRequest): Response<TransferMutationResponse>

    @POST("v1/warehouse/transfers/{id}/receive")
    suspend fun receiveTransfer(@Path("id") transferId: String): Response<TransferMutationResponse>

    @GET("v1/warehouse/replenishment/insights")
    suspend fun getReplenishmentInsights(): Response<ReplenishmentInsightsResponse>

    @POST("v1/warehouse/replenishment/insights/{id}/{action}")
    suspend fun replenishmentInsightAction(
        @Path("id") insightId: String,
        @Path("action") action: String,
    ): Response<ReplenishmentInsightActionResponse>

    @GET("v1/warehouse/ops/financials")
    suspend fun getOpsFinancials(@Query("period") period: String? = null): Response<OpsFinancialsResponse>

    @POST("v1/warehouse/ops/orders/{id}/delay")
    suspend fun delayOrder(
        @Path("id") orderId: String,
        @Body body: WarehouseOrderMutationRequest = WarehouseOrderMutationRequest(),
    ): Response<WarehouseOrderMutationResponse>

    @POST("v1/warehouse/ops/orders/{id}/reject")
    suspend fun rejectOrder(
        @Path("id") orderId: String,
        @Body body: WarehouseOrderMutationRequest,
    ): Response<WarehouseOrderMutationResponse>

    @POST("v1/warehouse/ops/orders/{id}/overflow")
    suspend fun overflowOrder(
        @Path("id") orderId: String,
        @Body body: WarehouseOrderMutationRequest = WarehouseOrderMutationRequest(),
    ): Response<WarehouseOrderMutationResponse>

    @GET("v1/warehouse/ops/fleet/live-map")
    suspend fun getFleetLiveMap(
        @Query("warehouse_id") warehouseId: String? = null,
    ): Response<WarehouseFleetLiveMapResponse>
}
