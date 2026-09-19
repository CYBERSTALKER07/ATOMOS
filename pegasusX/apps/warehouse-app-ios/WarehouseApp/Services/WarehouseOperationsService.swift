import Foundation

/// Portal-parity warehouse ops API for native screens.
enum WarehouseOperationsService {
    private static let api = APIClient.shared

    static func emergencyTransfer(totalVolumeVu: Double, notes: String? = nil) async throws -> TransferMutationResponse {
        try await api.post(
            "v1/warehouse/transfers/emergency",
            body: EmergencyTransferRequest(totalVolumeVu: totalVolumeVu, notes: notes),
            idempotencyKey: WarehouseIdempotency.emergencyTransfer(volumeVu: totalVolumeVu, notes: notes)
        )
    }

    static func forceReceive(
        factoryId: String? = nil,
        totalVolumeVu: Double,
        notes: String? = nil
    ) async throws -> TransferMutationResponse {
        try await api.post(
            "v1/warehouse/transfers/force-receive",
            body: ForceReceiveRequest(factoryId: factoryId, totalVolumeVu: totalVolumeVu, notes: notes),
            idempotencyKey: WarehouseIdempotency.forceReceive(volumeVu: totalVolumeVu, notes: notes, factoryId: factoryId)
        )
    }

    static func receiveTransfer(transferId: String) async throws -> TransferMutationResponse {
        try await api.postEmpty(
            "v1/warehouse/transfers/\(transferId)/receive",
            idempotencyKey: WarehouseIdempotency.receiveTransfer(transferId: transferId)
        )
    }

    static func replenishmentInsights() async throws -> [ReplenishmentInsight] {
        let response: ReplenishmentInsightsResponse = try await api.get("v1/warehouse/replenishment/insights")
        return response.rows
    }

    static func replenishmentInsightAction(
        insightId: String,
        action: String
    ) async throws -> ReplenishmentInsightActionResponse {
        try await api.postEmpty(
            "v1/warehouse/replenishment/insights/\(insightId)/\(action)",
            idempotencyKey: WarehouseIdempotency.replenishmentInsightAction(insightId: insightId, action: action)
        )
    }

    static func opsFinancials(period: String? = nil) async throws -> OpsFinancialsResponse {
        var query: [String: String] = [:]
        if let period, !period.isEmpty { query["period"] = period }
        return try await api.get("v1/warehouse/ops/financials", query: query)
    }

    static func delayOrder(orderId: String, reason: String? = nil) async throws -> WarehouseOrderMutationResponse {
        try await api.post(
            "v1/warehouse/ops/orders/\(orderId)/delay",
            body: WarehouseOrderMutationRequest(reason: reason),
            idempotencyKey: WarehouseIdempotency.orderDelay(orderId: orderId)
        )
    }

    static func rejectOrder(orderId: String, reason: String) async throws -> WarehouseOrderMutationResponse {
        try await api.post(
            "v1/warehouse/ops/orders/\(orderId)/reject",
            body: WarehouseOrderMutationRequest(reason: reason),
            idempotencyKey: WarehouseIdempotency.orderReject(orderId: orderId, reason: reason)
        )
    }

    static func proposeOrderDelivery(
        orderId: String,
        proposedDeliveryDate: String,
        reason: String
    ) async throws -> WarehouseOrderMutationResponse {
        try await api.post(
            "v1/warehouse/ops/orders/\(orderId)/propose-delivery",
            body: WarehouseProposeDeliveryRequest(proposedDeliveryDate: proposedDeliveryDate, reason: reason),
            idempotencyKey: WarehouseIdempotency.orderProposeDelivery(
                orderId: orderId,
                proposedDate: proposedDeliveryDate,
                reason: reason
            )
        )
    }

    static func proposePreorderDelivery(
        orderId: String,
        proposedDeliveryDate: String,
        reason: String
    ) async throws -> WarehouseOrderMutationResponse {
        try await api.post(
            "v1/warehouse/ops/preorders/\(orderId)/propose-delivery",
            body: WarehouseProposeDeliveryRequest(proposedDeliveryDate: proposedDeliveryDate, reason: reason),
            idempotencyKey: WarehouseIdempotency.orderProposeDelivery(
                orderId: orderId,
                proposedDate: proposedDeliveryDate,
                reason: reason
            )
        )
    }

    static func rejectPreorder(orderId: String, reason: String) async throws -> WarehouseOrderMutationResponse {
        try await api.post(
            "v1/warehouse/ops/preorders/\(orderId)/reject",
            body: WarehouseOrderMutationRequest(reason: reason),
            idempotencyKey: WarehouseIdempotency.orderReject(orderId: orderId, reason: reason)
        )
    }

    static func overflowOrder(orderId: String, reason: String? = nil) async throws -> WarehouseOrderMutationResponse {
        try await api.post(
            "v1/warehouse/ops/orders/\(orderId)/overflow",
            body: WarehouseOrderMutationRequest(reason: reason),
            idempotencyKey: WarehouseIdempotency.orderOverflow(orderId: orderId)
        )
    }

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

    static func refreshToken(_ refreshToken: String) async throws -> AuthResponse {
        try await api.post("v1/auth/warehouse/refresh", body: RefreshTokenRequest(refreshToken: refreshToken))
    }

    static func issuePaymentBypass(orderId: String) async throws -> PaymentBypassResponse {
        try await api.post(
            "v1/warehouse/ops/orders/payment-bypass",
            body: PaymentBypassRequest(orderId: orderId)
        )
    }

    static func getEarlyCompleteRequest(driverId: String) async throws -> [String: String] {
        try await api.get("v1/warehouse/ops/orders/early-complete/\(driverId)")
    }

    static func approveEarlyComplete(driverId: String, action: String, newWindowStart: String? = nil, newWindowEnd: String? = nil) async throws -> [String: String] {
        var body: [String: String] = [
            "driver_id": driverId,
            "action": action
        ]
        if let s = newWindowStart { body["newWindowStart"] = s }
        if let e = newWindowEnd { body["newWindowEnd"] = e }
        return try await api.post("v1/warehouse/ops/orders/early-complete/approve", body: body)
    }
}

// Models


struct PaymentBypassRequest: Encodable {
    let orderId: String
    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
    }
}

struct PaymentBypassResponse: Decodable {
    let status: String?
    let bypassToken: String?
    let orderId: String?
    enum CodingKeys: String, CodingKey {
        case status
        case bypassToken = "bypass_token"
        case orderId = "order_id"
    }
}

struct EarlyCompleteRequest: Encodable {
    let truckId: String
    enum CodingKeys: String, CodingKey {
        case truckId = "truck_id"
    }
}

struct EarlyCompleteResponse: Decodable {
    let status: String?
}
