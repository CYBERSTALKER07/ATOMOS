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
    suspend fun setupWarehouse(@Body body: com.pegasusx.warehouse.data.model.WarehouseSetupRequest): Response<com.pegasusx.warehouse.data.model.WarehouseSetupResponse>


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

    @GET("v1/warehouse/orders/{orderId}/receipt")
    suspend fun getOrderReceipt(
        @Path("orderId") orderId: String,
        @Query("format") format: String = "json",
    ): Response<OrderReceiptMeta>

    @POST("v1/warehouse/recommend-reassign")
    suspend fun recommendReassign(
        @Body req: RecommendReassignRequest,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): Response<RecommendReassignResponse>

    @POST("v1/warehouse/reassign-order")
    suspend fun reassignOrder(
        @Body req: ReassignOrderRequest,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): Response<StatusResponse>

    // ── Drivers ──
    @GET("v1/warehouse/ops/drivers")
    suspend fun getDrivers(): Response<DriverListResponse>

    @POST("v1/warehouse/ops/drivers")
    suspend fun createDriver(
        @Body body: CreateDriverRequest,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Response<CreateDriverResponse>

    @PATCH("v1/warehouse/ops/drivers/{id}/assign-vehicle")
    suspend fun assignDriverVehicle(
        @Path("id") id: String,
        @Body body: AssignVehicleRequest,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Response<AssignVehicleResponse>

    // ── Vehicles ──
    @GET("v1/warehouse/ops/vehicles")
    suspend fun getVehicles(): Response<VehicleListResponse>

    @GET("v1/warehouse/ops/vehicles/{id}")
    suspend fun getVehicle(@Path("id") id: String): Response<VehicleDetailResponse>

    @POST("v1/warehouse/ops/vehicles")
    suspend fun createVehicle(
        @Body body: CreateVehicleRequest,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Response<Vehicle>

    @PATCH("v1/warehouse/ops/vehicles/{id}")
    suspend fun updateVehicle(
        @Path("id") id: String,
        @Body body: UpdateVehicleRequest,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Response<VehicleMutationResponse>

    // ── Inventory ──
    @GET("v1/warehouse/ops/inventory")
    suspend fun getInventory(
        @Query("search") search: String? = null,
        @Query("low_stock") lowStock: Boolean? = null,
    ): Response<InventoryListResponse>

    @PATCH("v1/warehouse/ops/inventory")
    suspend fun adjustInventory(
        @Body body: InventoryAdjustRequest,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Response<Unit>

    @PATCH("v1/warehouse/ops/inventory/{productId}/policy")
    suspend fun patchInventoryPolicy(
        @Path("productId") productId: String,
        @Body body: InventoryPolicyPatchRequest,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Response<Unit>

    @GET("v1/warehouse/ops/bins")
    suspend fun listBins(): Response<WarehouseBinListResponse>

    @POST("v1/warehouse/ops/bins")
    suspend fun createBin(
        @Body body: WarehouseBinCreateRequest,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Response<WarehouseBinLocation>

    @POST("v1/warehouse/ops/lots/putaway")
    suspend fun putawayLot(
        @Body body: StockLotPutawayRequest,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Response<StockLotPutawayResponse>

    @GET("v1/warehouse/ops/pick-waves")
    suspend fun listPickWaves(
        @Query("manifest_id") manifestId: String? = null,
        @Query("status") status: String? = null,
    ): Response<PickWaveListResponse>

    @POST("v1/warehouse/ops/pick-waves")
    suspend fun createPickWave(
        @Body body: PickWaveCreateRequest,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Response<PickWave>

    @GET("v1/warehouse/ops/pick-waves/{waveId}")
    suspend fun getPickWave(
        @Path("waveId") waveId: String,
    ): Response<PickWave>

    @POST("v1/warehouse/ops/pick-waves/{waveId}/tasks/{taskId}/confirm")
    suspend fun confirmPickTask(
        @Path("waveId") waveId: String,
        @Path("taskId") taskId: String,
        @Body body: PickTaskConfirmRequest,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Response<PickWave>

    @GET("v1/warehouse/ops/cycle-counts")
    suspend fun listCycleCounts(
        @Query("status") status: String? = null,
    ): Response<CycleCountListResponse>

    @POST("v1/warehouse/ops/cycle-counts")
    suspend fun createCycleCount(
        @Body body: CycleCountCreateRequest,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Response<CycleCount>

    @POST("v1/warehouse/ops/cycle-counts/{countId}/submit")
    suspend fun submitCycleCount(
        @Path("countId") countId: String,
        @Body body: CycleCountSubmitRequest,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Response<CycleCount>

    @GET("v1/warehouse/ops/settings")
    suspend fun getOpsSettings(): Response<WarehouseOpsSettingsResponse>

    @PATCH("v1/warehouse/ops/settings")
    suspend fun patchOpsSettings(
        @Body body: WarehouseOpsSettingsPatchRequest,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Response<Map<String, String>>

    @GET("v1/warehouse/return-policy")
    suspend fun getReturnPolicy(
        @Query("warehouse_id") warehouseId: String? = null,
        @Query("supplier_id") supplierId: String? = null,
    ): Response<WarehouseReturnPolicy>

    @PUT("v1/warehouse/return-policy")
    suspend fun putReturnPolicy(
        @Body body: WarehouseReturnPolicy,
        @Header("Idempotency-Key") idempotencyKey: String,
        @Query("warehouse_id") warehouseId: String? = null,
        @Query("supplier_id") supplierId: String? = null,
    ): Response<WarehouseReturnPolicy>

    @GET("v1/warehouse/ops/location")
    suspend fun getWarehouseLocation(): Response<WarehouseLocationResponse>

    @PATCH("v1/warehouse/ops/location")
    suspend fun patchWarehouseLocation(
        @Body body: WarehouseLocationPatchRequest,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Response<WarehouseLocationResponse>

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

    @GET("v1/returns/inbound")
    suspend fun getInboundReturns(
        @Query("physical_status") physicalStatus: String = "OPEN",
        @Query("limit") limit: Int = 100,
    ): Response<InboundReturnListResponse>

    @GET("v1/warehouse/reverse-logistics")
    suspend fun getReverseLogistics(
        @Query("status") status: String = "OPEN",
        @Query("warehouse_id") warehouseId: String? = null,
    ): Response<ReverseLogisticsListResponse>

    @POST("v1/warehouse/reverse-logistics/{taskId}/receive")
    suspend fun receiveReverseLogistics(
        @Path("taskId") taskId: String,
        @Body body: ReverseLogisticsReceiveRequest,
    ): Response<ReverseLogisticsReceiveResponse>

    @GET("v1/warehouse/ops/exceptions")
    suspend fun getOpsExceptions(): Response<WarehouseOpsExceptionsResponse>

    @GET("v1/supplier/claims")
    suspend fun getSupplierClaims(
        @Query("status") status: String? = "OPEN",
        @Query("limit") limit: Int = 50,
    ): Response<WarehouseClaimsResponse>

    @POST("v1/returns/inbound/sessions")
    suspend fun createInboundSession(
        @Body body: com.google.gson.JsonObject = com.google.gson.JsonObject(),
    ): Response<com.google.gson.JsonObject>

    @POST("v1/returns/inbound/scan")
    suspend fun scanInboundReturn(
        @Header("Idempotency-Key") idempotencyKey: String,
        @Body body: com.google.gson.JsonObject,
    ): Response<com.google.gson.JsonObject>

    @POST("v1/returns/inbound/confirm")
    suspend fun confirmInboundReturns(
        @Header("Idempotency-Key") idempotencyKey: String,
        @Body body: com.google.gson.JsonObject,
    ): Response<com.google.gson.JsonObject>

    @GET("v1/returns/history")
    suspend fun getReturnsHistory(
        @Query("limit") limit: Int = 50,
    ): Response<InboundReturnListResponse>

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
    suspend fun executeDispatch(
        @Header("Idempotency-Key") idempotencyKey: String,
        @Body body: com.google.gson.JsonElement,
    ): Response<DispatchExecuteResponse>

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

    @GET("v1/warehouse/ops/preorders")
    suspend fun getPreorders(): Response<com.pegasusx.warehouse.data.model.WarehousePreordersResponse>

    @POST("v1/warehouse/ops/preorders/{id}/reject")
    suspend fun rejectPreorder(
        @Path("id") orderId: String,
        @Body body: WarehouseOrderMutationRequest,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Response<WarehouseOrderMutationResponse>

    @POST("v1/warehouse/ops/preorders/{id}/edit")
    suspend fun editPreorder(
        @Path("id") orderId: String,
        @Body body: WarehousePreorderEditRequest,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Response<WarehouseOrderMutationResponse>

    @POST("v1/warehouse/ops/orders/{id}/propose-delivery")
    suspend fun proposeOrderDelivery(
        @Path("id") orderId: String,
        @Body body: com.pegasusx.warehouse.data.model.WarehouseProposeDeliveryRequest,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Response<WarehouseOrderMutationResponse>

    @POST("v1/warehouse/ops/preorders/{id}/propose-delivery")
    suspend fun proposePreorderDelivery(
        @Path("id") orderId: String,
        @Body body: com.pegasusx.warehouse.data.model.WarehouseProposeDeliveryRequest,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Response<WarehouseOrderMutationResponse>

    @GET("v1/warehouse/ops/stock-commitments")
    suspend fun getStockCommitments(): Response<com.pegasusx.warehouse.data.model.StockCommitmentsResponse>

    @GET("v1/warehouse/ops/dispatch/settings")
    suspend fun getDispatchSettings(): Response<DispatchSettingsResponse>

    @PATCH("v1/warehouse/ops/dispatch/settings")
    suspend fun patchDispatchSettings(
        @Body body: DispatchSettingsPatchRequest,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Response<Map<String, String>>

    @POST("v1/warehouse/supply-requests")
    suspend fun createSupplyRequest(
        @Header("Idempotency-Key") idempotencyKey: String,
        @Body body: CreateWarehouseSupplyRequestRequest,
    ): Response<CreateWarehouseSupplyRequestResponse>

    @PATCH("v1/warehouse/supply-requests/{id}")
    suspend fun transitionSupplyRequest(
        @Path("id") id: String,
        @Header("Idempotency-Key") idempotencyKey: String,
        @Body body: WarehouseSupplyRequestTransitionRequest,
    ): Response<WarehouseSupplyRequestTransitionResponse>

    @GET("v1/warehouse/dispatch-locks")
    suspend fun getDispatchLocks(): Response<DispatchLockListResponse>

    @POST("v1/warehouse/dispatch-lock")
    suspend fun createDispatchLock(
        @Body body: CreateWarehouseDispatchLockRequest,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Response<CreateWarehouseDispatchLockResponse>

    @DELETE("v1/warehouse/dispatch-lock")
    suspend fun releaseDispatchLock(
        @Query("lock_id") lockId: String,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Response<ReleaseWarehouseDispatchLockResponse>

    // ── Staff ──
    @GET("v1/warehouse/ops/staff")
    suspend fun getStaff(): Response<StaffListResponse>

    @POST("v1/warehouse/ops/staff")
    suspend fun createStaff(
        @Body body: CreateStaffRequest,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Response<CreateStaffResponse>

    // ── Payment Config ──
    @GET("v1/warehouse/ops/payment-config")
    suspend fun getPaymentConfig(): Response<PaymentConfigResponse>

    // ── P1-03 ops depth ──
    @POST("v1/warehouse/transfers/emergency")
    suspend fun emergencyTransfer(
        @Body body: EmergencyTransferRequest,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Response<TransferMutationResponse>

    @POST("v1/warehouse/transfers/force-receive")
    suspend fun forceReceive(
        @Body body: ForceReceiveRequest,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Response<TransferMutationResponse>

    @POST("v1/warehouse/transfers/{id}/receive")
    suspend fun receiveTransfer(
        @Path("id") transferId: String,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Response<TransferMutationResponse>

    @GET("v1/warehouse/replenishment/insights")
    suspend fun getReplenishmentInsights(): Response<ReplenishmentInsightsResponse>

    @GET("v1/warehouse/ops/board")
    suspend fun getOpsBoard(@Query("date") date: String): Response<WarehouseOpsBoardResponse>

    @POST("v1/warehouse/replenishment/insights/{id}/{action}")
    suspend fun replenishmentInsightAction(
        @Path("id") insightId: String,
        @Path("action") action: String,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Response<ReplenishmentInsightActionResponse>

    @GET("v1/warehouse/ops/financials")
    suspend fun getOpsFinancials(@Query("period") period: String? = null): Response<OpsFinancialsResponse>

    @GET("v1/warehouse/ops/broadcast/templates")
    suspend fun getBroadcastTemplates(): Response<BroadcastTemplatesResponse>

    @POST("v1/warehouse/ops/broadcast/templates")
    suspend fun createBroadcastTemplate(
        @Body body: WarehouseBroadcastTemplateCreateRequest,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Response<BroadcastTemplate>

    @DELETE("v1/warehouse/ops/broadcast/templates/{id}")
    suspend fun deleteBroadcastTemplate(
        @Path("id") templateId: String,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Response<BroadcastTemplateDeleteResponse>

    @POST("v1/warehouse/ops/broadcast")
    suspend fun postBroadcast(
        @Body body: WarehouseBroadcastRequest,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Response<WarehouseBroadcastResponse>

    @POST("v1/warehouse/ops/pricing/retailer-overrides/preview")
    suspend fun previewRetailerPriceOverride(
        @Body body: RetailerOverridePreviewRequest,
    ): Response<RetailerOverridePreview>

    @POST("v1/warehouse/ops/orders/{id}/delay")
    suspend fun delayOrder(
        @Path("id") orderId: String,
        @Body body: WarehouseOrderMutationRequest = WarehouseOrderMutationRequest(),
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Response<WarehouseOrderMutationResponse>

    @POST("v1/warehouse/ops/orders/{id}/reject")
    suspend fun rejectOrder(
        @Path("id") orderId: String,
        @Body body: WarehouseOrderMutationRequest,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Response<WarehouseOrderMutationResponse>

    @POST("v1/warehouse/ops/orders/{id}/overflow")
    suspend fun overflowOrder(
        @Path("id") orderId: String,
        @Body body: WarehouseOrderMutationRequest = WarehouseOrderMutationRequest(),
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Response<WarehouseOrderMutationResponse>

    @GET("v1/warehouse/ops/fleet/live-map")
    suspend fun getFleetLiveMap(
        @Query("warehouse_id") warehouseId: String? = null,
    ): Response<WarehouseFleetLiveMapResponse>

    @GET("v1/warehouse/ops/pulse")
    suspend fun getPulse(): Response<PulseResponse>

    // ── Cold chain ──
    @GET("v1/warehouse/ops/temperature-readings")
    suspend fun listTemperatureReadings(
        @Query("manifest_id") manifestId: String,
    ): Response<TemperatureReadingListResponse>

    @POST("v1/warehouse/ops/temperature-readings")
    suspend fun ingestTemperatureReading(
        @Body body: TemperatureReadingIngestRequest,
    ): Response<TemperatureReading>

    // ── Labor capacity ──
    @GET("v1/labor-capacity/zone-capacity")
    suspend fun listLaborZoneCapacity(
        @Query("date") date: String,
    ): Response<LaborZoneCapacityListResponse>

    @GET("v1/labor-capacity/driver-score/{driverId}")
    suspend fun getLaborDriverScore(
        @Path("driverId") driverId: String,
    ): Response<LaborDriverScore>

    @POST("v1/labor-capacity/driver-availability")
    suspend fun setLaborDriverAvailability(
        @Body body: LaborDriverAvailabilityRequest,
    ): Response<LaborDriverAvailabilityResponse>

    // ── Rescue Operations ──
    @POST("v1/warehouse/ops/dispatch/rescue/preview")
    suspend fun previewRescue(
        @Body body: RescuePreviewRequest,
    ): Response<RescuePreviewResponse>

    @POST("v1/warehouse/ops/dispatch/rescue/propose")
    suspend fun proposeRescue(
        @Header("Idempotency-Key") idempotencyKey: String,
        @Body body: RescueProposeRequest,
    ): Response<RescueProposeResponse>


    // ── Notifications + client policy ──
    @GET("v1/user/notifications")
    suspend fun getNotifications(
        @Query("limit") limit: Int = 50,
        @Query("offset") offset: Int = 0,
    ): Response<NotificationsResponse>

    @POST("v1/user/notifications/read")
    suspend fun markNotificationsRead(
        @Body body: MarkNotificationsReadRequest,
    ): Response<Map<String, String>>

    @GET("v1/platform/client-policy")
    suspend fun getClientPolicy(
        @Query("role") role: String = "WAREHOUSE",
        @Query("platform") platform: String,
        @Query("version") version: String,
        @Query("channel") channel: String = "production",
    ): Response<ClientPolicyResponse>

    @POST("v1/user/device-token")
    suspend fun registerDeviceToken(
        @Body body: Map<String, String>,
    ): Response<Map<String, String>>
}
