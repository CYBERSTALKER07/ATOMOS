import Foundation

enum WarehouseService {
    private static let api = APIClient.shared

    // MARK: - Auth
    static func login(phone: String, pin: String) async throws -> AuthResponse {
        try await api.post("v1/auth/warehouse/login", body: LoginRequest(phone: phone, pin: pin))
    }

    static func login(idToken: String) async throws -> AuthResponse {
        try await api.post("v1/auth/warehouse/login", body: LoginRequest(idToken: idToken))
    }

    static func setup(
        name: String,
        address: String,
        placeId: String?,
        lat: Double,
        lng: Double
    ) async throws -> WarehouseSetupResponse {
        try await api.post(
            "v1/warehouse/setup",
            body: WarehouseSetupRequest(
                name: name,
                address: address,
                placeId: placeId,
                lat: lat,
                lng: lng
            )
        )
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

    static func vehicle(id: String) async throws -> VehicleDetailResponse {
        try await api.get("v1/warehouse/ops/vehicles/\(id)")
    }

    static func createVehicle(label: String, licensePlate: String, vehicleClass: String) async throws -> Vehicle {
        try await api.post(
            "v1/warehouse/ops/vehicles",
            body: CreateVehicleRequest(label: label, licensePlate: licensePlate, vehicleClass: vehicleClass),
            idempotencyKey: WarehouseIdempotency.createVehicle(licensePlate: licensePlate)
        )
    }

    static func updateVehicleAvailability(
        vehicleId: String,
        isActive: Bool,
        unavailableReason: String? = nil,
        unavailableNote: String? = nil
    ) async throws -> VehicleMutationResponse {
        try await api.patch(
            "v1/warehouse/ops/vehicles/\(vehicleId)",
            body: UpdateVehicleRequest(
                isActive: isActive,
                unavailableReason: unavailableReason,
                unavailableNote: unavailableNote
            ),
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

    static func patchInventoryPolicy(productId: String, policy: String) async throws {
        try await api.patchVoid(
            "v1/warehouse/ops/inventory/\(productId)/policy",
            body: InventoryPolicyPatchRequest(outOfStockPolicy: policy),
            idempotencyKey: WarehouseIdempotency.inventoryPolicy(productId: productId, policy: policy)
        )
    }

    static func opsSettings() async throws -> WarehouseOpsSettingsResponse {
        try await api.get("v1/warehouse/ops/settings")
    }

    static func preorders() async throws -> WarehousePreordersResponse {
        try await api.get("v1/warehouse/ops/preorders")
    }

    static func stockCommitments() async throws -> StockCommitmentsResponse {
        try await api.get("v1/warehouse/ops/stock-commitments")
    }

    static func patchOpsSettings(_ body: WarehouseOpsSettingsPatchRequest) async throws {
        try await api.patchVoid(
            "v1/warehouse/ops/settings",
            body: body,
            idempotencyKey: WarehouseIdempotency.opsSettings()
        )
    }

    static func patchOpsSettings(policy: String, operatingSchedule: [String: AnyCodable]) async throws {
        try await patchOpsSettings(WarehouseOpsSettingsPatchRequest(
            defaultOutOfStockPolicy: policy,
            showStockCountsToRetailers: nil,
            operatingSchedule: operatingSchedule,
            preorderMinLeadDays: nil,
            preorderMaxLeadDays: nil,
            orderLineMinQuantity: nil,
            orderLineMaxQuantity: nil,
            clearOrderLineMinQuantity: nil,
            clearOrderLineMaxQuantity: nil,
            expressEnabled: nil,
            expressStockFloor: nil,
            deliveryFeeRules: nil,
            clearDeliveryFeeRules: nil
        ))
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

    static func inboundReturns(physicalStatus: String = "OPEN", limit: Int = 100) async throws -> InboundReturnListResponse {
        try await api.get(
            "v1/returns/inbound",
            query: ["physical_status": physicalStatus, "limit": String(limit)]
        )
    }

    static func returnsHistory(limit: Int = 50) async throws -> InboundReturnListResponse {
        try await api.get("v1/returns/history", query: ["limit": String(limit)])
    }

    static func reverseLogistics(status: String = "OPEN", warehouseId: String? = nil) async throws -> ReverseLogisticsListResponse {
        var query = ["status": status]
        if let warehouseId, !warehouseId.isEmpty {
            query["warehouse_id"] = warehouseId
        }
        return try await api.get("v1/warehouse/reverse-logistics", query: query)
    }

    static func receiveReverseLogistics(
        taskId: String,
        warehouseId: String,
        receivedQty: [String: Int]
    ) async throws -> [String: String] {
        let body = ReverseLogisticsReceiveRequest(warehouseId: warehouseId, receivedQty: receivedQty)
        return try await api.post(
            "v1/warehouse/reverse-logistics/\(taskId)/receive",
            body: body
        )
    }

    static func opsExceptions() async throws -> WarehouseOpsExceptionsResponse {
        try await api.get("v1/warehouse/ops/exceptions")
    }

    static func supplierClaims(status: String? = "OPEN", limit: Int = 50) async throws -> WarehouseClaimsResponse {
        var query: [String: String] = ["limit": String(limit)]
        if let status, !status.isEmpty {
            query["status"] = status
        }
        return try await api.get("v1/supplier/claims", query: query)
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

    /// POST /v1/warehouse/ops/dispatch/rescue/preview — { broken_driver_id }
    static func previewRescue(brokenDriverId: String) async throws -> RescuePreviewResponse {
        try await api.post(
            "v1/warehouse/ops/dispatch/rescue/preview",
            body: RescuePreviewRequest(brokenDriverId: brokenDriverId)
        )
    }

    /// POST /v1/warehouse/ops/dispatch/rescue/propose — { rescue_id, broken_driver_id, rescue_driver_id, force_capacity? }
    static func proposeRescue(
        rescueId: String,
        brokenDriverId: String,
        rescueDriverId: String,
        forceCapacity: Bool = false
    ) async throws -> [String: String] {
        let body = RescueProposeRequest(
            rescueId: rescueId,
            brokenDriverId: brokenDriverId,
            rescueDriverId: rescueDriverId,
            forceCapacity: forceCapacity
        )
        return try await api.post(
            "v1/warehouse/ops/dispatch/rescue/propose",
            body: body,
            idempotencyKey: WarehouseIdempotency.rescuePropose(
                rescueId: rescueId,
                brokenDriverId: brokenDriverId,
                rescueDriverId: rescueDriverId
            )
        )
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

    static func createSupplyRequest(form: SupplyRequestFormData) async throws -> CreateWarehouseSupplyRequestResponse {
        let mode = form.useDemandForecast ? "FORECAST" : "MANUAL"
        return try await api.post(
            "v1/warehouse/supply-requests",
            body: CreateWarehouseSupplyRequestRequest(
                factoryId: form.factoryId,
                priority: form.priority,
                notes: form.notes,
                items: form.items,
                useDemandForecast: form.useDemandForecast,
                requestedDeliveryDate: form.requestedDeliveryDate
            ),
            idempotencyKey: WarehouseIdempotency.createSupplyRequest(factoryId: form.factoryId, mode: mode, notes: form.notes)
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

    static func opsBoard(date: String) async throws -> WarehouseOpsBoardResponse {
        try await api.get("v1/warehouse/ops/board", query: ["date": date])
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

    static func broadcastTemplates() async throws -> BroadcastTemplatesResponse {
        try await api.get("v1/warehouse/ops/broadcast/templates")
    }

    static func createBroadcastTemplate(
        _ request: WarehouseBroadcastTemplateCreateRequest,
        idempotencyKey: String
    ) async throws -> BroadcastTemplate {
        try await api.post(
            "v1/warehouse/ops/broadcast/templates",
            body: request,
            idempotencyKey: idempotencyKey
        )
    }

    static func deleteBroadcastTemplate(templateId: String, idempotencyKey: String) async throws -> BroadcastTemplateDeleteResponse {
        try await api.delete(
            "v1/warehouse/ops/broadcast/templates/\(templateId)",
            idempotencyKey: idempotencyKey
        )
    }

    static func postBroadcast(_ request: WarehouseBroadcastRequest, idempotencyKey: String) async throws -> WarehouseBroadcastResponse {
        try await api.post(
            "v1/warehouse/ops/broadcast",
            body: request,
            idempotencyKey: idempotencyKey
        )
    }

    static func previewRetailerPriceOverride(_ request: RetailerOverridePreviewRequest) async throws -> RetailerOverridePreview {
        try await api.post(
            "v1/warehouse/ops/pricing/retailer-overrides/preview",
            body: request
        )
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

    static func pulse() async throws -> WarehousePulseResponse {
        try await api.get("v1/warehouse/ops/pulse")
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

    static func warehouseLocation() async throws -> WarehouseLocationResponse {
        try await api.get("v1/warehouse/ops/location")
    }

    static func patchWarehouseLocation(address: String, placeId: String?, lat: Double, lng: Double) async throws -> WarehouseLocationResponse {
        try await api.patch(
            "v1/warehouse/ops/location",
            body: WarehouseLocationPatchRequest(address: address, placeId: placeId, lat: lat, lng: lng),
            idempotencyKey: WarehouseIdempotency.opsLocation(lat: lat, lng: lng, placeId: placeId)
        )
    }

    // MARK: - Reassignment
    static func recommendReassign(orderId: String) async throws -> RecommendReassignResponse {
        try await api.post(
            "v1/warehouse/recommend-reassign",
            body: RecommendReassignRequest(orderId: orderId)
        )
    }

    static func reassignOrder(_ request: ReassignOrderRequest, idempotencyKey: String) async throws {
        try await api.postVoid(
            "v1/warehouse/reassign-order",
            body: request
        )
    }
}
