package com.pegasusx.supplier.data.remote

import com.pegasusx.supplier.data.model.*
import kotlinx.serialization.json.JsonElement
import okhttp3.RequestBody
import retrofit2.Response
import retrofit2.http.*

interface SupplierApi {
    @POST("v1/auth/supplier/login")
    suspend fun login(@Body body: LoginRequest): Response<LoginResponse>

    @POST("v1/auth/supplier/register")
    suspend fun register(@Body body: JsonElement): Response<JsonElement>

    @POST("v1/auth/supplier/refresh")
    suspend fun refreshToken(@Body body: RefreshTokenRequest): Response<LoginResponse>

    @POST("v1/supplier/configure")
    suspend fun configureSupplier(@Body body: JsonElement): Response<JsonElement>

    @POST("v1/supplier/business/setup")
    suspend fun setupBusiness(@Body body: JsonElement): Response<JsonElement>

    @GET("v1/supplier/dashboard")
    suspend fun getDashboard(): Response<SupplierDashboard>

    @GET("v1/supplier/profile")
    suspend fun getProfile(): Response<SupplierProfile>

    @PUT("v1/supplier/profile")
    suspend fun updateProfile(@Body body: Map<String, String>): Response<SupplierProfile>

    @GET("v1/supplier/org/members")
    suspend fun getOrgMembers(): Response<SupplierOrgMembersResponse>

    @POST("v1/supplier/org/members")
    suspend fun createOrgMember(
        @Header("X-Idempotency-Key") idempotencyKey: String,
        @Body body: SupplierOrgMemberCreateRequest,
    ): Response<SupplierOrgMembersResponse>

    @PATCH("v1/supplier/org/members/{userId}")
    suspend fun updateOrgMember(
        @Path("userId") userId: String,
        @Header("X-Idempotency-Key") idempotencyKey: String,
        @Body body: SupplierOrgMemberUpdateRequest,
    ): Response<SupplierOrgMembersResponse>

    @DELETE("v1/supplier/org/members/{userId}")
    suspend fun deactivateOrgMember(
        @Path("userId") userId: String,
        @Header("X-Idempotency-Key") idempotencyKey: String,
    ): Response<SupplierOrgMembersResponse>

    @GET("v1/supplier/orders")
    suspend fun getOrders(
        @Query("status") status: String? = null,
        @Query("filter") filter: String? = null,
        @Query("limit") limit: Int? = null,
        @Query("offset") offset: Int? = null,
    ): Response<SupplierOrdersResponse>

    @GET("v1/supplier/returns")
    suspend fun getReturns(
        @Query("status") status: String = "PENDING",
        @Query("limit") limit: Int = 100,
        @Query("offset") offset: Int = 0,
    ): Response<SupplierReturnsResponse>

    @POST("v1/supplier/returns/resolve")
    suspend fun resolveReturn(
        @Header("Idempotency-Key") idempotencyKey: String,
        @Body body: ResolveReturnRequest,
    ): Response<JsonElement>

    @POST("v1/supplier/orders/vet")
    suspend fun vetOrder(
        @Header("Idempotency-Key") idempotencyKey: String,
        @Body body: JsonElement,
    ): Response<JsonElement>

    @POST("v1/supplier/orders/payment-bypass")
    suspend fun issuePaymentBypass(
        @Header("X-Idempotency-Key") idempotencyKey: String,
        @Body body: PaymentBypassRequest,
    ): Response<PaymentBypassResponse>

    @GET("v1/supplier/analytics/velocity")
    suspend fun getAnalyticsVelocity(): Response<SupplierAnalyticsVelocityResponse>

    @GET("v1/supplier/analytics/revenue")
    suspend fun getAnalyticsRevenue(): Response<SupplierAnalyticsRevenueResponse>

    @GET("v1/supplier/analytics/demand/today")
    suspend fun getDemandToday(): Response<SupplierDemandSummaryResponse>

    @GET("v1/supplier/analytics/demand/history")
    suspend fun getDemandHistory(): Response<DemandHistoryResponse>

    @POST("v1/payment/chargeback")
    suspend fun recordChargeback(
        @Header("Idempotency-Key") idempotencyKey: String,
        @Body body: JsonElement,
    ): Response<JsonElement>

    @POST("v1/payment/chargeback/reversal")
    suspend fun recordChargebackReversal(
        @Header("Idempotency-Key") idempotencyKey: String,
        @Body body: JsonElement,
    ): Response<JsonElement>

    @POST("v1/supplier/inventory/imports")
    suspend fun createImportSession(@Body body: ImportSessionCreateRequest): Response<ImportSessionCreateResponse>

    @GET("v1/supplier/inventory/imports/{sessionId}")
    suspend fun getImportSession(@Path("sessionId") sessionId: String): Response<JsonElement>

    @POST("v1/supplier/inventory/imports/{sessionId}/ingest")
    @Headers("Content-Type: text/csv")
    suspend fun ingestImportSession(
        @Path("sessionId") sessionId: String,
        @Body body: RequestBody,
    ): Response<JsonElement>

    @GET("v1/supplier/inventory/imports/{sessionId}/mapping")
    suspend fun getImportMapping(@Path("sessionId") sessionId: String): Response<JsonElement>

    @POST("v1/supplier/inventory/imports/{sessionId}/mapping")
    suspend fun postImportMapping(
        @Path("sessionId") sessionId: String,
        @Body body: JsonElement,
    ): Response<JsonElement>

    @POST("v1/supplier/inventory/imports/{sessionId}/approve")
    suspend fun approveImportSession(@Path("sessionId") sessionId: String): Response<JsonElement>

    @POST("v1/supplier/inventory/imports/{sessionId}/apply")
    suspend fun applyImportSession(@Path("sessionId") sessionId: String): Response<JsonElement>

    @POST("v1/supplier/route/approve-early-complete")
    suspend fun approveEarlyComplete(
        @Header("X-Idempotency-Key") idempotencyKey: String,
        @Body body: ApproveEarlyCompleteRequest,
    ): Response<JsonElement>

    @GET("v1/supplier/fleet/drivers")
    suspend fun getFleetDrivers(): Response<FleetDriversResponse>

    @POST("v1/supplier/fleet/drivers")
    suspend fun createFleetDriver(
        @Header("X-Idempotency-Key") idempotencyKey: String,
        @Body body: FleetDriverCreateRequest,
    ): Response<FleetDriversResponse>

    @GET("v1/supplier/fleet/vehicles")
    suspend fun getFleetVehicles(): Response<FleetVehiclesResponse>

    @POST("v1/supplier/fleet/vehicles")
    suspend fun createFleetVehicle(
        @Header("X-Idempotency-Key") idempotencyKey: String,
        @Body body: FleetVehicleCreateRequest,
    ): Response<FleetVehiclesResponse>

    @GET("v1/supplier/fleet/orders")
    suspend fun getFleetOrders(): Response<List<SupplierFleetOrderRow>>

    @GET("v1/supplier/fleet/live-map")
    suspend fun getFleetLiveMap(): Response<SupplierFleetLiveMapResponse>

    @GET("v1/catalog/products")
    suspend fun listCatalogProducts(): Response<List<CatalogProduct>>

    @GET("v1/catalog/products/{productId}")
    suspend fun getCatalogProduct(@Path("productId") productId: String): Response<CatalogProduct>

    @GET("v1/catalog/categories")
    suspend fun listCatalogCategories(
        @Query("supplier_id") supplierId: String? = null,
    ): Response<List<CatalogCategory>>

    @GET("v1/catalog/products/upload-ticket")
    suspend fun getCatalogUploadTicket(@Query("ext") ext: String): Response<CatalogUploadTicket>

    @POST("v1/catalog/products")
    suspend fun createCatalogProduct(@Body body: CatalogProductCreateRequest): Response<CatalogProduct>

    @PUT("v1/catalog/products/{productId}")
    suspend fun updateCatalogProduct(
        @Path("productId") productId: String,
        @Body body: CatalogProductUpdateRequest,
    ): Response<CatalogProduct>

    @GET("v1/supplier/inventory")
    suspend fun getInventory(): Response<InventoryListResponse>

    @PATCH("v1/supplier/inventory")
    suspend fun updateInventory(@Body body: JsonElement): Response<JsonElement>

    @GET("v1/supplier/inventory/audit")
    suspend fun getInventoryAudit(): Response<JsonElement>

    @GET("v1/supplier/earnings")
    suspend fun getEarnings(): Response<SupplierEarnings>

    @POST("v1/supplier/billing/setup")
    suspend fun configureBilling(@Body body: BillingSetupRequest): Response<BillingSetupResponse>

    @GET("v1/supplier/exceptions")
    suspend fun getExceptions(): Response<SupplierExceptionsResponse>

    @GET("v1/supplier/shop-closed/active")
    suspend fun getShopClosedActive(
        @Query("limit") limit: Int = 500,
        @Query("offset") offset: Int = 0,
    ): Response<ShopClosedActiveResponse>

    @GET("v1/supplier/negotiations/pending")
    suspend fun getNegotiationsPending(
        @Query("limit") limit: Int = 500,
        @Query("offset") offset: Int = 0,
    ): Response<NegotiationPendingResponse>

    @POST("v1/supplier/shop-closed/resolve")
    suspend fun resolveShopClosed(
        @Body body: ShopClosedResolveRequest,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Response<NegotiationResolveResponse>

    @POST("v1/supplier/negotiate/resolve")
    suspend fun resolveNegotiation(
        @Body body: NegotiationResolveRequest,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Response<NegotiationResolveResponse>

    @GET("v1/supplier/manifests")
    suspend fun getManifests(): Response<SupplierManifestsResponse>

    @GET("v1/supplier/manifests/{manifestId}")
    suspend fun getManifestDetail(@Path("manifestId") manifestId: String): Response<SupplierManifestDetail>

    @POST("v1/supplier/manifests/{manifestId}/start-loading")
    suspend fun startManifestLoading(
        @Path("manifestId") manifestId: String,
        @Header("X-Idempotency-Key") idempotencyKey: String,
    ): Response<JsonElement>

    @POST("v1/supplier/manifests/{manifestId}/inject-order")
    suspend fun injectManifestOrder(
        @Path("manifestId") manifestId: String,
        @Header("X-Idempotency-Key") idempotencyKey: String,
        @Body body: SupplierManifestInjectOrderRequest,
    ): Response<JsonElement>

    @POST("v1/supplier/manifests/{manifestId}/seal")
    suspend fun sealManifest(
        @Path("manifestId") manifestId: String,
        @Header("X-Idempotency-Key") idempotencyKey: String,
    ): Response<JsonElement>

    @GET("v1/supplier/manifest-exceptions")
    suspend fun getManifestExceptions(
        @Query("escalated") escalated: Boolean? = null,
    ): Response<SupplierManifestExceptionsResponse>

    @GET("v1/supplier/dispatch/preview")
    suspend fun getDispatchPreview(@Query("warehouse_id") warehouseId: String? = null): Response<SupplierDispatchPreview>

    @POST("v1/supplier/dispatch/preview")
    suspend fun createDispatchPreview(@Body body: JsonElement): Response<SupplierDispatchPreview>

    @POST("v1/supplier/dispatch/execute")
    suspend fun executeDispatch(
        @Query("warehouse_id") warehouseId: String? = null,
        @Header("Idempotency-Key") idempotencyKey: String,
        @Body body: JsonElement,
    ): Response<JsonElement>

    @GET("v1/supplier/pricing/rules")
    suspend fun getPricingRules(): Response<SupplierPricingRule>

    @PATCH("v1/supplier/pricing/rules")
    suspend fun updatePricingRules(@Body body: JsonElement): Response<SupplierPricingRule>

    @GET("v1/supplier/pricing/retailer-overrides")
    suspend fun listRetailerPriceOverrides(
        @Query("retailer_id") retailerId: String? = null,
        @Query("product_id") productId: String? = null,
    ): Response<RetailerPriceOverridesResponse>

    @POST("v1/supplier/pricing/retailer-overrides")
    suspend fun createRetailerPriceOverride(
        @Body body: CreateRetailerPriceOverrideRequest,
    ): Response<CreateRetailerPriceOverrideResponse>

    @DELETE("v1/supplier/pricing/retailer-overrides/{overrideId}")
    suspend fun deleteRetailerPriceOverride(
        @Path("overrideId") overrideId: String,
    ): Response<JsonElement>

    @GET("v1/supplier/topology")
    suspend fun getTopology(): Response<SupplierTopologyResponse>

    @PUT("v1/supplier/topology")
    suspend fun updateTopology(@Body body: SupplierTopologyUpdateRequest): Response<SupplierTopologyResponse>

    @GET("v1/supplier/supply-lanes")
    suspend fun getSupplyLanes(): Response<SupplierSupplyLanesResponse>

    @GET("v1/supplier/activity")
    suspend fun getActivity(): Response<SupplierActivityResponse>

    @GET("v1/supplier/ai/recommendations")
    suspend fun getAiRecommendations(
        @Query("status") status: String? = null,
        @Query("limit") limit: Int = 50,
    ): Response<SupplierAIRecommendationsResponse>

    @POST("v1/supplier/ai/recommendations")
    suspend fun recordAiRecommendationDecision(
        @Header("X-Idempotency-Key") idempotencyKey: String,
        @Body body: SupplierAIRecommendationDecisionRequest,
    ): Response<SupplierAIRecommendationDecisionResponse>

    @GET("v1/supplier/empathy/adoption")
    suspend fun getEmpathyAdoption(): Response<SupplierEmpathyAdoption>

    @POST("v1/supplier/broadcast")
    suspend fun postBroadcast(
        @Header("X-Idempotency-Key") idempotencyKey: String,
        @Body body: SupplierBroadcastRequest,
    ): Response<SupplierBroadcastResponse>

    @GET("v1/supplier/ws-session")
    suspend fun getWsSession(): Response<SupplierWsSessionResponse>

    @GET("v1/payment/ledger")
    suspend fun getPaymentLedger(@Query("currency") currency: String? = null): Response<PaymentLedgerResponse>

    @GET("v1/payment/settlement/authority")
    suspend fun getPaymentSettlementAuthority(
        @Query("group_limit") groupLimit: Int = 200,
    ): Response<SettlementAuthorityResponse>

    @GET("v1/payment/reconciliation/mismatches")
    suspend fun getPaymentReconciliationMismatches(
        @Query("group_limit") groupLimit: Int = 200,
        @Query("mismatch_threshold_minor") mismatchThresholdMinor: Int = 1,
    ): Response<ReconciliationMismatchResponse>

    @POST("v1/supplier/replenishment/trigger")
    suspend fun triggerReplenishment(): Response<SupplierReplenishmentTriggerResponse>

    @GET("v1/supplier/promotions")
    suspend fun getPromotions(): Response<SupplierPromotionsResponse>

    @POST("v1/supplier/promotions")
    suspend fun createPromotion(@Body body: SupplierPromotionUpsertRequest): Response<SupplierPromotion>

    @PATCH("v1/supplier/promotions/{promotionId}")
    suspend fun updatePromotion(
        @retrofit2.http.Path("promotionId") promotionId: String,
        @Body body: SupplierPromotionUpsertRequest,
    ): Response<SupplierPromotion>

    @DELETE("v1/supplier/promotions/{promotionId}")
    suspend fun deactivatePromotion(@Path("promotionId") promotionId: String): Response<Map<String, String>>

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
        @Query("role") role: String = "ADMIN",
        @Query("platform") platform: String,
        @Query("version") version: String,
        @Query("channel") channel: String = "production",
    ): Response<ClientPolicyResponse>

    @GET("v1/warehouse/ops/orders/{orderId}")
    suspend fun getWarehouseOrder(
        @Path("orderId") orderId: String,
        @Query("warehouse_id") warehouseId: String?,
    ): Response<WarehouseOrderDetail>

    @POST("v1/warehouse/ops/orders/{orderId}/delay")
    suspend fun delayWarehouseOrder(
        @Path("orderId") orderId: String,
        @Query("warehouse_id") warehouseId: String,
        @Header("X-Idempotency-Key") idempotencyKey: String,
        @Body body: WarehouseOrderMutationRequest,
    ): Response<WarehouseOrderMutationResponse>

    @POST("v1/warehouse/ops/orders/{orderId}/reject")
    suspend fun rejectWarehouseOrder(
        @Path("orderId") orderId: String,
        @Query("warehouse_id") warehouseId: String,
        @Header("X-Idempotency-Key") idempotencyKey: String,
        @Body body: WarehouseOrderMutationRequest,
    ): Response<WarehouseOrderMutationResponse>
}
