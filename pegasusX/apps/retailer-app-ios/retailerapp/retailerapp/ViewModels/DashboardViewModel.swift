import Foundation
import SwiftUI
import Observation

@Observable
@MainActor
final class DashboardViewModel {
    var activeOrders: [Order] = []
    var predictions: [RetailerAIPrediction] = []
    var reorderProducts: [Product] = []
    var isLoading = false
    var actingId: String?
    var orderActionPending = false
    var loadError: String?
    var pulseEvents: [RetailerPulseEvent] = []
    var pulseLoading = false
    var pulseError: String?
    var commandPulse: ControlTowerPulseWire?
    var commandPulseError: String?
    var commandJump: CommandStatusJump?

    private let api = APIClient.shared

    func loadData(silent: Bool = false) async {
        let rid = AuthManager.shared.currentUser?.id ?? ""
        if !silent { isLoading = true }
        if !silent { loadError = nil }
        pulseLoading = true
        pulseError = nil
        do {
            let orders: [Order] = try await api.get(path: "/v1/retailers/\(rid)/orders")
            activeOrders = orders.filter { $0.status.isActive }
        } catch {
            activeOrders = []
            if !silent {
                loadError = error.localizedDescription
            }
        }

        do {
            let p = try await api.getRetailerAIPredictions()
            predictions = p
        } catch {
            predictions = []
        }

        do {
            let products: [Product] = try await api.get(path: "/v1/catalog/products")
            reorderProducts = Array(products.prefix(6))
        } catch {
            reorderProducts = []
        }

        do {
            let pulse = try await api.getRetailerPulse()
            let result = PulseHonesty.apply(ok: true, incoming: pulse.events, previous: pulseEvents)
            pulseEvents = result.events
            pulseError = result.error
        } catch {
            pulseEvents = []
            pulseError = error.localizedDescription
        }
        await loadCommandPulse()
        pulseLoading = false
        if !silent { isLoading = false }
    }

    func pollCommandPulseLoop() async {
        while !Task.isCancelled {
            try? await Task.sleep(nanoseconds: 60_000_000_000)
            guard !Task.isCancelled else { break }
            if NetworkLifecycleEngine.shared.isOnline {
                await loadCommandPulse()
            }
        }
    }

    func loadCommandPulse() async {
        do {
            let incoming = try await api.getControlTowerPulse()
            let result = PulseHonesty.applyObject(ok: true, incoming: incoming, previous: commandPulse)
            commandPulse = result.value
            commandPulseError = result.error
        } catch {
            commandPulse = nil
            commandPulseError = error.localizedDescription
        }
    }

    func cancelOrder(_ orderId: String) async {
        let retailerId = AuthManager.shared.currentUser?.id ?? ""
        do {
            let _: [String: String] = try await api.post(
                path: "/v1/order/cancel",
                body: [
                    "order_id": orderId,
                    "retailer_id": retailerId,
                ],
                headers: ["Idempotency-Key": RetailerIdempotency.cancel(orderId: orderId)]
            )
            withAnimation(AnimationConstants.fluid) {
                activeOrders.removeAll { $0.id == orderId }
            }
        } catch {
            loadError = error.localizedDescription
        }
    }

    func confirmAiOrder(_ orderId: String) async {
        actingId = orderId
        orderActionPending = true
        defer {
            actingId = nil
            orderActionPending = false
        }
        do {
            try await APIClient.shared.confirmAiOrder(orderId: orderId)
            await loadData()
        } catch {
            loadError = "Failed to confirm AI order"
        }
    }

    func rejectAiOrder(_ orderId: String) async {
        actingId = orderId
        orderActionPending = true
        defer {
            actingId = nil
            orderActionPending = false
        }
        do {
            try await APIClient.shared.rejectAiOrder(orderId: orderId, reason: "Retailer rejected")
            await loadData()
        } catch {
            loadError = "Failed to reject AI order"
        }
    }

    func confirmPreorder(_ orderId: String) async {
        orderActionPending = true
        defer { orderActionPending = false }
        do {
            try await APIClient.shared.confirmPreorder(orderId: orderId)
            await loadData()
        } catch {
            loadError = "Failed to confirm preorder"
        }
    }

    func editPreorder(_ orderId: String) async {
        orderActionPending = true
        defer { orderActionPending = false }
        guard let order = await findOrder(orderId) else { return }
        let deliveryDate = order.deliverBefore ?? order.autoConfirmAt ?? order.estimatedDelivery ?? ""
        guard !deliveryDate.isEmpty else { return }
        let lineItems = order.items.map { item in
            APIClient.EditPreorderItem(
                sku: item.productId.isEmpty ? item.id : item.productId,
                name: item.productName,
                quantity: Int64(item.quantity),
                unitPriceMinor: item.unitPrice
            )
        }
        guard !lineItems.isEmpty else { return }
        do {
            try await APIClient.shared.editPreorder(orderId: orderId, deliveryDate: deliveryDate, items: lineItems)
            await loadData()
        } catch {
            print("Failed to edit preorder")
        }
    }

    func findOrder(_ orderId: String) async -> Order? {
        if let order = activeOrders.first(where: { $0.id == orderId }) {
            return order
        }
        let retailerId = AuthManager.shared.currentUser?.id ?? ""
        guard !retailerId.isEmpty else { return nil }
        let orders: [Order] = (try? await api.get(path: "/v1/retailers/\(retailerId)/orders")) ?? []
        return orders.first { $0.id == orderId }
    }
}
