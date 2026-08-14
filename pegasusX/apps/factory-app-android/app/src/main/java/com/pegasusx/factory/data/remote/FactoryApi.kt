package com.pegasusx.factory.data.remote

import com.pegasusx.factory.data.model.*
import kotlinx.serialization.json.JsonElement
import retrofit2.Response
import retrofit2.http.*

interface FactoryApi {

    // ── Auth ──
    @POST("v1/auth/factory/login")
    suspend fun login(@Body body: LoginRequest): Response<AuthResponse>

    @POST("v1/auth/factory/register")
    suspend fun register(@Body body: JsonElement): Response<AuthResponse>

    @POST("v1/auth/factory/refresh")
    suspend fun refreshToken(): Response<AuthResponse>

    @POST("v1/factory/setup")
    suspend fun setupFactory(@Body body: com.pegasusx.factory.data.model.FactorySetupRequest): Response<com.pegasusx.factory.data.model.FactorySetupResponse>

    // ── Dashboard ──
    @GET("v1/factory/dashboard")
    suspend fun getDashboard(): Response<DashboardStats>

    @GET("v1/factory/profile")
    suspend fun getFactoryProfile(): Response<JsonElement>

    @GET("v1/factory/analytics/overview")
    suspend fun getFactoryAnalyticsOverview(
        @Query("from") from: String? = null,
        @Query("to") to: String? = null,
    ): Response<FactoryAnalyticsOverview>

    // ── Transfers ──
    @GET("v1/factory/transfers")
    suspend fun getTransfers(
        @Query("state") state: String? = null,
        @Query("limit") limit: Int = 50,
        @Query("offset") offset: Int = 0,
    ): Response<TransferListResponse>

    @GET("v1/factory/transfers/{id}")
    suspend fun getTransfer(@Path("id") id: String): Response<Transfer>

    @POST("v1/factory/transfers/create")
    suspend fun createTransfer(
        @Body body: CreateTransferRequest,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Response<CreateTransferResponse>

    @POST("v1/factory/transfers/{id}/transition")
    suspend fun transitionTransfer(
        @Path("id") id: String,
        @Body body: TransitionRequest,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Response<Transfer>

    // ── Loading Bay (transfers filtered by loading states) ──
    @GET("v1/factory/transfers")
    suspend fun getLoadingBayTransfers(
        @Query("states") states: String = "APPROVED,LOADING,DISPATCHED",
        @Query("limit") limit: Int = 100,
    ): Response<TransferListResponse>

    // ── Dispatch ──
    @POST("v1/factory/dispatch")
    suspend fun dispatch(
        @Body body: DispatchRequest,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Response<DispatchResponse>

    @GET("v1/factory/pulse")
    suspend fun getPulse(): Response<PulseResponse>

    // ── Supply Requests ──
    @GET("v1/factory/supply-requests")
    suspend fun getSupplyRequests(
        @Query("state") state: String? = null,
    ): Response<SupplyRequestListResponse>

    @POST("v1/factory/supply-requests/{id}/accept")
    suspend fun acceptSupplyRequest(
        @Path("id") id: String,
        @Header("Idempotency-Key") idempotencyKey: String,
        @Body body: JsonElement = kotlinx.serialization.json.JsonObject(emptyMap()),
    ): Response<JsonElement>

    @PATCH("v1/factory/supply-requests/{id}")
    suspend fun transitionSupplyRequest(
        @Path("id") id: String,
        @Header("Idempotency-Key") idempotencyKey: String,
        @Body body: SupplyRequestTransitionRequest,
    ): Response<SupplyRequestTransitionResponse>

    @GET("v1/factory/supply-requests/{id}/fulfill-options")
    suspend fun getSupplyFulfillOptions(@Path("id") id: String): Response<SupplyFulfillOptions>

    @GET("v1/factory/supply-requests/{id}/qc")
    suspend fun getSupplyRequestQC(@Path("id") id: String): Response<SupplyRequestQCResponse>

    @POST("v1/factory/supply-requests/{id}/qc")
    suspend fun postSupplyRequestQC(
        @Path("id") id: String,
        @Header("Idempotency-Key") idempotencyKey: String,
        @Body body: SupplyRequestQCRequest,
    ): Response<SupplyRequestQCResponse>

    // ── Payload Override / Manifests ──
    @GET("v1/factory/manifests")
    suspend fun getManifests(
        @Query("state") state: String? = null,
    ): Response<ManifestListResponse>

    @GET("v1/factory/manifests/{id}")
    suspend fun getManifestDetail(@Path("id") id: String): Response<ManifestDetailResponse>

    @POST("v1/factory/manifests/{id}/start-loading")
    suspend fun startManifestLoading(
        @Path("id") id: String,
        @Header("Idempotency-Key") idempotencyKey: String,
        @Body body: ManifestTransitionRequest = ManifestTransitionRequest(),
    ): Response<ManifestTransitionResponse>

    @POST("v1/factory/manifests/{id}/seal")
    suspend fun sealManifest(
        @Path("id") id: String,
        @Header("Idempotency-Key") idempotencyKey: String,
        @Body body: ManifestTransitionRequest = ManifestTransitionRequest(),
    ): Response<ManifestTransitionResponse>

    @POST("v1/factory/manifests/{id}/dispatch")
    suspend fun dispatchManifest(
        @Path("id") id: String,
        @Header("Idempotency-Key") idempotencyKey: String,
        @Body body: ManifestTransitionRequest = ManifestTransitionRequest(),
    ): Response<ManifestTransitionResponse>

    @POST("v1/factory/manifests/{id}/complete")
    suspend fun completeManifest(
        @Path("id") id: String,
        @Header("Idempotency-Key") idempotencyKey: String,
        @Body body: ManifestTransitionRequest = ManifestTransitionRequest(),
    ): Response<ManifestTransitionResponse>

    @POST("v1/factory/manifests/rebalance")
    suspend fun rebalanceManifest(
        @Body body: ManifestRebalanceRequest,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Response<ManifestRebalanceResponse>

    @POST("v1/factory/manifests/cancel-transfer")
    suspend fun cancelManifestTransfer(
        @Body body: ManifestCancelTransferRequest,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Response<ManifestCancelTransferResponse>

    @POST("v1/factory/manifests/cancel")
    suspend fun cancelManifest(
        @Body body: ManifestCancelRequest,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Response<ManifestCancelResponse>

    // ── Fleet ──
    @GET("v1/factory/fleet")
    suspend fun getFleet(): Response<VehicleListResponse>

    @GET("v1/factory/fleet/live-map")
    suspend fun getFleetLiveMap(): Response<FactoryFleetLiveMapResponse>

    @GET("v1/factory/fleet/drivers")
    suspend fun getFleetDrivers(): Response<FleetDriverListResponse>

    @GET("v1/factory/fleet/vehicles")
    suspend fun getFleetVehicles(): Response<FleetVehicleListResponse>

    // ── Staff ──
    @GET("v1/factory/staff")
    suspend fun getStaff(): Response<StaffListResponse>

    @POST("v1/factory/staff")
    suspend fun createStaff(@Body body: CreateStaffRequest): Response<StaffMember>

    @GET("v1/factory/staff/{id}")
    suspend fun getStaffDetail(@Path("id") id: String): Response<StaffMember>

    @POST("v1/factory/staff/{id}/set-password")
    suspend fun setStaffPassword(
        @Path("id") id: String,
        @Body body: Map<String, String>,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Response<JsonElement>

    // ── Insights ──
    @GET("v1/warehouse/replenishment/insights")
    suspend fun getInsights(
        @Query("limit") limit: Int = 100,
    ): Response<InsightListResponse>

    // ── Manifest exceptions ──
    @GET("v1/factory/manifest-exceptions")
    suspend fun getManifestExceptions(
        @Query("escalated") escalated: String? = null,
    ): Response<ManifestExceptionListResponse>

    @POST("v1/factory/manifest-exceptions/{exceptionID}/resolve")
    suspend fun resolveManifestException(
        @Path("exceptionID") exceptionId: String,
        @Body body: ResolveManifestExceptionRequest = ResolveManifestExceptionRequest(),
    ): Response<ResolveManifestExceptionResponse>

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
        @Query("role") role: String = "FACTORY",
        @Query("platform") platform: String,
        @Query("version") version: String,
        @Query("channel") channel: String = "production",
    ): Response<ClientPolicyResponse>

    // ── Location (factory-scoped staff have full read/write) ──
    @GET("v1/factory/ops/location")
    suspend fun getFactoryLocation(): Response<com.pegasusx.factory.data.model.FactoryLocationResponse>

    @PATCH("v1/factory/ops/location")
    suspend fun patchFactoryLocation(
        @Body body: com.pegasusx.factory.data.model.FactoryLocationPatchRequest,
        @Header("Idempotency-Key") idempotencyKey: String,
    ): Response<com.pegasusx.factory.data.model.FactoryLocationResponse>
}
