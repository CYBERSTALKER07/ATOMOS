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

    /// POST /v1/order/complete — capture → FISCALIZING (ADR-009)
    func completeOrder(orderId: String) async throws

    /// POST /v1/order/collect-cash — cash capture → FISCALIZING (ADR-009)
    func collectCash(orderId: String) async throws -> CollectCashResponse

    /// POST /v1/order/{id}/fiscal/retry
    func retryFiscal(orderId: String) async throws -> CollectCashResponse

    /// GET /v1/order-items/{orderId} → [LineItem]
    func fetchOrderLineItems(orderId: String) async throws -> [LineItem]

    /// POST /v1/order/amend — partial-quantity reconciliation. rejectedQty 0 = fully accepted, item.quantity = fully rejected.
    func amendOrder(orderId: String, driverId: String, items: [(lineItemId: String, rejectedQty: Int, status: LineItemStatus, reason: String, customReason: String?)]) async throws

    /// POST /v1/delivery/verify-handshake — geofence + token verification at scan time.
    func verifyHandshake(orderId: String, token: String, latitude: Double, longitude: Double) async throws -> VerifyHandshakeResponse

    /// POST /v1/delivery/update-order-during-delivery — in-delivery reconciliation edge.
    func updateOrderDuringDelivery(orderId: String, latitude: Double, longitude: Double) async throws -> UpdateOrderDuringDeliveryResponse

    /// POST /v1/delivery/credit-delivery — mark delivered on credit.
    func markCreditDelivery(orderId: String, photoProofUrl: String?) async throws -> [String: String]

    /// POST /v1/delivery/split-payment — record cash/card split.
    func splitPayment(orderId: String, cashMinor: Int64, cardMinor: Int64, currency: String?) async throws -> SplitPaymentResponse

    /// POST /v1/driver/ops/rescue/request — Driver requests rescue
    func requestRescue(reason: String, note: String) async throws

    /// POST /v1/driver/ops/rescue/respond — Driver responds to a rescue proposal
    func respondRescue(rescueId: String, accept: Bool) async throws

    /// POST /v1/fleet/orders/{id}/reassign-handshake — Driver handshakes reassigned order
    func reassignHandshake(orderId: String) async throws
}

