import Foundation

enum WarehouseService {
    private static let api = APIClient.shared

    // MARK: - Auth
    static func login(phone: String, pin: String) async throws -> AuthResponse {
        try await api.post("v1/auth/warehouse/login", body: LoginRequest(phone: phone, pin: pin))
    }

    static func setup(body: [String: String]) async throws {
        try await api.postVoid("v1/warehouse/setup", body: body)
    }

    // MARK: - Dashboard
    static func dashboard() async throws -> DashboardData {
        try await api.get("v1/warehouse/ops/dashboard")
    }

    // MARK: - Orders
    static func orders(state: String? = nil) async throws -> OrderListResponse {
        var query: [String: String] = [:]
        if let state { query["state"] = state }
        return try await api.get("v1/warehouse/ops/orders", query: query)
    }

    static func order(id: String) async throws -> Order {
        try await api.get("v1/warehouse/ops/orders/\(id)")
    }

    // MARK: - Drivers
    static func drivers() async throws -> DriverListResponse {
        try await api.get("v1/warehouse/ops/drivers")
    }

    static func createDriver(name: String, phone: String) async throws -> CreateDriverResponse {
        try await api.post(
            "v1/warehouse/ops/drivers",
            body: CreateDriverRequest(name: name, phone: phone),
            idempotencyKey: WarehouseIdempotency.createDriver(phone: phone)
        )
    }

    static func assignDriver(driverId: String, vehicleId: String?) async throws -> AssignDriverVehicleResponse {
        try await api.patch(
            "v1/warehouse/ops/drivers/\(driverId)/assign-vehicle",
            body: AssignDriverVehicleRequest(vehicleId: vehicleId),
            idempotencyKey: WarehouseIdempotency.assignDriverVehicle(driverId: driverId, vehicleId: vehicleId)
        )
    }

    // MARK: - Vehicles
    static func vehicles() async throws -> VehicleListResponse {
        try await api.get("v1/warehouse/ops/vehicles")
    }

    static func createVehicle(label: String, licensePlate: String, vehicleClass: String) async throws -> Vehicle {
        try await api.post(
            "v1/warehouse/ops/vehicles",
            body: CreateVehicleRequest(label: label, licensePlate: licensePlate, vehicleClass: vehicleClass),
            idempotencyKey: WarehouseIdempotency.createVehicle(licensePlate: licensePlate)
        )
    }

    static func updateVehicleAvailability(vehicleId: String, isActive: Bool, unavailableReason: String? = nil) async throws -> VehicleMutationResponse {
        try await api.patch(
            "v1/warehouse/ops/vehicles/\(vehicleId)",
            body: UpdateVehicleRequest(isActive: isActive, unavailableReason: unavailableReason),
            idempotencyKey: WarehouseIdempotency.updateVehicle(
                vehicleId: vehicleId,
                isActive: isActive,
                unavailableReason: unavailableReason
            )
        )
    }

    // MARK: - Inventory
    static func inventory(lowStock: Bool = false) async throws -> InventoryListResponse {
        var query: [String: String] = [:]
        if lowStock { query["low_stock"] = "true" }
        return try await api.get("v1/warehouse/ops/inventory", query: query)
    }

    static func adjustInventory(productId: String, quantity: Int) async throws {
        try await api.patchVoid(
            "v1/warehouse/ops/inventory",
            body: InventoryAdjustRequest(productId: productId, quantity: quantity),
            idempotencyKey: WarehouseIdempotency.adjustInventory(productId: productId, quantity: quantity)
        )
    }

    // MARK: - Products
    static func products() async throws -> ProductListResponse {
        try await api.get("v1/warehouse/ops/products")
    }

    // MARK: - Manifests
    static func manifests() async throws -> ManifestListResponse {
        try await api.get("v1/warehouse/ops/manifests")
    }

    // MARK: - Analytics
    static func analytics(period: String = "30d") async throws -> AnalyticsData {
        try await api.get("v1/warehouse/ops/analytics", query: ["period": period])
    }

    // MARK: - CRM
    static func retailers() async throws -> RetailerListResponse {
        try await api.get("v1/warehouse/ops/crm")
    }

    // MARK: - Returns
    static func returns() async throws -> ReturnListResponse {
        try await api.get("v1/warehouse/ops/returns")
    }

    static func inboundReturns(physicalStatus: String = "ARRIVED", limit: Int = 100) async throws -> InboundReturnListResponse {
        try await api.get(
            "v1/returns/inbound",
            query: ["physical_status": physicalStatus, "limit": String(limit)]
        )
    }

    static func returnsHistory(limit: Int = 50) async throws -> InboundReturnListResponse {
        try await api.get("v1/returns/history", query: ["limit": String(limit)])
    }

    static func createInboundSession() async throws -> String {
        let resp: InboundSessionResponse = try await api.post("v1/returns/inbound/sessions", body: [String: String]())
        return resp.sessionId
    }

    static func scanInboundBarcode(barcode: String, qty: Int = 1, sessionId: String? = nil, idempotencyKey: String) async throws -> InboundScanResponse {
        let body = InboundScanBody(barcode: barcode, qty: qty, sessionId: sessionId ?? "")
        return try await api.post("v1/returns/inbound/scan", body: body, idempotencyKey: idempotencyKey)
    }

    static func scanInboundReturn(returnId: String, qty: Int = 1, sessionId: String? = nil, idempotencyKey: String) async throws -> InboundScanResponse {
        struct ScanByReturnBody: Encodable {
            let returnId: String
            let qty: Int
            let sessionId: String
            enum CodingKeys: String, CodingKey {
                case returnId = "return_id"
                case qty
                case sessionId = "session_id"
            }
        }
        let body = ScanByReturnBody(returnId: returnId, qty: qty, sessionId: sessionId ?? "")
        return try await api.post("v1/returns/inbound/scan", body: body, idempotencyKey: idempotencyKey)
    }

    static func confirmInboundReturns(returnIds: [String], disposition: String, sessionId: String, idempotencyKey: String) async throws -> InboundConfirmResponse {
        let body = InboundConfirmBody(
            lines: returnIds.map { InboundConfirmLine(returnId: $0, disposition: disposition) },
            sessionId: sessionId
        )
        return try await api.post("v1/returns/inbound/confirm", body: body, idempotencyKey: idempotencyKey)
    }

    // MARK: - Treasury
    static func treasuryOverview() async throws -> TreasuryOverview {
        try await api.get("v1/warehouse/ops/treasury", query: ["view": "overview"])
    }

    static func treasuryInvoices() async throws -> InvoiceListResponse {
        try await api.get("v1/warehouse/ops/treasury", query: ["view": "invoices"])
    }

    // MARK: - Dispatch
    static func dispatchPreview() async throws -> DispatchPreview {
        try await api.get("v1/warehouse/ops/dispatch/preview")
    }

    static func createDispatchPreview(body: [String: String]) async throws -> DispatchPreview {
        try await api.post("v1/warehouse/ops/dispatch/preview", body: body)
    }

    static func executeDispatch(body: DispatchExecuteRequest, idempotencyKey: String) async throws -> DispatchExecuteResponse {
        try await api.post("v1/warehouse/ops/dispatch/execute", body: body, idempotencyKey: idempotencyKey)
    }

    static func supplyRequests(state: String? = nil) async throws -> [WarehouseSupplyRequest] {
        var query: [String: String] = [:]
        if let state { query["state"] = state }
        let response: SupplyRequestListResponse = try await api.get("v1/warehouse/supply-requests", query: query)
        return response.requests
    }

    static func supplyRequest(id: String) async throws -> WarehouseSupplyRequest {
        try await api.get("v1/warehouse/supply-requests/\(id)")
    }

    static func demandForecast(days: Int = 7) async throws -> DemandForecastResponse {
        try await api.get("v1/warehouse/demand/forecast", query: ["days": "\(days)"])
    }

    static func dispatchLocks() async throws -> [WarehouseDispatchLock] {
        try await api.get("v1/warehouse/dispatch-locks")
    }

    static func createSupplyRequest(factoryId: String, priority: String, notes: String) async throws -> CreateWarehouseSupplyRequestResponse {
        try await api.post(
            "v1/warehouse/supply-requests",
            body: CreateWarehouseSupplyRequestRequest(
                factoryId: factoryId,
                priority: priority,
                notes: notes,
                items: [],
                useDemandForecast: true
            ),
            idempotencyKey: WarehouseIdempotency.createSupplyRequest(factoryId: factoryId, priority: priority, notes: notes)
        )
    }

    static func cancelSupplyRequest(id: String) async throws -> WarehouseSupplyRequestTransitionResponse {
        try await api.patch(
            "v1/warehouse/supply-requests/\(id)",
            body: WarehouseSupplyRequestTransitionRequest(action: "CANCEL", transferOrderId: nil),
            idempotencyKey: WarehouseIdempotency.supplyRequestTransition(requestId: id, action: "CANCEL")
        )
    }

    static func acquireDispatchLock(lockType: String = "MANUAL_DISPATCH") async throws -> CreateWarehouseDispatchLockResponse {
        try await api.post(
            "v1/warehouse/dispatch-lock",
            body: CreateWarehouseDispatchLockRequest(lockType: lockType),
            idempotencyKey: WarehouseIdempotency.dispatchLockAcquire()
        )
    }

    static func releaseDispatchLock(lockId: String) async throws -> ReleaseWarehouseDispatchLockResponse {
        try await api.delete(
            "v1/warehouse/dispatch-lock",
            query: ["lock_id": lockId],
            idempotencyKey: WarehouseIdempotency.dispatchLockRelease(lockId: lockId)
        )
    }

    // MARK: - Staff
    static func staff() async throws -> StaffListResponse {
        try await api.get("v1/warehouse/ops/staff")
    }

    static func createStaff(name: String, phone: String, role: String) async throws -> CreateStaffResponse {
        try await api.post(
            "v1/warehouse/ops/staff",
            body: CreateStaffRequest(name: name, phone: phone, role: role),
            idempotencyKey: WarehouseIdempotency.createStaff(phone: phone)
        )
    }

    // MARK: - Payment Config
    static func paymentConfig() async throws -> PaymentConfigResponse {
        try await api.get("v1/warehouse/ops/payment-config")
    }

    // MARK: - P1-03 ops depth
    static func replenishmentInsights() async throws -> ReplenishmentInsightsResponse {
        try await api.get("v1/warehouse/replenishment/insights")
    }

    static func replenishmentInsightAction(insightId: String, action: String) async throws -> ReplenishmentInsightActionResponse {
        try await api.postEmpty(
            "v1/warehouse/replenishment/insights/\(insightId)/\(action)",
            idempotencyKey: WarehouseIdempotency.replenishmentInsightAction(insightId: insightId, action: action)
        )
    }

    static func emergencyTransfer(body: EmergencyTransferRequest) async throws -> TransferMutationResponse {
        try await api.post("v1/warehouse/transfers/emergency", body: body)
    }

    static func forceReceive(body: ForceReceiveRequest) async throws -> TransferMutationResponse {
        try await api.post("v1/warehouse/transfers/force-receive", body: body)
    }

    static func receiveTransfer(transferId: String) async throws -> TransferMutationResponse {
        try await api.postEmpty("v1/warehouse/transfers/\(transferId)/receive")
    }

    static func opsFinancials(period: String? = nil) async throws -> OpsFinancialsResponse {
        var query: [String: String] = [:]
        if let period { query["period"] = period }
        return try await api.get("v1/warehouse/ops/financials", query: query)
    }

    static func delayOrder(orderId: String, body: WarehouseOrderMutationRequest) async throws -> WarehouseOrderMutationResponse {
        try await api.post("v1/warehouse/ops/orders/\(orderId)/delay", body: body)
    }

    static func rejectOrder(orderId: String, body: WarehouseOrderMutationRequest) async throws -> WarehouseOrderMutationResponse {
        try await api.post("v1/warehouse/ops/orders/\(orderId)/reject", body: body)
    }

    static func overflowOrder(orderId: String, body: WarehouseOrderMutationRequest) async throws -> WarehouseOrderMutationResponse {
        try await api.post("v1/warehouse/ops/orders/\(orderId)/overflow", body: body)
    }

    static func fleetLiveMap(warehouseId: String? = nil) async throws -> WarehouseFleetLiveMapResponse {
        var query: [String: String] = [:]
        if let warehouseId, !warehouseId.isEmpty {
            query["warehouse_id"] = warehouseId
        }
        return try await api.get("v1/warehouse/ops/fleet/live-map", query: query)
    }

    static func dispatchSettings() async throws -> DispatchSettingsResponse {
        try await api.get("v1/warehouse/ops/dispatch/settings")
    }

    static func patchDispatchSettings(enabled: Bool) async throws {
        try await api.patchVoid(
            "v1/warehouse/ops/dispatch/settings",
            body: DispatchSettingsPatchRequest(autoDispatchEnabled: enabled),
            idempotencyKey: WarehouseIdempotency.dispatchSettings(autoDispatchEnabled: enabled)
        )
    }
}
