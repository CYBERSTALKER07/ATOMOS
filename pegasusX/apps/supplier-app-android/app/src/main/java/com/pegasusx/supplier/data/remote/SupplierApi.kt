package com.pegasusx.supplier.data.remote

import com.pegasusx.supplier.data.model.*
import retrofit2.Response
import retrofit2.http.*

interface SupplierApi {
    @POST("v1/auth/supplier/login")
    suspend fun login(@Body body: LoginRequest): Response<LoginResponse>

    @POST("v1/auth/supplier/register")
    suspend fun register(@Body body: com.google.gson.JsonElement): Response<com.google.gson.JsonElement>

    @POST("v1/auth/supplier/refresh")
    suspend fun refreshToken(@Body body: RefreshTokenRequest): Response<LoginResponse>

    @POST("v1/supplier/configure")
    suspend fun configureSupplier(@Body body: com.google.gson.JsonElement): Response<com.google.gson.JsonElement>

    @POST("v1/supplier/business/setup")
    suspend fun setupBusiness(@Body body: com.google.gson.JsonElement): Response<com.google.gson.JsonElement>

    @GET("v1/supplier/dashboard")
    suspend fun getDashboard(): Response<SupplierDashboard>

    @GET("v1/supplier/profile")
    suspend fun getProfile(): Response<SupplierProfile>

    @PUT("v1/supplier/profile")
    suspend fun updateProfile(@Body body: Map<String, String>): Response<SupplierProfile>

    @GET("v1/supplier/org/members")
    suspend fun getOrgMembers(): Response<com.google.gson.JsonElement>

    @POST("v1/supplier/org/members")
    suspend fun createOrgMember(@Body body: com.google.gson.JsonElement): Response<com.google.gson.JsonElement>

    @GET("v1/supplier/orders")
    suspend fun getOrders(
        @Query("status") status: String? = null,
        @Query("filter") filter: String? = null,
        @Query("limit") limit: Int? = null,
        @Query("offset") offset: Int? = null,
    ): Response<SupplierOrdersResponse>

    @POST("v1/supplier/orders/vet")
    suspend fun vetOrder(@Body body: com.google.gson.JsonElement): Response<com.google.gson.JsonElement>

    @POST("v1/supplier/orders/payment-bypass")
    suspend fun issuePaymentBypass(@Body body: com.google.gson.JsonElement): Response<com.google.gson.JsonElement>

    @POST("v1/supplier/route/approve-early-complete")
    suspend fun approveEarlyComplete(@Body body: com.google.gson.JsonElement): Response<com.google.gson.JsonElement>

    @GET("v1/supplier/fleet/drivers")
    suspend fun getFleetDrivers(): Response<FleetDriversResponse>

    @POST("v1/supplier/fleet/drivers")
    suspend fun createFleetDriver(@Body body: FleetDriverCreateRequest): Response<FleetDriversResponse>

    @GET("v1/supplier/fleet/vehicles")
    suspend fun getFleetVehicles(): Response<FleetVehiclesResponse>

    @POST("v1/supplier/fleet/vehicles")
    suspend fun createFleetVehicle(@Body body: FleetVehicleCreateRequest): Response<FleetVehiclesResponse>

    @GET("v1/supplier/fleet/orders")
    suspend fun getFleetOrders(): Response<List<SupplierFleetOrderRow>>

    @GET("v1/supplier/inventory")
    suspend fun getInventory(): Response<InventoryListResponse>

    @PATCH("v1/supplier/inventory")
    suspend fun updateInventory(@Body body: com.google.gson.JsonElement): Response<com.google.gson.JsonElement>

    @GET("v1/supplier/inventory/audit")
    suspend fun getInventoryAudit(): Response<com.google.gson.JsonElement>

    @GET("v1/supplier/earnings")
    suspend fun getEarnings(): Response<SupplierEarnings>

    @POST("v1/supplier/billing/setup")
    suspend fun configureBilling(@Body body: BillingSetupRequest): Response<BillingSetupResponse>

    @GET("v1/supplier/exceptions")
    suspend fun getExceptions(): Response<SupplierExceptionsResponse>

    @GET("v1/supplier/shop-closed/active")
    suspend fun getShopClosedActive(): Response<ShopClosedActiveResponse>

    @GET("v1/supplier/negotiations/pending")
    suspend fun getNegotiationsPending(): Response<NegotiationPendingResponse>

    @POST("v1/supplier/shop-closed/resolve")
    suspend fun resolveShopClosed(@Body body: ShopClosedResolveRequest): Response<NegotiationResolveResponse>

    @POST("v1/supplier/negotiate/resolve")
    suspend fun resolveNegotiation(@Body body: NegotiationResolveRequest): Response<NegotiationResolveResponse>

    @GET("v1/supplier/manifests")
    suspend fun getManifests(): Response<SupplierManifestsResponse>

    @GET("v1/supplier/dispatch/preview")
    suspend fun getDispatchPreview(@Query("warehouse_id") warehouseId: String? = null): Response<SupplierDispatchPreview>

    @POST("v1/supplier/dispatch/preview")
    suspend fun createDispatchPreview(@Body body: com.google.gson.JsonElement): Response<SupplierDispatchPreview>

    @POST("v1/supplier/dispatch/execute")
    suspend fun executeDispatch(@Body body: com.google.gson.JsonElement): Response<com.google.gson.JsonElement>

    @GET("v1/supplier/pricing/rules")
    suspend fun getPricingRules(): Response<SupplierPricingRule>

    @PATCH("v1/supplier/pricing/rules")
    suspend fun updatePricingRules(@Body body: com.google.gson.JsonElement): Response<SupplierPricingRule>

    @GET("v1/supplier/topology")
    suspend fun getTopology(): Response<SupplierTopologyResponse>

    @PUT("v1/supplier/topology")
    suspend fun updateTopology(@Body body: com.google.gson.JsonElement): Response<SupplierTopologyResponse>

    @GET("v1/supplier/supply-lanes")
    suspend fun getSupplyLanes(): Response<SupplierSupplyLanesResponse>

    @GET("v1/supplier/activity")
    suspend fun getActivity(): Response<SupplierActivityResponse>

    @GET("v1/supplier/ai/recommendations")
    suspend fun getAiRecommendations(): Response<com.google.gson.JsonElement>

    @POST("v1/supplier/ai/recommendations")
    suspend fun recordAiRecommendationDecision(@Body body: com.google.gson.JsonElement): Response<com.google.gson.JsonElement>

    @GET("v1/supplier/empathy/adoption")
    suspend fun getEmpathyAdoption(): Response<com.google.gson.JsonElement>

    @POST("v1/supplier/broadcast")
    suspend fun postBroadcast(@Body body: com.google.gson.JsonElement): Response<com.google.gson.JsonElement>

    @GET("v1/supplier/ws-session")
    suspend fun getWsSession(): Response<SupplierWsSessionResponse>

    @GET("v1/payment/ledger")
    suspend fun getPaymentLedger(@Query("currency") currency: String? = null): Response<PaymentLedgerResponse>

    @POST("v1/supplier/replenishment/trigger")
    suspend fun triggerReplenishment(): Response<SupplierReplenishmentTriggerResponse>
}
