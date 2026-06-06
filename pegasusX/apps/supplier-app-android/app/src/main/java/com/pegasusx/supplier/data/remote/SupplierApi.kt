package com.pegasusx.supplier.data.remote

import com.pegasusx.supplier.data.model.*
import retrofit2.Response
import retrofit2.http.*

interface SupplierApi {
    @POST("v1/auth/supplier/login")
    suspend fun login(@Body body: LoginRequest): Response<LoginResponse>

    @POST("v1/auth/supplier/refresh")
    suspend fun refreshToken(@Body body: RefreshTokenRequest): Response<LoginResponse>

    @GET("v1/supplier/dashboard")
    suspend fun getDashboard(): Response<SupplierDashboard>

    @GET("v1/supplier/profile")
    suspend fun getProfile(): Response<SupplierProfile>

    @PUT("v1/supplier/profile")
    suspend fun updateProfile(@Body body: Map<String, String>): Response<SupplierProfile>

    @GET("v1/supplier/orders")
    suspend fun getOrders(
        @Query("status") status: String? = null,
        @Query("filter") filter: String? = null,
        @Query("limit") limit: Int? = null,
        @Query("offset") offset: Int? = null,
    ): Response<SupplierOrdersResponse>

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

    @GET("v1/supplier/pricing/rules")
    suspend fun getPricingRules(): Response<SupplierPricingRule>

    @GET("v1/supplier/topology")
    suspend fun getTopology(): Response<SupplierTopologyResponse>

    @GET("v1/supplier/supply-lanes")
    suspend fun getSupplyLanes(): Response<SupplierSupplyLanesResponse>

    @GET("v1/supplier/activity")
    suspend fun getActivity(): Response<SupplierActivityResponse>

    @GET("v1/supplier/ws-session")
    suspend fun getWsSession(): Response<SupplierWsSessionResponse>

    @GET("v1/payment/ledger")
    suspend fun getPaymentLedger(@Query("currency") currency: String? = null): Response<PaymentLedgerResponse>

    @POST("v1/supplier/replenishment/trigger")
    suspend fun triggerReplenishment(): Response<SupplierReplenishmentTriggerResponse>
}
