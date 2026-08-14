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

// MARK: - Auth Response (C1.2/C1.3 multi-org aware)

struct RetailerMembershipDTO: Codable, Identifiable, Hashable {
    var id: String { "\(retailerId)|\(userId)" }
    let userId: String
    let retailerId: String
    let retailerRole: String
    let name: String?
    let phone: String?
    let isActive: Bool

    enum CodingKeys: String, CodingKey {
        case userId = "user_id"
        case retailerId = "retailer_id"
        case retailerRole = "retailer_role"
        case name, phone
        case isActive = "is_active"
    }
}

struct AuthResponse: Codable {
    let token: String
    let tokenType: String?
    let user: User?
    let firebaseToken: String?
    let isConfigured: Bool?
    let memberships: [RetailerMembershipDTO]?
    let expiresInSec: Int?
    let retailerId: String?
    let refreshToken: String?

    enum CodingKeys: String, CodingKey {
        case token, user, memberships
        case tokenType = "token_type"
        case firebaseToken = "firebase_token"
        case isConfigured = "is_configured"
        case expiresInSec = "expires_in_sec"
        case retailerId = "retailer_id"
        case refreshToken = "refresh_token"
    }

    var isPendingOrgSelect: Bool {
        (tokenType ?? "").lowercased() == "pending_org_select"
    }
}

struct SelectOrgRequest: Encodable {
    let retailerId: String
    enum CodingKeys: String, CodingKey { case retailerId = "retailer_id" }
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
    let blocked: Bool?
    let blockedReason: String?
    let label: String?

    enum CodingKeys: String, CodingKey {
        case id
        case productId = "product_id"
        case productName = "product_name"
        case predictedQuantity = "predicted_quantity"
        case confidence
        case reasoning
        case suggestedOrderDate = "suggested_order_date"
        case blocked
        case blockedReason = "blocked_reason"
        case label
    }

    var confidencePercent: String {
        String(format: "%.0f%%", confidence * 100)
    }

    var isBlocked: Bool {
        blocked == true || label == "insufficient_history" || (blockedReason?.isEmpty == false)
    }
}

extension DemandForecast {
    static let samples: [DemandForecast] = [
        DemandForecast(id: "fc-001", productId: "prod-001", productName: "Organic Whole Milk", predictedQuantity: 24, confidence: 0.89, reasoning: "Steady weekly demand, slight uptick on weekends.", suggestedOrderDate: "2026-03-19", blocked: nil, blockedReason: nil, label: nil),
        DemandForecast(id: "fc-002", productId: "prod-003", productName: "Free-Range Eggs", predictedQuantity: 12, confidence: 0.76, reasoning: "Holiday season approaching, expect higher traffic.", suggestedOrderDate: "2026-03-18", blocked: nil, blockedReason: nil, label: nil),
        DemandForecast(id: "fc-003", productId: "prod-005", productName: "Sparkling Water", predictedQuantity: 36, confidence: 0.92, reasoning: "Trending product with repeat buyers.", suggestedOrderDate: "2026-03-20", blocked: nil, blockedReason: nil, label: nil)
    ]
}

// MARK: - Retailer AI predictions (pending AI preorders; not SKU DemandForecast)

struct RetailerAILineItem: Codable, Hashable {
    let sku: String
    let name: String?
    let quantity: Int64
    let unitPriceMinor: Int64?

    enum CodingKeys: String, CodingKey {
        case sku, name, quantity
        case unitPriceMinor = "unit_price_minor"
    }
}

struct RetailerAIPrediction: Codable, Identifiable, Hashable {
    var id: String { orderId }
    let orderId: String
    let supplierId: String?
    let orderSource: String?
    let confirmationStatus: String?
    let requestedDeliveryDate: String?
    let autoConfirmAt: String?
    let totalMinor: Int64
    let currency: String
    let derivedFromOrderId: String?
    let updatedAt: String
    let lineItems: [RetailerAILineItem]?

    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
        case supplierId = "supplier_id"
        case orderSource = "order_source"
        case confirmationStatus = "confirmation_status"
        case requestedDeliveryDate = "requested_delivery_date"
        case autoConfirmAt = "auto_confirm_at"
        case totalMinor = "total_minor"
        case currency
        case derivedFromOrderId = "derived_from_order_id"
        case updatedAt = "updated_at"
        case lineItems = "line_items"
    }

    var title: String {
        if let first = lineItems?.first {
            let label = (first.name?.isEmpty == false ? first.name : nil) ?? first.sku
            if !label.isEmpty { return label }
        }
        return orderId
    }

    var quantity: Int64 {
        (lineItems ?? []).reduce(0) { $0 + $1.quantity }
    }

    var statusLabel: String {
        let raw = (confirmationStatus?.isEmpty == false ? confirmationStatus : nil) ?? "PENDING"
        return raw.replacingOccurrences(of: "_", with: " ")
    }

    var deliveryLabel: String {
        guard let date = requestedDeliveryDate, !date.isEmpty else { return orderId }
        return String(date.prefix(10))
    }

    var formattedTotal: String {
        let units = totalMinor / 100
        if currency.isEmpty { return "\(units)" }
        return "\(units) \(currency)"
    }
}

struct RetailerAIPredictionsResponse: Codable {
    let items: [RetailerAIPrediction]
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

// Retail OS Phase 0
struct RetailerCapabilityPack: Codable, Identifiable {
    let id: String
    let name: String
    let description: String
    let hardDeps: [String]?
    let softDeps: [String]?
    let alwaysOn: Bool?
    let enabled: Bool

    enum CodingKeys: String, CodingKey {
        case id, name, description, enabled
        case hardDeps = "hard_deps"
        case softDeps = "soft_deps"
        case alwaysOn = "always_on"
    }
}

struct RetailerCapabilitiesResponse: Codable {
    let retailerId: String
    let capabilities: [String]
    let packs: [RetailerCapabilityPack]

    enum CodingKeys: String, CodingKey {
        case capabilities, packs
        case retailerId = "retailer_id"
    }
}

struct RetailerMeResponse: Codable {
    let userId: String
    let retailerId: String
    let retailerOrgId: String
    let retailerRole: String
    let name: String
    let permissions: [String]
    let capabilities: [String]

    enum CodingKeys: String, CodingKey {
        case name, permissions, capabilities
        case userId = "user_id"
        case retailerId = "retailer_id"
        case retailerOrgId = "retailer_org_id"
        case retailerRole = "retailer_role"
    }
}

struct RetailerCapabilityMutationResponse: Codable {
    let status: String?
    let packId: String?
    let enabled: Bool?
    let message: String?
    let capabilities: [String]?

    enum CodingKeys: String, CodingKey {
        case status, enabled, message, capabilities
        case packId = "pack_id"
    }
}

// Retail OS Phase 2 locations
struct RetailerLocationDTO: Codable, Identifiable {
    let locationId: String
    let retailerId: String?
    let name: String
    let deliveryAddress: String?
    let lat: Double?
    let lng: Double?
    let isPrimary: Bool
    let isActive: Bool

    var id: String { locationId }

    enum CodingKeys: String, CodingKey {
        case name, lat, lng
        case locationId = "location_id"
        case retailerId = "retailer_id"
        case deliveryAddress = "delivery_address"
        case isPrimary = "is_primary"
        case isActive = "is_active"
    }
}

struct RetailerLocationsResponse: Codable {
    let retailerId: String?
    let activeLocationId: String?
    let items: [RetailerLocationDTO]

    enum CodingKeys: String, CodingKey {
        case items
        case retailerId = "retailer_id"
        case activeLocationId = "active_location_id"
    }
}

// Retail OS Phase 1 team
struct RetailerOrgMember: Codable, Identifiable {
    let userId: String
    let retailerId: String
    let name: String
    let phone: String
    let retailerRole: String
    let isOwner: Bool
    let isActive: Bool

    var id: String { userId }

    enum CodingKeys: String, CodingKey {
        case name, phone
        case userId = "user_id"
        case retailerId = "retailer_id"
        case retailerRole = "retailer_role"
        case isOwner = "is_owner"
        case isActive = "is_active"
    }
}

struct RetailerOrgMembersResponse: Codable {
    let retailerId: String
    let items: [RetailerOrgMember]
    let updatedAt: String?

    enum CodingKeys: String, CodingKey {
        case items
        case retailerId = "retailer_id"
        case updatedAt = "updated_at"
    }
}

struct FamilyMemberRequest: Encodable {
    let name: String
    let phone: String?
    let photoUrl: String?

    enum CodingKeys: String, CodingKey {
        case name
        case phone
        case photoUrl = "photo_url"
    }
}

struct FamilyMemberResponse: Codable, Identifiable {
    let memberId: String
    let retailerId: String?
    let name: String
    let phone: String?
    let photoUrl: String?
    let createdAt: String?

    var id: String { memberId }
    /// Legacy alias used by older UI bindings.
    var nickname: String { name }

    enum CodingKeys: String, CodingKey {
        case name, phone
        case nickname
        case memberId = "member_id"
        case retailerId = "retailer_id"
        case photoUrl = "photo_url"
        case createdAt = "created_at"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        memberId = try c.decode(String.self, forKey: .memberId)
        retailerId = try c.decodeIfPresent(String.self, forKey: .retailerId)
        let decodedName = try c.decodeIfPresent(String.self, forKey: .name)
        let decodedNick = try c.decodeIfPresent(String.self, forKey: .nickname)
        name = decodedName ?? decodedNick ?? "Member"
        phone = try c.decodeIfPresent(String.self, forKey: .phone)
        photoUrl = try c.decodeIfPresent(String.self, forKey: .photoUrl)
        createdAt = try c.decodeIfPresent(String.self, forKey: .createdAt)
    }

    func encode(to encoder: Encoder) throws {
        var c = encoder.container(keyedBy: CodingKeys.self)
        try c.encode(memberId, forKey: .memberId)
        try c.encodeIfPresent(retailerId, forKey: .retailerId)
        try c.encode(name, forKey: .name)
        try c.encodeIfPresent(phone, forKey: .phone)
        try c.encodeIfPresent(photoUrl, forKey: .photoUrl)
        try c.encodeIfPresent(createdAt, forKey: .createdAt)
    }
}

struct FamilyMembersListResponse: Codable {
    let members: [FamilyMemberResponse]
    let familyWrites: String?
    let migrate: String?

    enum CodingKeys: String, CodingKey {
        case members
        case familyWrites = "family_writes"
        case migrate
    }
}

struct FamilyMigrateItem: Codable, Identifiable {
    let memberId: String
    let userId: String
    let phone: String
    let name: String
    let retailerRole: String
    let tempPassword: String?

    var id: String { userId }

    enum CodingKeys: String, CodingKey {
        case phone, name
        case memberId = "member_id"
        case userId = "user_id"
        case retailerRole = "retailer_role"
        case tempPassword = "temp_password"
    }
}

struct FamilyMigrateSkipped: Codable, Identifiable {
    let memberId: String
    let phone: String?
    let reason: String

    var id: String { memberId }

    enum CodingKeys: String, CodingKey {
        case phone, reason
        case memberId = "member_id"
    }
}

struct FamilyMigrateResult: Codable {
    let retailerId: String
    let migrated: [FamilyMigrateItem]
    let skipped: [FamilyMigrateSkipped]
    let familyRemaining: Int
    let familyWrites: String

    enum CodingKeys: String, CodingKey {
        case migrated, skipped
        case retailerId = "retailer_id"
        case familyRemaining = "family_remaining"
        case familyWrites = "family_writes"
    }
}

struct AutoOrderSkip: Codable {
    let sku: String?
    let reason: String
}

struct AutoOrderPlacedOrder: Codable, Identifiable {
    let orderId: String
    let supplierId: String?
    let lineCount: Int
    let totalMinor: Int64?
    let skus: [String]?

    var id: String { orderId }

    enum CodingKeys: String, CodingKey {
        case skus
        case orderId = "order_id"
        case supplierId = "supplier_id"
        case lineCount = "line_count"
        case totalMinor = "total_minor"
    }
}

struct AutoOrderRun: Codable, Identifiable {
    let runId: String
    let retailerId: String
    let startedAt: String
    let finishedAt: String?
    let mode: String
    let draftLines: Int
    let placedLines: Int?
    let placedOrders: [AutoOrderPlacedOrder]?
    let skipped: [AutoOrderSkip]?
    let status: String
    let message: String?
    let suggestionsSeen: Int?
    let scheduleBucket: String?
    let candidateSource: String?

    var id: String { runId }

    enum CodingKeys: String, CodingKey {
        case mode, status, message, skipped
        case runId = "run_id"
        case retailerId = "retailer_id"
        case startedAt = "started_at"
        case finishedAt = "finished_at"
        case draftLines = "draft_lines"
        case placedLines = "placed_lines"
        case placedOrders = "placed_orders"
        case suggestionsSeen = "suggestions_seen"
        case scheduleBucket = "schedule_bucket"
        case candidateSource = "candidate_source"
    }
}

struct AutoOrderRunsResponse: Codable {
    let items: [AutoOrderRun]
}

struct RetailerReorderSuggestion: Codable, Identifiable {
    let sku: String
    let suggestedQty: Int64
    let adjustedDemandPerDay: Double?
    let currentStock: Int64?
    let inFlightQty: Int64?
    let safetyStock: Double?
    let status: String?
    let sources: [String]?
    let sellThroughVelocity: Double?
    let baseDemandPerDay: Double?

    var id: String { sku }

    enum CodingKeys: String, CodingKey {
        case sku, status, sources
        case suggestedQty = "suggested_qty"
        case adjustedDemandPerDay = "adjusted_demand_per_day"
        case currentStock = "current_stock"
        case inFlightQty = "in_flight_qty"
        case safetyStock = "safety_stock"
        case sellThroughVelocity = "sell_through_velocity"
        case baseDemandPerDay = "base_demand_per_day"
    }
}

struct RetailerReorderSuggestionsResponse: Codable {
    let items: [RetailerReorderSuggestion]
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

struct OrderTimelineEntry: Codable, Identifiable {
    var id: String { transitionId }
    let transitionId: String
    let orderId: String
    let previousStatus: String?
    let newStatus: String
    let reason: String?
    let actorRole: String?
    let actorId: String?
    let eventKind: String?
    let metadata: [String: String]?
    let createdAt: String

    enum CodingKeys: String, CodingKey {
        case transitionId = "transition_id"
        case orderId = "order_id"
        case previousStatus = "previous_status"
        case newStatus = "new_status"
        case reason
        case actorRole = "actor_role"
        case actorId = "actor_id"
        case eventKind = "event_kind"
        case metadata
        case createdAt = "created_at"
    }
}

struct OrderTimelineResponse: Codable {
    let orderId: String
    let items: [OrderTimelineEntry]

    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
        case items
    }
}

struct CreditProfile: Decodable {
    let retailerId: String
    let supplierId: String
    let creditLimitMinor: Int64
    let currentBalanceMinor: Int64
    let availableCreditMinor: Int64
    let riskScore: Int64?
    let riskTier: String?
    let delinquencyCount: Int64?
    let status: String
    let version: Int64?

    enum CodingKeys: String, CodingKey {
        case retailerId = "retailer_id"
        case supplierId = "supplier_id"
        case creditLimitMinor = "credit_limit_minor"
        case currentBalanceMinor = "current_balance_minor"
        case availableCreditMinor = "available_credit_minor"
        case riskScore = "risk_score"
        case riskTier = "risk_tier"
        case delinquencyCount = "delinquency_count"
        case status
        case version
    }
}
