package com.pegasusx.warehouse.data.remote

import com.pegasusx.warehouse.data.model.*
import retrofit2.Response
import javax.inject.Inject
import javax.inject.Singleton

/** Portal-parity warehouse ops API facade for native screens. */
@Singleton
class WarehouseOperationsRepository @Inject constructor(
    private val api: WarehouseApi,
) {
    suspend fun emergencyTransfer(body: EmergencyTransferRequest): Response<TransferMutationResponse> =
        api.emergencyTransfer(body)

    suspend fun forceReceive(body: ForceReceiveRequest): Response<TransferMutationResponse> =
        api.forceReceive(body)

    suspend fun receiveTransfer(transferId: String): Response<TransferMutationResponse> =
        api.receiveTransfer(transferId)

    suspend fun getReplenishmentInsights(): Response<ReplenishmentInsightsResponse> =
        api.getReplenishmentInsights()

    suspend fun replenishmentInsightAction(
        insightId: String,
        action: String,
    ): Response<ReplenishmentInsightActionResponse> =
        api.replenishmentInsightAction(insightId, action)

    suspend fun getOpsFinancials(period: String? = null): Response<OpsFinancialsResponse> =
        api.getOpsFinancials(period)

    suspend fun delayOrder(
        orderId: String,
        reason: String? = null,
    ): Response<WarehouseOrderMutationResponse> =
        api.delayOrder(orderId, WarehouseOrderMutationRequest(reason))

    suspend fun rejectOrder(
        orderId: String,
        reason: String,
    ): Response<WarehouseOrderMutationResponse> =
        api.rejectOrder(orderId, WarehouseOrderMutationRequest(reason))

    suspend fun overflowOrder(
        orderId: String,
        reason: String? = null,
    ): Response<WarehouseOrderMutationResponse> =
        api.overflowOrder(orderId, WarehouseOrderMutationRequest(reason))

    suspend fun refreshToken(refreshToken: String): Response<AuthResponse> =
        api.refreshToken(RefreshTokenRequest(refreshToken))
}
