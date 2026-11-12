import Foundation
import SwiftUI
import Observation

@Observable
@MainActor
final class ActiveDeliveriesViewModel {
    var orders: [Order] = []
    var isLoading = false
    var loadError = false
    var selectedOrder: Order?
    var qrOverlayOrder: Order?
    var showMap = false
    var approachingOrderIds: Set<String> = []

    private let api = APIClient.shared
    private let ws = RetailerWebSocket.shared

    func loadOrders() async {
        let rid = AuthManager.shared.currentUser?.id ?? ""
        isLoading = true
        do {
            let result: [Order] = try await api.get(path: "/v1/retailers/\(rid)/orders")
            orders = result.filter { $0.status.isActive }
            loadError = false
        } catch {
            orders = []
            loadError = true
        }
        isLoading = false
    }

    func listenForApproaching() async {
        for await event in ws.eventStream() {
            if case .driverApproaching(let orderId, _, _, _, _, _) = event {
                approachingOrderIds.insert(orderId)
                if let order = orders.first(where: { $0.id == orderId && $0.status.hasDeliveryToken }) {
                    qrOverlayOrder = order
                }
            }
        }
    }
}
