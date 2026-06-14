import Foundation

/// Portal-parity supplier ops API for native screens (exceptions, dispatch, treasury reads).
enum SupplierOperationsService {
    static func refreshToken(_ refreshToken: String) async throws -> LoginResponse {
        try await APIClient.shared.post(
            "v1/auth/supplier/refresh",
            body: RefreshTokenRequest(refreshToken: refreshToken)
        )
    }

    static func exceptions() async throws -> [SupplierExceptionRow] {
        let resp: SupplierExceptionsResponse = try await APIClient.shared.get("v1/supplier/exceptions")
        return resp.exceptions
    }

    static func shopClosedActive(limit: Int = 500, offset: Int = 0) async throws -> [ShopClosedAttemptRow] {
        let resp: ShopClosedActiveResponse = try await APIClient.shared.get(
            "v1/supplier/shop-closed/active?limit=\(limit)&offset=\(offset)"
        )
        return resp.data
    }

    static func negotiationsPending(limit: Int = 500, offset: Int = 0) async throws -> [NegotiationProposalRow] {
        let resp: NegotiationPendingResponse = try await APIClient.shared.get(
            "v1/supplier/negotiations/pending?limit=\(limit)&offset=\(offset)"
        )
        return resp.data
    }

    static func resolveShopClosed(_ request: ShopClosedResolveRequest) async throws -> NegotiationResolveResponse {
        try await APIClient.shared.post("v1/supplier/shop-closed/resolve", body: request)
    }

    static func resolveNegotiation(_ request: NegotiationResolveRequest) async throws -> NegotiationResolveResponse {
        try await APIClient.shared.post("v1/supplier/negotiate/resolve", body: request)
    }

    static func manifests() async throws -> [SupplierManifestRow] {
        let resp: SupplierManifestsResponse = try await APIClient.shared.get("v1/supplier/manifests")
        return resp.manifests
    }

    static func manifestDetail(_ manifestId: String) async throws -> SupplierManifestDetail {
        try await APIClient.shared.get("v1/supplier/manifests/\(manifestId)")
    }

    static func startManifestLoading(_ manifestId: String, idempotencyKey: String) async throws {
        try await APIClient.shared.postVoid(
            "v1/supplier/manifests/\(manifestId)/start-loading",
            body: [String: String](),
            idempotencyKey: idempotencyKey
        )
    }

    static func injectManifestOrder(
        _ manifestId: String,
        request: SupplierManifestInjectOrderRequest,
        idempotencyKey: String
    ) async throws {
        try await APIClient.shared.postVoid(
            "v1/supplier/manifests/\(manifestId)/inject-order",
            body: request,
            idempotencyKey: idempotencyKey
        )
    }

    static func sealManifest(_ manifestId: String, idempotencyKey: String) async throws {
        try await APIClient.shared.postVoid(
            "v1/supplier/manifests/\(manifestId)/seal",
            body: [String: String](),
            idempotencyKey: idempotencyKey
        )
    }

    static func manifestExceptions(escalatedOnly: Bool = false) async throws -> [SupplierManifestExceptionRow] {
        var query: [String: String] = [:]
        if escalatedOnly { query["escalated"] = "true" }
        let resp: SupplierManifestExceptionsResponse = try await APIClient.shared.get(
            "v1/supplier/manifest-exceptions",
            query: query
        )
        return resp.exceptions
    }

    static func dispatchPreview(warehouseId: String? = nil) async throws -> SupplierDispatchPreview {
        var query: [String: String] = [:]
        if let warehouseId, !warehouseId.isEmpty { query["warehouse_id"] = warehouseId }
        return try await APIClient.shared.get("v1/supplier/dispatch/preview", query: query)
    }

    static func createDispatchPreview(body: [String: String]) async throws -> SupplierDispatchPreview {
        try await APIClient.shared.post("v1/supplier/dispatch/preview", body: body)
    }

    static func executeDispatch(warehouseId: String? = nil) async throws {
        var query: [String: String] = [:]
        if let warehouseId, !warehouseId.isEmpty {
            query["warehouse_id"] = warehouseId
        }
        try await APIClient.shared.postVoid(
            "v1/supplier/dispatch/execute",
            body: ["mode": "AUTO"],
            query: query
        )
    }

    static func pricingRules() async throws -> SupplierPricingRule {
        try await APIClient.shared.get("v1/supplier/pricing/rules")
    }

    static func listRetailerPriceOverrides(
        retailerId: String? = nil,
        productId: String? = nil
    ) async throws -> RetailerPriceOverridesResponse {
        var query: [String: String] = [:]
        if let retailerId, !retailerId.isEmpty {
            query["retailer_id"] = retailerId
        }
        if let productId, !productId.isEmpty {
            query["product_id"] = productId
        }
        return try await APIClient.shared.get("v1/supplier/pricing/retailer-overrides", query: query)
    }

    static func createRetailerPriceOverride(
        _ request: CreateRetailerPriceOverrideRequest
    ) async throws -> CreateRetailerPriceOverrideResponse {
        try await APIClient.shared.post("v1/supplier/pricing/retailer-overrides", body: request)
    }

    static func deleteRetailerPriceOverride(overrideId: String) async throws {
        try await APIClient.shared.deleteVoid("v1/supplier/pricing/retailer-overrides/\(overrideId)")
    }

    static func topology() async throws -> SupplierTopologyResponse {
        try await APIClient.shared.get("v1/supplier/topology")
    }

    static func updateTopology(_ request: SupplierTopologyUpdateRequest) async throws -> SupplierTopologyResponse {
        try await APIClient.shared.put("v1/supplier/topology", body: request)
    }

    static func supplyLanes() async throws -> [SupplierSupplyLaneRow] {
        let resp: SupplierSupplyLanesResponse = try await APIClient.shared.get("v1/supplier/supply-lanes")
        return resp.lanes
    }

    static func activity() async throws -> [SupplierActivityEvent] {
        let resp: SupplierActivityResponse = try await APIClient.shared.get("v1/supplier/activity")
        return resp.events
    }

    static func fleetOrders() async throws -> [SupplierFleetOrderRow] {
        try await APIClient.shared.get("v1/supplier/fleet/orders")
    }

    static func fleetLiveMap() async throws -> SupplierFleetLiveMapResponse {
        try await APIClient.shared.get("v1/supplier/fleet/live-map")
    }

    static func wsSession() async throws -> SupplierWsSessionResponse {
        try await APIClient.shared.get("v1/supplier/ws-session")
    }

    static func orders(
        status: String? = nil,
        filter: String? = nil,
        limit: Int? = nil,
        offset: Int? = nil
    ) async throws -> SupplierOrdersResponse {
        var query: [String: String] = [:]
        if let status, !status.isEmpty { query["status"] = status }
        if let filter, !filter.isEmpty { query["filter"] = filter }
        if let limit { query["limit"] = String(limit) }
        if let offset { query["offset"] = String(offset) }
        return try await APIClient.shared.get("v1/supplier/orders", query: query)
    }

    static func paymentLedger(currency: String? = nil) async throws -> PaymentLedgerResponse {
        var query: [String: String] = [:]
        if let currency, !currency.isEmpty { query["currency"] = currency }
        return try await APIClient.shared.get("v1/payment/ledger", query: query)
    }

    static func paymentSettlementAuthority() async throws -> SettlementAuthorityResponse {
        try await APIClient.shared.get("v1/payment/settlement/authority", query: ["group_limit": "200"])
    }

    static func paymentReconciliationMismatches() async throws -> ReconciliationMismatchResponse {
        try await APIClient.shared.get(
            "v1/payment/reconciliation/mismatches",
            query: ["group_limit": "200", "mismatch_threshold_minor": "1"]
        )
    }

    static func orgMembers() async throws -> [SupplierOrgMember] {
        let resp: SupplierOrgMembersResponse = try await APIClient.shared.get("v1/supplier/org/members")
        return resp.items
    }

    static func createOrgMember(_ request: SupplierOrgMemberCreateRequest, idempotencyKey: String) async throws -> [SupplierOrgMember] {
        let resp: SupplierOrgMembersResponse = try await APIClient.shared.post(
            "v1/supplier/org/members",
            body: request,
            idempotencyKey: idempotencyKey
        )
        return resp.items
    }

    static func deactivateOrgMember(_ userId: String, idempotencyKey: String) async throws -> [SupplierOrgMember] {
        let resp: SupplierOrgMembersResponse = try await APIClient.shared.delete(
            "v1/supplier/org/members/\(userId)",
            idempotencyKey: idempotencyKey
        )
        return resp.items
    }

    static func triggerReplenishment() async throws -> SupplierReplenishmentTriggerResponse {
        try await APIClient.shared.postEmpty("v1/supplier/replenishment/trigger")
    }

    static func createFleetDriver(_ request: FleetDriverCreateRequest, idempotencyKey: String) async throws -> FleetDriversResponse {
        try await APIClient.shared.post(
            "v1/supplier/fleet/drivers",
            body: request,
            idempotencyKey: idempotencyKey
        )
    }

    static func createFleetVehicle(_ request: FleetVehicleCreateRequest, idempotencyKey: String) async throws -> FleetVehiclesResponse {
        try await APIClient.shared.post(
            "v1/supplier/fleet/vehicles",
            body: request,
            idempotencyKey: idempotencyKey
        )
    }

    static func updateProfile(_ request: SupplierProfileUpdateRequest) async throws -> SupplierProfile {
        try await APIClient.shared.put("v1/supplier/profile", body: request)
    }

    static func updateInventory(body: [String: String]) async throws {
        try await APIClient.shared.patchVoid("v1/supplier/inventory", body: body)
    }

    static func inventoryAudit() async throws -> [String: String] { // placeholder
        try await APIClient.shared.get("v1/supplier/inventory/audit")
    }

    static func vetOrder(body: [String: String], idempotencyKey: String) async throws {
        try await APIClient.shared.postVoid("v1/supplier/orders/vet", body: body, idempotencyKey: idempotencyKey)
    }

    static func aiRecommendations(status: String? = nil, limit: Int = 50) async throws -> SupplierAIRecommendationsResponse {
        var query: [String: String] = ["limit": String(limit)]
        if let status, !status.isEmpty, status.uppercased() != "ALL" {
            query["status"] = status
        }
        return try await APIClient.shared.get("v1/supplier/ai/recommendations", query: query)
    }

    static func recordAiRecommendationDecision(
        _ request: SupplierAIRecommendationDecisionRequest,
        idempotencyKey: String
    ) async throws -> SupplierAIRecommendationDecisionResponse {
        try await APIClient.shared.post(
            "v1/supplier/ai/recommendations",
            body: request,
            idempotencyKey: idempotencyKey
        )
    }

    static func analyticsVelocity() async throws -> SupplierAnalyticsVelocityResponse {
        try await APIClient.shared.get("v1/supplier/analytics/velocity")
    }

    static func analyticsRevenue() async throws -> SupplierAnalyticsRevenueResponse {
        try await APIClient.shared.get("v1/supplier/analytics/revenue")
    }

    static func demandToday() async throws -> SupplierDemandSummaryResponse {
        try await APIClient.shared.get("v1/supplier/analytics/demand/today")
    }

    static func issuePaymentBypass(_ request: PaymentBypassRequest) async throws -> PaymentBypassResponse {
        try await APIClient.shared.post("v1/supplier/orders/payment-bypass", body: request)
    }

    static func approveEarlyComplete(driverId: String) async throws {
        let key = "supplier-approve-early-complete:\(driverId)"
        try await APIClient.shared.postVoid(
            "v1/supplier/route/approve-early-complete",
            body: ApproveEarlyCompleteRequest(driverId: driverId),
            idempotencyKey: key
        )
    }

    static func empathyAdoption() async throws -> SupplierEmpathyAdoption {
        try await APIClient.shared.get("v1/supplier/empathy/adoption")
    }

    static func broadcast(_ request: SupplierBroadcastRequest) async throws -> SupplierBroadcastResponse {
        try await APIClient.shared.post("v1/supplier/broadcast", body: request)
    }
}
