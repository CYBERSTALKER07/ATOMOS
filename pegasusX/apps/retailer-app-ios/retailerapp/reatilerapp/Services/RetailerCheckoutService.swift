import Foundation
import UIKit

/// Orchestrates cart unified checkout and per-order card/cash payment initiation.
enum RetailerCheckoutService {
    static func completeCheckout(
        api: APIClient,
        payload: UnifiedCheckoutPayload,
        gateway: String,
        idempotencyKey: String
    ) async throws -> CheckoutResponse {
        let preview: CheckoutPreviewResponse = try await api.post(
            path: "/v1/checkout/preview",
            body: payload
        )
        if preview.blocked == true {
            throw CheckoutPreviewError.blocked(preview.message ?? "Checkout blocked by stock policy")
        }
        let response: CheckoutResponse = try await api.post(
            path: "/v1/checkout/unified",
            body: payload,
            headers: ["Idempotency-Key": idempotencyKey]
        )
        try await completeSupplierOrderPayments(
            api: api,
            supplierOrders: response.supplierOrders ?? [],
            gateway: gateway,
            invoiceId: response.invoiceId,
            backorderOrderId: response.backorderOrderId
        )
        return response
    }

    static func completeSupplierOrderPayments(
        api: APIClient,
        supplierOrders: [CheckoutResponse.SupplierOrderResult],
        gateway: String,
        invoiceId: String,
        backorderOrderId: String? = nil
    ) async throws {
        let normalizedGateway = gateway.trimmingCharacters(in: .whitespacesAndNewlines).uppercased()
        let isCash = normalizedGateway == "CASH"

        for supplierOrder in supplierOrders {
            if let backorderOrderId, supplierOrder.orderId == backorderOrderId {
                continue
            }
            if isCash {
                let _: CashCheckoutResponse = try await api.post(
                    path: "/v1/order/cash-checkout",
                    body: CashCheckoutRequest(orderId: supplierOrder.orderId),
                    headers: ["Idempotency-Key": "retailer-cash-checkout:\(supplierOrder.orderId)"]
                )
                continue
            }

            let card: CardCheckoutResponse = try await api.post(
                path: "/v1/order/card-checkout",
                body: CardCheckoutRequest(
                    orderId: supplierOrder.orderId,
                    gateway: normalizedGateway,
                    amount: supplierOrder.total,
                    invoiceId: invoiceId
                ),
                headers: ["Idempotency-Key": "retailer-card-checkout:\(supplierOrder.orderId):\(normalizedGateway)"]
            )
            if let url = URL(string: card.paymentUrl), !card.paymentUrl.isEmpty {
                await MainActor.run {
                    UIApplication.shared.open(url)
                }
            }
        }
    }
}

enum CheckoutPreviewError: LocalizedError {
    case blocked(String)
    var errorDescription: String? {
        switch self {
        case .blocked(let message): return message
        }
    }
}

struct CheckoutPreviewResponse: Decodable {
    let ok: Bool
    let blocked: Bool?
    let message: String?
}

private struct CashCheckoutRequest: Encodable {
    let orderId: String

    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
    }
}

private struct CardCheckoutRequest: Encodable {
    let orderId: String
    let gateway: String
    let amount: Int64
    let invoiceId: String

    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
        case gateway
        case amount
        case invoiceId = "invoice_id"
    }
}

private struct CashCheckoutResponse: Decodable {
    let orderId: String
    let state: String
    let amount: Int64
    let retailerId: String
    let message: String

    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
        case state
        case amount
        case retailerId = "retailer_id"
        case message
    }
}

private struct CardCheckoutResponse: Decodable {
    let orderId: String
    let state: String
    let amount: Int64
    let gateway: String
    let paymentUrl: String
    let invoiceId: String
    let sessionId: String?
    let attemptId: String?
    let retailerId: String
    let message: String

    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
        case state
        case amount
        case gateway
        case paymentUrl = "payment_url"
        case invoiceId = "invoice_id"
        case sessionId = "session_id"
        case attemptId = "attempt_id"
        case retailerId = "retailer_id"
        case message
    }
}
