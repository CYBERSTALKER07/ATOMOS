import Foundation
import SwiftData

@Observable
final class OrdersViewModel {
    var allOrders: [Order] = []
    var predictions: [DemandForecast] = []
    var isLoading = false
    var loadError: String?
    var orderActionPending = false

    private let api = APIClient.shared

    var activeOrders: [Order] {
        allOrders.filter {
            $0.status == .loaded || $0.status == .dispatched || $0.status == .inTransit || $0.status == .arrived
        }
    }

    var pendingOrders: [Order] {
        allOrders.filter {
            $0.status == .pending || $0.status == .pendingReview || $0.status == .scheduled
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
            if !silent {
                allOrders = []
                loadError = "Could not load orders. Pull to refresh or try again."
            }
        }
        do {
            let forecasts: [DemandForecast] = try await api.get(path: "/v1/ai/predictions?retailer_id=\(rid)")
            predictions = forecasts
        } catch {
            if !silent { predictions = [] }
        }
    }

    func listenWebSocket(modelContext: ModelContext) async {
        let rid = AuthManager.shared.currentUser?.id ?? ""
        guard !rid.isEmpty else { return }
        RetailerWebSocket.shared.connect(retailerId: rid)
        for await event in RetailerWebSocket.shared.eventStream() {
            switch event {
            case .paymentRequired, .driverApproaching, .orderCompleted, .paymentSettled,
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
        let descriptor = FetchDescriptor<PendingOrder>(sortBy: [SortDescriptor(\.createdAt)])
        guard let pending = try? modelContext.fetch(descriptor), !pending.isEmpty else { return }
        for order in pending {
            do {
                if try await replayPendingOrder(order) {
                    modelContext.delete(order)
                }
            } catch {
                order.lastError = RetailerErrorSupport.retryQueuedMessage(
                    for: error,
                    fallback: "Pending order retry failed."
                )
                order.retryCount += 1
            }
        }
        try? modelContext.save()
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
            await loadData()
        } catch {
            loadError = useRequestCancel ? "Failed to request cancellation" : "Failed to cancel order"
        }
    }

    func confirmAiOrder(_ orderId: String) async {
        loadError = "AI orders are not available"
    }

    func rejectAiOrder(_ orderId: String) async {
        loadError = "AI orders are not available"
    }

    func confirmPreorder(_ orderId: String) async {
        guard !orderActionPending else { return }
        orderActionPending = true
        defer { orderActionPending = false }
        do {
            try await api.confirmPreorder(orderId: orderId)
            await loadData()
        } catch {
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
            await loadData()
        } catch {
            loadError = "Failed to edit preorder"
        }
    }

    func preorder(_ forecast: DemandForecast) async {
        do {
            struct PreorderBody: Codable {
                let productId: String
                let quantity: Int
                enum CodingKeys: String, CodingKey { case productId = "product_id"; case quantity }
            }
            let _: Order = try await api.post(
                path: "/v1/ai/preorder",
                body: PreorderBody(productId: forecast.productId, quantity: forecast.predictedQuantity),
                headers: ["Idempotency-Key": "retailer-ai-preorder:\(forecast.id):\(forecast.predictedQuantity)"]
            )
            Haptics.success()
        } catch {
            Haptics.error()
        }
    }

    private func replayPendingOrder(_ order: PendingOrder) async throws -> Bool {
        guard order.method == "POST" else {
            order.lastError = "Unsupported pending mutation \(order.method) \(order.endpoint)"
            order.retryCount += 1
            return false
        }
        guard let data = order.payloadJson.data(using: .utf8) else {
            return true
        }

        switch order.endpoint {
        case "/v1/checkout/unified":
            guard let payload = try? JSONDecoder().decode(UnifiedCheckoutPayload.self, from: data) else {
                return true
            }
            _ = try await RetailerCheckoutService.completeCheckout(
                api: api,
                payload: payload,
                gateway: payload.paymentGateway,
                idempotencyKey: pendingOrderIdempotencyKey(order)
            )
            return true
        case "/v1/order/create":
            guard let payload = try? JSONDecoder().decode(ProcurementOrderRequest.self, from: data) else {
                return true
            }
            let _: ProcurementOrderResponse = try await api.post(
                path: order.endpoint,
                body: payload,
                headers: ["Idempotency-Key": pendingOrderIdempotencyKey(order)]
            )
            return true
        default:
            order.lastError = "Unsupported pending mutation \(order.method) \(order.endpoint)"
            order.retryCount += 1
            return false
        }
    }

    private func pendingOrderIdempotencyKey(_ order: PendingOrder) -> String {
        if !order.idempotencyKey.isEmpty {
            return order.idempotencyKey
        }
        return "retailer-checkout-pending:\(Int64(order.createdAt.timeIntervalSince1970 * 1000))"
    }
}
