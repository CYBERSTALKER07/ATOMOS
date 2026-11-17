import Foundation
import Observation
import SwiftUI

struct SupplierDockGroup: Identifiable {
    var id: String { supplierId }
    let supplierId: String
    let supplierName: String
    let orders: [TrackingOrder]
    let totalAmount: Int
    let hasApproaching: Bool
    let hasArrived: Bool
}

@Observable
@MainActor
final class ArrivalViewModel {
    var orders: [TrackingOrder] = []
    var expandedSupplierIds: Set<String> = []
    var revealedTokenOrderIds: Set<String> = []
    var activeQrOrder: TrackingOrder?
    var autoOpenedOrderIds: Set<String> = []
    var isLoading = false
    var isRefreshing = false
    var loadError: String?

    private let api = APIClient.shared
    private let ws = RetailerWebSocket.shared
    private let dockStates: Set<String> = [
        "DISPATCHED", "IN_TRANSIT", "ARRIVING", "ARRIVED", "AWAITING_PAYMENT",
    ]

    var activeOrders: [TrackingOrder] {
        orders.filter { dockStates.contains($0.state.uppercased()) }
    }

    var supplierGroups: [SupplierDockGroup] {
        var map: [String: SupplierDockGroup] = [:]
        for order in activeOrders {
            if var group = map[order.supplierId] {
                group = SupplierDockGroup(
                    supplierId: group.supplierId,
                    supplierName: group.supplierName,
                    orders: group.orders + [order],
                    totalAmount: group.totalAmount + order.totalAmount,
                    hasApproaching: group.hasApproaching || order.isApproaching || order.state == "ARRIVING",
                    hasArrived: group.hasArrived || order.state == "ARRIVED" || order.state == "AWAITING_PAYMENT"
                )
                map[order.supplierId] = group
            } else {
                map[order.supplierId] = SupplierDockGroup(
                    supplierId: order.supplierId,
                    supplierName: order.supplierName.isEmpty ? String(order.supplierId.prefix(8)) : order.supplierName,
                    orders: [order],
                    totalAmount: order.totalAmount,
                    hasApproaching: order.isApproaching || order.state == "ARRIVING",
                    hasArrived: order.state == "ARRIVED" || order.state == "AWAITING_PAYMENT"
                )
            }
        }
        return map.values.sorted { $0.totalAmount > $1.totalAmount }
    }

    func toggleSupplierExpanded(_ supplierId: String) {
        Haptics.selection()
        if expandedSupplierIds.contains(supplierId) {
            expandedSupplierIds.remove(supplierId)
        } else {
            expandedSupplierIds.insert(supplierId)
        }
    }

    func toggleRevealToken(_ orderId: String) {
        Haptics.light()
        if revealedTokenOrderIds.contains(orderId) {
            revealedTokenOrderIds.remove(orderId)
        } else {
            revealedTokenOrderIds.insert(orderId)
        }
    }

    func loadOrders(refreshing: Bool = false) async {
        if refreshing { isRefreshing = true } else { isLoading = true }
        loadError = nil
        do {
            let response: TrackingResponse = try await api.get(path: "/v1/retailer/tracking")
            orders = response.orders.filter { !["COMPLETED", "CANCELLED"].contains($0.state.uppercased()) }
            if expandedSupplierIds.isEmpty {
                expandedSupplierIds = Set(supplierGroups.map(\.supplierId))
            } else {
                let available = Set(supplierGroups.map(\.supplierId))
                expandedSupplierIds = expandedSupplierIds.intersection(available)
            }
            maybeAutoOpenQr()
        } catch {
            orders = []
            if expandedSupplierIds.isEmpty {
                expandedSupplierIds = Set(supplierGroups.map(\.supplierId))
            }
            loadError = error.localizedDescription
        }
        isLoading = false
        isRefreshing = false
    }

    func maybeAutoOpenQr() {
        guard let candidate = activeOrders.first(where: {
            !$0.deliveryToken.isEmpty &&
            ($0.isApproaching || $0.state == "ARRIVED") &&
            !autoOpenedOrderIds.contains($0.orderId)
        }) else { return }

        autoOpenedOrderIds.insert(candidate.orderId)
        expandedSupplierIds.insert(candidate.supplierId)
        revealedTokenOrderIds.insert(candidate.orderId)
        activeQrOrder = candidate
    }

    func observeWebSocket() async {
        for await event in ws.eventStream() {
            switch event {
            case .driverApproaching(let orderId, let deliveryToken, _, _, _, _):
                orders = orders.map { order in
                    guard order.orderId == orderId else { return order }
                    return TrackingOrder(
                        orderId: order.orderId,
                        supplierId: order.supplierId,
                        supplierName: order.supplierName,
                        warehouseId: order.warehouseId,
                        warehouseName: order.warehouseName,
                        driverId: order.driverId,
                        state: order.state == "IN_TRANSIT" ? "ARRIVING" : order.state,
                        totalAmount: order.totalAmount,
                        orderSource: order.orderSource,
                        driverLatitude: order.driverLatitude,
                        driverLongitude: order.driverLongitude,
                        liveLocationAvailable: order.liveLocationAvailable,
                        isApproaching: true,
                        deliveryToken: deliveryToken.isEmpty ? order.deliveryToken : deliveryToken,
                        createdAt: order.createdAt,
                        items: order.items
                    )
                }
                maybeAutoOpenQr()
            case .orderStatusChanged(let orderId, let newState):
                orders = orders.map { order in
                    guard order.orderId == orderId else { return order }
                    return TrackingOrder(
                        orderId: order.orderId,
                        supplierId: order.supplierId,
                        supplierName: order.supplierName,
                        warehouseId: order.warehouseId,
                        warehouseName: order.warehouseName,
                        driverId: order.driverId,
                        state: newState,
                        totalAmount: order.totalAmount,
                        orderSource: order.orderSource,
                        driverLatitude: order.driverLatitude,
                        driverLongitude: order.driverLongitude,
                        liveLocationAvailable: order.liveLocationAvailable,
                        isApproaching: order.isApproaching,
                        deliveryToken: order.deliveryToken,
                        createdAt: order.createdAt,
                        items: order.items
                    )
                }
            case .orderCompleted(let event):
                orders.removeAll { $0.orderId == event.orderId }
                if activeQrOrder?.orderId == event.orderId { activeQrOrder = nil }
            default:
                break
            }
        }
    }
}
