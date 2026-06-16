package com.pegasusx.warehouse.data.remote

import com.pegasusx.warehouse.data.model.*
import com.pegasusx.warehouse.util.WarehouseIdempotencyKeys
import retrofit2.Response
import javax.inject.Inject
import javax.inject.Singleton

/** Portal-parity warehouse ops API facade for native screens. */
@Singleton
class WarehouseOperationsRepository @Inject constructor(
    private val api: WarehouseApi,
) {
    suspend fun emergencyTransfer(body: EmergencyTransferRequest): Response<TransferMutationResponse> =
        api.emergencyTransfer(
            body,
            WarehouseIdempotencyKeys.emergencyTransfer(body.totalVolumeVu, body.notes),
        )

    suspend fun forceReceive(body: ForceReceiveRequest): Response<TransferMutationResponse> =
        api.forceReceive(
            body,
            WarehouseIdempotencyKeys.forceReceive(body.totalVolumeVu, body.notes, body.factoryId),
        )

    suspend fun receiveTransfer(transferId: String): Response<TransferMutationResponse> =
        api.receiveTransfer(transferId, WarehouseIdempotencyKeys.receiveTransfer(transferId))

    suspend fun getReplenishmentInsights(): Response<ReplenishmentInsightsResponse> =
        api.getReplenishmentInsights()

    suspend fun replenishmentInsightAction(
        insightId: String,
        action: String,
    ): Response<ReplenishmentInsightActionResponse> =
        api.replenishmentInsightAction(
            insightId,
            action,
            WarehouseIdempotencyKeys.replenishmentInsightAction(insightId, action),
        )

    suspend fun getOpsFinancials(period: String? = null): Response<OpsFinancialsResponse> =
        api.getOpsFinancials(period)

    suspend fun delayOrder(
        orderId: String,
        reason: String? = null,
    ): Response<WarehouseOrderMutationResponse> =
        api.delayOrder(
            orderId,
            WarehouseOrderMutationRequest(reason),
            WarehouseIdempotencyKeys.orderDelay(orderId),
        )

    suspend fun rejectOrder(
        orderId: String,
        reason: String,
    ): Response<WarehouseOrderMutationResponse> =
        api.rejectOrder(
            orderId,
            WarehouseOrderMutationRequest(reason),
            WarehouseIdempotencyKeys.orderReject(orderId, reason),
        )

    suspend fun overflowOrder(
        orderId: String,
        reason: String? = null,
    ): Response<WarehouseOrderMutationResponse> =
        api.overflowOrder(
            orderId,
            WarehouseOrderMutationRequest(reason),
            WarehouseIdempotencyKeys.orderOverflow(orderId),
        )

    suspend fun refreshToken(refreshToken: String): Response<AuthResponse> =
        api.refreshToken(RefreshTokenRequest(refreshToken))

    suspend fun getFleetLiveMap(warehouseId: String? = null): Response<WarehouseFleetLiveMapResponse> =
        api.getFleetLiveMap(warehouseId)

    suspend fun getDispatchSettings(): Response<DispatchSettingsResponse> =
        api.getDispatchSettings()

    suspend fun patchDispatchSettings(enabled: Boolean): Response<Map<String, String>> =
        api.patchDispatchSettings(
            DispatchSettingsPatchRequest(autoDispatchEnabled = enabled),
            WarehouseIdempotencyKeys.dispatchSettings(enabled),
        )

    suspend fun getSupplyRequest(id: String): Response<WarehouseSupplyRequest> =
        api.getSupplyRequest(id)

    suspend fun getPaymentConfig(): Response<PaymentConfigResponse> =
        api.getPaymentConfig()
}
