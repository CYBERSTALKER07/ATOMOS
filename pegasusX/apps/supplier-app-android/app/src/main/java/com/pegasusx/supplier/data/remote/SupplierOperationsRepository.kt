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

    suspend fun getManifestDetail(manifestId: String): Response<SupplierManifestDetail> =
        api.getManifestDetail(manifestId)

    suspend fun startManifestLoading(manifestId: String, idempotencyKey: String): Response<JsonElement> =
        api.startManifestLoading(manifestId, idempotencyKey)

    suspend fun injectManifestOrder(
        manifestId: String,
        body: SupplierManifestInjectOrderRequest,
        idempotencyKey: String,
    ): Response<JsonElement> = api.injectManifestOrder(manifestId, idempotencyKey, body)

    suspend fun sealManifest(manifestId: String, idempotencyKey: String): Response<JsonElement> =
        api.sealManifest(manifestId, idempotencyKey)

    suspend fun getManifestExceptions(escalated: Boolean = false): Response<SupplierManifestExceptionsResponse> =
        api.getManifestExceptions(if (escalated) true else null)

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

    suspend fun listRetailerPriceOverrides(
        retailerId: String? = null,
        productId: String? = null,
    ): Response<RetailerPriceOverridesResponse> = api.listRetailerPriceOverrides(retailerId, productId)

    suspend fun createRetailerPriceOverride(
        body: CreateRetailerPriceOverrideRequest,
    ): Response<CreateRetailerPriceOverrideResponse> = api.createRetailerPriceOverride(body)

    suspend fun deleteRetailerPriceOverride(overrideId: String): Response<JsonElement> =
        api.deleteRetailerPriceOverride(overrideId)

    suspend fun getDashboard(): Response<SupplierDashboard> = api.getDashboard()

    suspend fun getTopology(): Response<SupplierTopologyResponse> = api.getTopology()

    suspend fun updateTopology(body: SupplierTopologyUpdateRequest): Response<SupplierTopologyResponse> =
        api.updateTopology(body)

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

    suspend fun updateOrgMember(
        userId: String,
        body: SupplierOrgMemberUpdateRequest,
        idempotencyKey: String,
    ): Response<SupplierOrgMembersResponse> = api.updateOrgMember(userId, idempotencyKey, body)

    suspend fun deactivateOrgMember(
        userId: String,
        idempotencyKey: String,
    ): Response<SupplierOrgMembersResponse> = api.deactivateOrgMember(userId, idempotencyKey)

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

    suspend fun getAnalyticsVelocity(): Response<SupplierAnalyticsVelocityResponse> =
        api.getAnalyticsVelocity()

    suspend fun getAnalyticsRevenue(): Response<SupplierAnalyticsRevenueResponse> =
        api.getAnalyticsRevenue()

    suspend fun getDemandToday(): Response<SupplierDemandSummaryResponse> =
        api.getDemandToday()

    suspend fun getEmpathyAdoption(): Response<SupplierEmpathyAdoption> =
        api.getEmpathyAdoption()

    suspend fun postBroadcast(body: SupplierBroadcastRequest): Response<SupplierBroadcastResponse> =
        api.postBroadcast(body)

    suspend fun issuePaymentBypass(body: PaymentBypassRequest): Response<PaymentBypassResponse> =
        api.issuePaymentBypass(body)
}
