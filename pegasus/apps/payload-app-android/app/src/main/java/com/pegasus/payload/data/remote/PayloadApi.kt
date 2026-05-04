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
import com.pegasus.payload.data.model.ManifestsResponse
import com.pegasus.payload.data.model.MarkReadRequest
import com.pegasus.payload.data.model.MissingItemsRequest
import com.pegasus.payload.data.model.NotificationsResponse
import com.pegasus.payload.data.model.RecommendReassignRequest
import com.pegasus.payload.data.model.RecommendReassignResponse
import com.pegasus.payload.data.model.SealManifestResponse
import com.pegasus.payload.data.model.SealOrderRequest
import com.pegasus.payload.data.model.SealOrderResponse
import com.pegasus.payload.data.model.StatusResponse
import com.pegasus.payload.data.model.Truck
import retrofit2.http.Body
import retrofit2.http.DELETE
import retrofit2.http.GET
import retrofit2.http.Header
import retrofit2.http.POST
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

    // ── Trucks / Orders ──────────────────────────────────────────────────────
    @GET("v1/payloader/trucks")
    suspend fun trucks(): List<Truck>

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

    // ── Manifest lifecycle ───────────────────────────────────────────────────
    @GET("v1/supplier/manifests")
    suspend fun manifests(
        @Query("state") state: String = "DRAFT",
        @Query("truck_id") truckId: String? = null,
    ): ManifestsResponse

    @GET("v1/supplier/manifests/{id}")
    suspend fun manifestDetail(@Path("id") manifestId: String): Manifest

    @POST("v1/supplier/manifests/{id}/start-loading")
    suspend fun startLoading(
        @Path("id") manifestId: String,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): StatusResponse

    @POST("v1/supplier/manifests/{id}/seal")
    suspend fun sealManifest(
        @Path("id") manifestId: String,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): SealManifestResponse

    @POST("v1/supplier/manifests/{id}/inject-order")
    suspend fun injectOrder(
        @Path("id") manifestId: String,
        @Body req: InjectOrderRequest,
        @Header("Idempotency-Key") idempotencyKey: String? = null,
    ): StatusResponse

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
    suspend fun notifications(@Query("limit") limit: Int = 50): NotificationsResponse

    @POST("v1/user/notifications/read")
    suspend fun markRead(@Body req: MarkReadRequest): StatusResponse

    // ── FCM device-token lifecycle ───────────────────────────────────────────
    @POST("v1/user/device-token")
    suspend fun registerDeviceToken(@Body req: DeviceTokenRequest): StatusResponse

    @DELETE("v1/user/device-token")
    suspend fun unregisterDeviceToken(@Query("platform") platform: String = "ANDROID"): StatusResponse
}
