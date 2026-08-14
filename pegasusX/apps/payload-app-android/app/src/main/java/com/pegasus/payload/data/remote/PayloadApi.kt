package com.pegasus.payload.data.remote

import com.pegasus.payload.data.model.DeviceTokenRequest
import com.pegasus.payload.data.model.FleetReassignRequest
import com.pegasus.payload.data.model.FleetReassignResponse
import com.pegasus.payload.data.model.InjectOrderRequest
import com.pegasus.payload.data.model.LiveOrder
import com.pegasus.payload.data.model.LoginRequest
import com.pegasus.payload.data.model.LoginResponse
import com.pegasus.payload.data.model.Manifest
import com.pegasus.payload.data.model.ManifestExceptionRequest
import com.pegasus.payload.data.model.ManifestExceptionResponse
import com.pegasus.payload.data.model.ClientPolicyResponse
import com.pegasus.payload.data.model.ManifestExceptionsResponse
import com.pegasus.payload.data.model.ManifestsResponse
import com.pegasus.payload.data.model.MarkReadRequest
import com.pegasus.payload.data.model.MissingItemsRequest
import com.pegasus.payload.data.model.NotificationsResponse
import com.pegasus.payload.data.model.PulseResponse
import com.pegasus.payload.data.model.RecommendReassignRequest
import com.pegasus.payload.data.model.RecommendReassignResponse
import com.pegasus.payload.data.model.SealCompletedManifestsRequest
import com.pegasus.payload.data.model.SealCompletedManifestsResponse
import com.pegasus.payload.data.model.SealManifestResponse
import com.pegasus.payload.data.model.SealOrderRequest
import com.pegasus.payload.data.model.SealOrderResponse
import com.pegasus.payload.data.model.StatusResponse
import com.pegasus.payload.data.model.Truck
import retrofit2.Response
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
 * PayloadApi — single Retrofit interface for every endpoint the Expo
 * payload-terminal currently consumes. All routes verified against
 * [authroutes/], [payloaderroutes/], [adminroutes/], [deliveryroutes/],
 * [fleetroutes/], [userroutes/]. No backend changes required.
 */
interface PayloadApi {

    // ── Auth ─────────────────────────────────────────────────────────────────
    @POST("v1/auth/payloader/login")
    suspend fun login(@Body req: LoginRequest): LoginResponse

    @POST("v1/auth/payloader/refresh")
    suspend fun refreshToken(@Body req: com.pegasus.payload.data.model.RefreshTokenRequest): com.pegasus.payload.data.model.RefreshTokenResponse

    // ── Trucks / Orders ──────────────────────────────────────────────────────
    @GET("v1/payloader/trucks")
    suspend fun trucks(): List<Truck>

    @GET("v1/payloader/pulse")
    suspend fun getPulse(): PulseResponse

    @GET("v1/payloader/orders")
    suspend fun orders(
        @Query("vehicle_id") vehicleId: String? = null,
        @Query("state") state: String? = null,
    ): List<LiveOrder>

    @POST("v1/payloader/recommend-reassign")
    suspend fun recommendReassign(
        @Body req: RecommendReassignRequest,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): RecommendReassignResponse

    @POST("v1/payloader/reassign-order")
    suspend fun reassignOrder(
        @Body req: com.pegasus.payload.data.model.ReassignOrderRequest,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): StatusResponse

    // ── Manifest lifecycle ───────────────────────────────────────────────────
    @GET("v1/payloader/manifests")
    suspend fun manifests(
        @Query("state") state: String = "DRAFT",
        @Query("truck_id") truckId: String? = null,
    ): ManifestsResponse

    @GET("v1/payloader/manifests/{id}")
    suspend fun manifestDetail(@Path("id") manifestId: String): Manifest

    @POST("v1/payloader/manifests/{id}/start-loading")
    suspend fun startLoading(
        @Path("id") manifestId: String,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): StatusResponse

    @POST("v1/payloader/manifests/seal-completed")
    suspend fun sealCompletedManifests(
        @Body req: SealCompletedManifestsRequest,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): SealCompletedManifestsResponse

    @POST("v1/payloader/manifests/seal-all")
    suspend fun sealAllManifests(
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): SealCompletedManifestsResponse

    @POST("v1/payloader/manifests/{id}/seal")
    suspend fun sealManifest(
        @Path("id") manifestId: String,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): SealManifestResponse

    @POST("v1/payloader/manifests/{id}/inject-order")
    suspend fun injectOrder(
        @Path("id") manifestId: String,
        @Body req: InjectOrderRequest,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): StatusResponse

    @GET("v1/supplier/manifests")
    suspend fun supplierManifests(
        @Query("state") state: String = "DRAFT",
    ): ManifestsResponse

    @GET("v1/supplier/manifests/{id}")
    suspend fun supplierManifestDetail(@Path("id") manifestId: String): Manifest

    @POST("v1/supplier/manifests/{id}/start-loading")
    suspend fun supplierStartLoading(
        @Path("id") manifestId: String,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): StatusResponse

    @POST("v1/supplier/manifests/{id}/seal")
    suspend fun supplierSealManifest(
        @Path("id") manifestId: String,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): SealManifestResponse

    @POST("v1/supplier/manifests/{id}/inject-order")
    suspend fun supplierInjectOrder(
        @Path("id") manifestId: String,
        @Body req: InjectOrderRequest,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): StatusResponse

    // ── Factory loading-bay (P1-18 / P2-25 parity with Expo terminal) ─────────
    @GET("v1/factory/manifests")
    suspend fun factoryManifests(
        @Query("state") state: String = "DRAFT",
    ): ManifestsResponse

    @GET("v1/factory/manifests/{id}")
    suspend fun factoryManifestDetail(@Path("id") manifestId: String): Manifest

    @POST("v1/factory/manifests/{id}/start-loading")
    suspend fun factoryStartLoading(
        @Path("id") manifestId: String,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): StatusResponse

    @POST("v1/factory/manifests/{id}/seal")
    suspend fun factorySealManifest(
        @Path("id") manifestId: String,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): SealManifestResponse

    // ── Per-order seal / exception ───────────────────────────────────────────
    @POST("v1/payload/seal")
    suspend fun sealOrder(
        @Body req: SealOrderRequest,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): SealOrderResponse

    @POST("v1/payload/manifest-exception")
    suspend fun manifestException(
        @Body req: ManifestExceptionRequest,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): ManifestExceptionResponse

    @GET("v1/payloader/manifest-exceptions")
    suspend fun manifestExceptionsList(
        @Query("limit") limit: Int = 50,
        @Query("offset") offset: Int = 0,
    ): ManifestExceptionsResponse

    @POST("v1/delivery/missing-items")
    suspend fun missingItems(
        @Body req: MissingItemsRequest,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): StatusResponse

    // ── Fleet reassign ───────────────────────────────────────────────────────
    @POST("v1/fleet/reassign")
    suspend fun fleetReassign(
        @Body req: FleetReassignRequest,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): FleetReassignResponse

    // ── Notifications ────────────────────────────────────────────────────────
    @GET("v1/user/notifications")
    suspend fun notifications(
        @Query("limit") limit: Int = 100,
        @Query("offset") offset: Int = 0,
    ): NotificationsResponse

    @POST("v1/user/notifications/read")
    suspend fun markRead(@Body req: MarkReadRequest): StatusResponse

    // ── FCM device-token lifecycle ───────────────────────────────────────────
    @POST("v1/user/device-token")
    suspend fun registerDeviceToken(@Body req: DeviceTokenRequest): StatusResponse

    @DELETE("v1/user/device-token")
    suspend fun unregisterDeviceToken(@Query("platform") platform: String = "ANDROID"): StatusResponse

    // ── Platform policy ──────────────────────────────────────────────────────
    @GET("v1/platform/client-policy")
    suspend fun getClientPolicy(
        @Query("role") role: String = "PAYLOAD",
        @Query("platform") platform: String,
        @Query("version") version: String,
        @Query("channel") channel: String = "production",
    ): Response<ClientPolicyResponse>

    // ── Inbound returns gate ─────────────────────────────────────────────────
    @GET("v1/returns/inbound")
    suspend fun getInboundReturns(
        @Query("physical_status") physicalStatus: String = "ARRIVED",
        @Query("limit") limit: Int = 100,
    ): Response<Map<String, @JvmSuppressWildcards Any>>

    @POST("v1/returns/inbound/sessions")
    suspend fun createInboundSession(
        @Body body: Map<String, @JvmSuppressWildcards Any> = emptyMap(),
    ): Response<Map<String, @JvmSuppressWildcards Any>>

    @POST("v1/returns/inbound/scan")
    suspend fun scanInboundReturn(
        @Body body: Map<String, @JvmSuppressWildcards Any>,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): Response<Map<String, @JvmSuppressWildcards Any>>

    @POST("v1/returns/inbound/confirm")
    suspend fun confirmInboundReturns(
        @Body body: Map<String, @JvmSuppressWildcards Any>,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): Response<Map<String, @JvmSuppressWildcards Any>>

    @GET("v1/returns/history")
    suspend fun getReturnsHistory(@Query("limit") limit: Int = 50): Response<Map<String, @JvmSuppressWildcards Any>>

    @GET("v1/catalog/barcode/{ean}")
    suspend fun lookupBarcode(@Path("ean") ean: String): Response<Map<String, @JvmSuppressWildcards Any>>

    // ── Raw queue sync ───────────────────────────────────────────────────────
    @POST("{endpoint}")
    suspend fun rawPost(
        @Path(value = "endpoint", encoded = true) endpoint: String,
        @Body body: okhttp3.RequestBody?,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): Response<okhttp3.ResponseBody>

    @PUT("{endpoint}")
    suspend fun rawPut(
        @Path(value = "endpoint", encoded = true) endpoint: String,
        @Body body: okhttp3.RequestBody?,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): Response<okhttp3.ResponseBody>

    @PATCH("{endpoint}")
    suspend fun rawPatch(
        @Path(value = "endpoint", encoded = true) endpoint: String,
        @Body body: okhttp3.RequestBody?,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): Response<okhttp3.ResponseBody>

    @DELETE("{endpoint}")
    suspend fun rawDelete(
        @Path(value = "endpoint", encoded = true) endpoint: String,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): Response<okhttp3.ResponseBody>
}
