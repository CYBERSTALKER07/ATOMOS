import Foundation
import MapKit
import Observation
import SwiftUI

enum TrackingLoadIssue {
    case restricted
    case offline
    case error
}

struct SupplierFilter: Identifiable {
    let id: String
    let name: String
}

@Observable
@MainActor
final class DeliveryMapViewModel {
    var orders: [TrackingOrder] = []
    var recentReceipts: [TrackingOrder] = []
    var fiscalQRReceipt: TrackingOrder?
    var suppliers: [SupplierFilter] = []
    var selectedSupplierIds: Set<String> = []
    var isLoading = false
    var activeFulfillmentCount = 0
    var loadIssue: TrackingLoadIssue?
    var selectedOrder: TrackingOrder?
    var hasFittedInitialCamera = false
    var cameraPosition: MapCameraPosition = .region(
        MKCoordinateRegion(
            center: CLLocationCoordinate2D(latitude: packMapCoordinate().lat, longitude: packMapCoordinate().lng),
            span: MKCoordinateSpan(latitudeDelta: 0.1, longitudeDelta: 0.1)
        )
    )

    private let api = APIClient.shared
    private let ws = RetailerWebSocket.shared
    private let network = NetworkLifecycleEngine.shared
    private let pollingInterval: TimeInterval = 15

    var selectedOrderId: String? {
        get { selectedOrder?.orderId }
        set {
            selectedOrder = newValue.flatMap { id in
                visibleOrders.first { $0.orderId == id }
            }
        }
    }

    var visibleOrders: [TrackingOrder] {
        orders.filter { order in
            order.hasDriverLocation && (selectedSupplierIds.isEmpty || selectedSupplierIds.contains(order.supplierId))
        }
    }

    var activeDeliveryCount: Int {
        if activeFulfillmentCount > 0 || orders.isEmpty {
            return activeFulfillmentCount
        }
        return orders.count
    }

    var emptyStateMessage: String {
        switch loadIssue {
        case .restricted:
            return "Your account cannot view retailer tracking right now"
        case .offline:
            return "Live tracking is offline. Reconnect to refresh driver positions"
        case .error:
            return "Tracking data could not be loaded right now"
        case nil:
            return activeDeliveryCount > 0
                ? "Active deliveries exist, but live driver location is not available yet"
                : "No active deliveries with driver location"
        }
    }

    func receiptHTMLURL(for receipt: TrackingOrder) -> URL? {
        let qr = receipt.fiscalQr.trimmingCharacters(in: .whitespacesAndNewlines)
        if !qr.isEmpty {
            if qr.contains("format=") { return URL(string: qr) }
            let sep = qr.contains("?") ? "&" : "?"
            return URL(string: qr + sep + "format=html")
        }
        let id = receipt.latestFiscalReceiptId.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !id.isEmpty else { return nil }
        let base = api.baseURL.hasSuffix("/") ? api.baseURL : api.baseURL + "/"
        return URL(string: "\(base)v1/platform/receipts/\(id)?format=html")
    }

    func receiptPDFURL(for receipt: TrackingOrder) -> URL? {
        let id = receipt.latestFiscalReceiptId.trimmingCharacters(in: .whitespacesAndNewlines)
        let base = api.baseURL.hasSuffix("/") ? api.baseURL : api.baseURL + "/"
        if !id.isEmpty {
            return URL(string: "\(base)v1/platform/receipts/\(id)?format=pdf")
        }
        return nil
    }

    func startPolling() async {
        while !Task.isCancelled {
            if network.isOnline {
                await refreshTrackingState()
            }
            try? await Task.sleep(for: .seconds(pollingInterval))
        }
    }

    func refreshTrackingState() async {
        async let trackingTask: Void = fetchTracking()
        async let fulfillmentsTask: Void = fetchActiveFulfillments()
        _ = await (trackingTask, fulfillmentsTask)
    }

    func fetchTracking() async {
        guard network.isOnline else { return }
        if orders.isEmpty { isLoading = true }
        defer { isLoading = false }
        do {
            let response = try await api.getTracking()
            orders = response.orders
            recentReceipts = response.recentReceipts ?? []
            loadIssue = nil
            let unique = Dictionary(grouping: orders, by: \.supplierId)
                .compactMap { (id, group) -> SupplierFilter? in
                    guard let name = group.first?.supplierName else { return nil }
                    return SupplierFilter(id: id, name: name)
                }
                .sorted { $0.name < $1.name }
            suppliers = unique
            fitCamera()
        } catch {
            if orders.isEmpty {
                loadIssue = resolveLoadIssue(error)
            }
        }
    }

    func fetchActiveFulfillments() async {
        guard network.isOnline else { return }
        do {
            let response = try await api.getActiveFulfillments()
            activeFulfillmentCount = response.count
            loadIssue = nil
        } catch {
            if activeFulfillmentCount == 0 {
                loadIssue = resolveLoadIssue(error)
            }
        }
    }

    func observeWebSocket() async {
        for await event in ws.eventStream() {
            switch event {
            case .orderCompleted(let e):
                orders.removeAll { $0.orderId == e.orderId }
                await fetchActiveFulfillments()
            case .driverApproaching(let orderId, _, let lat, let lng, _, _):
                if let idx = orders.firstIndex(where: { $0.orderId == orderId }) {
                    let old = orders[idx]
                    let updated = TrackingOrder(
                        orderId: old.orderId, supplierId: old.supplierId, supplierName: old.supplierName,
                        warehouseId: old.warehouseId, warehouseName: old.warehouseName,
                        driverId: old.driverId, state: old.state, totalAmount: old.totalAmount,
                        orderSource: old.orderSource, driverLatitude: lat ?? old.driverLatitude,
                        driverLongitude: lng ?? old.driverLongitude,
                        liveLocationAvailable: old.liveLocationAvailable,
                        isApproaching: true,
                        deliveryToken: old.deliveryToken, createdAt: old.createdAt, items: old.items
                    )
                    orders[idx] = updated
                }
            case .orderStatusChanged, .orderReassigned:
                await refreshTrackingState()
            default:
                break
            }
        }
    }

    func toggleSupplier(_ id: String) {
        Haptics.selection()
        if selectedSupplierIds.contains(id) {
            selectedSupplierIds.remove(id)
        } else {
            selectedSupplierIds.insert(id)
        }
    }

    func fitCamera() {
        let points = visibleOrders.compactMap { order -> CLLocationCoordinate2D? in
            guard let lat = order.driverLatitude, let lng = order.driverLongitude else { return nil }
            return CLLocationCoordinate2D(latitude: lat, longitude: lng)
        }
        guard !points.isEmpty else { return }
        hasFittedInitialCamera = true
        if points.count == 1 {
            withAnimation {
                cameraPosition = .region(MKCoordinateRegion(
                    center: points[0],
                    span: MKCoordinateSpan(latitudeDelta: 0.02, longitudeDelta: 0.02)
                ))
            }
        } else {
            var minLat = points[0].latitude, maxLat = points[0].latitude
            var minLng = points[0].longitude, maxLng = points[0].longitude
            for p in points {
                minLat = min(minLat, p.latitude); maxLat = max(maxLat, p.latitude)
                minLng = min(minLng, p.longitude); maxLng = max(maxLng, p.longitude)
            }
            let center = CLLocationCoordinate2D(latitude: (minLat + maxLat) / 2, longitude: (minLng + maxLng) / 2)
            let span = MKCoordinateSpan(latitudeDelta: (maxLat - minLat) * 1.4 + 0.01, longitudeDelta: (maxLng - minLng) * 1.4 + 0.01)
            withAnimation {
                cameraPosition = .region(MKCoordinateRegion(center: center, span: span))
            }
        }
    }

    private func resolveLoadIssue(_ error: Error) -> TrackingLoadIssue {
        if let apiError = error as? APIError,
           case let .serverError(statusCode, _) = apiError,
           statusCode == 401 || statusCode == 403 {
            return .restricted
        }

        if error is URLError || (error as NSError).domain == NSURLErrorDomain {
            return .offline
        }

        return .error
    }
}
