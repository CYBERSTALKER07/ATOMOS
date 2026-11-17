import Foundation
import Observation
import SwiftData

@Observable
@MainActor
final class OrdersViewModel {
    var allOrders: [Order] = []
    var predictions: [RetailerAIPrediction] = []
    var isLoading = false
    var loadError: String?
    var orderActionPending = false
    var commandStatus: String?
    var commandSupplierId: String?

    var selectedTab: OrderTab = .active
    var selectedOrder: Order?
    var qrOverlayOrder: Order?
    var deliveryProposalOrder: Order?

    private let api = APIClient.shared

    var showLoadErrorAlert: Bool {
        get { loadError != nil }
        set {
            if !newValue {
                loadError = nil
            }
        }
    }

    var commandFilteredOrders: [Order] {
        guard let commandStatus else { return allOrders }
        return allOrders.filter {
            retailerOrderMatchesCommand(
                statusRaw: $0.status.rawValue,
                supplierId: $0.supplierId,
                commandStatus: commandStatus,
                commandSupplierId: commandSupplierId
            )
        }
    }

    func applyCommandFilter(status: String, supplierId: String? = nil) {
        commandStatus = canonicalizeOrderStatus(status)
        commandSupplierId = supplierId
    }

    func clearCommandFilter() {
        Haptics.light()
        commandStatus = nil
        commandSupplierId = nil
    }

    var activeOrders: [Order] {
        allOrders.filter {
            $0.status.customerStage == .outForDelivery ||
                $0.status.customerStage == .exceptionActionRequired ||
                $0.status == .loaded || $0.status == .dispatched
        }
    }

    var pendingOrders: [Order] {
        allOrders.filter {
            $0.status.customerStage == .placed ||
                $0.status == .pending || $0.status == .delayed
        }
    }

    func loadData(silent: Bool = false) async {
        let rid = AuthManager.shared.currentUser?.id ?? ""
        if !silent { isLoading = true }
        defer { if !silent { isLoading = false } }
        do {
            let orders: [Order] = try await api.get(path: "/v1/retailers/\(rid)/orders")
            allOrders = orders
        } catch {
            allOrders = []
        }
        do {
            predictions = try await api.getRetailerAIPredictions()
        } catch {
            predictions = []
        }
    }

    func listenWebSocket(modelContext: ModelContext) async {
        let rid = AuthManager.shared.currentUser?.id ?? ""
        guard !rid.isEmpty else { return }
        RetailerWebSocket.shared.connect(retailerId: rid)
        for await event in RetailerWebSocket.shared.eventStream() {
            switch event {
            case .paymentRequired, .driverApproaching, .orderCompleted, .fiscalSucceeded,
                 .fiscalizing, .paymentSettled,
                 .paymentFailed, .paymentExpired, .orderStatusChanged, .orderReassigned,
                 .preOrderAutoAccepted, .preOrderConfirmed, .preOrderEdited,
                 .preOrderNudge, .preOrderConfirmationPush,
                 .preOrderDateProposed, .preOrderDateAccepted, .preOrderDateRejected, .preOrderCancelled:
                await loadData(silent: !allOrders.isEmpty)
            case .transportReconnected:
                await flushPendingOrders(modelContext: modelContext)
                await loadData(silent: !allOrders.isEmpty)
            case .shopClosedAlert, .cartSyncUpdated, .promotionChanged:
                break
            }
        }
    }

    func flushPendingOrders(modelContext: ModelContext) async {
        await PendingOrderReplayer.flush(modelContext: modelContext, api: api)
    }

    func cancelOrder(_ order: Order) async {
        guard !orderActionPending else { return }
        orderActionPending = true
        defer { orderActionPending = false }
        let rid = AuthManager.shared.currentUser?.id ?? ""
        guard !rid.isEmpty else { return }

        let requestCancelStates: Set<OrderStatus> = [.dispatched, .inTransit, .arrived]
        let useRequestCancel = requestCancelStates.contains(order.status)

        do {
            if useRequestCancel {
                try await api.requestCancelOrder(orderId: order.id, retailerId: rid)
            } else {
                let _: [String: String] = try await api.post(
                    path: "/v1/order/cancel",
                    body: [
                        "order_id": order.id,
                        "retailer_id": rid,
                        "reason": "Retailer requested cancellation",
                    ],
                    headers: ["Idempotency-Key": RetailerIdempotency.cancel(orderId: order.id)]
                )
            }
            Haptics.success()
            await loadData()
        } catch {
            Haptics.error()
            loadError = useRequestCancel ? "Failed to request cancellation" : "Failed to cancel order"
        }
    }

    func confirmAiOrder(_ orderId: String) async {
        guard !orderActionPending else { return }
        orderActionPending = true
        defer { orderActionPending = false }
        do {
            try await api.confirmAiOrder(orderId: orderId)
            Haptics.success()
            await loadData()
        } catch {
            Haptics.error()
            loadError = "Failed to confirm AI order"
        }
    }

    func rejectAiOrder(_ orderId: String) async {
        guard !orderActionPending else { return }
        orderActionPending = true
        defer { orderActionPending = false }
        do {
            try await api.rejectAiOrder(orderId: orderId, reason: "Retailer rejected")
            Haptics.light()
            await loadData()
        } catch {
            Haptics.error()
            loadError = "Failed to reject AI order"
        }
    }

    func confirmPreorder(_ orderId: String) async {
        guard !orderActionPending else { return }
        orderActionPending = true
        defer { orderActionPending = false }
        do {
            try await api.confirmPreorder(orderId: orderId)
            Haptics.success()
            await loadData()
        } catch {
            Haptics.error()
            loadError = "Failed to confirm preorder"
        }
    }

    func acceptDeliveryProposal(_ orderId: String) async {
        guard !orderActionPending else { return }
        orderActionPending = true
        defer { orderActionPending = false }
        do {
            try await api.acceptDeliveryProposal(orderId: orderId)
            Haptics.success()
            await loadData()
        } catch {
            Haptics.error()
            loadError = "Failed to accept delivery proposal"
        }
    }

    func rejectDeliveryProposal(_ orderId: String, reason: String? = "Retailer rejected proposed date") async {
        guard !orderActionPending else { return }
        orderActionPending = true
        defer { orderActionPending = false }
        do {
            try await api.rejectDeliveryProposal(orderId: orderId, reason: reason)
            Haptics.light()
            await loadData()
        } catch {
            Haptics.error()
            loadError = "Failed to reject delivery proposal"
        }
    }

    func editPreorder(_ order: Order) async {
        guard !orderActionPending else { return }
        orderActionPending = true
        defer { orderActionPending = false }
        let deliveryDate = order.deliverBefore ?? order.autoConfirmAt ?? ""
        let items = order.items.map { item in
            APIClient.EditPreorderItem(
                sku: item.productId.isEmpty ? item.id : item.productId,
                name: item.productName,
                quantity: Int64(item.quantity),
                unitPriceMinor: Int64(item.unitPrice)
            )
        }
        do {
            try await api.editPreorder(orderId: order.id, deliveryDate: deliveryDate, items: items)
            Haptics.success()
            await loadData()
        } catch {
            Haptics.error()
            loadError = "Failed to edit preorder"
        }
    }
}
