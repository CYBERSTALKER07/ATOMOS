import Foundation

enum FactoryService {
    private static let api = APIClient.shared

    // MARK: - Auth
    static func login(phone: String, password: String) async throws -> AuthResponse {
        try await api.post("v1/auth/factory/login", body: LoginRequest(phone: phone, password: password))
    }

    static func login(idToken: String) async throws -> AuthResponse {
        try await api.post("v1/auth/factory/login", body: LoginRequest(idToken: idToken))
    }

    static func register(body: [String: String]) async throws -> AuthResponse {
        try await api.post("v1/auth/factory/register", body: body)
    }

    static func refresh() async throws -> AuthResponse {
        struct EmptyBody: Encodable {}
        return try await api.post("v1/auth/factory/refresh", body: EmptyBody())
    }

    static func setup(
        factoryName: String,
        address: String,
        placeId: String?,
        lat: Double,
        lng: Double,
        facilityType: String = "MANUFACTURING"
    ) async throws -> FactorySetupResponse {
        try await api.post(
            "v1/factory/setup",
            body: FactorySetupRequest(
                factoryName: factoryName,
                address: address,
                placeId: placeId,
                lat: lat,
                lng: lng,
                facilityType: facilityType
            )
        )
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
        try await api.post(
            "v1/factory/transfers/\(id)/transition",
            body: TransitionRequest(targetState: target),
            idempotencyKey: FactoryIdempotency.transferTransition(transferId: id, targetState: target)
        )
    }

    static func createTransfer(_ req: FactoryCreateTransferRequest) async throws -> FactoryCreateTransferResponse {
        try await api.post(
            "v1/factory/transfers/create",
            body: req,
            idempotencyKey: FactoryIdempotency.transferCreate(
                orderId: req.orderId ?? "",
                totalVu: Int64(req.totalVu),
                driverId: req.driverId ?? "",
                vehicleId: req.vehicleId ?? ""
            )
        )
    }

    // MARK: - Loading Bay
    static func loadingBayTransfers() async throws -> TransferListResponse {
        try await api.get("v1/factory/transfers", query: ["states": "APPROVED,LOADING,DISPATCHED", "limit": "100"])
    }

    static func pulse() async throws -> FactoryPulseResponse {
        try await api.get("v1/factory/pulse")
    }

    // MARK: - Dispatch
    static func dispatch(transferIds: [String]) async throws -> DispatchResponse {
        try await api.post(
            "v1/factory/dispatch",
            body: DispatchRequest(transferIds: transferIds),
            idempotencyKey: FactoryIdempotency.batchDispatch(transferIds: transferIds)
        )
    }

    // MARK: - Supply Requests
    static func supplyRequests() async throws -> [SupplyRequest] {
        let response: SupplyRequestListResponse = try await api.get("v1/factory/supply-requests")
        return response.requests
    }

    static func acceptSupplyRequest(id: String, body: [String: String] = [:]) async throws {
        try await api.postVoid(
            "v1/factory/supply-requests/\(id)/accept",
            body: body,
            idempotencyKey: FactoryIdempotency.supplyRequestAccept(requestId: id)
        )
    }

    static func transitionSupplyRequest(id: String, action: String) async throws -> SupplyRequestTransitionResponse {
        try await api.patch(
            "v1/factory/supply-requests/\(id)",
            body: SupplyRequestTransitionRequest(action: action, transferOrderId: nil),
            idempotencyKey: FactoryIdempotency.supplyRequestTransition(requestId: id, action: action)
        )
    }

    static func supplyFulfillOptions(id: String) async throws -> SupplyFulfillOptions {
        try await api.get("v1/factory/supply-requests/\(id)/fulfill-options")
    }

    // MARK: - Payload Override / Manifests
    static func loadingManifests() async throws -> ManifestListResponse {
        try await api.get("v1/factory/manifests", query: ["state": "LOADING"])
    }

    static func manifests() async throws -> ManifestListResponse {
        try await api.get("v1/factory/manifests")
    }

    static func manifestDetail(id: String) async throws -> ManifestDetailSnapshot {
        try await api.get("v1/factory/manifests/\(id)")
    }

    static func transitionManifest(id: String, action: String) async throws -> FactoryManifestTransitionResponse {
        struct EmptyBody: Encodable {}
        let idempotencyKey = FactoryIdempotency.forLifecycleAction(action, manifestId: id)
        return try await api.post(
            "v1/factory/manifests/\(id)/\(action)",
            body: EmptyBody(),
            idempotencyKey: idempotencyKey
        )
    }

    static func rebalanceManifestTransfer(sourceManifestId: String, targetManifestId: String, transferId: String) async throws -> ManifestRebalanceResponse {
        try await api.post(
            "v1/factory/manifests/rebalance",
            body: ManifestRebalanceRequest(
                sourceManifestId: sourceManifestId,
                targetManifestId: targetManifestId,
                transferIds: [transferId]
            ),
            idempotencyKey: FactoryIdempotency.rebalance(
                manifestId: sourceManifestId,
                transferId: transferId,
                targetManifestId: targetManifestId
            )
        )
    }

    static func cancelManifestTransfer(manifestId: String, transferId: String) async throws -> ManifestCancelTransferResponse {
        try await api.post(
            "v1/factory/manifests/cancel-transfer",
            body: ManifestCancelTransferRequest(manifestId: manifestId, transferId: transferId),
            idempotencyKey: FactoryIdempotency.cancelTransfer(manifestId: manifestId, transferId: transferId)
        )
    }

    static func cancelManifest(manifestId: String) async throws -> ManifestCancelResponse {
        try await api.post(
            "v1/factory/manifests/cancel",
            body: ManifestCancelRequest(manifestId: manifestId),
            idempotencyKey: FactoryIdempotency.cancelManifest(manifestId: manifestId)
        )
    }

    // MARK: - Fleet
    static func fleet() async throws -> VehicleListResponse {
        try await api.get("v1/factory/fleet")
    }

    static func fleetLiveMap() async throws -> FactoryFleetLiveMapResponse {
        try await api.get("v1/factory/fleet/live-map")
    }

    static func fleetDrivers() async throws -> [FactoryFleetDriverRow] {
        let response: FactoryFleetDriversEnvelope = try await api.get("v1/factory/fleet/drivers")
        return response.drivers
    }

    static func fleetVehicles() async throws -> [FactoryFleetVehicleRow] {
        let response: FactoryFleetVehiclesEnvelope = try await api.get("v1/factory/fleet/vehicles")
        return response.vehicles
    }

    // MARK: - Staff
    static func staff() async throws -> StaffListResponse {
        try await api.get("v1/factory/staff")
    }

    static func createStaff(name: String, role: String) async throws -> StaffMember {
        try await api.post(
            "v1/factory/staff",
            body: CreateStaffRequest(name: name, role: role)
        )
    }

    static func staffDetail(id: String) async throws -> StaffMember {
        try await api.get("v1/factory/staff/\(id)")
    }

    // MARK: - Insights
    static func insights() async throws -> InsightListResponse {
        try await api.get("v1/warehouse/replenishment/insights", query: ["limit": "100"])
    }

    // MARK: - Manifest exceptions
    static func manifestExceptions(escalatedOnly: Bool = false) async throws -> ManifestExceptionListResponse {
        var query: [String: String] = [:]
        if escalatedOnly {
            query["escalated"] = "true"
        }
        return try await api.get("v1/factory/manifest-exceptions", query: query)
    }

    static func resolveManifestException(
        exceptionId: String,
        resolution: String = "RESOLVED",
        note: String = ""
    ) async throws -> ResolveManifestExceptionResponse {
        try await api.post(
            "v1/factory/manifest-exceptions/\(exceptionId)/resolve",
            body: ResolveManifestExceptionRequest(resolution: resolution, note: note)
        )
    }

    // MARK: - Location (all factory-scoped staff may read/write)
    static func factoryLocation() async throws -> FactoryLocationResponse {
        try await api.get("v1/factory/ops/location")
    }

    static func patchFactoryLocation(address: String, placeId: String?, lat: Double, lng: Double) async throws -> FactoryLocationResponse {
        try await api.patch(
            "v1/factory/ops/location",
            body: FactoryLocationPatchRequest(address: address, placeId: placeId, lat: lat, lng: lng),
            idempotencyKey: FactoryIdempotency.opsLocation(lat: lat, lng: lng, placeId: placeId)
        )
    }
}
