import Foundation

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
        return response
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

