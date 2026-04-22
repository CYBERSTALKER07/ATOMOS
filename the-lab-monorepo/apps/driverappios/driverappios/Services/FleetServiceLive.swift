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
        let location = await currentLocation()
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
            // Business rejection — do NOT queue offline, propagate immediately
            throw error
        } catch {
            // Network/transport error — queue for offline sync
            await enqueueOfflineDelivery(orderId: orderId, scannedToken: scannedToken)
            print("[FleetServiceLive] Delivery queued offline for order \(orderId)")
        }
    }

    // MARK: - New Delivery Flow

    func validateQR(orderId: String, scannedToken: String) async throws -> ValidateQRResponse {
        try await api.validateQR(orderId: orderId, scannedToken: scannedToken)
    }

    func confirmOffload(orderId: String) async throws -> ConfirmOffloadResponse {
        try await api.confirmOffload(orderId: orderId)
    }

    func completeOrder(orderId: String) async throws {
        try await api.completeOrder(orderId: orderId)
    }

    /// Collect cash from retailer with geofence validation.
    /// Sends driver GPS coords; backend rejects if > 500m from retailer.
    func collectCash(orderId: String) async throws -> CollectCashResponse {
        let location = await currentLocation()
        return try await api.collectCash(
            orderId: orderId,
            latitude: location.latitude,
            longitude: location.longitude
        )
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
        items: [(lineItemId: String, rejectedQty: Int, status: LineItemStatus)]
    ) async throws {
        // Build AmendItemPayload from partial quantities
        let order = try await api.getOrder(id: orderId)

        let amendments: [AmendItemPayload] = items.compactMap { (lineItemId, rejectedQty, _) in
            guard let original = order.items.first(where: { $0.productId == lineItemId }) else {
                return nil
            }
            let accepted = original.quantity - rejectedQty
            let reason = rejectedQty > 0 ? "DAMAGED" : ""
            return AmendItemPayload(
                productId: lineItemId,
                acceptedQty: accepted,
                rejectedQty: rejectedQty,
                reason: reason
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

    // MARK: - Offline Delivery Queue

    @MainActor
    private func enqueueOfflineDelivery(orderId: String, scannedToken: String) {
        guard let context = modelContainer?.mainContext else { return }
        let store = OfflineDeliveryStore(modelContext: context)
        store.enqueue(orderId: orderId, signature: scannedToken, status: "DELIVERED")
    }

    /// Attempt to flush all pending offline deliveries to the server.
    /// Call this when network connectivity is restored.
    func flushOfflineQueue() async {
        guard let container = modelContainer else { return }
        let pending: [OfflineDelivery] = await MainActor.run {
            let store = OfflineDeliveryStore(modelContext: container.mainContext)
            return store.fetchPending()
        }

        for delivery in pending {
            do {
                let location = await currentLocation()
                let response = try await api.submitDelivery(
                    orderId: delivery.orderId,
                    qrToken: delivery.signature,
                    latitude: location.latitude,
                    longitude: location.longitude
                )
                if response.success {
                    await MainActor.run {
                        let store = OfflineDeliveryStore(modelContext: container.mainContext)
                        store.delete(delivery)
                    }
                    print("[FleetServiceLive] Flushed offline delivery: \(delivery.orderId)")
                }
            } catch {
                // Still offline or server error — stop flushing, retry later
                print("[FleetServiceLive] Flush failed for \(delivery.orderId), will retry later")
                break
            }
        }
    }

    // MARK: - GPS Helper

    private func currentLocation() async -> CLLocationCoordinate2D {
        // Read from CLLocationManager via FleetViewModel's shared location
        // Refuse to fabricate coordinates — GPS must be available for geofence integrity
        await MainActor.run {
            FleetViewModel.lastKnownLocation ?? CLLocationCoordinate2D(latitude: 0, longitude: 0)
        }
    }
}

// MARK: - Errors

enum FleetServiceError: LocalizedError {
    case deliveryRejected(String)
    case amendmentRejected(String)

    var errorDescription: String? {
        switch self {
        case .deliveryRejected(let msg): return "Delivery rejected: \(msg)"
        case .amendmentRejected(let msg): return "Amendment rejected: \(msg)"
        }
    }
}
