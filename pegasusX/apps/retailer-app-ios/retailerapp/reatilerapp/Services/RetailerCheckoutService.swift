import Foundation

/// Orchestrates cart unified checkout and per-order card/cash payment initiation.
enum RetailerCheckoutService {
    static func fetchPreview(
        api: APIClient,
        payload: UnifiedCheckoutPayload
    ) async throws -> CheckoutPreviewResponse {
        let preview: CheckoutPreviewResponse = try await api.post(
            path: "/v1/checkout/preview",
            body: payload
        )
        if preview.blocked == true {
            throw CheckoutPreviewError.blocked(
                message: preview.message ?? "Checkout blocked by stock policy",
                oosItems: preview.oosItems ?? preview.rejectedSkus ?? []
            )
        }
        return preview
    }

    static func completeCheckout(
        api: APIClient,
        payload: UnifiedCheckoutPayload,
        gateway: String,
        idempotencyKey: String,
        skipBackorderConfirm: Bool = false,
        onBackorderWarning: ((CheckoutPreviewResponse) -> Bool)? = nil
    ) async throws -> CheckoutResponse {
        let preview = try await fetchPreview(api: api, payload: payload)
        if !preview.stockWarnings.isEmpty && !skipBackorderConfirm {
            let proceed = onBackorderWarning?(preview) ?? false
            if !proceed {
                throw CheckoutPreviewError.backorderConfirmationRequired(preview)
            }
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
    case blocked(message: String, oosItems: [String])
    case backorderConfirmationRequired(CheckoutPreviewResponse)

    var errorDescription: String? {
        switch self {
        case .blocked(let message, _):
            return message
        case .backorderConfirmationRequired:
            return "Some items will be backordered. Confirm to proceed."
        }
    }
}

struct CheckoutPreviewResponse: Decodable {
    let ok: Bool
    let blocked: Bool?
    let code: String?
    let message: String?
    let rejectedSkus: [String]?
    let oosItems: [String]?
    let shortfall: [String: Int64]?
    let stockWarnings: [StockWarning]
    let maxQuantities: [String: Int64]?
    let orderableQuantities: [String: Int64]?
    let backorderedItemCount: Int?
    let showStockCounts: Bool?
    let deliveryFeeMinor: Int64?
    let deliveryDistanceKm: Double?
    let preorderMinLeadDays: Int64?
    let preorderMaxLeadDays: Int64?
    let orderLineMinQuantity: Int64?
    let orderLineMaxQuantity: Int64?
    let defaultOutOfStockPolicy: String?
    let checkoutPolicyToken: String?
    let checkoutPolicyExpiresAt: String?
    let orderAcceptanceOpen: Bool?
    let orderAcceptanceWindowLabel: String?
    let nextOrderAcceptanceAt: String?

    enum CodingKeys: String, CodingKey {
        case ok, blocked, code, message, shortfall
        case rejectedSkus = "rejected_skus"
        case oosItems = "oos_items"
        case stockWarnings = "stock_warnings"
        case maxQuantities = "max_quantities"
        case orderableQuantities = "orderable_quantities"
        case backorderedItemCount = "backordered_item_count"
        case showStockCounts = "show_stock_counts"
        case deliveryFeeMinor = "delivery_fee_minor"
        case deliveryDistanceKm = "delivery_distance_km"
        case preorderMinLeadDays = "preorder_min_lead_days"
        case preorderMaxLeadDays = "preorder_max_lead_days"
        case orderLineMinQuantity = "order_line_min_quantity"
        case orderLineMaxQuantity = "order_line_max_quantity"
        case defaultOutOfStockPolicy = "default_out_of_stock_policy"
        case checkoutPolicyToken = "checkout_policy_token"
        case checkoutPolicyExpiresAt = "checkout_policy_expires_at"
        case orderAcceptanceOpen = "order_acceptance_open"
        case orderAcceptanceWindowLabel = "order_acceptance_window_label"
        case nextOrderAcceptanceAt = "next_order_acceptance_at"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        ok = try container.decode(Bool.self, forKey: .ok)
        blocked = try container.decodeIfPresent(Bool.self, forKey: .blocked)
        code = try container.decodeIfPresent(String.self, forKey: .code)
        message = try container.decodeIfPresent(String.self, forKey: .message)
        rejectedSkus = try container.decodeIfPresent([String].self, forKey: .rejectedSkus)
        oosItems = try container.decodeIfPresent([String].self, forKey: .oosItems)
        shortfall = try container.decodeIfPresent([String: Int64].self, forKey: .shortfall)
        stockWarnings = try container.decodeIfPresent([StockWarning].self, forKey: .stockWarnings) ?? []
        maxQuantities = try container.decodeIfPresent([String: Int64].self, forKey: .maxQuantities)
        orderableQuantities = try container.decodeIfPresent([String: Int64].self, forKey: .orderableQuantities)
        backorderedItemCount = try container.decodeIfPresent(Int.self, forKey: .backorderedItemCount)
        showStockCounts = try container.decodeIfPresent(Bool.self, forKey: .showStockCounts)
        deliveryFeeMinor = try container.decodeIfPresent(Int64.self, forKey: .deliveryFeeMinor)
        deliveryDistanceKm = try container.decodeIfPresent(Double.self, forKey: .deliveryDistanceKm)
        preorderMinLeadDays = try container.decodeIfPresent(Int64.self, forKey: .preorderMinLeadDays)
        preorderMaxLeadDays = try container.decodeIfPresent(Int64.self, forKey: .preorderMaxLeadDays)
        orderLineMinQuantity = try container.decodeIfPresent(Int64.self, forKey: .orderLineMinQuantity)
        orderLineMaxQuantity = try container.decodeIfPresent(Int64.self, forKey: .orderLineMaxQuantity)
        defaultOutOfStockPolicy = try container.decodeIfPresent(String.self, forKey: .defaultOutOfStockPolicy)
        checkoutPolicyToken = try container.decodeIfPresent(String.self, forKey: .checkoutPolicyToken)
        checkoutPolicyExpiresAt = try container.decodeIfPresent(String.self, forKey: .checkoutPolicyExpiresAt)
        orderAcceptanceOpen = try container.decodeIfPresent(Bool.self, forKey: .orderAcceptanceOpen)
        orderAcceptanceWindowLabel = try container.decodeIfPresent(String.self, forKey: .orderAcceptanceWindowLabel)
        nextOrderAcceptanceAt = try container.decodeIfPresent(String.self, forKey: .nextOrderAcceptanceAt)
    }
}
