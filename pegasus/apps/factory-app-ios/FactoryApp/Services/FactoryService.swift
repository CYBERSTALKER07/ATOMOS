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

    // MARK: - Loading Bay
    static func loadingBayTransfers() async throws -> TransferListResponse {
        try await api.get("v1/factory/transfers", query: ["states": "APPROVED,LOADING,DISPATCHED", "limit": "100"])
    }

    // MARK: - Dispatch
    static func dispatch(transferIds: [String]) async throws -> DispatchResponse {
        try await api.post("v1/factory/dispatch", body: DispatchRequest(transferIds: transferIds))
    }

    // MARK: - Fleet
    static func fleet() async throws -> VehicleListResponse {
        try await api.get("v1/factory/fleet")
    }

    // MARK: - Staff
    static func staff() async throws -> StaffListResponse {
        try await api.get("v1/factory/staff")
    }

    // MARK: - Insights
    static func insights() async throws -> InsightListResponse {
        try await api.get("v1/warehouse/replenishment/insights", query: ["limit": "100"])
    }
}
