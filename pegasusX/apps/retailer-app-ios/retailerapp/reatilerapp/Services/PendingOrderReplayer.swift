import Foundation
import SwiftData

/// Drains SwiftData pending checkout/procurement mutations after reconnect.
enum PendingOrderReplayer {
    @MainActor
    static func flush(modelContext: ModelContext, api: APIClient = .shared) async {
        let descriptor = FetchDescriptor<PendingOrder>(sortBy: [SortDescriptor(\.createdAt)])
        guard let pending = try? modelContext.fetch(descriptor), !pending.isEmpty else { return }

        for order in pending {
            do {
                if try await replay(order, api: api) {
                    modelContext.delete(order)
                }
            } catch {
                order.lastError = RetailerErrorSupport.retryQueuedMessage(
                    for: error,
                    fallback: "Pending order retry failed.",
                )
                order.retryCount += 1
            }
        }
        try? modelContext.save()
    }

    private static func replay(_ order: PendingOrder, api: APIClient) async throws -> Bool {
        guard order.method == "POST" else {
            order.lastError = "Unsupported pending mutation \(order.method) \(order.endpoint)"
            order.retryCount += 1
            return false
        }
        guard let data = order.payloadJson.data(using: .utf8) else {
            return true
        }

        let idempotencyKey = pendingOrderIdempotencyKey(order)

        switch order.endpoint {
        case "/v1/checkout/unified":
            guard let payload = try? JSONDecoder().decode(UnifiedCheckoutPayload.self, from: data) else {
                return true
            }
            _ = try await RetailerCheckoutService.completeCheckout(
                api: api,
                payload: payload,
                gateway: payload.paymentGateway,
                idempotencyKey: idempotencyKey
            )
            return true
        case "/v1/order/create":
            guard let payload = try? JSONDecoder().decode(ProcurementOrderRequest.self, from: data) else {
                return true
            }
            let _: ProcurementOrderResponse = try await api.post(
                path: order.endpoint,
                body: payload,
                headers: ["Idempotency-Key": idempotencyKey]
            )
            return true
        default:
            order.lastError = "Unsupported pending mutation \(order.method) \(order.endpoint)"
            order.retryCount += 1
            return false
        }
    }

    private static func pendingOrderIdempotencyKey(_ order: PendingOrder) -> String {
        if !order.idempotencyKey.isEmpty {
            return order.idempotencyKey
        }
        return "retailer-checkout-pending:\(Int64(order.createdAt.timeIntervalSince1970 * 1000))"
    }
}
