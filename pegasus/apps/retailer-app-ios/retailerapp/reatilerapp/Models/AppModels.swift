import Foundation

// MARK: - User

struct User: Codable, Identifiable {
    let id: String
    let name: String
    let company: String
    let email: String?
    let avatarURL: String?

    enum CodingKeys: String, CodingKey {
        case id, name, company, email
        case avatarURL = "avatar_url"
    }
}

extension User {
    static let sample = User(
        id: "retailer-123",
        name: "Shakhzod",
        company: "The Lab Retail",
        email: nil,
        avatarURL: nil
    )
}

// MARK: - Auth Response

struct AuthResponse: Codable {
    let token: String
    let user: User
    let firebaseToken: String?
    
    enum CodingKeys: String, CodingKey {
        case token, user
        case firebaseToken = "firebase_token"
    }
}

// MARK: - Demand Forecast (AI)

struct DemandForecast: Codable, Identifiable, Hashable {
    let id: String
    let productId: String
    let productName: String
    let predictedQuantity: Int
    let confidence: Double
    let reasoning: String
    let suggestedOrderDate: String

    enum CodingKeys: String, CodingKey {
        case id
        case productId = "product_id"
        case productName = "product_name"
        case predictedQuantity = "predicted_quantity"
        case confidence
        case reasoning
        case suggestedOrderDate = "suggested_order_date"
    }

    var confidencePercent: String {
        String(format: "%.0f%%", confidence * 100)
    }
}

extension DemandForecast {
    static let samples: [DemandForecast] = [
        DemandForecast(id: "fc-001", productId: "prod-001", productName: "Organic Whole Milk", predictedQuantity: 24, confidence: 0.89, reasoning: "Steady weekly demand, slight uptick on weekends.", suggestedOrderDate: "2026-03-19"),
        DemandForecast(id: "fc-002", productId: "prod-003", productName: "Free-Range Eggs", predictedQuantity: 12, confidence: 0.76, reasoning: "Holiday season approaching, expect higher traffic.", suggestedOrderDate: "2026-03-18"),
        DemandForecast(id: "fc-003", productId: "prod-005", productName: "Sparkling Water", predictedQuantity: 36, confidence: 0.92, reasoning: "Trending product with repeat buyers.", suggestedOrderDate: "2026-03-20")
    ]
}

// MARK: - Cart Item

struct CartItem: Identifiable, Hashable {
    let id: String  // product_id + variant_id
    let product: Product
    let variant: Variant
    var quantity: Int

    var totalPrice: Double {
        Double(quantity) * variant.price
    }
}

// MARK: - API Generic Response

struct APIResponse<T: Codable>: Codable {
    let data: T?
    let error: String?
    let message: String?
}

// MARK: - Device Token

struct DeviceTokenPayload: Codable {
    let token: String
    let platform: String
    let retailerId: String

    enum CodingKeys: String, CodingKey {
        case token, platform
        case retailerId = "retailer_id"
    }
}
