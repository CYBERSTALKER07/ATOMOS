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

    suspend fun getComplianceDashboard(limit: Int = 100): Response<ComplianceDashboardResponse> =
        api.getComplianceDashboard(limit)

    suspend fun getOrderReceipt(orderId: String): Response<OrderReceiptMeta> =
        api.getOrderReceipt(orderId)

    suspend fun getCashReconciliations(): Response<CashReconciliationsResponse> =
        api.getCashReconciliations()

    suspend fun acceptCashReconciliation(
        id: String,
        idempotencyKey: String,
        note: String? = null,
    ): Response<StatusResponse> =
        api.acceptCashReconciliation(
            id = id,
            idempotencyKey = idempotencyKey,
            body = if (note.isNullOrBlank()) emptyMap() else mapOf("note" to note),
        )

    suspend fun getCreditNotes(): Response<CreditNotesResponse> =
        api.getCreditNotes()

    suspend fun issueCreditNote(id: String, idempotencyKey: String): Response<StatusResponse> =
        api.issueCreditNote(id, idempotencyKey)

    suspend fun getCreditProfiles(
        status: String? = null,
        limit: Int = 100,
    ): Response<CreditProfilesResponse> =
        api.getCreditProfiles(status = status?.ifBlank { null }, limit = limit)

    suspend fun patchRetailerCreditProfile(
        body: RetailerCreditProfilePatchRequest,
        idempotencyKey: String,
    ): Response<StatusResponse> =
        api.patchRetailerCreditProfile(body, idempotencyKey)

    suspend fun resolveException(
        kind: String,
        id: String,
        body: Map<String, String> = emptyMap(),
    ): Response<StatusResponse> =
        api.resolveException(kind, id, body)

    suspend fun getRoutePerformance(): Response<RoutePerformanceResponse> =
        api.getRoutePerformance()

    suspend fun getNotificationPreferences(): Response<NotificationPreferencesResponse> =
        api.getNotificationPreferences()

    suspend fun patchNotificationPreferences(body: NotificationPreferencesPatchRequest): Response<StatusResponse> =
        api.patchNotificationPreferences(body)

    suspend fun getNegotiationsPending(): Response<NegotiationPendingResponse> = api.getNegotiationsPending()

    suspend fun resolveShopClosed(
        body: ShopClosedResolveRequest,
        idempotencyKey: String,
    ): Response<NegotiationResolveResponse> =
        api.resolveShopClosed(body, idempotencyKey)

    suspend fun resolveNegotiation(
        body: NegotiationResolveRequest,
        idempotencyKey: String,
    ): Response<NegotiationResolveResponse> =
        api.resolveNegotiation(body, idempotencyKey)

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

    suspend fun executeDispatch(
        warehouseId: String? = null,
        idempotencyKey: String,
        mode: String = "AUTO",
        forceCapacity: Boolean = false,
        routes: List<SupplierDispatchManualRoute> = emptyList(),
    ): Response<JsonElement> =
        api.executeDispatch(
            warehouseId = warehouseId,
            idempotencyKey = idempotencyKey,
            body = kotlinx.serialization.json.buildJsonObject {
                put("mode", kotlinx.serialization.json.JsonPrimitive(mode))
                if (forceCapacity) {
                    put("force_capacity", kotlinx.serialization.json.JsonPrimitive(true))
                }
                if (routes.isNotEmpty()) {
                    put(
                        "routes",
                        kotlinx.serialization.json.JsonArray(
                            routes.map { route ->
                                kotlinx.serialization.json.buildJsonObject {
                                    put("driver_id", kotlinx.serialization.json.JsonPrimitive(route.driverId))
                                    put(
                                        "order_ids",
                                        kotlinx.serialization.json.JsonArray(
                                            route.orderIds.map { kotlinx.serialization.json.JsonPrimitive(it) },
                                        ),
                                    )
                                }
                            },
                        ),
                    )
                }
            },
        )

    suspend fun getPricingRules(): Response<SupplierPricingRule> = api.getPricingRules()

    suspend fun updatePricingRules(
        body: JsonElement,
        idempotencyKey: String,
    ): Response<SupplierPricingRule> = api.updatePricingRules(body, idempotencyKey)

    suspend fun listRetailerPriceOverrides(
        retailerId: String? = null,
        productId: String? = null,
    ): Response<RetailerPriceOverridesResponse> = api.listRetailerPriceOverrides(retailerId, productId)

    suspend fun createRetailerPriceOverride(
        body: CreateRetailerPriceOverrideRequest,
        idempotencyKey: String,
    ): Response<CreateRetailerPriceOverrideResponse> = api.createRetailerPriceOverride(body, idempotencyKey)

    suspend fun deleteRetailerPriceOverride(
        overrideId: String,
        idempotencyKey: String,
    ): Response<JsonElement> = api.deleteRetailerPriceOverride(overrideId, idempotencyKey)

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

    suspend fun getPulse(): Response<PulseResponse> = api.getPulse()

    suspend fun getFleetOrders(): Response<List<SupplierFleetOrderRow>> = api.getFleetOrders()

    suspend fun getFleetLiveMap(): Response<SupplierFleetLiveMapResponse> = api.getFleetLiveMap()

    suspend fun getMEIONetworkSummary(): Response<SupplierMEIONetworkSummary> = api.getMEIONetworkSummary()

    suspend fun getControlTowerZoneOverrides(): Response<ControlTowerZoneOverridesResponse> =
        api.getControlTowerZoneOverrides()

    suspend fun getPlanningSAndOP(): Response<PlanningSAndOPSnapshot> = api.getPlanningSAndOP()

    suspend fun runPlanningScenario(
        body: PlanningScenarioInput,
        idempotencyKey: String,
    ): Response<PlanningScenarioResult> = api.runPlanningScenario(body, idempotencyKey)

    suspend fun getSeasonalOverrides(): Response<SeasonalTemplatesResponse> = api.getSeasonalOverrides()

    suspend fun createSeasonalOverride(
        body: SeasonalOverrideInput,
        idempotencyKey: String,
    ): Response<SeasonalOverrideRow> = api.createSeasonalOverride(body, idempotencyKey)

    suspend fun getReturnPolicy(): Response<SupplierReturnPolicy> = api.getReturnPolicy()

    suspend fun putReturnPolicy(
        body: SupplierReturnPolicy,
        idempotencyKey: String,
    ): Response<SupplierReturnPolicy> = api.putReturnPolicy(body, idempotencyKey)

    suspend fun getKnowledgeGraph(): Response<SupplierKnowledgeGraph> = api.getKnowledgeGraph()

    suspend fun getReplenishmentPolicies(): Response<SupplierReplenishmentPolicy> = api.getReplenishmentPolicies()

    suspend fun createControlTowerZoneOverride(
        body: ControlTowerZoneOverrideCreateRequest,
        idempotencyKey: String,
    ): Response<ControlTowerZoneOverride> = api.createControlTowerZoneOverride(body, idempotencyKey)

    suspend fun getExceptionMap(windowHours: Int = 24): Response<ExceptionMapResponse> =
        api.getExceptionMap(windowHours)

    suspend fun previewRetailerPriceOverride(body: RetailerOverridePreviewRequest): Response<RetailerOverridePreview> =
        api.previewRetailerPriceOverride(body)

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

    suspend fun getWarehouseOrder(orderId: String, warehouseId: String): Response<WarehouseOrderDetail> =
        api.getWarehouseOrder(orderId, warehouseId)

    suspend fun proposeWarehouseOrder(
        orderId: String,
        warehouseId: String,
        proposedDeliveryDate: String,
        reason: String,
        idempotencyKey: String,
    ): Response<WarehouseOrderMutationResponse> =
        api.proposeWarehouseOrderDelivery(
            orderId,
            warehouseId,
            idempotencyKey,
            WarehouseProposeDeliveryRequest(proposedDeliveryDate = proposedDeliveryDate, reason = reason),
        )

    suspend fun rejectWarehouseOrder(
        orderId: String,
        warehouseId: String,
        reason: String,
        idempotencyKey: String,
    ): Response<WarehouseOrderMutationResponse> =
        api.rejectWarehouseOrder(
            orderId,
            warehouseId,
            idempotencyKey,
            WarehouseOrderMutationRequest(reason = reason),
        )

    suspend fun getReturns(
        status: String = "PENDING",
        limit: Int = 100,
        offset: Int = 0,
    ): Response<SupplierReturnsResponse> = api.getReturns(status, limit, offset)

    suspend fun resolveReturn(
        body: ResolveReturnRequest,
        idempotencyKey: String,
    ): Response<kotlinx.serialization.json.JsonElement> = api.resolveReturn(idempotencyKey, body)

    suspend fun updateProfile(
        body: Map<String, String>,
        idempotencyKey: String,
    ): Response<SupplierProfile> = api.updateProfile(body, idempotencyKey)

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

    suspend fun postBroadcast(
        body: SupplierBroadcastRequest,
        idempotencyKey: String,
    ): Response<SupplierBroadcastResponse> =
        api.postBroadcast(idempotencyKey, body)

    suspend fun issuePaymentBypass(
        body: PaymentBypassRequest,
        idempotencyKey: String,
    ): Response<PaymentBypassResponse> =
        api.issuePaymentBypass(idempotencyKey, body)

    suspend fun vetOrder(body: JsonElement, idempotencyKey: String): Response<JsonElement> =
        api.vetOrder(idempotencyKey, body)

    suspend fun updateInventory(
        body: JsonElement,
        idempotencyKey: String,
    ): Response<JsonElement> = api.updateInventory(body, idempotencyKey)

    suspend fun getDemandHistory(): Response<DemandHistoryResponse> =
        api.getDemandHistory()

    suspend fun recordChargeback(body: JsonElement, idempotencyKey: String): Response<JsonElement> =
        api.recordChargeback(idempotencyKey, body)

    suspend fun recordChargebackReversal(body: JsonElement, idempotencyKey: String): Response<JsonElement> =
        api.recordChargebackReversal(idempotencyKey, body)

    suspend fun listSupplierClaims(status: String? = null, limit: Int = 50): Response<SupplierClaimsListResponse> =
        api.listSupplierClaims(status = status?.ifBlank { null }, limit = limit)

    suspend fun approveClaim(claimId: String, body: ApproveClaimRequest): Response<ApproveClaimResponse> =
        api.approveClaim(claimId, body)

    suspend fun rejectClaim(claimId: String, body: RejectClaimRequest): Response<SupplierClaim> =
        api.rejectClaim(claimId, body)

    suspend fun listClaimChargebacks(limit: Int = 100, orderId: String? = null): Response<ClaimChargebacksResponse> =
        api.listClaimChargebacks(limit = limit, orderId = orderId?.ifBlank { null })

    suspend fun createImportSession(
        idempotencyKey: String,
        body: ImportSessionCreateRequest,
    ): Response<ImportSessionCreateResponse> = api.createImportSession(idempotencyKey, body)

    suspend fun getImportSession(sessionId: String): Response<JsonElement> =
        api.getImportSession(sessionId)

    suspend fun ingestImportSession(
        sessionId: String,
        idempotencyKey: String,
        body: okhttp3.RequestBody,
    ): Response<JsonElement> = api.ingestImportSession(sessionId, idempotencyKey, body)

    suspend fun getImportMapping(sessionId: String): Response<JsonElement> =
        api.getImportMapping(sessionId)

    suspend fun postImportMapping(sessionId: String, body: JsonElement): Response<JsonElement> =
        api.postImportMapping(sessionId, body)

    suspend fun approveImportSession(sessionId: String, idempotencyKey: String): Response<JsonElement> =
        api.approveImportSession(sessionId, idempotencyKey)

    suspend fun applyImportSession(sessionId: String, idempotencyKey: String): Response<JsonElement> =
        api.applyImportSession(sessionId, idempotencyKey)

    suspend fun getCatalogProduct(productId: String): Response<CatalogProduct> =
        api.getCatalogProduct(productId)

    suspend fun recommendReassign(orderId: String): Response<RecommendReassignResponse> =
        api.recommendReassign(
            idempotencyKey = "supplier-recommend-reassign-$orderId",
            body = mapOf("order_id" to orderId)
        )

    suspend fun applyReassign(orderId: String, driverId: String, partial: Boolean): Response<JsonElement> =
        api.applyReassign(
            idempotencyKey = "supplier-apply-reassign-$orderId-$driverId",
            body = ApplyReassignRequest(
                orderId = orderId,
                driverId = driverId,
                partial = partial
            )
        )
}
