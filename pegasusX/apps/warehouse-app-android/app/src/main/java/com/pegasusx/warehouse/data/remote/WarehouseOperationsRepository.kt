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

    suspend fun listBins(): Response<WarehouseBinListResponse> = api.listBins()

    suspend fun createBin(body: WarehouseBinCreateRequest): Response<WarehouseBinLocation> =
        api.createBin(body, WarehouseIdempotencyKeys.createBin(body.zone.orEmpty()))

    suspend fun putawayLot(body: StockLotPutawayRequest): Response<StockLotPutawayResponse> =
        api.putawayLot(
            body,
            WarehouseIdempotencyKeys.putawayLot(body.productId, body.locationId),
        )

    suspend fun listPickWaves(manifestId: String? = null): Response<PickWaveListResponse> =
        api.listPickWaves(manifestId = manifestId)

    suspend fun createPickWave(manifestId: String): Response<PickWave> =
        api.createPickWave(
            PickWaveCreateRequest(manifestId = manifestId),
            WarehouseIdempotencyKeys.createPickWave(manifestId),
        )

    suspend fun getPickWave(waveId: String): Response<PickWave> = api.getPickWave(waveId)

    suspend fun confirmPickTask(waveId: String, taskId: String, quantityPicked: Long? = null): Response<PickWave> =
        api.confirmPickTask(
            waveId,
            taskId,
            PickTaskConfirmRequest(quantityPicked = quantityPicked),
            WarehouseIdempotencyKeys.confirmPickTask(waveId, taskId),
        )

    suspend fun listCycleCounts(): Response<CycleCountListResponse> = api.listCycleCounts()

    suspend fun createCycleCount(locationId: String, productId: String, expectedQty: Long? = null): Response<CycleCount> =
        api.createCycleCount(
            CycleCountCreateRequest(locationId = locationId, productId = productId, expectedQty = expectedQty),
            "warehouse-cycle-count:${System.currentTimeMillis()}",
        )

    suspend fun submitCycleCount(countId: String, countedQty: Long): Response<CycleCount> =
        api.submitCycleCount(
            countId,
            CycleCountSubmitRequest(countedQty = countedQty),
            "warehouse-cycle-submit:$countId:${System.currentTimeMillis()}",
        )

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

    suspend fun proposeOrderDelivery(
        orderId: String,
        proposedDeliveryDate: String,
        reason: String,
    ): Response<WarehouseOrderMutationResponse> =
        api.proposeOrderDelivery(
            orderId,
            WarehouseProposeDeliveryRequest(proposedDeliveryDate = proposedDeliveryDate, reason = reason),
            WarehouseIdempotencyKeys.orderProposeDelivery(orderId, proposedDeliveryDate, reason),
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

    suspend fun recommendReassign(orderId: String): Response<RecommendReassignResponse> =
        api.recommendReassign(
            RecommendReassignRequest(orderId = orderId),
            WarehouseIdempotencyKeys.recommendReassign(orderId),
        )

    suspend fun reassignOrder(orderId: String, driverId: String, isPartial: Boolean): Response<StatusResponse> =
        api.reassignOrder(
            ReassignOrderRequest(orderId = orderId, toDriverId = driverId, isPartial = isPartial),
            WarehouseIdempotencyKeys.reassignOrder(orderId, driverId),
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
