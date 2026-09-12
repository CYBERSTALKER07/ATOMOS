//
//  FleetServiceLive.swift
//  driverappios
//
//  Real implementation of FleetServiceProtocol backed by APIClient.
//  Replaces FleetServiceStub for production use.
//

import Foundation
import CoreLocation
import SwiftData

final class FleetServiceLive: FleetServiceProtocol {

    static let shared = FleetServiceLive()

    private let api = APIClient.shared

    /// Lazily-initialized SwiftData container for offline delivery queue.
    private lazy var modelContainer: ModelContainer? = {
        try? ModelContainer(for: OfflineDelivery.self)
    }()

    // MARK: - Fetch Active Missions (bridged from Orders)

    func fetchActiveMissions(routeId: String) async throws -> [Mission] {
        let orders = try await api.getAssignedOrders()
        return orders.map { order in
            Mission(
                order_id: order.id,
                state: order.state.rawValue,
                target_lat: order.latitude,
                target_lng: order.longitude,
                amount: order.totalAmount,
                gateway: "CASH",
                estimated_arrival_at: order.estimatedArrivalAt,
                route_id: order.routeId,
                sequence_index: order.sequenceIndex
            )
        }
    }

    // MARK: - Deliver Order (with offline queue fallback)

    func deliverOrder(orderId: String, scannedToken: String) async throws {
        let location = try await requireCurrentLocation()
        do {
            let response = try await api.submitDelivery(
                orderId: orderId,
                qrToken: scannedToken,
                latitude: location.latitude,
                longitude: location.longitude
            )
            guard response.success else {
                throw FleetServiceError.deliveryRejected(response.message)
            }
        } catch let error as FleetServiceError {
            // Business rejection / queued — do NOT re-enqueue
            throw error
        } catch {
            // Transport / retryable only — never enqueue geofence or other business 4xx (P0-4)
            guard DriverOfflineActionCatalog.isNetworkEnqueueable(error) else {
                throw error
            }
            await enqueueOfflineDelivery(
                orderId: orderId,
                scannedToken: scannedToken,
                latitude: location.latitude,
                longitude: location.longitude
            )
            print("[FleetServiceLive] Delivery queued offline for order \(orderId)")
            throw FleetServiceError.queuedForSync(orderId)
        }
    }

    // MARK: - New Delivery Flow

    func validateQR(orderId: String, scannedToken: String) async throws -> ValidateQRResponse {
        try await api.validateQR(orderId: orderId, scannedToken: scannedToken)
    }

    func confirmOffload(orderId: String) async throws -> ConfirmOffloadResponse {
        try await api.confirmOffload(orderId: orderId)
    }

    func scanDeliveryQR(orderId: String, qrToken: String) async throws -> DeliveryScanQRResponse {
        try await api.scanDeliveryQR(orderId: orderId, qrToken: qrToken)
    }

    func completeOrder(orderId: String) async throws {
        try await api.completeOrder(orderId: orderId)
    }

    /// Collect cash from retailer with geofence validation.
    /// Sends driver GPS coords; backend rejects if > 500m from retailer.
    func collectCash(orderId: String, amountReceivedMinor: Int64? = nil) async throws -> CollectCashResponse {
        let location = try await requireCurrentLocation()
        return try await api.collectCash(
            orderId: orderId,
            latitude: location.latitude,
            longitude: location.longitude,
            amountReceivedMinor: amountReceivedMinor
        )
    }

    func retryFiscal(orderId: String) async throws -> CollectCashResponse {
        try await api.retryFiscal(orderId: orderId)
    }

    // MARK: - Fetch Order Line Items

    func fetchOrderLineItems(orderId: String) async throws -> [LineItem] {
        let order = try await api.getOrder(id: orderId)
        return order.items.map { item in
            LineItem(
                line_item_id: item.productId,
                sku_id: item.productId,
                quantity: item.quantity,
                unit_price: item.unitPrice,
                status: .DELIVERED
            )
        }
    }

    // MARK: - Amend Order (partial rejection → reconciliation)

    func amendOrder(
        orderId: String,
        driverId: String,
        items: [(lineItemId: String, rejectedQty: Int, status: LineItemStatus, reason: String, customReason: String?)]
    ) async throws {
        // Build AmendItemPayload from partial quantities
        let order = try await api.getOrder(id: orderId)

        let amendments: [AmendItemPayload] = items.compactMap { (lineItemId, rejectedQty, _, reason, customReason) in
            guard let original = order.items.first(where: { $0.productId == lineItemId }) else {
                return nil
            }
            let accepted = original.quantity - rejectedQty
            return AmendItemPayload(
                productId: lineItemId,
                acceptedQty: accepted,
                rejectedQty: rejectedQty,
                reason: rejectedQty > 0 ? (reason.isEmpty ? RejectionReason.DAMAGED.rawValue : reason) : "",
                customReason: customReason
            )
        }

        let request = AmendOrderRequest(
            orderId: orderId,
            items: amendments,
            driverNotes: ""
        )

        let response = try await api.amendOrder(request: request)
        guard response.success else {
            throw FleetServiceError.amendmentRejected(response.message)
        }
    }

    func verifyHandshake(orderId: String, token: String, latitude: Double, longitude: Double) async throws -> VerifyHandshakeResponse {
        try await api.verifyHandshake(
            orderId: orderId,
            token: token,
            latitude: latitude,
            longitude: longitude
        )
    }

    func updateOrderDuringDelivery(orderId: String, latitude: Double, longitude: Double) async throws -> UpdateOrderDuringDeliveryResponse {
        // G1.C: do not hit network — backend has no durable writer. Use amend / missing-items.
        struct MidDeliveryDisabled: LocalizedError {
            var errorDescription: String? {
                "Use delivery correction (amend / missing items) — mid-delivery update is not implemented"
            }
        }
        throw MidDeliveryDisabled()
    }

    func markCreditDelivery(orderId: String, photoProofUrl: String? = nil, signatureUrl: String? = nil) async throws -> [String: String] {
        try await api.markCreditDelivery(orderId: orderId, photoProofUrl: photoProofUrl, signatureUrl: signatureUrl)
    }

    func splitPayment(orderId: String, cashMinor: Int64, cardMinor: Int64, currency: String? = nil) async throws -> SplitPaymentResponse {
        try await api.splitPayment(
            orderId: orderId,
            cashMinor: cashMinor,
            cardMinor: cardMinor,
            currency: currency
        )
    }

    // MARK: - Rescue Operations

    func requestRescue(reason: String, note: String) async throws {
        _ = try await api.requestRescue(reason: reason, note: note)
    }

    func respondRescue(rescueId: String, accept: Bool) async throws {
        _ = try await api.respondRescue(rescueId: rescueId, accept: accept)
    }

    func reassignHandshake(orderId: String) async throws {
        _ = try await api.reassignHandshake(orderId: orderId)
    }

    // MARK: - Offline Delivery Queue

    @MainActor
    private func enqueueOfflineDelivery(
        orderId: String,
        scannedToken: String,
        latitude: Double,
        longitude: Double
    ) {
        if let container = modelContainer {
            DriverOfflineQueue.shared.attach(container: container)
        }
        let body: [String: Any] = [
            "order_id": orderId,
            "scanned_token": scannedToken,
            "signature": scannedToken,
            "latitude": latitude,
            "longitude": longitude,
            "client_timestamp": DriverOfflineActionCatalog.nowIso(),
        ]
        guard let data = try? JSONSerialization.data(withJSONObject: body),
              let json = String(data: data, encoding: .utf8) else { return }
        DriverOfflineQueue.shared.enqueue(
            endpoint: DriverOfflineActionCatalog.deliver,
            bodyJSON: json,
            idempotencyKey: DriverIdempotency.deliver(orderId: orderId),
            orderId: orderId
        )
        // Keep legacy store in sync for OfflineVerifier until fully migrated.
        if let context = modelContainer?.mainContext {
            OfflineDeliveryStore(modelContext: context)
                .enqueue(orderId: orderId, signature: scannedToken, status: "DELIVERED")
        }
    }

    /// Flush durable offline action queue (protocol-ordered).
    func flushOfflineQueue() async {
        if let container = modelContainer {
            await MainActor.run { DriverOfflineQueue.shared.attach(container: container) }
        }
        let result = await DriverOfflineQueue.shared.flush(api: api)
        print("[FleetServiceLive] Flush sent=\(result.sent) remaining=\(result.remaining)")
    }

    // MARK: - GPS Helper

    /// Fail-closed: never fabricate (0,0) — that skips backend geofence and poisons offline replay.
    private func requireCurrentLocation() async throws -> CLLocationCoordinate2D {
        let coord = await MainActor.run { FleetViewModel.lastKnownLocation }
        guard let coord,
              !(coord.latitude == 0 && coord.longitude == 0) else {
            throw FleetServiceError.locationUnavailable
        }
        return coord
    }
}

// MARK: - Errors

enum FleetServiceError: LocalizedError {
    case deliveryRejected(String)
    case amendmentRejected(String)
    case locationUnavailable
    /// Delivery was persisted for sync — not a completed delivery.
    case queuedForSync(String)

    var errorDescription: String? {
        switch self {
        case .deliveryRejected(let msg): return "Delivery rejected: \(msg)"
        case .amendmentRejected(let msg): return "Amendment rejected: \(msg)"
        case .locationUnavailable: return "GPS location unavailable — move outdoors and try again"
        case .queuedForSync(let orderId): return "Delivery queued for sync (order \(orderId))"
        }
    }
}
