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

    static func resolveShopClosed(
        _ request: ShopClosedResolveRequest,
        idempotencyKey: String
    ) async throws -> NegotiationResolveResponse {
        try await APIClient.shared.post(
            "v1/supplier/shop-closed/resolve",
            body: request,
            idempotencyKey: idempotencyKey
        )
    }

    static func resolveNegotiation(
        _ request: NegotiationResolveRequest,
        idempotencyKey: String
    ) async throws -> NegotiationResolveResponse {
        try await APIClient.shared.post(
            "v1/supplier/negotiate/resolve",
            body: request,
            idempotencyKey: idempotencyKey
        )
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

    static func executeDispatch(
        warehouseId: String? = nil,
        idempotencyKey: String
    ) async throws {
        var query: [String: String] = [:]
        if let warehouseId, !warehouseId.isEmpty {
            query["warehouse_id"] = warehouseId
        }
        try await APIClient.shared.postVoid(
            "v1/supplier/dispatch/execute",
            body: ["mode": "AUTO"],
            query: query,
            idempotencyKey: idempotencyKey
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
        try await APIClient.shared.post(
            "v1/supplier/pricing/retailer-overrides",
            body: request,
            idempotencyKey: SupplierIdempotencyKeys.retailerPriceOverrideCreate(
                scopeId: await SupplierIdempotencyKeys.supplierScopeId(),
                retailerId: request.retailerId,
                productId: request.productId,
                priceMinor: Int64(request.price)
            )
        )
    }

    static func deleteRetailerPriceOverride(overrideId: String) async throws {
        try await APIClient.shared.deleteVoid(
            "v1/supplier/pricing/retailer-overrides/\(overrideId)",
            idempotencyKey: SupplierIdempotencyKeys.retailerPriceOverrideDelete(
                scopeId: await SupplierIdempotencyKeys.supplierScopeId(),
                overrideId: overrideId
            )
        )
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

    static func returns(status: String = "PENDING", limit: Int = 100, offset: Int = 0) async throws -> SupplierReturnsResponse {
        try await APIClient.shared.get(
            "v1/supplier/returns",
            query: [
                "status": status,
                "limit": String(limit),
                "offset": String(offset),
            ]
        )
    }

    static func resolveReturn(
        returnId: String,
        resolution: String,
        notes: String = "",
        idempotencyKey: String
    ) async throws {
        let body = ResolveReturnRequest(
            returnId: returnId,
            lineItemId: returnId,
            resolution: resolution,
            notes: notes
        )
        try await APIClient.shared.postVoid(
            "v1/supplier/returns/resolve",
            body: body,
            idempotencyKey: idempotencyKey
        )
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

    static func demandHistory() async throws -> SupplierDemandHistoryResponse {
        try await APIClient.shared.get("v1/supplier/analytics/demand/history")
    }

    static func updateOrgMember(
        _ userId: String,
        request: SupplierOrgMemberUpdateRequest,
        idempotencyKey: String
    ) async throws -> [SupplierOrgMember] {
        let resp: SupplierOrgMembersResponse = try await APIClient.shared.patch(
            "v1/supplier/org/members/\(userId)",
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
        let fingerprint = [
            request.legalName ?? "",
            request.contactName ?? "",
            request.email ?? "",
            request.phone ?? "",
        ].joined(separator: "|")
        return try await APIClient.shared.put(
            "v1/supplier/profile",
            body: request,
            idempotencyKey: SupplierIdempotencyKeys.profileUpdate(
                scopeId: await SupplierIdempotencyKeys.supplierScopeId(),
                payloadFingerprint: fingerprint
            )
        )
    }

    static func updateInventory(_ request: InventoryPatchRequest) async throws {
        let scope = await SupplierIdempotencyKeys.supplierScopeId()
        let sku = request.sku ?? request.skuId ?? ""
        try await APIClient.shared.patchVoid(
            "v1/supplier/inventory",
            body: request,
            idempotencyKey: SupplierIdempotencyKeys.inventoryAdjust(
                scopeId: scope,
                skuId: sku,
                quantityDelta: request.quantityDelta ?? 0,
                version: request.quantity ?? 0
            )
        )
    }

    static func importInventoryCSV(_ csv: String, idempotencyKey: String) async throws -> SupplierInventoryImportResult {
        try await APIClient.shared.postRaw(
            "v1/supplier/inventory/import",
            body: Data(csv.utf8),
            contentType: "text/csv",
            idempotencyKey: idempotencyKey
        )
    }

    static func createImportSession(fileName: String, fileSizeBytes: Int, idempotencyKey: String) async throws -> SupplierImportSessionCreateResponse {
        struct CreateBody: Encodable {
            let fileName: String
            let fileSizeBytes: Int
            enum CodingKeys: String, CodingKey {
                case fileName = "file_name"
                case fileSizeBytes = "file_size_bytes"
            }
        }
        return try await APIClient.shared.post(
            "v1/supplier/inventory/imports",
            body: CreateBody(fileName: fileName, fileSizeBytes: fileSizeBytes),
            idempotencyKey: idempotencyKey
        )
    }

    static func ingestImportSession(sessionId: String, csv: String, idempotencyKey: String) async throws -> SupplierImportIngestResponse {
        try await APIClient.shared.postRaw(
            "v1/supplier/inventory/imports/\(sessionId)/ingest",
            body: Data(csv.utf8),
            contentType: "text/csv",
            idempotencyKey: idempotencyKey
        )
    }

    static func getImportSession(sessionId: String) async throws -> SupplierImportSession {
        try await APIClient.shared.get("v1/supplier/inventory/imports/\(sessionId)")
    }

    static func getImportMapping(sessionId: String) async throws -> SupplierImportMappingResponse {
        try await APIClient.shared.get("v1/supplier/inventory/imports/\(sessionId)/mapping")
    }

    static func approveImportSession(sessionId: String, idempotencyKey: String) async throws {
        struct ApproveResponse: Decodable {
            let sessionId: String
            enum CodingKeys: String, CodingKey { case sessionId = "session_id" }
        }
        let _: ApproveResponse = try await APIClient.shared.post(
            "v1/supplier/inventory/imports/\(sessionId)/approve",
            body: [String: String](),
            idempotencyKey: idempotencyKey
        )
    }

    static func applyImportSession(sessionId: String, idempotencyKey: String) async throws -> SupplierImportApplyResponse {
        try await APIClient.shared.post(
            "v1/supplier/inventory/imports/\(sessionId)/apply",
            body: [String: String](),
            idempotencyKey: idempotencyKey
        )
    }

    static func recordChargeback(_ request: PaymentChargebackRequest, idempotencyKey: String) async throws -> PaymentChargebackResponse {
        try await APIClient.shared.post("v1/payment/chargeback", body: request, idempotencyKey: idempotencyKey)
    }

    static func recordChargebackReversal(_ request: PaymentChargebackReversalRequest, idempotencyKey: String) async throws -> PaymentChargebackReversalResponse {
        try await APIClient.shared.post("v1/payment/chargeback/reversal", body: request, idempotencyKey: idempotencyKey)
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

    static func issuePaymentBypass(
        _ request: PaymentBypassRequest,
        idempotencyKey: String
    ) async throws -> PaymentBypassResponse {
        try await APIClient.shared.post(
            "v1/supplier/orders/payment-bypass",
            body: request,
            idempotencyKey: idempotencyKey
        )
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

    static func broadcast(
        _ request: SupplierBroadcastRequest,
        idempotencyKey: String
    ) async throws -> SupplierBroadcastResponse {
        try await APIClient.shared.post(
            "v1/supplier/broadcast",
            body: request,
            idempotencyKey: idempotencyKey
        )
    }

    static func exceptionMap(windowHours: Int = 24) async throws -> ExceptionMapResponse {
        try await APIClient.shared.get(
            "v1/supplier/ops/exception-map",
            query: ["window_hours": String(windowHours)]
        )
    }

    static func previewRetailerPriceOverride(
        _ request: RetailerOverridePreviewRequest
    ) async throws -> RetailerOverridePreview {
        try await APIClient.shared.post(
            "v1/supplier/pricing/retailer-overrides/preview",
            body: request
        )
    }

    static func getWarehouseOrder(orderId: String, warehouseId: String) async throws -> WarehouseOrderDetail {
        try await APIClient.shared.get(
            "v1/warehouse/ops/orders/\(orderId)",
            query: ["warehouse_id": warehouseId]
        )
    }

    static func proposeWarehouseOrder(
        orderId: String,
        warehouseId: String,
        proposedDeliveryDate: String,
        reason: String,
        idempotencyKey: String
    ) async throws -> WarehouseOrderMutationResponse {
        var components = URLComponents()
        components.queryItems = [URLQueryItem(name: "warehouse_id", value: warehouseId)]
        let query = components.percentEncodedQuery.map { "?\($0)" } ?? ""
        return try await APIClient.shared.post(
            "v1/warehouse/ops/orders/\(orderId)/propose-delivery\(query)",
            body: WarehouseProposeDeliveryRequest(proposedDeliveryDate: proposedDeliveryDate, reason: reason),
            idempotencyKey: idempotencyKey
        )
    }

    static func rejectWarehouseOrder(
        orderId: String,
        warehouseId: String,
        reason: String,
        idempotencyKey: String
    ) async throws -> WarehouseOrderMutationResponse {
        var components = URLComponents()
        components.queryItems = [URLQueryItem(name: "warehouse_id", value: warehouseId)]
        let query = components.percentEncodedQuery.map { "?\($0)" } ?? ""
        return try await APIClient.shared.post(
            "v1/warehouse/ops/orders/\(orderId)/reject\(query)",
            body: WarehouseOrderMutationRequest(reason: reason),
            idempotencyKey: idempotencyKey
        )
    }
}

/// Deterministic idempotency keys — aligned with @pegasusx/api-client idempotency.ts
enum SupplierIdempotency {
    static func resolveReturn(returnId: String, resolution: String) -> String {
        "supplier-resolve-return:\(returnId):\(resolution.uppercased())"
    }

    static func dispatch(
        supplierId: String,
        warehouseId: String,
        mode: String,
        routeFingerprint: String
    ) -> String {
        "supplier-dispatch:\(supplierId):\(warehouseId):\(mode):\(stableHash(routeFingerprint))"
    }

    static func broadcast(scopeId: String, role: String, title: String, body: String) -> String {
        "supplier-broadcast:\(scopeId):\(stableHash("\(role):\(title):\(body)"))"
    }

    static func paymentBypass(orderId: String, reason: String) -> String {
        "supplier-payment-bypass:\(orderId):\(stableHash(reason))"
    }

    private static func stableHash(_ input: String) -> String {
        var hash: UInt32 = 2166136261
        for scalar in input.unicodeScalars {
            hash ^= scalar.value
            hash = hash &* 16777619
        }
        return String(hash, radix: 36)
    }
}
