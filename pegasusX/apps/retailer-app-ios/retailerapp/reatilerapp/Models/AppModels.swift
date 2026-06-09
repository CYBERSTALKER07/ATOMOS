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
        company: "Pegasus Retail",
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

// MARK: - Phase 4 Retailer Ecosystem Data Sync

struct RetailerProfileRequest: Encodable {
    let name: String?
    let company: String?
    let phone: String?
    let location: String?
    let regionId: String?
    let receivingWindowOpen: String?
    let receivingWindowClose: String?
    
    enum CodingKeys: String, CodingKey {
        case name, company, phone, location
        case regionId = "region_id"
        case receivingWindowOpen = "receiving_window_open"
        case receivingWindowClose = "receiving_window_close"
    }
}

struct RetailerProfileResponse: Decodable, Identifiable {
    let id: String
    let name: String
    let phone: String
    let company: String
    let location: String?
    let regionId: String?
    let taxId: String?
    let status: String
    let receivingWindowOpen: String?
    let receivingWindowClose: String?
    
    enum CodingKeys: String, CodingKey {
        case id
        case retailerId = "retailer_id"
        case name, phone, company, location, status
        case regionId = "region_id"
        case taxId = "tax_id"
        case receivingWindowOpen = "receiving_window_open"
        case receivingWindowClose = "receiving_window_close"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        id = try container.decodeIfPresent(String.self, forKey: .id)
            ?? container.decodeIfPresent(String.self, forKey: .retailerId)
            ?? ""
        name = try container.decodeIfPresent(String.self, forKey: .name) ?? ""
        phone = try container.decodeIfPresent(String.self, forKey: .phone) ?? ""
        company = try container.decodeIfPresent(String.self, forKey: .company) ?? name
        location = try container.decodeIfPresent(String.self, forKey: .location)
        regionId = try container.decodeIfPresent(String.self, forKey: .regionId)
        taxId = try container.decodeIfPresent(String.self, forKey: .taxId)
        status = try container.decodeIfPresent(String.self, forKey: .status) ?? "ACTIVE"
        receivingWindowOpen = try container.decodeIfPresent(String.self, forKey: .receivingWindowOpen)
        receivingWindowClose = try container.decodeIfPresent(String.self, forKey: .receivingWindowClose)
    }
}

struct FamilyMemberRequest: Encodable {
    let nickname: String
    let photoUrl: String?
    
    enum CodingKeys: String, CodingKey {
        case nickname
        case photoUrl = "photo_url"
    }
}

struct FamilyMemberResponse: Codable, Identifiable {
    let memberId: String
    let retailerId: String
    let nickname: String
    let photoUrl: String?
    let createdAt: String
    
    var id: String { memberId }
    
    enum CodingKeys: String, CodingKey {
        case nickname
        case memberId = "member_id"
        case retailerId = "retailer_id"
        case photoUrl = "photo_url"
        case createdAt = "created_at"
    }
}

struct CartSyncItem: Codable {
    let cartId: String?
    let skuId: String
    let supplierId: String
    let quantity: Int64
    let unitPrice: Int64
    let currency: String
    
    enum CodingKeys: String, CodingKey {
        case cartId = "cart_id"
        case skuId = "sku_id"
        case supplierId = "supplier_id"
        case quantity
        case unitPrice = "unit_price"
        case currency
    }
}

struct CartSyncRequest: Encodable {
    let items: [CartSyncItem]
}

struct CartSyncResponse: Codable {
    let items: [CartSyncItem]
    let total: Int
}

struct RetailerSupplierResponse: Codable, Identifiable {
    let id: String
    let name: String
    let logoUrl: String
    let category: String
    let primaryCategoryId: String?
    let operatingCategoryIds: [String]?
    let operatingCategoryNames: [String]?
    let orderCount: Int64
    let isActive: Bool
    
    enum CodingKeys: String, CodingKey {
        case id, name, category
        case logoUrl = "logo_url"
        case primaryCategoryId = "primary_category_id"
        case operatingCategoryIds = "operating_category_ids"
        case operatingCategoryNames = "operating_category_names"
        case orderCount = "order_count"
        case isActive = "is_active"
    }
}
