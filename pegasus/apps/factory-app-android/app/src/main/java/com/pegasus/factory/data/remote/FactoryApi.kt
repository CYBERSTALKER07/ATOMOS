package com.pegasus.factory.data.remote

import com.pegasus.factory.data.model.*
import retrofit2.Response
import retrofit2.http.*

interface FactoryApi {

    // ── Auth ──
    @POST("v1/auth/factory/login")
    suspend fun login(@Body body: LoginRequest): Response<AuthResponse>

    @POST("v1/auth/factory/refresh")
    suspend fun refreshToken(): Response<AuthResponse>

    // ── Dashboard ──
    @GET("v1/factory/dashboard")
    suspend fun getDashboard(): Response<DashboardStats>

    // ── Transfers ──
    @GET("v1/factory/transfers")
    suspend fun getTransfers(
        @Query("state") state: String? = null,
        @Query("limit") limit: Int = 50,
        @Query("offset") offset: Int = 0,
    ): Response<TransferListResponse>

    @GET("v1/factory/transfers/{id}")
    suspend fun getTransfer(@Path("id") id: String): Response<Transfer>

    @POST("v1/factory/transfers/{id}/transition")
    suspend fun transitionTransfer(
        @Path("id") id: String,
        @Body body: TransitionRequest,
    ): Response<Transfer>

    // ── Loading Bay (transfers filtered by loading states) ──
    @GET("v1/factory/transfers")
    suspend fun getLoadingBayTransfers(
        @Query("states") states: String = "APPROVED,LOADING,DISPATCHED",
        @Query("limit") limit: Int = 100,
    ): Response<TransferListResponse>

    // ── Dispatch ──
    @POST("v1/factory/dispatch")
    suspend fun dispatch(@Body body: DispatchRequest): Response<DispatchResponse>

    // ── Fleet ──
    @GET("v1/factory/fleet")
    suspend fun getFleet(): Response<VehicleListResponse>

    // ── Staff ──
    @GET("v1/factory/staff")
    suspend fun getStaff(): Response<StaffListResponse>

    // ── Insights ──
    @GET("v1/warehouse/replenishment/insights")
    suspend fun getInsights(
        @Query("limit") limit: Int = 100,
    ): Response<InsightListResponse>
}
