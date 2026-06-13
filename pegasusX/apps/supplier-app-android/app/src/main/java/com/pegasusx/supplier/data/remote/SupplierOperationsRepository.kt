package com.pegasusx.supplier.data.remote

import com.pegasusx.supplier.data.model.*
import kotlinx.serialization.json.JsonElement
import retrofit2.Response
import javax.inject.Inject
import javax.inject.Singleton

/** Portal-parity supplier ops API facade for native screens. */
@Singleton
class SupplierOperationsRepository @Inject constructor(
    private val api: SupplierApi,
) {
    suspend fun refreshToken(refreshToken: String): Response<LoginResponse> =
        api.refreshToken(RefreshTokenRequest(refreshToken))

    suspend fun getExceptions(): Response<SupplierExceptionsResponse> = api.getExceptions()

    suspend fun getShopClosedActive(): Response<ShopClosedActiveResponse> = api.getShopClosedActive()

    suspend fun getNegotiationsPending(): Response<NegotiationPendingResponse> = api.getNegotiationsPending()

    suspend fun resolveShopClosed(body: ShopClosedResolveRequest): Response<NegotiationResolveResponse> =
        api.resolveShopClosed(body)

    suspend fun resolveNegotiation(body: NegotiationResolveRequest): Response<NegotiationResolveResponse> =
        api.resolveNegotiation(body)

    suspend fun getManifests(): Response<SupplierManifestsResponse> = api.getManifests()

    suspend fun getDispatchPreview(warehouseId: String? = null): Response<SupplierDispatchPreview> =
        api.getDispatchPreview(warehouseId)

    suspend fun executeDispatch(warehouseId: String? = null): Response<JsonElement> =
        api.executeDispatch(
            warehouseId = warehouseId,
            body = kotlinx.serialization.json.buildJsonObject {
                put("mode", kotlinx.serialization.json.JsonPrimitive("AUTO"))
            },
        )

    suspend fun getPricingRules(): Response<SupplierPricingRule> = api.getPricingRules()

    suspend fun updatePricingRules(body: JsonElement): Response<SupplierPricingRule> =
        api.updatePricingRules(body)

    suspend fun getDashboard(): Response<SupplierDashboard> = api.getDashboard()

    suspend fun getTopology(): Response<SupplierTopologyResponse> = api.getTopology()

    suspend fun getSupplyLanes(): Response<SupplierSupplyLanesResponse> = api.getSupplyLanes()

    suspend fun getAiRecommendations(
        status: String? = null,
        limit: Int = 50,
    ): Response<SupplierAIRecommendationsResponse> = api.getAiRecommendations(status, limit)

    suspend fun recordAiRecommendationDecision(
        body: SupplierAIRecommendationDecisionRequest,
        idempotencyKey: String,
    ): Response<SupplierAIRecommendationDecisionResponse> =
        api.recordAiRecommendationDecision(idempotencyKey, body)

    suspend fun getActivity(): Response<SupplierActivityResponse> = api.getActivity()

    suspend fun getFleetOrders(): Response<List<SupplierFleetOrderRow>> = api.getFleetOrders()

    suspend fun getFleetLiveMap(): Response<SupplierFleetLiveMapResponse> = api.getFleetLiveMap()

    suspend fun getWsSession(): Response<SupplierWsSessionResponse> = api.getWsSession()

    suspend fun getPaymentLedger(currency: String? = null): Response<PaymentLedgerResponse> =
        api.getPaymentLedger(currency)

    suspend fun getPaymentSettlementAuthority(): Response<SettlementAuthorityResponse> =
        api.getPaymentSettlementAuthority()

    suspend fun getPaymentReconciliationMismatches(): Response<ReconciliationMismatchResponse> =
        api.getPaymentReconciliationMismatches()

    suspend fun getOrgMembers(): Response<SupplierOrgMembersResponse> = api.getOrgMembers()

    suspend fun createOrgMember(
        body: SupplierOrgMemberCreateRequest,
        idempotencyKey: String,
    ): Response<SupplierOrgMembersResponse> = api.createOrgMember(idempotencyKey, body)

    suspend fun approveEarlyComplete(
        driverId: String,
        idempotencyKey: String,
    ): Response<JsonElement> = api.approveEarlyComplete(
        idempotencyKey,
        ApproveEarlyCompleteRequest(driverId),
    )

    suspend fun getOrders(
        status: String? = null,
        filter: String? = null,
        limit: Int? = null,
        offset: Int? = null,
    ): Response<SupplierOrdersResponse> = api.getOrders(status, filter, limit, offset)

    suspend fun updateProfile(body: Map<String, String>): Response<SupplierProfile> =
        api.updateProfile(body)

    suspend fun createFleetDriver(
        body: FleetDriverCreateRequest,
        idempotencyKey: String,
    ): Response<FleetDriversResponse> = api.createFleetDriver(idempotencyKey, body)

    suspend fun createFleetVehicle(
        body: FleetVehicleCreateRequest,
        idempotencyKey: String,
    ): Response<FleetVehiclesResponse> = api.createFleetVehicle(idempotencyKey, body)

    suspend fun triggerReplenishment(): Response<SupplierReplenishmentTriggerResponse> =
        api.triggerReplenishment()
}
