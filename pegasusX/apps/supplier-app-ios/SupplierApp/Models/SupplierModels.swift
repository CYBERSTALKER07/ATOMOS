import Foundation

// MARK: - Auth

struct LoginRequest: Encodable {
    let phone: String
    let password: String
}

struct LoginResponse: Decodable {
    let supplierId: String
    let isRegistered: Bool
    let isConfigured: Bool
    let nextStep: String
    let token: String?
    let refreshToken: String?

    enum CodingKeys: String, CodingKey {
        case supplierId = "supplier_id"
        case isRegistered = "is_registered"
        case isConfigured = "is_configured"
        case nextStep = "next_step"
        case token
        case refreshToken = "refresh_token"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        supplierId = try container.decode(String.self, forKey: .supplierId)
        isRegistered = try container.decodeIfPresent(Bool.self, forKey: .isRegistered) ?? false
        isConfigured = try container.decodeIfPresent(Bool.self, forKey: .isConfigured) ?? false
        nextStep = try container.decodeIfPresent(String.self, forKey: .nextStep) ?? ""
        token = try container.decodeIfPresent(String.self, forKey: .token)
        refreshToken = try container.decodeIfPresent(String.self, forKey: .refreshToken)
    }
}

struct DeviceTokenRequest: Encodable, Equatable {
    let token: String
    let platform: String
}

struct RegisterResponse: Decodable {
    let supplierId: String
    let legalName: String
    let isRegistered: Bool
    let isConfigured: Bool
    let nextStep: String
    let token: String?

    enum CodingKeys: String, CodingKey {
        case supplierId = "supplier_id"
        case legalName = "legal_name"
        case isRegistered = "is_registered"
        case isConfigured = "is_configured"
        case nextStep = "next_step"
        case token
    }
}

struct RefreshTokenRequest: Encodable {
    let refreshToken: String

    enum CodingKeys: String, CodingKey {
        case refreshToken = "refresh_token"
    }
}

// MARK: - Dashboard

struct SupplierDashboard: Decodable {
    let supplierId: String
    let isConfigured: Bool
    let inventorySKUs: Int
    let pendingOrders: Int
    let updatedAt: String
    let ordersByStatus: [String: Int]
    let todayRevenueMinor: Int64
    let deliveriesCompletedToday: Int
    let deliveriesAttemptedToday: Int
    let manifestsByState: [String: Int]

    enum CodingKeys: String, CodingKey {
        case supplierId = "supplier_id"
        case isConfigured = "is_configured"
        case inventorySKUs = "inventory_skus"
        case pendingOrders = "pending_orders"
        case updatedAt = "updated_at"
        case ordersByStatus = "orders_by_status"
        case todayRevenueMinor = "today_revenue_minor"
        case deliveriesCompletedToday = "deliveries_completed_today"
        case deliveriesAttemptedToday = "deliveries_attempted_today"
        case manifestsByState = "manifests_by_state"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        supplierId = try c.decode(String.self, forKey: .supplierId)
        isConfigured = try c.decode(Bool.self, forKey: .isConfigured)
        inventorySKUs = (try? c.decode(Int.self, forKey: .inventorySKUs)) ?? 0
        pendingOrders = (try? c.decode(Int.self, forKey: .pendingOrders)) ?? 0
        updatedAt = (try? c.decode(String.self, forKey: .updatedAt)) ?? ""
        ordersByStatus = (try? c.decode([String: Int].self, forKey: .ordersByStatus)) ?? [:]
        todayRevenueMinor = (try? c.decode(Int64.self, forKey: .todayRevenueMinor)) ?? 0
        deliveriesCompletedToday = (try? c.decode(Int.self, forKey: .deliveriesCompletedToday)) ?? 0
        deliveriesAttemptedToday = (try? c.decode(Int.self, forKey: .deliveriesAttemptedToday)) ?? 0
        manifestsByState = (try? c.decode([String: Int].self, forKey: .manifestsByState)) ?? [:]
    }
}

// MARK: - Profile

struct SupplierProfile: Decodable {
    let supplierId: String
    let legalName: String
    let contactName: String
    let email: String
    let phone: String
    let country: String
    let currency: String
    let categories: [String]
    let isRegistered: Bool
    let isConfigured: Bool
    let selectedGateways: [String]
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case supplierId = "supplier_id"
        case legalName = "legal_name"
        case contactName = "contact_name"
        case email, phone, country, currency, categories
        case isRegistered = "is_registered"
        case isConfigured = "is_configured"
        case selectedGateways = "selected_gateways"
        case updatedAt = "updated_at"
    }
}

// MARK: - Orders

struct SupplierOrdersResponse: Decodable {
    let orders: [SupplierOrder]
    let total: Int?
    let limit: Int?
    let offset: Int?
}

struct SupplierOrder: Decodable, Identifiable, Hashable {
    var id: String { orderId }
    let orderId: String
    let retailerId: String
    let warehouseId: String?
    let status: String
    let decision: String?
    let note: String?
    let totalMinor: Int64
    let currency: String
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
        case retailerId = "retailer_id"
        case warehouseId = "warehouse_id"
        case status, decision, note
        case totalMinor = "total_minor"
        case currency
        case updatedAt = "updated_at"
    }
}

struct WarehouseOrderLineItem: Decodable, Identifiable, Hashable {
    var id: String { productId ?? productName ?? UUID().uuidString }
    let productId: String?
    let productName: String?
    let quantity: Double?
    let unitPrice: Int64?

    enum CodingKeys: String, CodingKey {
        case productId = "product_id"
        case productName = "product_name"
        case quantity
        case unitPrice = "unit_price"
    }
}

struct WarehouseOrderDetail: Decodable {
    let orderId: String
    let retailerName: String?
    let state: String?
    let status: String?
    let totalUzs: Int64?
    let totalMinor: Int64?
    let lineItems: [WarehouseOrderLineItem]

    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
        case retailerName = "retailer_name"
        case state, status
        case totalUzs = "total_uzs"
        case totalMinor = "total_minor"
        case lineItems = "line_items"
    }
}

struct WarehouseProposeDeliveryRequest: Encodable {
    let proposedDeliveryDate: String
    let reason: String

    enum CodingKeys: String, CodingKey {
        case proposedDeliveryDate = "proposed_delivery_date"
        case reason
    }
}

struct WarehouseOrderMutationRequest: Encodable {
    let reason: String?
}

struct WarehouseOrderMutationResponse: Decodable {
    let status: String?
}

struct SupplierReturnRow: Decodable, Identifiable, Hashable {
    var id: String { returnId }
    let returnId: String
    let orderId: String
    let skuId: String
    let productName: String
    let quantity: Int64
    let unitPrice: Int64
    let status: String
    let physicalStatus: String
    let receivedQty: Int64
    let reason: String
    let driverName: String
    let createdAt: String

    enum CodingKeys: String, CodingKey {
        case returnId = "return_id"
        case orderId = "order_id"
        case skuId = "sku_id"
        case productName = "product_name"
        case quantity
        case unitPrice = "unit_price"
        case status
        case physicalStatus = "physical_status"
        case receivedQty = "received_qty"
        case reason
        case driverName = "driver_name"
        case createdAt = "created_at"
    }
}

struct SupplierReturnsResponse: Decodable {
    let data: [SupplierReturnRow]
}

struct ResolveReturnRequest: Encodable {
    let returnId: String
    let lineItemId: String
    let resolution: String
    let notes: String

    enum CodingKeys: String, CodingKey {
        case returnId = "return_id"
        case lineItemId = "line_item_id"
        case resolution
        case notes
    }
}

// MARK: - Fleet

struct FleetDriversResponse: Decodable {
    let supplierId: String
    let items: [FleetDriver]
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case supplierId = "supplier_id"
        case items
        case updatedAt = "updated_at"
    }
}

struct FleetDriver: Decodable, Identifiable {
    var id: String { driverId }
    let driverId: String
    let name: String
    let phone: String
    let homeNodeType: String
    let homeNodeId: String
    let isActive: Bool

    enum CodingKeys: String, CodingKey {
        case driverId = "driver_id"
        case name, phone
        case homeNodeType = "home_node_type"
        case homeNodeId = "home_node_id"
        case isActive = "is_active"
    }
}

struct FleetVehiclesResponse: Decodable {
    let supplierId: String
    let items: [FleetVehicle]
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case supplierId = "supplier_id"
        case items
        case updatedAt = "updated_at"
    }
}

struct FleetVehicle: Decodable, Identifiable {
    var id: String { vehicleId }
    let vehicleId: String
    let label: String?
    let licensePlate: String
    let homeNodeType: String
    let homeNodeId: String
    let isActive: Bool

    enum CodingKeys: String, CodingKey {
        case vehicleId = "vehicle_id"
        case label
        case licensePlate = "license_plate"
        case homeNodeType = "home_node_type"
        case homeNodeId = "home_node_id"
        case isActive = "is_active"
    }
}

// MARK: - Inventory

struct InventoryListResponse: Decodable {
    let items: [InventoryItem]
}

struct InventoryItem: Decodable, Identifiable {
    var id: String { sku }
    let sku: String
    let productName: String
    let quantity: Int64
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case sku
        case productName = "product_name"
        case quantity
        case updatedAt = "updated_at"
    }
}

struct InventoryPatchRequest: Encodable {
    let skuId: String?
    let sku: String?
    let quantityDelta: Int64?
    let quantity: Int64?
    let reason: String?

    enum CodingKeys: String, CodingKey {
        case skuId = "sku_id"
        case sku
        case quantityDelta = "quantity_delta"
        case quantity
        case reason
    }
}

// MARK: - Catalog

struct CatalogProduct: Decodable, Identifiable {
    let productId: String
    let name: String
    let categoryId: String
    let priceMinor: Int64
    let currency: String
    let unit: String
    let unitVolumeVu: Double
    let imageUrl: String?
    let barcode: String?
    let isActive: Bool
    let version: Int64

    var id: String { productId }

    enum CodingKeys: String, CodingKey {
        case productId = "product_id"
        case name
        case categoryId = "category_id"
        case priceMinor = "price_minor"
        case currency
        case unit
        case unitVolumeVu = "unit_volume_vu"
        case imageUrl = "image_url"
        case barcode
        case isActive = "is_active"
        case version
    }
}

struct CatalogCategory: Decodable, Identifiable {
    let categoryId: String
    let name: String

    var id: String { categoryId }

    enum CodingKeys: String, CodingKey {
        case categoryId = "category_id"
        case name
    }
}

struct CatalogUploadTicket: Decodable {
    let uploadUrl: String
    let imageUrl: String

    enum CodingKeys: String, CodingKey {
        case uploadUrl = "upload_url"
        case imageUrl = "image_url"
    }
}

struct CatalogProductCreateRequest: Encodable {
    let categoryId: String
    let name: String
    let description: String
    let priceMinor: Int64
    let currency: String
    let unitVolumeVu: Double
    let stockQuantity: Int64
    let unit: String
    let imageUrl: String?
    let barcode: String?

    enum CodingKeys: String, CodingKey {
        case categoryId = "category_id"
        case name
        case description
        case priceMinor = "price_minor"
        case currency
        case unitVolumeVu = "unit_volume_vu"
        case stockQuantity = "stock_quantity"
        case unit
        case imageUrl = "image_url"
        case barcode
    }
}

struct CatalogProductUpdateRequest: Encodable {
    let name: String
    let priceMinor: Int64
    let currency: String
    let unit: String
    let unitVolumeVu: Double
    let imageUrl: String?
    let barcode: String?
    let isActive: Bool
    let version: Int64

    enum CodingKeys: String, CodingKey {
        case name
        case priceMinor = "price_minor"
        case currency
        case unit
        case unitVolumeVu = "unit_volume_vu"
        case imageUrl = "image_url"
        case barcode
        case isActive = "is_active"
        case version
    }
}

// MARK: - Earnings

struct SupplierEarnings: Decodable {
    let currency: String
    let todayMinor: Int64
    let weekMinor: Int64
    let monthMinor: Int64
    let authoritative: Bool
    let updatedAt: String?

    enum CodingKeys: String, CodingKey {
        case currency
        case todayMinor = "today_minor"
        case weekMinor = "week_minor"
        case monthMinor = "month_minor"
        case authoritative
        case updatedAt = "updated_at"
    }
}

// MARK: - Promotions

struct SupplierPromotion: Decodable, Identifiable {
    let promotionId: String
    let supplierId: String
    let name: String
    let description: String?
    let discountBps: Int64
    let scopeType: String
    let scopeProductId: String?
    let retailerScope: String
    let isActive: Bool
    let priority: Int64

    var id: String { promotionId }

    enum CodingKeys: String, CodingKey {
        case promotionId = "promotion_id"
        case supplierId = "supplier_id"
        case name
        case description
        case discountBps = "discount_bps"
        case scopeType = "scope_type"
        case scopeProductId = "scope_product_id"
        case retailerScope = "retailer_scope"
        case isActive = "is_active"
        case priority
    }
}

struct SupplierPromotionsResponse: Decodable {
    let promotions: [SupplierPromotion]
}

struct SupplierPromotionUpsertRequest: Encodable {
    let name: String
    let description: String
    let discountBps: Int64
    let scopeType: String
    let retailerScope: String
    let scopeProductId: String?

    enum CodingKeys: String, CodingKey {
        case name
        case description
        case discountBps = "discount_bps"
        case scopeType = "scope_type"
        case retailerScope = "retailer_scope"
        case scopeProductId = "scope_product_id"
    }
}

// MARK: - Billing

struct BillingSetupRequest: Encodable {
    let bankName: String
    let accountHolder: String
    let accountNumber: String
    let swiftBic: String
    let iban: String?
    let selectedGateways: [String]
}

struct BillingSetupResponse: Decodable {
    let supplierId: String
    let isConfigured: Bool
    let selectedGateways: [String]

    enum CodingKeys: String, CodingKey {
        case supplierId = "supplier_id"
        case isConfigured = "is_configured"
        case selectedGateways = "selected_gateways"
    }
}

// MARK: - Reassignment

struct DriverRecommendation: Decodable, Identifiable {
    let driverId: String
    let driverName: String
    let currentLat: Double
    let currentLon: Double
    let distanceKm: Double
    let score: Double
    let vehicleClass: String
    let licensePlate: String

    var id: String { driverId }

    enum CodingKeys: String, CodingKey {
        case driverId = "driver_id"
        case driverName = "driver_name"
        case currentLat = "current_lat"
        case currentLon = "current_lon"
        case distanceKm = "distance_km"
        case score
        case vehicleClass = "vehicle_class"
        case licensePlate = "license_plate"
    }
}

struct RecommendReassignResponse: Decodable {
    let orderId: String
    let retailerName: String
    let orderVolumeVu: Double
    let recommendations: [DriverRecommendation]

    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
        case retailerName = "retailer_name"
        case orderVolumeVu = "order_volume_vu"
        case recommendations
    }
}

struct ApplyReassignRequest: Encodable {
    let driverId: String
    let isPartial: Bool

    enum CodingKeys: String, CodingKey {
        case driverId = "driver_id"
        case isPartial = "is_partial"
    }
}
