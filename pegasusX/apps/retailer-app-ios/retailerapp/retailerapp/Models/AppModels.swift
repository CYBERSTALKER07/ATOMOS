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
    var mustHave: Bool = false
    var allowSubstitutes: Bool = false

    var totalPriceMinor: Int64 {
        Int64(quantity) * variant.priceMinor
    }
}

// MARK: - API Generic Response

struct APIResponse<T: Codable>: Codable {
    let data: T?
    let error: String?
    let message: String?
}

// MARK: - Device Token

struct DeviceTokenPayload: Codable, Equatable {
    let token: String
    let platform: String
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
    let isOwner: Bool?
    let name: String
    let permissions: [String]
    let capabilities: [String]

    var resolvedIsOwner: Bool {
        isOwner == true || retailerRole.trimmingCharacters(in: .whitespacesAndNewlines).uppercased() == "OWNER"
    }

    enum CodingKeys: String, CodingKey {
        case name, permissions, capabilities
        case userId = "user_id"
        case retailerId = "retailer_id"
        case retailerOrgId = "retailer_org_id"
        case retailerRole = "retailer_role"
        case isOwner = "is_owner"
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
    let mustHave: Bool
    let allowSubstitutes: Bool

    enum CodingKeys: String, CodingKey {
        case cartId = "cart_id"
        case skuId = "sku_id"
        case supplierId = "supplier_id"
        case quantity
        case unitPrice = "unit_price"
        case currency
        case mustHave = "must_have"
        case allowSubstitutes = "allow_substitutes"
    }

    init(
        cartId: String?,
        skuId: String,
        supplierId: String,
        quantity: Int64,
        unitPrice: Int64,
        currency: String,
        mustHave: Bool = false,
        allowSubstitutes: Bool = false
    ) {
        self.cartId = cartId
        self.skuId = skuId
        self.supplierId = supplierId
        self.quantity = quantity
        self.unitPrice = unitPrice
        self.currency = currency
        self.mustHave = mustHave
        self.allowSubstitutes = allowSubstitutes
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        self.cartId = try c.decodeIfPresent(String.self, forKey: .cartId)
        self.skuId = try c.decode(String.self, forKey: .skuId)
        self.supplierId = try c.decode(String.self, forKey: .supplierId)
        self.quantity = try c.decode(Int64.self, forKey: .quantity)
        self.unitPrice = try c.decode(Int64.self, forKey: .unitPrice)
        self.currency = try c.decode(String.self, forKey: .currency)
        self.mustHave = try c.decodeIfPresent(Bool.self, forKey: .mustHave) ?? false
        self.allowSubstitutes = try c.decodeIfPresent(Bool.self, forKey: .allowSubstitutes) ?? false
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

struct LoyaltyTierView: Decodable {
    let enrolled: Bool
    let tier: String
    let lifetimePoints: Int64
    let availablePoints: Int64
    let nextTier: String
    let pointsToNext: Int64

    enum CodingKeys: String, CodingKey {
        case enrolled, tier
        case lifetimePoints = "lifetime_points"
        case availablePoints = "available_points"
        case nextTier = "next_tier"
        case pointsToNext = "points_to_next"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        enrolled = try c.decodeIfPresent(Bool.self, forKey: .enrolled) ?? false
        tier = try c.decodeIfPresent(String.self, forKey: .tier) ?? ""
        lifetimePoints = try c.decodeIfPresent(Int64.self, forKey: .lifetimePoints) ?? 0
        availablePoints = try c.decodeIfPresent(Int64.self, forKey: .availablePoints) ?? 0
        nextTier = try c.decodeIfPresent(String.self, forKey: .nextTier) ?? ""
        pointsToNext = try c.decodeIfPresent(Int64.self, forKey: .pointsToNext) ?? 0
    }
}

// MARK: - W2 Owner Approvals & Seller Mandates

struct RetailerApprovalItem: Codable, Identifiable, Equatable {
    let requestId: String
    let retailerId: String
    let supplierId: String
    let requestedByUser: String
    let requestedByName: String
    let clientOrderId: String?
    let amountMinor: Int64
    let trippedRule: String
    let status: String
    let payloadKind: String
    let paymentMethod: String
    let approvedByUser: String?
    let decisionReason: String?
    let expiresAt: String?
    let createdAt: String
    let updatedAt: String

    var id: String { requestId }

    enum CodingKeys: String, CodingKey {
        case status
        case requestId = "request_id"
        case retailerId = "retailer_id"
        case supplierId = "supplier_id"
        case requestedByUser = "requested_by_user"
        case requestedByName = "requested_by_name"
        case clientOrderId = "client_order_id"
        case amountMinor = "amount_minor"
        case trippedRule = "tripped_rule"
        case payloadKind = "payload_kind"
        case paymentMethod = "payment_method"
        case approvedByUser = "approved_by_user"
        case decisionReason = "decision_reason"
        case expiresAt = "expires_at"
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        requestId = try c.decodeIfPresent(String.self, forKey: .requestId) ?? ""
        retailerId = try c.decodeIfPresent(String.self, forKey: .retailerId) ?? ""
        supplierId = try c.decodeIfPresent(String.self, forKey: .supplierId) ?? ""
        requestedByUser = try c.decodeIfPresent(String.self, forKey: .requestedByUser) ?? ""
        requestedByName = try c.decodeIfPresent(String.self, forKey: .requestedByName) ?? ""
        clientOrderId = try c.decodeIfPresent(String.self, forKey: .clientOrderId)
        amountMinor = try c.decodeIfPresent(Int64.self, forKey: .amountMinor) ?? 0
        trippedRule = try c.decodeIfPresent(String.self, forKey: .trippedRule) ?? ""
        status = try c.decodeIfPresent(String.self, forKey: .status) ?? "PENDING_OWNER"
        payloadKind = try c.decodeIfPresent(String.self, forKey: .payloadKind) ?? ""
        paymentMethod = try c.decodeIfPresent(String.self, forKey: .paymentMethod) ?? ""
        approvedByUser = try c.decodeIfPresent(String.self, forKey: .approvedByUser)
        decisionReason = try c.decodeIfPresent(String.self, forKey: .decisionReason)
        expiresAt = try c.decodeIfPresent(String.self, forKey: .expiresAt)
        createdAt = try c.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
        updatedAt = try c.decodeIfPresent(String.self, forKey: .updatedAt) ?? ""
    }

    init(
        requestId: String,
        retailerId: String = "",
        supplierId: String = "",
        requestedByUser: String = "",
        requestedByName: String = "",
        clientOrderId: String? = nil,
        amountMinor: Int64 = 0,
        trippedRule: String = "",
        status: String = "PENDING_OWNER",
        payloadKind: String = "",
        paymentMethod: String = "",
        approvedByUser: String? = nil,
        decisionReason: String? = nil,
        expiresAt: String? = nil,
        createdAt: String = "",
        updatedAt: String = ""
    ) {
        self.requestId = requestId
        self.retailerId = retailerId
        self.supplierId = supplierId
        self.requestedByUser = requestedByUser
        self.requestedByName = requestedByName
        self.clientOrderId = clientOrderId
        self.amountMinor = amountMinor
        self.trippedRule = trippedRule
        self.status = status
        self.payloadKind = payloadKind
        self.paymentMethod = paymentMethod
        self.approvedByUser = approvedByUser
        self.decisionReason = decisionReason
        self.expiresAt = expiresAt
        self.createdAt = createdAt
        self.updatedAt = updatedAt
    }
}

struct RetailerApprovalsResponse: Decodable {
    let retailerId: String?
    let items: [RetailerApprovalItem]
    let nextCursor: String?

    enum CodingKeys: String, CodingKey {
        case items
        case retailerId = "retailer_id"
        case nextCursor = "next_cursor"
    }

    init(retailerId: String? = nil, items: [RetailerApprovalItem] = [], nextCursor: String? = nil) {
        self.retailerId = retailerId
        self.items = items
        self.nextCursor = nextCursor
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        retailerId = try c.decodeIfPresent(String.self, forKey: .retailerId)
        items = try c.decodeIfPresent([RetailerApprovalItem].self, forKey: .items) ?? []
        nextCursor = try c.decodeIfPresent(String.self, forKey: .nextCursor)
    }
}

struct ApprovalDecisionRequest: Encodable {
    let reason: String?
}

struct ApprovalDecisionResponse: Decodable {
    let requestId: String
    let status: String
    let orderId: String?
    let parentOrderId: String?
    let decisionReason: String?

    enum CodingKeys: String, CodingKey {
        case status
        case requestId = "request_id"
        case orderId = "order_id"
        case parentOrderId = "parent_order_id"
        case decisionReason = "decision_reason"
    }

    init(
        requestId: String = "",
        status: String = "",
        orderId: String? = nil,
        parentOrderId: String? = nil,
        decisionReason: String? = nil
    ) {
        self.requestId = requestId
        self.status = status
        self.orderId = orderId
        self.parentOrderId = parentOrderId
        self.decisionReason = decisionReason
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        requestId = try c.decodeIfPresent(String.self, forKey: .requestId) ?? ""
        status = try c.decodeIfPresent(String.self, forKey: .status) ?? ""
        orderId = try c.decodeIfPresent(String.self, forKey: .orderId)
        parentOrderId = try c.decodeIfPresent(String.self, forKey: .parentOrderId)
        decisionReason = try c.decodeIfPresent(String.self, forKey: .decisionReason)
    }
}

struct SellerMandateDTO: Codable, Equatable {
    let canReceiveGoods: Bool
    let canPayCash: Bool
    let canUseCredit: Bool
    let dailyCreditCapMinor: Int64
    let dailyCashCapMinor: Int64
    let maxSingleOrderMinor: Int64
    let updatedAt: String?
    let updatedBy: String?

    enum CodingKeys: String, CodingKey {
        case canReceiveGoods = "can_receive_goods"
        case canPayCash = "can_pay_cash"
        case canUseCredit = "can_use_credit"
        case dailyCreditCapMinor = "daily_credit_cap_minor"
        case dailyCashCapMinor = "daily_cash_cap_minor"
        case maxSingleOrderMinor = "max_single_order_minor"
        case updatedAt = "updated_at"
        case updatedBy = "updated_by"
    }

    init(
        canReceiveGoods: Bool = true,
        canPayCash: Bool = true,
        canUseCredit: Bool = true,
        dailyCreditCapMinor: Int64 = 0,
        dailyCashCapMinor: Int64 = 0,
        maxSingleOrderMinor: Int64 = 0,
        updatedAt: String? = nil,
        updatedBy: String? = nil
    ) {
        self.canReceiveGoods = canReceiveGoods
        self.canPayCash = canPayCash
        self.canUseCredit = canUseCredit
        self.dailyCreditCapMinor = dailyCreditCapMinor
        self.dailyCashCapMinor = dailyCashCapMinor
        self.maxSingleOrderMinor = maxSingleOrderMinor
        self.updatedAt = updatedAt
        self.updatedBy = updatedBy
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        canReceiveGoods = try c.decodeIfPresent(Bool.self, forKey: .canReceiveGoods) ?? true
        canPayCash = try c.decodeIfPresent(Bool.self, forKey: .canPayCash) ?? true
        canUseCredit = try c.decodeIfPresent(Bool.self, forKey: .canUseCredit) ?? true
        dailyCreditCapMinor = try c.decodeIfPresent(Int64.self, forKey: .dailyCreditCapMinor) ?? 0
        dailyCashCapMinor = try c.decodeIfPresent(Int64.self, forKey: .dailyCashCapMinor) ?? 0
        maxSingleOrderMinor = try c.decodeIfPresent(Int64.self, forKey: .maxSingleOrderMinor) ?? 0
        updatedAt = try c.decodeIfPresent(String.self, forKey: .updatedAt)
        updatedBy = try c.decodeIfPresent(String.self, forKey: .updatedBy)
    }
}

struct SellerDailyUsageDTO: Decodable, Equatable {
    let usageDate: String
    let spentCreditMinor: Int64
    let spentCashMinor: Int64
    let orderCount: Int

    enum CodingKeys: String, CodingKey {
        case usageDate = "usage_date"
        case spentCreditMinor = "spent_credit_minor"
        case spentCashMinor = "spent_cash_minor"
        case orderCount = "order_count"
    }

    init(usageDate: String = "", spentCreditMinor: Int64 = 0, spentCashMinor: Int64 = 0, orderCount: Int = 0) {
        self.usageDate = usageDate
        self.spentCreditMinor = spentCreditMinor
        self.spentCashMinor = spentCashMinor
        self.orderCount = orderCount
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        usageDate = try c.decodeIfPresent(String.self, forKey: .usageDate) ?? ""
        spentCreditMinor = try c.decodeIfPresent(Int64.self, forKey: .spentCreditMinor) ?? 0
        spentCashMinor = try c.decodeIfPresent(Int64.self, forKey: .spentCashMinor) ?? 0
        orderCount = try c.decodeIfPresent(Int.self, forKey: .orderCount) ?? 0
    }
}

struct SellerMandateResponse: Decodable {
    let userId: String
    let retailerId: String
    let retailerRole: String
    let isOwner: Bool
    let mandate: SellerMandateDTO
    let dailyUsage: SellerDailyUsageDTO

    enum CodingKeys: String, CodingKey {
        case mandate
        case userId = "user_id"
        case retailerId = "retailer_id"
        case retailerRole = "retailer_role"
        case isOwner = "is_owner"
        case dailyUsage = "daily_usage"
    }

    init(
        userId: String = "",
        retailerId: String = "",
        retailerRole: String = "",
        isOwner: Bool = false,
        mandate: SellerMandateDTO = SellerMandateDTO(),
        dailyUsage: SellerDailyUsageDTO = SellerDailyUsageDTO()
    ) {
        self.userId = userId
        self.retailerId = retailerId
        self.retailerRole = retailerRole
        self.isOwner = isOwner
        self.mandate = mandate
        self.dailyUsage = dailyUsage
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        userId = try c.decodeIfPresent(String.self, forKey: .userId) ?? ""
        retailerId = try c.decodeIfPresent(String.self, forKey: .retailerId) ?? ""
        retailerRole = try c.decodeIfPresent(String.self, forKey: .retailerRole) ?? ""
        isOwner = try c.decodeIfPresent(Bool.self, forKey: .isOwner) ?? false
        mandate = try c.decodeIfPresent(SellerMandateDTO.self, forKey: .mandate) ?? SellerMandateDTO()
        dailyUsage = try c.decodeIfPresent(SellerDailyUsageDTO.self, forKey: .dailyUsage) ?? SellerDailyUsageDTO()
    }
}

struct SellerMandatePatchBody: Encodable {
    let canReceiveGoods: Bool
    let canPayCash: Bool
    let canUseCredit: Bool
    let dailyCreditCapMinor: Int64
    let dailyCashCapMinor: Int64
    let maxSingleOrderMinor: Int64

    enum CodingKeys: String, CodingKey {
        case canReceiveGoods = "can_receive_goods"
        case canPayCash = "can_pay_cash"
        case canUseCredit = "can_use_credit"
        case dailyCreditCapMinor = "daily_credit_cap_minor"
        case dailyCashCapMinor = "daily_cash_cap_minor"
        case maxSingleOrderMinor = "max_single_order_minor"
    }
}

// MARK: - W5 Fast Home & Telegram Quick-Bind

struct FastHomeTomorrowDelivery: Codable, Equatable, Sendable {
    let orderId: String
    let supplierId: String?
    let supplierName: String?
    let status: String
    let etaText: String?
    let itemsCount: Int
    let totalMinor: Int64
    let currency: String

    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
        case supplierId = "supplier_id"
        case supplierName = "supplier_name"
        case status
        case state
        case etaText = "eta_text"
        case eta
        case deliveryDate = "delivery_date"
        case itemsCount = "items_count"
        case itemCount = "item_count"
        case totalMinor = "total_minor"
        case amountMinor = "amount_minor"
        case amount
        case currency
    }

    init(
        orderId: String,
        supplierId: String? = nil,
        supplierName: String? = nil,
        status: String,
        etaText: String? = nil,
        itemsCount: Int = 0,
        totalMinor: Int64 = 0,
        currency: String = "UZS"
    ) {
        self.orderId = orderId
        self.supplierId = supplierId
        self.supplierName = supplierName
        self.status = status
        self.etaText = etaText
        self.itemsCount = itemsCount
        self.totalMinor = totalMinor
        self.currency = currency
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        orderId = try c.decodeIfPresent(String.self, forKey: .orderId) ?? ""
        supplierId = try c.decodeIfPresent(String.self, forKey: .supplierId)
        supplierName = try c.decodeIfPresent(String.self, forKey: .supplierName)
        status = try c.decodeIfPresent(String.self, forKey: .status)
            ?? c.decodeIfPresent(String.self, forKey: .state)
            ?? "SCHEDULED"
        etaText = try c.decodeIfPresent(String.self, forKey: .etaText)
            ?? c.decodeIfPresent(String.self, forKey: .eta)
            ?? c.decodeIfPresent(String.self, forKey: .deliveryDate)
        itemsCount = try c.decodeIfPresent(Int.self, forKey: .itemsCount)
            ?? c.decodeIfPresent(Int.self, forKey: .itemCount)
            ?? 0
        totalMinor = try c.decodeIfPresent(Int64.self, forKey: .totalMinor)
            ?? c.decodeIfPresent(Int64.self, forKey: .amountMinor)
            ?? c.decodeIfPresent(Int64.self, forKey: .amount)
            ?? 0
        currency = try c.decodeIfPresent(String.self, forKey: .currency) ?? "UZS"
    }

    func encode(to encoder: Encoder) throws {
        var c = encoder.container(keyedBy: CodingKeys.self)
        try c.encode(orderId, forKey: .orderId)
        try c.encodeIfPresent(supplierId, forKey: .supplierId)
        try c.encodeIfPresent(supplierName, forKey: .supplierName)
        try c.encode(status, forKey: .status)
        try c.encodeIfPresent(etaText, forKey: .etaText)
        try c.encode(itemsCount, forKey: .itemsCount)
        try c.encode(totalMinor, forKey: .totalMinor)
        try c.encode(currency, forKey: .currency)
    }
}

struct FastHomeActivePromise: Codable, Equatable, Sendable {
    let promiseId: String?
    let promisedUntil: String?
    let amountMinor: Int64?
    let status: String?

    enum CodingKeys: String, CodingKey {
        case promiseId = "promise_id"
        case id
        case promisedUntil = "promised_until"
        case promisedDate = "promised_date"
        case dueDate = "due_date"
        case amountMinor = "amount_minor"
        case promisedAmountMinor = "promised_amount_minor"
        case status
    }

    init(
        promiseId: String? = nil,
        promisedUntil: String? = nil,
        amountMinor: Int64? = nil,
        status: String? = nil
    ) {
        self.promiseId = promiseId
        self.promisedUntil = promisedUntil
        self.amountMinor = amountMinor
        self.status = status
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        promiseId = try c.decodeIfPresent(String.self, forKey: .promiseId)
            ?? c.decodeIfPresent(String.self, forKey: .id)
        promisedUntil = try c.decodeIfPresent(String.self, forKey: .promisedUntil)
            ?? c.decodeIfPresent(String.self, forKey: .promisedDate)
            ?? c.decodeIfPresent(String.self, forKey: .dueDate)
        amountMinor = try c.decodeIfPresent(Int64.self, forKey: .amountMinor)
            ?? c.decodeIfPresent(Int64.self, forKey: .promisedAmountMinor)
        status = try c.decodeIfPresent(String.self, forKey: .status)
    }

    func encode(to encoder: Encoder) throws {
        var c = encoder.container(keyedBy: CodingKeys.self)
        try c.encodeIfPresent(promiseId, forKey: .promiseId)
        try c.encodeIfPresent(promisedUntil, forKey: .promisedUntil)
        try c.encodeIfPresent(amountMinor, forKey: .amountMinor)
        try c.encodeIfPresent(status, forKey: .status)
    }
}

struct FastHomeProposalItem: Codable, Identifiable, Equatable, Hashable, Sendable {
    var id: String { sku }
    let sku: String
    let name: String
    var quantity: Int64
    let unitPriceMinor: Int64
    let packSize: Int?

    enum CodingKeys: String, CodingKey {
        case sku
        case skuId = "sku_id"
        case name
        case skuName = "sku_name"
        case productName = "product_name"
        case quantity
        case unitPriceMinor = "unit_price_minor"
        case priceMinor = "price_minor"
        case packSize = "pack_size"
    }

    init(
        sku: String,
        name: String,
        quantity: Int64,
        unitPriceMinor: Int64,
        packSize: Int? = nil
    ) {
        self.sku = sku
        self.name = name
        self.quantity = quantity
        self.unitPriceMinor = unitPriceMinor
        self.packSize = packSize
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        let resolvedSku = try c.decodeIfPresent(String.self, forKey: .sku)
            ?? c.decodeIfPresent(String.self, forKey: .skuId)
            ?? UUID().uuidString
        sku = resolvedSku
        name = try c.decodeIfPresent(String.self, forKey: .name)
            ?? c.decodeIfPresent(String.self, forKey: .skuName)
            ?? c.decodeIfPresent(String.self, forKey: .productName)
            ?? resolvedSku
        quantity = try c.decodeIfPresent(Int64.self, forKey: .quantity) ?? 0
        unitPriceMinor = try c.decodeIfPresent(Int64.self, forKey: .unitPriceMinor)
            ?? c.decodeIfPresent(Int64.self, forKey: .priceMinor)
            ?? 0
        packSize = try c.decodeIfPresent(Int.self, forKey: .packSize)
    }

    func encode(to encoder: Encoder) throws {
        var c = encoder.container(keyedBy: CodingKeys.self)
        try c.encode(sku, forKey: .sku)
        try c.encode(name, forKey: .name)
        try c.encode(quantity, forKey: .quantity)
        try c.encode(unitPriceMinor, forKey: .unitPriceMinor)
        try c.encodeIfPresent(packSize, forKey: .packSize)
    }
}

struct FastHomePendingProposal: Codable, Equatable, Sendable {
    let orderId: String
    let supplierId: String?
    let supplierName: String?
    let orderSource: String?
    let confirmationStatus: String?
    let requestedDeliveryDate: String?
    let autoConfirmAt: String?
    let itemsCount: Int
    let totalMinor: Int64
    let currency: String
    let lineItems: [FastHomeProposalItem]

    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
        case id
        case supplierId = "supplier_id"
        case supplierName = "supplier_name"
        case orderSource = "order_source"
        case confirmationStatus = "confirmation_status"
        case requestedDeliveryDate = "requested_delivery_date"
        case autoConfirmAt = "auto_confirm_at"
        case itemsCount = "items_count"
        case totalMinor = "total_minor"
        case amountMinor = "amount_minor"
        case amount
        case currency
        case lineItems = "line_items"
        case items
    }

    init(
        orderId: String,
        supplierId: String? = nil,
        supplierName: String? = nil,
        orderSource: String? = nil,
        confirmationStatus: String? = nil,
        requestedDeliveryDate: String? = nil,
        autoConfirmAt: String? = nil,
        itemsCount: Int = 0,
        totalMinor: Int64 = 0,
        currency: String = "UZS",
        lineItems: [FastHomeProposalItem] = []
    ) {
        self.orderId = orderId
        self.supplierId = supplierId
        self.supplierName = supplierName
        self.orderSource = orderSource
        self.confirmationStatus = confirmationStatus
        self.requestedDeliveryDate = requestedDeliveryDate
        self.autoConfirmAt = autoConfirmAt
        self.itemsCount = itemsCount
        self.totalMinor = totalMinor
        self.currency = currency
        self.lineItems = lineItems
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        orderId = try c.decodeIfPresent(String.self, forKey: .orderId)
            ?? c.decodeIfPresent(String.self, forKey: .id)
            ?? ""
        supplierId = try c.decodeIfPresent(String.self, forKey: .supplierId)
        supplierName = try c.decodeIfPresent(String.self, forKey: .supplierName)
        orderSource = try c.decodeIfPresent(String.self, forKey: .orderSource)
        confirmationStatus = try c.decodeIfPresent(String.self, forKey: .confirmationStatus)
        requestedDeliveryDate = try c.decodeIfPresent(String.self, forKey: .requestedDeliveryDate)
        autoConfirmAt = try c.decodeIfPresent(String.self, forKey: .autoConfirmAt)
        let decodedItems = try c.decodeIfPresent([FastHomeProposalItem].self, forKey: .lineItems)
            ?? c.decodeIfPresent([FastHomeProposalItem].self, forKey: .items)
            ?? []
        lineItems = decodedItems
        let decodedCount = try c.decodeIfPresent(Int.self, forKey: .itemsCount)
        itemsCount = decodedCount ?? decodedItems.count
        totalMinor = try c.decodeIfPresent(Int64.self, forKey: .totalMinor)
            ?? c.decodeIfPresent(Int64.self, forKey: .amountMinor)
            ?? c.decodeIfPresent(Int64.self, forKey: .amount)
            ?? 0
        currency = try c.decodeIfPresent(String.self, forKey: .currency) ?? "UZS"
    }

    func encode(to encoder: Encoder) throws {
        var c = encoder.container(keyedBy: CodingKeys.self)
        try c.encode(orderId, forKey: .orderId)
        try c.encodeIfPresent(supplierId, forKey: .supplierId)
        try c.encodeIfPresent(supplierName, forKey: .supplierName)
        try c.encodeIfPresent(orderSource, forKey: .orderSource)
        try c.encodeIfPresent(confirmationStatus, forKey: .confirmationStatus)
        try c.encodeIfPresent(requestedDeliveryDate, forKey: .requestedDeliveryDate)
        try c.encodeIfPresent(autoConfirmAt, forKey: .autoConfirmAt)
        try c.encode(itemsCount, forKey: .itemsCount)
        try c.encode(totalMinor, forKey: .totalMinor)
        try c.encode(currency, forKey: .currency)
        try c.encode(lineItems, forKey: .lineItems)
    }
}

struct RetailerFastHomeResponse: Codable, Equatable, Sendable {
    let serverNow: String?
    let cutoffAt: String?
    let nextDeliveryDate: String?
    let tomorrowDelivery: FastHomeTomorrowDelivery?
    let balanceOwedMinor: Int64
    let overdueMinor: Int64
    let daysOverdue: Int
    let activePromise: FastHomeActivePromise?
    let creditHeadroomMinor: Int64
    let pendingProposal: FastHomePendingProposal?
    let pendingApprovalsCount: Int
    let locale: String?

    enum CodingKeys: String, CodingKey {
        case serverNow = "server_now"
        case cutoffAt = "cutoff_at"
        case nextDeliveryDate = "next_delivery_date"
        case tomorrowDelivery = "tomorrow_delivery"
        case balanceOwedMinor = "balance_owed_minor"
        case overdueMinor = "overdue_minor"
        case daysOverdue = "days_overdue"
        case activePromise = "active_promise"
        case creditHeadroomMinor = "credit_headroom_minor"
        case pendingProposal = "pending_proposal"
        case pendingApprovalsCount = "pending_approvals_count"
        case locale
    }

    init(
        serverNow: String? = nil,
        cutoffAt: String? = nil,
        nextDeliveryDate: String? = nil,
        tomorrowDelivery: FastHomeTomorrowDelivery? = nil,
        balanceOwedMinor: Int64 = 0,
        overdueMinor: Int64 = 0,
        daysOverdue: Int = 0,
        activePromise: FastHomeActivePromise? = nil,
        creditHeadroomMinor: Int64 = 0,
        pendingProposal: FastHomePendingProposal? = nil,
        pendingApprovalsCount: Int = 0,
        locale: String? = nil
    ) {
        self.serverNow = serverNow
        self.cutoffAt = cutoffAt
        self.nextDeliveryDate = nextDeliveryDate
        self.tomorrowDelivery = tomorrowDelivery
        self.balanceOwedMinor = balanceOwedMinor
        self.overdueMinor = overdueMinor
        self.daysOverdue = daysOverdue
        self.activePromise = activePromise
        self.creditHeadroomMinor = creditHeadroomMinor
        self.pendingProposal = pendingProposal
        self.pendingApprovalsCount = pendingApprovalsCount
        self.locale = locale
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        serverNow = try c.decodeIfPresent(String.self, forKey: .serverNow)
        cutoffAt = try c.decodeIfPresent(String.self, forKey: .cutoffAt)
        nextDeliveryDate = try c.decodeIfPresent(String.self, forKey: .nextDeliveryDate)
        tomorrowDelivery = try c.decodeIfPresent(FastHomeTomorrowDelivery.self, forKey: .tomorrowDelivery)
        balanceOwedMinor = try c.decodeIfPresent(Int64.self, forKey: .balanceOwedMinor) ?? 0
        overdueMinor = try c.decodeIfPresent(Int64.self, forKey: .overdueMinor) ?? 0
        daysOverdue = try c.decodeIfPresent(Int.self, forKey: .daysOverdue) ?? 0
        activePromise = try c.decodeIfPresent(FastHomeActivePromise.self, forKey: .activePromise)
        creditHeadroomMinor = try c.decodeIfPresent(Int64.self, forKey: .creditHeadroomMinor) ?? 0
        pendingProposal = try c.decodeIfPresent(FastHomePendingProposal.self, forKey: .pendingProposal)
        pendingApprovalsCount = try c.decodeIfPresent(Int.self, forKey: .pendingApprovalsCount) ?? 0
        locale = try c.decodeIfPresent(String.self, forKey: .locale)
    }
}

struct TelegramLinkStatusResponse: Codable, Equatable, Sendable {
    let linked: Bool
    let telegramChatId: String?
    let username: String?
    let botDeepLink: String?
    let linkedAt: String?
    let locale: String?

    enum CodingKeys: String, CodingKey {
        case linked
        case telegramChatId = "telegram_chat_id"
        case username
        case botDeepLink = "bot_deep_link"
        case linkedAt = "linked_at"
        case locale
    }

    init(
        linked: Bool = false,
        telegramChatId: String? = nil,
        username: String? = nil,
        botDeepLink: String? = nil,
        linkedAt: String? = nil,
        locale: String? = nil
    ) {
        self.linked = linked
        self.telegramChatId = telegramChatId
        self.username = username
        self.botDeepLink = botDeepLink
        self.linkedAt = linkedAt
        self.locale = locale
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        linked = try c.decodeIfPresent(Bool.self, forKey: .linked) ?? false
        if let strId = try? c.decodeIfPresent(String.self, forKey: .telegramChatId) {
            telegramChatId = strId
        } else if let intId = try? c.decodeIfPresent(Int64.self, forKey: .telegramChatId) {
            telegramChatId = String(intId)
        } else {
            telegramChatId = nil
        }
        username = try c.decodeIfPresent(String.self, forKey: .username)
        botDeepLink = try c.decodeIfPresent(String.self, forKey: .botDeepLink)
        linkedAt = try c.decodeIfPresent(String.self, forKey: .linkedAt)
        locale = try c.decodeIfPresent(String.self, forKey: .locale)
    }
}

struct TelegramLinkRequestResponse: Codable, Equatable, Sendable {
    let ok: Bool
    let otpCode: String
    let expiresAt: String?
    let botDeepLink: String?

    enum CodingKeys: String, CodingKey {
        case ok
        case otpCode = "otp_code"
        case expiresAt = "expires_at"
        case botDeepLink = "bot_deep_link"
    }

    init(
        ok: Bool = true,
        otpCode: String,
        expiresAt: String? = nil,
        botDeepLink: String? = nil
    ) {
        self.ok = ok
        self.otpCode = otpCode
        self.expiresAt = expiresAt
        self.botDeepLink = botDeepLink
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        otpCode = try c.decodeIfPresent(String.self, forKey: .otpCode) ?? ""
        ok = try c.decodeIfPresent(Bool.self, forKey: .ok) ?? !otpCode.isEmpty
        expiresAt = try c.decodeIfPresent(String.self, forKey: .expiresAt)
        botDeepLink = try c.decodeIfPresent(String.self, forKey: .botDeepLink)
    }
}

struct CutoffCountdownState: Equatable, Sendable {
    let totalSecondsRemaining: Int
    let formattedHMS: String
    let isExpired: Bool
}

enum FastHomeContractHelper {
    private static let isoFractionalFormatter: ISO8601DateFormatter = {
        let f = ISO8601DateFormatter()
        f.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return f
    }()

    private static let isoBasicFormatter: ISO8601DateFormatter = {
        let f = ISO8601DateFormatter()
        f.formatOptions = [.withInternetDateTime]
        return f
    }()

    private static let dayOnlyFormatter: DateFormatter = {
        let f = DateFormatter()
        f.calendar = Calendar(identifier: .gregorian)
        f.timeZone = TimeZone(secondsFromGMT: 5 * 3600) ?? .current
        f.locale = Locale(identifier: "en_US_POSIX")
        f.dateFormat = "yyyy-MM-dd"
        return f
    }()

    static func parseISO8601Date(_ raw: String?) -> Date? {
        guard let trimmed = raw?.trimmingCharacters(in: .whitespacesAndNewlines), !trimmed.isEmpty else {
            return nil
        }
        if let fractional = isoFractionalFormatter.date(from: trimmed) {
            return fractional
        }
        if let basic = isoBasicFormatter.date(from: trimmed) {
            return basic
        }
        if trimmed.count == 10 {
            return dayOnlyFormatter.date(from: trimmed)
        }
        return nil
    }

    static func formatISO8601Date(_ date: Date) -> String {
        isoBasicFormatter.string(from: date)
    }

    static func computeServerClockOffset(serverNowISO: String?, deviceNow: Date = Date()) -> TimeInterval {
        guard let serverDate = parseISO8601Date(serverNowISO) else { return 0 }
        return serverDate.timeIntervalSince(deviceNow)
    }

    static func computeCutoffCountdown(
        cutoffAtISO: String?,
        serverClockOffset: TimeInterval = 0,
        now: Date = Date()
    ) -> CutoffCountdownState? {
        guard let cutoffDate = parseISO8601Date(cutoffAtISO) else { return nil }
        let effectiveServerNow = now.addingTimeInterval(serverClockOffset)
        let remainingRaw = Int(floor(cutoffDate.timeIntervalSince(effectiveServerNow)))
        if remainingRaw <= 0 {
            return CutoffCountdownState(
                totalSecondsRemaining: 0,
                formattedHMS: "00:00:00",
                isExpired: true
            )
        }
        let hours = remainingRaw / 3600
        let minutes = (remainingRaw % 3600) / 60
        let seconds = remainingRaw % 60
        return CutoffCountdownState(
            totalSecondsRemaining: remainingRaw,
            formattedHMS: String(format: "%02d:%02d:%02d", hours, minutes, seconds),
            isExpired: false
        )
    }

    static func resolveDefaultWorkspaceMode(retailerRole: String?, isOwner: Bool? = nil) -> RetailerWorkspaceMode {
        if isOwner == true {
            return .full
        }
        let normalized = (retailerRole ?? "")
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .uppercased()
        return normalized == "OWNER" ? .full : .fast
    }

    static func resolveEffectiveWorkspaceMode(
        persistedOverride: RetailerWorkspaceMode?,
        retailerRole: String?,
        isOwner: Bool? = nil
    ) -> RetailerWorkspaceMode {
        persistedOverride ?? resolveDefaultWorkspaceMode(retailerRole: retailerRole, isOwner: isOwner)
    }

    static func buildFastHomeFallback(
        orders: [Order],
        predictions: [RetailerAIPrediction],
        creditProfile: CreditProfile?,
        creditRelationships: [CreditRelationshipWire] = [],
        pendingApprovalsCount: Int = 0,
        cutoffAtISO: String? = nil,
        locale: String? = nil,
        now: Date = Date()
    ) -> RetailerFastHomeResponse {
        let activeCandidates = orders.filter { order in
            order.status != .cancelled
                && order.status != .completed
                && order.status != .quarantine
                && (order.status.isActive || order.status == .pending || order.status == .pendingReview || order.status == .scheduled)
        }

        let sortedDeliveryCandidates = activeCandidates.sorted { lhs, rhs in
            let lKey = lhs.estimatedDelivery ?? lhs.deliverBefore ?? lhs.proposedDeliveryDate ?? lhs.createdAt
            let rKey = rhs.estimatedDelivery ?? rhs.deliverBefore ?? rhs.proposedDeliveryDate ?? rhs.createdAt
            return lKey < rKey
        }

        let tomorrowDelivery: FastHomeTomorrowDelivery? = sortedDeliveryCandidates.first.map { order in
            FastHomeTomorrowDelivery(
                orderId: order.id,
                supplierId: order.supplierId,
                supplierName: order.supplierName,
                status: order.status.rawValue,
                etaText: order.estimatedDelivery ?? order.deliverBefore ?? order.proposedDeliveryDate,
                itemsCount: order.items.count,
                totalMinor: order.totalAmount,
                currency: order.currency.isEmpty ? "UZS" : order.currency
            )
        }

        let relationshipBalance = creditRelationships.reduce(Int64(0)) { $0 + max(0, $1.currentBalanceMinor) }
        let relationshipHeadroom = creditRelationships.reduce(Int64(0)) { $0 + max(0, $1.availableCreditMinor) }
        let balanceOwedMinor: Int64 = creditRelationships.isEmpty
            ? max(0, creditProfile?.currentBalanceMinor ?? 0)
            : relationshipBalance
        let creditHeadroomMinor: Int64 = creditRelationships.isEmpty
            ? max(0, creditProfile?.availableCreditMinor ?? 0)
            : relationshipHeadroom
        let daysOverdue = Int(max(0, creditProfile?.delinquencyCount ?? 0))
        let overdueMinor: Int64 = daysOverdue > 0 ? balanceOwedMinor : 0

        let actionablePrediction = predictions.first { prediction in
            let st = (prediction.confirmationStatus ?? "").trimmingCharacters(in: .whitespacesAndNewlines).uppercased()
            return st.isEmpty || st == "PENDING" || st == "PENDING_REVIEW" || st == "PROPOSED"
        }

        let pendingProposal: FastHomePendingProposal?
        if let prediction = actionablePrediction {
            let items = (prediction.lineItems ?? []).map { item in
                FastHomeProposalItem(
                    sku: item.sku,
                    name: (item.name?.isEmpty == false ? item.name! : item.sku),
                    quantity: item.quantity,
                    unitPriceMinor: item.unitPriceMinor ?? 0,
                    packSize: nil
                )
            }
            pendingProposal = FastHomePendingProposal(
                orderId: prediction.orderId,
                supplierId: prediction.supplierId,
                supplierName: nil,
                orderSource: prediction.orderSource ?? "AI",
                confirmationStatus: prediction.confirmationStatus ?? "PENDING",
                requestedDeliveryDate: prediction.requestedDeliveryDate,
                autoConfirmAt: prediction.autoConfirmAt,
                itemsCount: items.count,
                totalMinor: prediction.totalMinor,
                currency: prediction.currency.isEmpty ? "UZS" : prediction.currency,
                lineItems: items
            )
        } else if let pendingOrder = orders.first(where: {
            $0.status == .pending
                || $0.status == .pendingReview
                || $0.status == .scheduled
                || ($0.confirmationStatus ?? "").uppercased() == "PENDING"
        }) {
            let items = pendingOrder.items.map { item in
                FastHomeProposalItem(
                    sku: item.productId.isEmpty ? item.variantId : item.productId,
                    name: item.productName,
                    quantity: Int64(item.quantity),
                    unitPriceMinor: item.unitPrice,
                    packSize: nil
                )
            }
            pendingProposal = FastHomePendingProposal(
                orderId: pendingOrder.id,
                supplierId: pendingOrder.supplierId,
                supplierName: pendingOrder.supplierName,
                orderSource: pendingOrder.orderSource ?? "RETAILER",
                confirmationStatus: pendingOrder.confirmationStatus ?? pendingOrder.status.rawValue,
                requestedDeliveryDate: pendingOrder.deliverBefore ?? pendingOrder.proposedDeliveryDate ?? pendingOrder.estimatedDelivery,
                autoConfirmAt: pendingOrder.autoConfirmAt,
                itemsCount: items.count,
                totalMinor: pendingOrder.totalAmount,
                currency: pendingOrder.currency.isEmpty ? "UZS" : pendingOrder.currency,
                lineItems: items
            )
        } else {
            pendingProposal = nil
        }

        return RetailerFastHomeResponse(
            serverNow: formatISO8601Date(now),
            cutoffAt: cutoffAtISO ?? pendingProposal?.autoConfirmAt,
            nextDeliveryDate: tomorrowDelivery?.etaText,
            tomorrowDelivery: tomorrowDelivery,
            balanceOwedMinor: balanceOwedMinor,
            overdueMinor: overdueMinor,
            daysOverdue: daysOverdue,
            activePromise: nil,
            creditHeadroomMinor: creditHeadroomMinor,
            pendingProposal: pendingProposal,
            pendingApprovalsCount: pendingApprovalsCount,
            locale: locale
        )
    }
}
