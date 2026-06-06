import Foundation

// MARK: - Auth

struct LoginRequest: Encodable {
    let phone: String
    let password: String
}

struct LoginResponse: Decodable {
    let supplierId: String
    let isConfigured: Bool
    let nextStep: String
    let token: String?
    let refreshToken: String?

    enum CodingKeys: String, CodingKey {
        case supplierId = "supplier_id"
        case isConfigured = "is_configured"
        case nextStep = "next_step"
        case token
        case refreshToken = "refresh_token"
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

    enum CodingKeys: String, CodingKey {
        case supplierId = "supplier_id"
        case isConfigured = "is_configured"
        case inventorySKUs = "inventory_skus"
        case pendingOrders = "pending_orders"
        case updatedAt = "updated_at"
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
    let status: String
    let decision: String?
    let totalMinor: Int64
    let currency: String
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
        case retailerId = "retailer_id"
        case status, decision
        case totalMinor = "total_minor"
        case currency
        case updatedAt = "updated_at"
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
