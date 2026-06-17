import Foundation

// MARK: - Auth
struct LoginRequest: Encodable {
    var phone: String = ""
    var password: String = ""
    var idToken: String = ""

    enum CodingKeys: String, CodingKey {
        case phone, password
        case idToken = "id_token"
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        if !phone.isEmpty { try container.encode(phone, forKey: .phone) }
        if !password.isEmpty { try container.encode(password, forKey: .password) }
        if !idToken.isEmpty { try container.encode(idToken, forKey: .idToken) }
    }
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

struct SupplyRequestListResponse: Decodable {
    let requests: [SupplyRequest]
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
        case stopCount = "stop_count", transferCount = "transfer_count"
        case regionCode = "region_code"
        case createdAt = "created_at"
        case transfers
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        id = try c.decode(String.self, forKey: .id)
        factoryId = try c.decodeIfPresent(String.self, forKey: .factoryId) ?? ""
        driverId = try c.decodeIfPresent(String.self, forKey: .driverId) ?? ""
        driverName = try c.decodeIfPresent(String.self, forKey: .driverName) ?? ""
        vehicleId = try c.decodeIfPresent(String.self, forKey: .vehicleId) ?? ""
        vehicleLabel = try c.decodeIfPresent(String.self, forKey: .vehicleLabel) ?? ""
        truckId = try c.decodeIfPresent(String.self, forKey: .truckId) ?? ""
        truckPlate = try c.decodeIfPresent(String.self, forKey: .truckPlate) ?? ""
        state = try c.decodeIfPresent(String.self, forKey: .state) ?? ""
        status = try c.decodeIfPresent(String.self, forKey: .status) ?? ""
        totalVolumeVU = try c.decodeIfPresent(Double.self, forKey: .totalVolumeVU) ?? 0
        maxVolumeVU = try c.decodeIfPresent(Double.self, forKey: .maxVolumeVU) ?? 0
        maxCapacityVU = try c.decodeIfPresent(Double.self, forKey: .maxCapacityVU) ?? 0
        stopCount = try c.decodeIfPresent(Int.self, forKey: .stopCount)
            ?? c.decodeIfPresent(Int.self, forKey: .transferCount) ?? 0
        regionCode = try c.decodeIfPresent(String.self, forKey: .regionCode) ?? ""
        createdAt = try c.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
        transfers = try c.decodeIfPresent([ManifestTransfer].self, forKey: .transfers) ?? []
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
        case volumeVU = "volume_vu", totalVU = "total_vu"
        case state
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        id = try c.decode(String.self, forKey: .id)
        productName = try c.decodeIfPresent(String.self, forKey: .productName) ?? ""
        quantity = try c.decodeIfPresent(Int.self, forKey: .quantity) ?? 0
        volumeVU = try c.decodeIfPresent(Double.self, forKey: .volumeVU)
            ?? Double(c.decodeIfPresent(Int.self, forKey: .totalVU) ?? 0)
        state = try c.decodeIfPresent(String.self, forKey: .state) ?? ""
    }
}

struct ManifestListResponse: Decodable {
    let manifests: [Manifest]
    let total: Int
}

struct ManifestDetailCore: Decodable, Identifiable {
    let manifestId: String
    let state: String
    let transferCount: Int
    let totalVolumeVU: Int
    let maxVolumeVU: Int
    let driverId: String
    let vehicleId: String
    let updatedAt: String

    var id: String { manifestId }

    enum CodingKeys: String, CodingKey {
        case manifestId = "manifest_id"
        case state
        case transferCount = "transfer_count"
        case totalVolumeVU = "total_volume_vu"
        case maxVolumeVU = "max_volume_vu"
        case driverId = "driver_id"
        case vehicleId = "vehicle_id"
        case updatedAt = "updated_at"
    }
}

struct ManifestTransitionRow: Decodable, Identifiable {
    let action: String
    let fromState: String
    let toState: String
    let at: String
    let reason: String

    var id: String { "\(action)-\(at)" }

    enum CodingKeys: String, CodingKey {
        case action
        case fromState = "from_state"
        case toState = "to_state"
        case at
        case reason
    }
}

struct ManifestDetailSnapshot: Decodable {
    let manifest: ManifestDetailCore
    let transfers: [ManifestTransfer]
    let transitions: [ManifestTransitionRow]
    let exceptions: [ManifestException]
    let routeId: String
    let stopCount: Int
    let orderCount: Int

    enum CodingKeys: String, CodingKey {
        case manifest, transfers, transitions, exceptions
        case routeId = "route_id"
        case stopCount = "stop_count"
        case orderCount = "order_count"
    }
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

// MARK: - Manifest exceptions
struct ManifestException: Decodable, Identifiable {
    let exceptionId: String
    let manifestId: String
    let transferId: String
    let reason: String
    let metadata: String
    let attemptCount: Int
    let escalated: Bool
    let createdAt: String
    let correlationId: String

    var id: String { exceptionId }

    enum CodingKeys: String, CodingKey {
        case exceptionId = "exception_id"
        case manifestId = "manifest_id"
        case transferId = "transfer_id"
        case reason
        case metadata
        case attemptCount = "attempt_count"
        case escalated
        case createdAt = "created_at"
        case correlationId = "correlation_id"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        exceptionId = try container.decode(String.self, forKey: .exceptionId)
        manifestId = try container.decodeIfPresent(String.self, forKey: .manifestId) ?? ""
        transferId = try container.decodeIfPresent(String.self, forKey: .transferId) ?? ""
        reason = try container.decodeIfPresent(String.self, forKey: .reason) ?? ""
        metadata = try container.decodeIfPresent(String.self, forKey: .metadata) ?? ""
        attemptCount = try container.decodeIfPresent(Int.self, forKey: .attemptCount) ?? 0
        escalated = try container.decodeIfPresent(Bool.self, forKey: .escalated) ?? false
        createdAt = try container.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
        correlationId = try container.decodeIfPresent(String.self, forKey: .correlationId) ?? ""
    }
}

struct ManifestExceptionListResponse: Decodable {
    let exceptions: [ManifestException]
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

// MARK: - Extended Factory Contracts
struct FactoryProfile: Decodable {
    let factoryId: String
    let supplierId: String
    let name: String
    let address: String
    let lat: Double
    let lng: Double
    let h3Index: String
    let regionCode: String
    let leadTimeDays: Int
    let productionCapacityVU: Double
    let productTypes: [String]
    let isActive: Bool
    let createdAt: String

    enum CodingKeys: String, CodingKey {
        case factoryId = "factory_id"
        case supplierId = "supplier_id"
        case name, address, lat, lng
        case h3Index = "h3_index"
        case regionCode = "region_code"
        case leadTimeDays = "lead_time_days"
        case productionCapacityVU = "production_capacity_vu"
        case productTypes = "product_types"
        case isActive = "is_active"
        case createdAt = "created_at"
    }
}

struct FactoryAnalyticsDayBucket: Decodable {
    let date: String
    let transfers: Int64

    enum CodingKeys: String, CodingKey {
        case date
        case transfers
    }
}

struct FactoryAnalyticsOverview: Decodable {
    let dailyActivity: [FactoryAnalyticsDayBucket]
    let transfersTotal: Int64
    let manifestsActive: Int64
    let exceptionQueue: Int64
    let avgLeadTimeMins: Double

    enum CodingKeys: String, CodingKey {
        case dailyActivity = "daily_activity"
        case transfersTotal = "transfers_total"
        case manifestsActive = "manifests_active"
        case exceptionQueue = "exception_queue"
        case avgLeadTimeMins = "avg_lead_time_mins"
    }
}

struct FactoryCreateTransferRequest: Encodable {
    let orderId: String?
    let totalVu: Int
    let driverId: String?
    let vehicleId: String?

    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
        case totalVu = "total_vu"
        case driverId = "driver_id"
        case vehicleId = "vehicle_id"
    }
}

struct FactoryCreateTransferResponse: Decodable {
    let transferId: String
    let state: String
    let totalVu: Int

    enum CodingKeys: String, CodingKey {
        case transferId = "transfer_id"
        case state
        case totalVu = "total_vu"
    }
}

struct FactoryFleetDriverRow: Decodable, Identifiable {
    let driverId: String
    let name: String
    let onShift: Bool

    var id: String { driverId }

    enum CodingKeys: String, CodingKey {
        case driverId = "driver_id"
        case name
        case onShift = "on_shift"
    }
}

struct FactoryFleetDriversEnvelope: Decodable {
    let drivers: [FactoryFleetDriverRow]
}

struct FactoryFleetVehicleRow: Decodable, Identifiable {
    let vehicleId: String
    let plateNo: String
    let state: String

    var id: String { vehicleId }

    enum CodingKeys: String, CodingKey {
        case vehicleId = "vehicle_id"
        case plateNo = "plate_no"
        case state
    }
}

struct FactoryFleetVehiclesEnvelope: Decodable {
    let vehicles: [FactoryFleetVehicleRow]
}

struct FactoryFleetDriver: Decodable, Identifiable {
    let id: String
    let name: String
    let phone: String
    let driverType: String
    let vehicleType: String
    let licensePlate: String
    let isActive: Bool
    let truckStatus: String
    let createdAt: String
    let vehicleId: String

    enum CodingKeys: String, CodingKey {
        case id = "driver_id"
        case name, phone
        case driverType = "driver_type"
        case vehicleType = "vehicle_type"
        case licensePlate = "license_plate"
        case isActive = "is_active"
        case truckStatus = "truck_status"
        case createdAt = "created_at"
        case vehicleId = "vehicle_id"
    }
}

struct FactoryFleetDriverListResponse: Decodable {
    let data: [FactoryFleetDriver]
}

struct FactoryFleetVehicle: Decodable, Identifiable {
    let id: String
    let supplierId: String
    let vehicleClass: String
    let maxVolumeVU: Double
    let licensePlate: String
    let isActive: Bool
    let createdAt: String

    enum CodingKeys: String, CodingKey {
        case id = "vehicle_id"
        case supplierId = "supplier_id"
        case vehicleClass = "vehicle_class"
        case maxVolumeVU = "max_volume_vu"
        case licensePlate = "license_plate"
        case isActive = "is_active"
        case createdAt = "created_at"
    }
}

struct FactoryFleetVehicleListResponse: Decodable {
    let data: [FactoryFleetVehicle]
}

struct FactoryStaffDetail: Decodable {
    let staffId: String
    let factoryId: String
    let name: String
    let phone: String
    let staffRole: String
    let isActive: Bool
    let createdAt: String

    enum CodingKeys: String, CodingKey {
        case staffId = "staff_id"
        case factoryId = "factory_id"
        case name
        case phone
        case staffRole = "staff_role"
        case isActive = "is_active"
        case createdAt = "created_at"
    }
}

struct FactoryManifestTransitionResponse: Decodable {
    let status: String
    let manifestId: String?
    let state: String?

    enum CodingKeys: String, CodingKey {
        case status
        case manifestId = "manifest_id"
        case state
    }
}

enum ManifestLifecycleAction: String {
    case startLoading = "start-loading"
    case seal
    case dispatch
    case complete

    var label: String {
        switch self {
        case .startLoading: "Start loading"
        case .seal: "Seal manifest"
        case .dispatch: "Dispatch"
        case .complete: "Complete"
        }
    }

    static func next(for state: String) -> ManifestLifecycleAction? {
        switch state.uppercased() {
        case "DRAFT": .startLoading
        case "LOADING": .seal
        case "SEALED": .dispatch
        case "DISPATCHED": .complete
        default: nil
        }
    }
}
