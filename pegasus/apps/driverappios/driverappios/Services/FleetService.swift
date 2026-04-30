//
//  FleetService.swift
//  driverappios
//

import Foundation

// MARK: - Protocol

protocol FleetServiceProtocol {
    /// GET /v1/fleet/active?route_id={routeId} → [Mission]
    func fetchActiveMissions(routeId: String) async throws -> [Mission]

    /// POST /v1/order/deliver { "order_id", "scanned_token" } — LEGACY
    func deliverOrder(orderId: String, scannedToken: String) async throws

    /// POST /v1/order/validate-qr — Validates QR token, returns order info
    func validateQR(orderId: String, scannedToken: String) async throws -> ValidateQRResponse

    /// POST /v1/order/confirm-offload — ARRIVED → AWAITING_PAYMENT
    func confirmOffload(orderId: String) async throws -> ConfirmOffloadResponse

    /// POST /v1/order/complete — AWAITING_PAYMENT → COMPLETED
    func completeOrder(orderId: String) async throws

    /// POST /v1/order/collect-cash — PENDING_CASH_COLLECTION → COMPLETED with geofence
    func collectCash(orderId: String) async throws -> CollectCashResponse

    /// GET /v1/order-items/{orderId} → [LineItem]
    func fetchOrderLineItems(orderId: String) async throws -> [LineItem]

    /// POST /v1/order/amend — partial-quantity reconciliation. rejectedQty 0 = fully accepted, item.quantity = fully rejected.
    func amendOrder(orderId: String, driverId: String, items: [(lineItemId: String, rejectedQty: Int, status: LineItemStatus)]) async throws
}

// MARK: - Stub Implementation

final class FleetServiceStub: FleetServiceProtocol {

    static let shared = FleetServiceStub()

    func fetchActiveMissions(routeId: String) async throws -> [Mission] {
        try await Task.sleep(nanoseconds: 500_000_000)
        return Mission.mockMissions
    }

    func deliverOrder(orderId: String, scannedToken: String) async throws {
        try await Task.sleep(nanoseconds: 500_000_000)
    }

    func validateQR(orderId: String, scannedToken: String) async throws -> ValidateQRResponse {
        try await Task.sleep(nanoseconds: 500_000_000)
        return ValidateQRResponse(orderId: orderId, retailerName: "Mock Store", totalAmount: 100_000, state: "ARRIVED", items: [])
    }

    func confirmOffload(orderId: String) async throws -> ConfirmOffloadResponse {
        try await Task.sleep(nanoseconds: 500_000_000)
        return ConfirmOffloadResponse(orderId: orderId, state: "AWAITING_PAYMENT", paymentMethod: "CASH", amount: 100_000, invoiceId: nil, retailerId: "RET-001", message: "Collect 100000")
    }

    func completeOrder(orderId: String) async throws {
        try await Task.sleep(nanoseconds: 500_000_000)
    }

    func collectCash(orderId: String) async throws -> CollectCashResponse {
        try await Task.sleep(nanoseconds: 500_000_000)
        return CollectCashResponse(orderId: orderId, state: "COMPLETED", amount: 100_000, distanceM: 25.0, message: "Cash collected")
    }

    func fetchOrderLineItems(orderId: String) async throws -> [LineItem] {
        try await Task.sleep(nanoseconds: 500_000_000)
        return LineItem.mockLineItems
    }

    func amendOrder(orderId: String, driverId: String, items: [(lineItemId: String, rejectedQty: Int, status: LineItemStatus)]) async throws {
        try await Task.sleep(nanoseconds: 500_000_000)
        // Stub: always succeeds
    }
}
