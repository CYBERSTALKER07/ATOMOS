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

    static func shopClosedActive() async throws -> [ShopClosedAttemptRow] {
        let resp: ShopClosedActiveResponse = try await APIClient.shared.get("v1/supplier/shop-closed/active")
        return resp.data
    }

    static func negotiationsPending() async throws -> [NegotiationProposalRow] {
        let resp: NegotiationPendingResponse = try await APIClient.shared.get("v1/supplier/negotiations/pending")
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

    static func dispatchPreview(warehouseId: String? = nil) async throws -> SupplierDispatchPreview {
        var query: [String: String] = [:]
        if let warehouseId, !warehouseId.isEmpty { query["warehouse_id"] = warehouseId }
        return try await APIClient.shared.get("v1/supplier/dispatch/preview", query: query)
    }

    static func pricingRules() async throws -> SupplierPricingRule {
        try await APIClient.shared.get("v1/supplier/pricing/rules")
    }

    static func topology() async throws -> SupplierTopologyResponse {
        try await APIClient.shared.get("v1/supplier/topology")
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

    static func triggerReplenishment() async throws -> SupplierReplenishmentTriggerResponse {
        try await APIClient.shared.postEmpty("v1/supplier/replenishment/trigger")
    }

    static func createFleetDriver(_ request: FleetDriverCreateRequest) async throws -> FleetDriversResponse {
        try await APIClient.shared.post("v1/supplier/fleet/drivers", body: request)
    }

    static func createFleetVehicle(_ request: FleetVehicleCreateRequest) async throws -> FleetVehiclesResponse {
        try await APIClient.shared.post("v1/supplier/fleet/vehicles", body: request)
    }

    static func updateProfile(_ request: SupplierProfileUpdateRequest) async throws -> SupplierProfile {
        try await APIClient.shared.put("v1/supplier/profile", body: request)
    }
}
