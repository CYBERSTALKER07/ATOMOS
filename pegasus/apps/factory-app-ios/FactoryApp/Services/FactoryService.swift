import Foundation

enum FactoryService {
    private static let api = APIClient.shared

    // MARK: - Auth
    static func login(phone: String, password: String) async throws -> AuthResponse {
        try await api.post("v1/auth/factory/login", body: LoginRequest(phone: phone, password: password))
    }

    // MARK: - Dashboard
    static func dashboard() async throws -> DashboardStats {
        try await api.get("v1/factory/dashboard")
    }

    static func profile() async throws -> FactoryProfile {
        try await api.get("v1/factory/profile")
    }

    static func analyticsOverview(from: String? = nil, to: String? = nil) async throws -> FactoryAnalyticsOverview {
        var query: [String: String] = [:]
        if let from { query["from"] = from }
        if let to { query["to"] = to }
        return try await api.get("v1/factory/analytics/overview", query: query)
    }

    // MARK: - Transfers
    static func transfers(state: String? = nil, limit: Int = 50) async throws -> TransferListResponse {
        var query: [String: String] = ["limit": "\(limit)"]
        if let state { query["state"] = state }
        return try await api.get("v1/factory/transfers", query: query)
    }

    static func transfer(id: String) async throws -> Transfer {
        try await api.get("v1/factory/transfers/\(id)")
    }

    static func transitionTransfer(id: String, target: String) async throws -> Transfer {
        try await api.post("v1/factory/transfers/\(id)/transition", body: TransitionRequest(targetState: target))
    }

    static func createTransfer(_ req: FactoryCreateTransferRequest) async throws -> FactoryCreateTransferResponse {
        try await api.post("v1/factory/transfers/create", body: req)
    }

    // MARK: - Loading Bay
    static func loadingBayTransfers() async throws -> TransferListResponse {
        try await api.get("v1/factory/transfers", query: ["states": "APPROVED,LOADING,DISPATCHED", "limit": "100"])
    }

    // MARK: - Dispatch
    static func dispatch(transferIds: [String]) async throws -> DispatchResponse {
        try await api.post("v1/factory/dispatch", body: DispatchRequest(transferIds: transferIds))
    }

    // MARK: - Supply Requests
    static func supplyRequests() async throws -> [SupplyRequest] {
        try await api.get("v1/factory/supply-requests")
    }

    static func transitionSupplyRequest(id: String, action: String) async throws -> SupplyRequestTransitionResponse {
        try await api.patch(
            "v1/factory/supply-requests/\(id)",
            body: SupplyRequestTransitionRequest(action: action, transferOrderId: nil)
        )
    }

    // MARK: - Payload Override / Manifests
    static func loadingManifests() async throws -> ManifestListResponse {
        try await api.get("v1/factory/manifests", query: ["state": "LOADING"])
    }

    static func manifestDetail(id: String) async throws -> Manifest {
        try await api.get("v1/factory/manifests/\(id)")
    }

    static func transitionManifest(id: String, action: String) async throws -> FactoryManifestTransitionResponse {
        struct EmptyBody: Encodable {}
        return try await api.post("v1/factory/manifests/\(id)/\(action)", body: EmptyBody())
    }

    static func rebalanceManifestTransfer(sourceManifestId: String, targetManifestId: String, transferId: String) async throws -> ManifestRebalanceResponse {
        try await api.post(
            "v1/factory/manifests/rebalance",
            body: ManifestRebalanceRequest(
                sourceManifestId: sourceManifestId,
                targetManifestId: targetManifestId,
                transferIds: [transferId]
            )
        )
    }

    static func cancelManifestTransfer(manifestId: String, transferId: String) async throws -> ManifestCancelTransferResponse {
        try await api.post(
            "v1/factory/manifests/cancel-transfer",
            body: ManifestCancelTransferRequest(manifestId: manifestId, transferId: transferId)
        )
    }

    static func cancelManifest(manifestId: String) async throws -> ManifestCancelResponse {
        try await api.post(
            "v1/factory/manifests/cancel",
            body: ManifestCancelRequest(manifestId: manifestId)
        )
    }

    // MARK: - Fleet
    static func fleet() async throws -> VehicleListResponse {
        try await api.get("v1/factory/fleet")
    }

    static func fleetDrivers() async throws -> FactoryFleetDriverListResponse {
        try await api.get("v1/factory/fleet/drivers")
    }

    static func fleetVehicles() async throws -> FactoryFleetVehicleListResponse {
        try await api.get("v1/factory/fleet/vehicles")
    }

    // MARK: - Staff
    static func staff() async throws -> StaffListResponse {
        try await api.get("v1/factory/staff")
    }

    static func staffDetail(id: String) async throws -> FactoryStaffDetail {
        try await api.get("v1/factory/staff/\(id)")
    }

    // MARK: - Insights
    static func insights() async throws -> InsightListResponse {
        try await api.get("v1/warehouse/replenishment/insights", query: ["limit": "100"])
    }
}
