import Foundation

// MARK: - Auth
struct LoginRequest: Encodable {
    let phone: String
    let password: String
}

struct AuthResponse: Decodable {
    let token: String
    let refreshToken: String
    let factoryId: String
    let factoryName: String

    enum CodingKeys: String, CodingKey {
        case token
        case refreshToken = "refresh_token"
        case factoryId = "factory_id"
        case factoryName = "factory_name"
    }
}

// MARK: - Dashboard
struct DashboardStats: Decodable {
    let pendingTransfers: Int
    let loadingTransfers: Int
    let activeManifests: Int
    let dispatchedToday: Int
    let vehiclesTotal: Int
    let vehiclesAvailable: Int
    let staffOnShift: Int
    let criticalInsights: Int

    enum CodingKeys: String, CodingKey {
        case pendingTransfers = "pending_transfers"
        case loadingTransfers = "loading_transfers"
        case activeManifests = "active_manifests"
        case dispatchedToday = "dispatched_today"
        case vehiclesTotal = "vehicles_total"
        case vehiclesAvailable = "vehicles_available"
        case staffOnShift = "staff_on_shift"
        case criticalInsights = "critical_insights"
    }

    static let empty = DashboardStats(
        pendingTransfers: 0, loadingTransfers: 0, activeManifests: 0,
        dispatchedToday: 0, vehiclesTotal: 0, vehiclesAvailable: 0,
        staffOnShift: 0, criticalInsights: 0
    )
}

// MARK: - Transfer
struct Transfer: Decodable, Identifiable {
    let id: String
    let factoryId: String
    let warehouseId: String
    let warehouseName: String
    let state: String
    let priority: String
    let totalItems: Int
    let totalVolumeL: Double
    let notes: String
    let createdAt: String
    let updatedAt: String
    let items: [TransferItem]

    enum CodingKeys: String, CodingKey {
        case id
        case factoryId = "factory_id"
        case warehouseId = "warehouse_id"
        case warehouseName = "warehouse_name"
        case state, priority
        case totalItems = "total_items"
        case totalVolumeL = "total_volume_l"
        case notes
        case createdAt = "created_at"
        case updatedAt = "updated_at"
        case items
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        id = try c.decode(String.self, forKey: .id)
        factoryId = try c.decodeIfPresent(String.self, forKey: .factoryId) ?? ""
        warehouseId = try c.decodeIfPresent(String.self, forKey: .warehouseId) ?? ""
        warehouseName = try c.decodeIfPresent(String.self, forKey: .warehouseName) ?? ""
        state = try c.decodeIfPresent(String.self, forKey: .state) ?? ""
        priority = try c.decodeIfPresent(String.self, forKey: .priority) ?? ""
        totalItems = try c.decodeIfPresent(Int.self, forKey: .totalItems) ?? 0
        totalVolumeL = try c.decodeIfPresent(Double.self, forKey: .totalVolumeL) ?? 0
        notes = try c.decodeIfPresent(String.self, forKey: .notes) ?? ""
        createdAt = try c.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
        updatedAt = try c.decodeIfPresent(String.self, forKey: .updatedAt) ?? ""
        items = try c.decodeIfPresent([TransferItem].self, forKey: .items) ?? []
    }
}

struct TransferItem: Decodable, Identifiable {
    let id: String
    let productId: String
    let productName: String
    let quantity: Int
    let quantityAvailable: Int
    let unitVolumeL: Double

    enum CodingKeys: String, CodingKey {
        case id
        case productId = "product_id"
        case productName = "product_name"
        case quantity
        case quantityAvailable = "quantity_available"
        case unitVolumeL = "unit_volume_l"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        id = try c.decode(String.self, forKey: .id)
        productId = try c.decodeIfPresent(String.self, forKey: .productId) ?? ""
        productName = try c.decodeIfPresent(String.self, forKey: .productName) ?? ""
        quantity = try c.decodeIfPresent(Int.self, forKey: .quantity) ?? 0
        quantityAvailable = try c.decodeIfPresent(Int.self, forKey: .quantityAvailable) ?? 0
        unitVolumeL = try c.decodeIfPresent(Double.self, forKey: .unitVolumeL) ?? 0
    }
}

struct TransferListResponse: Decodable {
    let transfers: [Transfer]
    let total: Int
}

struct TransitionRequest: Encodable {
    let targetState: String

    enum CodingKeys: String, CodingKey {
        case targetState = "target_state"
    }
}

// MARK: - Supply Requests
struct SupplyRequest: Decodable, Identifiable {
    let id: String
    let warehouseId: String
    let factoryId: String
    let supplierId: String
    let state: String
    let priority: String
    let requestedDeliveryDate: String?
    let totalVolumeVU: Double
    let notes: String
    let transferOrderId: String
    let createdBy: String
    let createdAt: String
    let updatedAt: String?

    enum CodingKeys: String, CodingKey {
        case id = "request_id"
        case warehouseId = "warehouse_id"
        case factoryId = "factory_id"
        case supplierId = "supplier_id"
        case state
        case priority
        case requestedDeliveryDate = "requested_delivery_date"
        case totalVolumeVU = "total_volume_vu"
        case notes
        case transferOrderId = "transfer_order_id"
        case createdBy = "created_by"
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }
}

struct SupplyRequestTransitionRequest: Encodable {
    let action: String
    let transferOrderId: String?

    enum CodingKeys: String, CodingKey {
        case action
        case transferOrderId = "transfer_order_id"
    }
}

struct SupplyRequestTransitionResponse: Decodable {
    let requestId: String
    let state: String

    enum CodingKeys: String, CodingKey {
        case requestId = "request_id"
        case state
    }
}

// MARK: - Manifests / Override
struct Manifest: Decodable, Identifiable {
    let id: String
    let factoryId: String
    let driverId: String
    let driverName: String
    let vehicleId: String
    let vehicleLabel: String
    let truckId: String
    let truckPlate: String
    let state: String
    let status: String
    let totalVolumeVU: Double
    let maxVolumeVU: Double
    let maxCapacityVU: Double
    let stopCount: Int
    let regionCode: String
    let createdAt: String
    let transfers: [ManifestTransfer]

    enum CodingKeys: String, CodingKey {
        case id = "manifest_id"
        case factoryId = "factory_id"
        case driverId = "driver_id"
        case driverName = "driver_name"
        case vehicleId = "vehicle_id"
        case vehicleLabel = "vehicle_label"
        case truckId = "truck_id"
        case truckPlate = "truck_plate"
        case state
        case status
        case totalVolumeVU = "total_volume_vu"
        case maxVolumeVU = "max_volume_vu"
        case maxCapacityVU = "max_capacity_vu"
        case stopCount = "stop_count"
        case regionCode = "region_code"
        case createdAt = "created_at"
        case transfers
    }
}

struct ManifestTransfer: Decodable, Identifiable {
    let id: String
    let productName: String
    let quantity: Int
    let volumeVU: Double
    let state: String

    enum CodingKeys: String, CodingKey {
        case id = "transfer_id"
        case productName = "product_name"
        case quantity
        case volumeVU = "volume_vu"
        case state
    }
}

struct ManifestListResponse: Decodable {
    let manifests: [Manifest]
    let total: Int
}

struct ManifestRebalanceRequest: Encodable {
    let sourceManifestId: String
    let targetManifestId: String
    let transferIds: [String]

    enum CodingKeys: String, CodingKey {
        case sourceManifestId = "source_manifest_id"
        case targetManifestId = "target_manifest_id"
        case transferIds = "transfer_ids"
    }
}

struct ManifestRebalanceResponse: Decodable {
    let sourceManifestId: String
    let targetManifestId: String
    let transfersMoved: Int
    let volumeMovedVU: Double
    let reason: String

    enum CodingKeys: String, CodingKey {
        case sourceManifestId = "source_manifest_id"
        case targetManifestId = "target_manifest_id"
        case transfersMoved = "transfers_moved"
        case volumeMovedVU = "volume_moved_vu"
        case reason
    }
}

struct ManifestCancelTransferRequest: Encodable {
    let manifestId: String
    let transferId: String

    enum CodingKeys: String, CodingKey {
        case manifestId = "manifest_id"
        case transferId = "transfer_id"
    }
}

struct ManifestCancelTransferResponse: Decodable {
    let manifestId: String
    let transferId: String
    let status: String

    enum CodingKeys: String, CodingKey {
        case manifestId = "manifest_id"
        case transferId = "transfer_id"
        case status
    }
}

struct ManifestCancelRequest: Encodable {
    let manifestId: String

    enum CodingKeys: String, CodingKey {
        case manifestId = "manifest_id"
    }
}

struct ManifestCancelResponse: Decodable {
    let manifestId: String
    let status: String
    let transfersReleased: Int

    enum CodingKeys: String, CodingKey {
        case manifestId = "manifest_id"
        case status
        case transfersReleased = "transfers_released"
    }
}

// MARK: - Vehicle
struct Vehicle: Decodable, Identifiable {
    let id: String
    let plateNumber: String
    let driverName: String
    let status: String
    let capacityKg: Double
    let capacityL: Double
    let currentRoute: String

    enum CodingKeys: String, CodingKey {
        case id
        case plateNumber = "plate_number"
        case driverName = "driver_name"
        case status
        case capacityKg = "capacity_kg"
        case capacityL = "capacity_l"
        case currentRoute = "current_route"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        id = try c.decode(String.self, forKey: .id)
        plateNumber = try c.decodeIfPresent(String.self, forKey: .plateNumber) ?? ""
        driverName = try c.decodeIfPresent(String.self, forKey: .driverName) ?? ""
        status = try c.decodeIfPresent(String.self, forKey: .status) ?? ""
        capacityKg = try c.decodeIfPresent(Double.self, forKey: .capacityKg) ?? 0
        capacityL = try c.decodeIfPresent(Double.self, forKey: .capacityL) ?? 0
        currentRoute = try c.decodeIfPresent(String.self, forKey: .currentRoute) ?? ""
    }
}

struct VehicleListResponse: Decodable {
    let vehicles: [Vehicle]
}

// MARK: - Staff
struct StaffMember: Decodable, Identifiable {
    let id: String
    let name: String
    let phone: String
    let role: String
    let status: String
    let joinedAt: String

    enum CodingKeys: String, CodingKey {
        case id, name, phone, role, status
        case joinedAt = "joined_at"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        id = try c.decode(String.self, forKey: .id)
        name = try c.decodeIfPresent(String.self, forKey: .name) ?? ""
        phone = try c.decodeIfPresent(String.self, forKey: .phone) ?? ""
        role = try c.decodeIfPresent(String.self, forKey: .role) ?? ""
        status = try c.decodeIfPresent(String.self, forKey: .status) ?? ""
        joinedAt = try c.decodeIfPresent(String.self, forKey: .joinedAt) ?? ""
    }
}

struct StaffListResponse: Decodable {
    let staff: [StaffMember]
}

// MARK: - Insight
struct Insight: Decodable, Identifiable {
    let id: String
    let warehouseId: String
    let warehouseName: String
    let productId: String
    let productName: String
    let urgency: String
    let currentStock: Int
    let avgDailyVelocity: Double
    let daysUntilStockout: Int
    let reorderQuantity: Int
    let status: String

    enum CodingKeys: String, CodingKey {
        case id
        case warehouseId = "warehouse_id"
        case warehouseName = "warehouse_name"
        case productId = "product_id"
        case productName = "product_name"
        case urgency
        case currentStock = "current_stock"
        case avgDailyVelocity = "avg_daily_velocity"
        case daysUntilStockout = "days_until_stockout"
        case reorderQuantity = "reorder_quantity"
        case status
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        id = try c.decode(String.self, forKey: .id)
        warehouseId = try c.decodeIfPresent(String.self, forKey: .warehouseId) ?? ""
        warehouseName = try c.decodeIfPresent(String.self, forKey: .warehouseName) ?? ""
        productId = try c.decodeIfPresent(String.self, forKey: .productId) ?? ""
        productName = try c.decodeIfPresent(String.self, forKey: .productName) ?? ""
        urgency = try c.decodeIfPresent(String.self, forKey: .urgency) ?? ""
        currentStock = try c.decodeIfPresent(Int.self, forKey: .currentStock) ?? 0
        avgDailyVelocity = try c.decodeIfPresent(Double.self, forKey: .avgDailyVelocity) ?? 0
        daysUntilStockout = try c.decodeIfPresent(Int.self, forKey: .daysUntilStockout) ?? 0
        reorderQuantity = try c.decodeIfPresent(Int.self, forKey: .reorderQuantity) ?? 0
        status = try c.decodeIfPresent(String.self, forKey: .status) ?? ""
    }
}

struct InsightListResponse: Decodable {
    let insights: [Insight]
}

// MARK: - Dispatch
struct DispatchRequest: Encodable {
    let transferIds: [String]

    enum CodingKeys: String, CodingKey {
        case transferIds = "transfer_ids"
    }
}

struct DispatchResponse: Decodable {
    let manifestId: String
    let truckPlate: String
    let stopCount: Int

    enum CodingKeys: String, CodingKey {
        case manifestId = "manifest_id"
        case truckPlate = "truck_plate"
        case stopCount = "stop_count"
    }
}
