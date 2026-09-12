import Foundation

// MARK: - Auth

struct LoginRequest: Encodable {
    let phone: String
    let pin: String
    let idToken: String

    enum CodingKeys: String, CodingKey {
        case phone
        case pin
        case idToken = "id_token"
    }

    init(phone: String = "", pin: String = "", idToken: String = "") {
        self.phone = phone
        self.pin = pin
        self.idToken = idToken
    }
}

struct AuthResponse: Decodable {
    let token: String
    let refreshToken: String
    let warehouseId: String
    let isConfigured: Bool

    enum CodingKeys: String, CodingKey {
        case token
        case refreshToken = "refresh_token"
        case warehouseId = "warehouse_id"
        case isConfigured = "is_configured"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        token = try container.decode(String.self, forKey: .token)
        refreshToken = try container.decode(String.self, forKey: .refreshToken)
        warehouseId = try container.decodeIfPresent(String.self, forKey: .warehouseId) ?? ""
        isConfigured = try container.decodeIfPresent(Bool.self, forKey: .isConfigured) ?? false
    }
}

// MARK: - Dashboard

struct FleetStatusEntry: Decodable, Hashable {
    let status: String
    let count: Int64
}

struct HoldReasonEntry: Decodable, Hashable {
    let code: String
    let count: Int
}

struct DashboardData: Decodable {
    let activeOrders: Int64
    let completedToday: Int64
    let pendingDispatch: Int64
    let driversOnRoute: Int64
    let driversIdle: Int64
    let totalDrivers: Int64
    let totalVehicles: Int64
    let todayRevenue: Int64
    let completedTodayAvailable: Bool
    let todayRevenueAvailable: Bool
    let lowStockCount: Int64
    let totalStaff: Int64
    let fleetStatus: [FleetStatusEntry]
    let ordersByStatus: [String: Int]
    let truckDuty: [String: Int]
    let holdReasons: [HoldReasonEntry]
    let demandSource: String
    let historyAvailable: Bool

    enum CodingKeys: String, CodingKey {
        case activeOrders = "active_orders"
        case completedToday = "completed_today"
        case pendingDispatch = "pending_dispatch"
        case driversOnRoute = "drivers_on_route"
        case driversIdle = "drivers_idle"
        case totalDrivers = "total_drivers"
        case totalVehicles = "total_vehicles"
        case todayRevenue = "today_revenue"
        case completedTodayAvailable = "completed_today_available"
        case todayRevenueAvailable = "today_revenue_available"
        case lowStockCount = "low_stock_count"
        case totalStaff = "total_staff"
        case fleetStatus = "fleet_status"
        case ordersByStatus = "orders_by_status"
        case truckDuty = "truck_duty"
        case holdReasons = "hold_reasons"
        case demandSource = "demand_source"
        case historyAvailable = "history_available"
    }

    init(
        activeOrders: Int64 = 0,
        completedToday: Int64 = 0,
        pendingDispatch: Int64 = 0,
        driversOnRoute: Int64 = 0,
        driversIdle: Int64 = 0,
        totalDrivers: Int64 = 0,
        totalVehicles: Int64 = 0,
        todayRevenue: Int64 = 0,
        completedTodayAvailable: Bool = false,
        todayRevenueAvailable: Bool = false,
        lowStockCount: Int64 = 0,
        totalStaff: Int64 = 0,
        fleetStatus: [FleetStatusEntry] = [],
        ordersByStatus: [String: Int] = [:],
        truckDuty: [String: Int] = [:],
        holdReasons: [HoldReasonEntry] = [],
        demandSource: String = "empty",
        historyAvailable: Bool = false
    ) {
        self.activeOrders = activeOrders
        self.completedToday = completedToday
        self.pendingDispatch = pendingDispatch
        self.driversOnRoute = driversOnRoute
        self.driversIdle = driversIdle
        self.totalDrivers = totalDrivers
        self.totalVehicles = totalVehicles
        self.todayRevenue = todayRevenue
        self.completedTodayAvailable = completedTodayAvailable
        self.todayRevenueAvailable = todayRevenueAvailable
        self.lowStockCount = lowStockCount
        self.totalStaff = totalStaff
        self.fleetStatus = fleetStatus
        self.ordersByStatus = ordersByStatus
        self.truckDuty = truckDuty
        self.holdReasons = holdReasons
        self.demandSource = demandSource
        self.historyAvailable = historyAvailable
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        activeOrders = (try? c.decode(Int64.self, forKey: .activeOrders)) ?? 0
        completedToday = (try? c.decode(Int64.self, forKey: .completedToday)) ?? 0
        pendingDispatch = (try? c.decode(Int64.self, forKey: .pendingDispatch)) ?? 0
        driversOnRoute = (try? c.decode(Int64.self, forKey: .driversOnRoute)) ?? 0
        driversIdle = (try? c.decode(Int64.self, forKey: .driversIdle)) ?? 0
        totalDrivers = (try? c.decode(Int64.self, forKey: .totalDrivers)) ?? 0
        totalVehicles = (try? c.decode(Int64.self, forKey: .totalVehicles)) ?? 0
        todayRevenue = (try? c.decode(Int64.self, forKey: .todayRevenue)) ?? 0
        completedTodayAvailable = (try? c.decode(Bool.self, forKey: .completedTodayAvailable)) ?? false
        todayRevenueAvailable = (try? c.decode(Bool.self, forKey: .todayRevenueAvailable)) ?? false
        lowStockCount = (try? c.decode(Int64.self, forKey: .lowStockCount)) ?? 0
        totalStaff = (try? c.decode(Int64.self, forKey: .totalStaff)) ?? 0
        fleetStatus = (try? c.decode([FleetStatusEntry].self, forKey: .fleetStatus)) ?? []
        ordersByStatus = (try? c.decode([String: Int].self, forKey: .ordersByStatus)) ?? [:]
        truckDuty = (try? c.decode([String: Int].self, forKey: .truckDuty)) ?? [:]
        holdReasons = (try? c.decode([HoldReasonEntry].self, forKey: .holdReasons)) ?? []
        demandSource = (try? c.decode(String.self, forKey: .demandSource)) ?? "empty"
        historyAvailable = (try? c.decode(Bool.self, forKey: .historyAvailable)) ?? false
    }

    static let empty = DashboardData()
}

// MARK: - Order

struct Order: Decodable, Identifiable {
    var id: String { orderId }
    let orderId: String
    let retailerName: String
    let state: String
    let totalUzs: Int
    let lineItems: [LineItem]

    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
        case retailerName = "retailer_name"
        case state
        case totalUzs = "total_uzs"
        case lineItems = "line_items"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        orderId = try c.decode(String.self, forKey: .orderId)
        retailerName = try c.decodeIfPresent(String.self, forKey: .retailerName) ?? ""
        state = try c.decodeIfPresent(String.self, forKey: .state) ?? ""
        totalUzs = try c.decodeIfPresent(Int.self, forKey: .totalUzs) ?? 0
        lineItems = try c.decodeIfPresent([LineItem].self, forKey: .lineItems) ?? []
    }
}

struct OrderReceiptMeta: Decodable {
    let receiptId: String
    let htmlUrl: String
    let pdfUrl: String
    let qrUrl: String
    let partyCopy: String

    enum CodingKeys: String, CodingKey {
        case receiptId = "receipt_id"
        case htmlUrl = "html_url"
        case pdfUrl = "pdf_url"
        case qrUrl = "qr_url"
        case partyCopy = "party_copy"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        receiptId = try c.decodeIfPresent(String.self, forKey: .receiptId) ?? ""
        htmlUrl = try c.decodeIfPresent(String.self, forKey: .htmlUrl) ?? ""
        pdfUrl = try c.decodeIfPresent(String.self, forKey: .pdfUrl) ?? ""
        qrUrl = try c.decodeIfPresent(String.self, forKey: .qrUrl) ?? ""
        partyCopy = try c.decodeIfPresent(String.self, forKey: .partyCopy) ?? ""
    }
}

struct LineItem: Decodable, Identifiable {
    var id: String { productId }
    let productId: String
    let productName: String
    let quantity: Int
    let unitPrice: Int

    enum CodingKeys: String, CodingKey {
        case productId = "product_id"
        case productName = "product_name"
        case quantity
        case unitPrice = "unit_price"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        productId = try c.decode(String.self, forKey: .productId)
        productName = try c.decodeIfPresent(String.self, forKey: .productName) ?? ""
        quantity = try c.decodeIfPresent(Int.self, forKey: .quantity) ?? 0
        unitPrice = try c.decodeIfPresent(Int.self, forKey: .unitPrice) ?? 0
    }
}

struct OrderListResponse: Decodable {
    let orders: [Order]
}

// MARK: - Driver

struct Driver: Decodable, Identifiable {
    var id: String { driverId }
    let driverId: String
    let name: String
    let phone: String
    let truckStatus: String
    let isActive: Bool
    let vehicleId: String?
    let vehicleClass: String?
    let vehicleIsActive: Bool
    let vehicleUnavailableReason: String?

    enum CodingKeys: String, CodingKey {
        case driverId = "driver_id"
        case name, phone
        case truckStatus = "truck_status"
        case isActive = "is_active"
        case vehicleId = "vehicle_id"
        case vehicleClass = "vehicle_class"
        case vehicleIsActive = "vehicle_is_active"
        case vehicleUnavailableReason = "vehicle_unavailable_reason"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        driverId = try c.decode(String.self, forKey: .driverId)
        name = try c.decodeIfPresent(String.self, forKey: .name) ?? ""
        phone = try c.decodeIfPresent(String.self, forKey: .phone) ?? ""
        truckStatus = try c.decodeIfPresent(String.self, forKey: .truckStatus) ?? ""
        isActive = try c.decodeIfPresent(Bool.self, forKey: .isActive) ?? true
        vehicleId = try c.decodeIfPresent(String.self, forKey: .vehicleId)
        vehicleClass = try c.decodeIfPresent(String.self, forKey: .vehicleClass)
        vehicleIsActive = try c.decodeIfPresent(Bool.self, forKey: .vehicleIsActive) ?? false
        vehicleUnavailableReason = try c.decodeIfPresent(String.self, forKey: .vehicleUnavailableReason)
    }
}

struct DriverListResponse: Decodable {
    let drivers: [Driver]
}

struct CreateDriverRequest: Encodable {
    let name: String
    let phone: String
}

struct CreateDriverResponse: Decodable {
    let driverId: String
    let pin: String

    enum CodingKeys: String, CodingKey {
        case driverId = "driver_id"
        case pin
    }
}

struct AssignDriverVehicleRequest: Encodable {
    let vehicleId: String?

    enum CodingKeys: String, CodingKey {
        case vehicleId = "vehicle_id"
    }
}

struct AssignDriverVehicleResponse: Decodable {
    let status: String
    let driverId: String
    let vehicleId: String?
    let previouslyAssignedDriver: String?

    enum CodingKeys: String, CodingKey {
        case status
        case driverId = "driver_id"
        case vehicleId = "vehicle_id"
        case previouslyAssignedDriver = "previously_assigned_driver"
    }
}

// MARK: - Vehicle

enum VehicleUnavailableReasonOption: String, CaseIterable, Identifiable {
    case maintenance = "MAINTENANCE"
    case truckDamaged = "TRUCK_DAMAGED"
    case regulatoryHold = "REGULATORY_HOLD"
    case manualHold = "MANUAL_HOLD"
    case other = "OTHER"

    var id: String { rawValue }

    var title: String {
        switch self {
        case .maintenance:
            return "Maintenance"
        case .truckDamaged:
            return "Truck Damaged"
        case .regulatoryHold:
            return "Regulatory Hold"
        case .manualHold:
            return "Manual Hold"
        case .other:
            return "Other"
        }
    }
}

func formatUnavailableReason(_ reason: String?, note: String? = nil) -> String {
    guard let reason, !reason.isEmpty else { return note?.trimmingCharacters(in: .whitespacesAndNewlines) ?? "" }
    if reason.uppercased() == "OTHER", let note, !note.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
        return note.trimmingCharacters(in: .whitespacesAndNewlines)
    }
    return vehicleUnavailableReasonLabel(reason)
}

func vehicleUnavailableReasonLabel(_ reason: String) -> String {
    VehicleUnavailableReasonOption(rawValue: reason)?.title
        ?? reason.replacingOccurrences(of: "_", with: " ").capitalized
}

struct Vehicle: Decodable, Identifiable {
    var id: String { vehicleId }
    let vehicleId: String
    let label: String
    let licensePlate: String
    let vehicleClass: String
    let capacityVu: Int
    let status: String
    let isActive: Bool
    let unavailableReason: String?
    let unavailableNote: String?
    let assignedDriverId: String?
    let assignedDriverName: String?

    enum CodingKeys: String, CodingKey {
        case vehicleId = "vehicle_id"
        case label
        case licensePlate = "license_plate"
        case vehicleClass = "vehicle_class"
        case capacityVu = "capacity_vu"
        case status
        case isActive = "is_active"
        case unavailableReason = "unavailable_reason"
        case unavailableNote = "unavailable_note"
        case assignedDriverId = "assigned_driver_id"
        case assignedDriverName = "assigned_driver_name"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        vehicleId = try c.decode(String.self, forKey: .vehicleId)
        label = try c.decodeIfPresent(String.self, forKey: .label) ?? ""
        licensePlate = try c.decodeIfPresent(String.self, forKey: .licensePlate) ?? ""
        vehicleClass = try c.decodeIfPresent(String.self, forKey: .vehicleClass) ?? ""
        capacityVu = try c.decodeIfPresent(Int.self, forKey: .capacityVu) ?? 0
        status = try c.decodeIfPresent(String.self, forKey: .status) ?? ""
        isActive = try c.decodeIfPresent(Bool.self, forKey: .isActive) ?? true
        unavailableReason = try c.decodeIfPresent(String.self, forKey: .unavailableReason)
        unavailableNote = try c.decodeIfPresent(String.self, forKey: .unavailableNote)
        assignedDriverId = try c.decodeIfPresent(String.self, forKey: .assignedDriverId)
        assignedDriverName = try c.decodeIfPresent(String.self, forKey: .assignedDriverName)
    }
}

struct VehicleListResponse: Decodable {
    let vehicles: [Vehicle]
}

struct VehicleDetailResponse: Decodable {
    let vehicle: Vehicle
}

struct CreateVehicleRequest: Encodable {
    let label: String
    let licensePlate: String
    let vehicleClass: String

    enum CodingKeys: String, CodingKey {
        case label
        case licensePlate = "license_plate"
        case vehicleClass = "vehicle_class"
    }
}

struct UpdateVehicleRequest: Encodable {
    let isActive: Bool?
    let unavailableReason: String?
    let unavailableNote: String?

    enum CodingKeys: String, CodingKey {
        case isActive = "is_active"
        case unavailableReason = "unavailable_reason"
        case unavailableNote = "unavailable_note"
    }
}

struct VehicleMutationResponse: Decodable {
    let status: String
    let vehicleId: String
    let unavailableReason: String?

    enum CodingKeys: String, CodingKey {
        case status
        case vehicleId = "vehicle_id"
        case unavailableReason = "unavailable_reason"
    }
}

// MARK: - Inventory

struct InventoryItem: Decodable, Identifiable {
    var id: String { productId }
    let productId: String
    let productName: String
    let quantity: Int
    let reorderThreshold: Int
    let outOfStockPolicy: String?

    enum CodingKeys: String, CodingKey {
        case productId = "product_id"
        case productName = "product_name"
        case quantity
        case reorderThreshold = "reorder_threshold"
        case outOfStockPolicy = "out_of_stock_policy"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        productId = try c.decode(String.self, forKey: .productId)
        productName = try c.decodeIfPresent(String.self, forKey: .productName) ?? ""
        quantity = try c.decodeIfPresent(Int.self, forKey: .quantity) ?? 0
        reorderThreshold = try c.decodeIfPresent(Int.self, forKey: .reorderThreshold) ?? 0
        outOfStockPolicy = try c.decodeIfPresent(String.self, forKey: .outOfStockPolicy)
    }
}

struct InventoryPolicyPatchRequest: Encodable {
    let outOfStockPolicy: String

    enum CodingKeys: String, CodingKey {
        case outOfStockPolicy = "out_of_stock_policy"
    }
}

struct DeliveryFeeTier: Codable, Equatable {
    let maxKm: Double?
    let feeMinor: Int64

    enum CodingKeys: String, CodingKey {
        case maxKm = "max_km"
        case feeMinor = "fee_minor"
    }
}

struct DeliveryFeeRules: Codable, Equatable {
    let currency: String
    let baseFeeMinor: Int64
    let tiers: [DeliveryFeeTier]

    enum CodingKeys: String, CodingKey {
        case currency
        case baseFeeMinor = "base_fee_minor"
        case tiers
    }
}

struct WarehouseOpsSettingsResponse: Decodable {
    let warehouseId: String
    let name: String
    let defaultOutOfStockPolicy: String
    let showStockCountsToRetailers: Bool
    let operatingSchedule: [String: AnyCodable]?
    let opsAlwaysAvailable: Bool
    let expressEnabled: Bool
    let expressStockFloor: Int64
    let preorderMinLeadDays: Int64
    let preorderMaxLeadDays: Int64
    let orderLineMinQuantity: Int64?
    let orderLineMaxQuantity: Int64?
    let deliveryFeeRules: DeliveryFeeRules?

    enum CodingKeys: String, CodingKey {
        case warehouseId = "warehouse_id"
        case name
        case defaultOutOfStockPolicy = "default_out_of_stock_policy"
        case showStockCountsToRetailers = "show_stock_counts_to_retailers"
        case operatingSchedule = "operating_schedule"
        case opsAlwaysAvailable = "ops_always_available"
        case expressEnabled = "express_enabled"
        case expressStockFloor = "express_stock_floor"
        case preorderMinLeadDays = "preorder_min_lead_days"
        case preorderMaxLeadDays = "preorder_max_lead_days"
        case orderLineMinQuantity = "order_line_min_quantity"
        case orderLineMaxQuantity = "order_line_max_quantity"
        case deliveryFeeRules = "delivery_fee_rules"
    }
}

/// GET/PUT /v1/warehouse/return-policy
struct WarehouseReturnPolicy: Codable {
    var supplierId: String
    var reverseDockSlaHours: Int64?
    var retailerFileWindowHours: Int64?
    var canOverrideRetailerWindow: Bool

    enum CodingKeys: String, CodingKey {
        case supplierId = "supplier_id"
        case reverseDockSlaHours = "reverse_dock_sla_hours"
        case retailerFileWindowHours = "retailer_file_window_hours"
        case canOverrideRetailerWindow = "can_override_retailer_window"
    }
}

struct WarehouseOpsSettingsPatchRequest: Encodable {
    let defaultOutOfStockPolicy: String
    let showStockCountsToRetailers: Bool?
    let operatingSchedule: [String: AnyCodable]
    let preorderMinLeadDays: Int64?
    let preorderMaxLeadDays: Int64?
    let orderLineMinQuantity: Int64?
    let orderLineMaxQuantity: Int64?
    let clearOrderLineMinQuantity: Bool?
    let clearOrderLineMaxQuantity: Bool?
    let expressEnabled: Bool?
    let expressStockFloor: Int64?
    let deliveryFeeRules: DeliveryFeeRules?
    let clearDeliveryFeeRules: Bool?

    enum CodingKeys: String, CodingKey {
        case defaultOutOfStockPolicy = "default_out_of_stock_policy"
        case showStockCountsToRetailers = "show_stock_counts_to_retailers"
        case operatingSchedule = "operating_schedule"
        case preorderMinLeadDays = "preorder_min_lead_days"
        case preorderMaxLeadDays = "preorder_max_lead_days"
        case orderLineMinQuantity = "order_line_min_quantity"
        case orderLineMaxQuantity = "order_line_max_quantity"
        case clearOrderLineMinQuantity = "clear_order_line_min_quantity"
        case clearOrderLineMaxQuantity = "clear_order_line_max_quantity"
        case expressEnabled = "express_enabled"
        case expressStockFloor = "express_stock_floor"
        case deliveryFeeRules = "delivery_fee_rules"
        case clearDeliveryFeeRules = "clear_delivery_fee_rules"
    }
}

struct WarehousePreorderRow: Decodable, Identifiable {
    var id: String { orderId }
    let orderId: String
    let status: String
    let orderSource: String?
    let confirmationStatus: String?
    let requestedDeliveryDate: String?
    let proposedDeliveryDate: String?
    let deliveryProposalReason: String?
    let preorderBadge: String?

    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
        case status
        case orderSource = "order_source"
        case confirmationStatus = "confirmation_status"
        case requestedDeliveryDate = "requested_delivery_date"
        case proposedDeliveryDate = "proposed_delivery_date"
        case deliveryProposalReason = "delivery_proposal_reason"
        case preorderBadge = "preorder_badge"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        orderId = try c.decode(String.self, forKey: .orderId)
        status = try c.decodeIfPresent(String.self, forKey: .status) ?? ""
        orderSource = try c.decodeIfPresent(String.self, forKey: .orderSource)
        confirmationStatus = try c.decodeIfPresent(String.self, forKey: .confirmationStatus)
        requestedDeliveryDate = try c.decodeIfPresent(String.self, forKey: .requestedDeliveryDate)
        proposedDeliveryDate = try c.decodeIfPresent(String.self, forKey: .proposedDeliveryDate)
        deliveryProposalReason = try c.decodeIfPresent(String.self, forKey: .deliveryProposalReason)
        preorderBadge = try c.decodeIfPresent(String.self, forKey: .preorderBadge)
    }
}

struct WarehousePreordersResponse: Decodable {
    let preorders: [WarehousePreorderRow]
    let items: [WarehousePreorderRow]

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        preorders = try c.decodeIfPresent([WarehousePreorderRow].self, forKey: .preorders) ?? []
        items = try c.decodeIfPresent([WarehousePreorderRow].self, forKey: .items) ?? []
    }

    enum CodingKeys: String, CodingKey { case preorders, items }
}

struct StockCommitmentRow: Decodable, Identifiable {
    var id: String { skuId }
    let skuId: String
    let name: String?
    let imageUrl: String?
    let onHand: Int64
    let availableQty: Int64
    let reservedAsap: Int64
    let reservedScheduled: Int64
    let deficitQty: Int64

    enum CodingKeys: String, CodingKey {
        case skuId = "sku_id"
        case name
        case imageUrl = "image_url"
        case onHand = "on_hand"
        case availableQty = "available_qty"
        case reservedAsap = "reserved_asap"
        case reservedScheduled = "reserved_scheduled"
        case deficitQty = "deficit_qty"
    }
}

struct StockCommitmentsResponse: Decodable {
    let items: [StockCommitmentRow]
    let skus: [StockCommitmentRow]

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        items = try c.decodeIfPresent([StockCommitmentRow].self, forKey: .items) ?? []
        skus = try c.decodeIfPresent([StockCommitmentRow].self, forKey: .skus) ?? []
    }

    enum CodingKeys: String, CodingKey { case items, skus }
}

struct InventoryListResponse: Decodable {
    let items: [InventoryItem]
}

struct InventoryAdjustRequest: Encodable {
    let productId: String
    let quantity: Int

    enum CodingKeys: String, CodingKey {
        case productId = "product_id"
        case quantity
    }
}

// MARK: - §8.7 Wave 1A bins / lots

struct WarehouseBinCreateRequest: Encodable {
    var locationId: String?
    var zone: String?
    var locationType: String
    var pickSequence: Int?

    enum CodingKeys: String, CodingKey {
        case locationId = "location_id"
        case zone
        case locationType = "location_type"
        case pickSequence = "pick_sequence"
    }
}

struct WarehouseBinLocation: Decodable {
    let warehouseId: String?
    let locationId: String
    let zone: String?
    let locationType: String?

    enum CodingKeys: String, CodingKey {
        case warehouseId = "warehouse_id"
        case locationId = "location_id"
        case zone
        case locationType = "location_type"
    }
}

struct StockLotPutawayRequest: Encodable {
    let productId: String
    let locationId: String
    let quantity: Int
    var lotCode: String?
    var expiryDate: String?

    enum CodingKeys: String, CodingKey {
        case productId = "product_id"
        case locationId = "location_id"
        case quantity
        case lotCode = "lot_code"
        case expiryDate = "expiry_date"
    }
}

struct StockLotPutawayResponse: Decodable {
    let lotId: String
    let productId: String?
    let locationId: String?
    let quantityOnHand: Int?
    let status: String?

    enum CodingKeys: String, CodingKey {
        case lotId = "lot_id"
        case productId = "product_id"
        case locationId = "location_id"
        case quantityOnHand = "quantity_on_hand"
        case status
    }
}

// MARK: - §8.7 Wave 1B pick waves

struct PickTask: Decodable, Identifiable {
    var id: String { taskId }
    let taskId: String
    let orderId: String?
    let productId: String
    let lotId: String
    let locationId: String?
    let quantityRequested: Int
    let quantityPicked: Int?
    let status: String
    let pickSequence: Int?

    enum CodingKeys: String, CodingKey {
        case taskId = "task_id"
        case orderId = "order_id"
        case productId = "product_id"
        case lotId = "lot_id"
        case locationId = "location_id"
        case quantityRequested = "quantity_requested"
        case quantityPicked = "quantity_picked"
        case status
        case pickSequence = "pick_sequence"
    }
}

struct PickWave: Decodable {
    let waveId: String
    let warehouseId: String?
    let manifestId: String
    let status: String
    let tasks: [PickTask]?

    enum CodingKeys: String, CodingKey {
        case waveId = "wave_id"
        case warehouseId = "warehouse_id"
        case manifestId = "manifest_id"
        case status
        case tasks
    }
}

struct PickWaveListResponse: Decodable {
    let waves: [PickWave]
    let pickWavesEnabled: Bool?

    enum CodingKeys: String, CodingKey {
        case waves
        case pickWavesEnabled = "pick_waves_enabled"
    }
}

struct PickWaveCreateRequest: Encodable {
    let manifestId: String

    enum CodingKeys: String, CodingKey {
        case manifestId = "manifest_id"
    }
}

struct PickTaskConfirmRequest: Encodable {
    var quantityPicked: Int?

    enum CodingKeys: String, CodingKey {
        case quantityPicked = "quantity_picked"
    }
}

struct CycleCount: Decodable, Identifiable {
    var id: String { countId }
    let countId: String
    let locationId: String?
    let productId: String
    let expectedQty: Int
    let countedQty: Int?
    let status: String

    enum CodingKeys: String, CodingKey {
        case countId = "count_id"
        case locationId = "location_id"
        case productId = "product_id"
        case expectedQty = "expected_qty"
        case countedQty = "counted_qty"
        case status
    }
}

struct CycleCountListResponse: Decodable {
    let counts: [CycleCount]
}

struct CycleCountCreateRequest: Encodable {
    let locationId: String
    let productId: String
    var expectedQty: Int?

    enum CodingKeys: String, CodingKey {
        case locationId = "location_id"
        case productId = "product_id"
        case expectedQty = "expected_qty"
    }
}

struct CycleCountSubmitRequest: Encodable {
    let countedQty: Int

    enum CodingKeys: String, CodingKey {
        case countedQty = "counted_qty"
    }
}

// MARK: - Product

struct Product: Decodable, Identifiable {
    var id: String { productId }
    let productId: String
    let name: String
    let skuId: String
    let category: String
    let priceUzs: Int

    enum CodingKeys: String, CodingKey {
        case productId = "product_id"
        case name
        case skuId = "sku_id"
        case category
        case priceUzs = "price_uzs"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        productId = try c.decode(String.self, forKey: .productId)
        name = try c.decodeIfPresent(String.self, forKey: .name) ?? ""
        skuId = try c.decodeIfPresent(String.self, forKey: .skuId) ?? ""
        category = try c.decodeIfPresent(String.self, forKey: .category) ?? ""
        priceUzs = try c.decodeIfPresent(Int.self, forKey: .priceUzs) ?? 0
    }
}

struct ProductListResponse: Decodable {
    let products: [Product]
}

// MARK: - Manifest

struct Manifest: Decodable, Identifiable {
    var id: String { manifestId }
    let manifestId: String
    let driverName: String
    let vehicleLabel: String
    let stopCount: Int
    let createdAt: String

    enum CodingKeys: String, CodingKey {
        case manifestId = "manifest_id"
        case driverName = "driver_name"
        case vehicleLabel = "vehicle_label"
        case stopCount = "stop_count"
        case createdAt = "created_at"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        manifestId = try c.decode(String.self, forKey: .manifestId)
        driverName = try c.decodeIfPresent(String.self, forKey: .driverName) ?? ""
        vehicleLabel = try c.decodeIfPresent(String.self, forKey: .vehicleLabel) ?? ""
        stopCount = try c.decodeIfPresent(Int.self, forKey: .stopCount) ?? 0
        createdAt = try c.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
    }
}

struct ManifestListResponse: Decodable {
    let manifests: [Manifest]
}

// MARK: - Analytics

struct DailyMetric: Decodable, Identifiable {
    var id: String { date }
    let date: String
    let revenue: Int
    let orders: Int
    let completed: Int

    enum CodingKeys: String, CodingKey {
        case date
        case revenue
        case orders
        case completed
    }

    init(date: String = "", revenue: Int = 0, orders: Int = 0, completed: Int = 0) {
        self.date = date
        self.revenue = revenue
        self.orders = orders
        self.completed = completed
    }
}

struct AnalyticsData: Decodable {
    let period: String
    let totalOrders: Int
    let totalRevenue: Int
    let completedOrders: Int
    let cancelledOrders: Int
    let avgOrderValue: Double
    let fleetUtilizationPct: Double
    let topProducts: [TopProduct]
    let dailyBreakdown: [DailyMetric]
    let daily: [DailyMetric]
    let importFreshness: ImportFreshness
    let importAnomalyQueue: ImportAnomalyQueue

    var chartDaily: [DailyMetric] {
        dailyBreakdown.isEmpty ? daily : dailyBreakdown
    }

    enum CodingKeys: String, CodingKey {
        case period
        case totalOrders = "total_orders"
        case totalRevenue = "total_revenue"
        case completedOrders = "completed_orders"
        case cancelledOrders = "cancelled_orders"
        case avgOrderValue = "avg_order_value"
        case fleetUtilization = "fleet_utilization"
        case fleetUtilizationPct = "fleet_utilization_pct"
        case topProducts = "top_products"
        case dailyBreakdown = "daily_breakdown"
        case daily
        case importFreshness = "import_freshness"
        case importAnomalyQueue = "import_anomaly_queue"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        period = try c.decodeIfPresent(String.self, forKey: .period) ?? ""
        totalOrders = try c.decodeIfPresent(Int.self, forKey: .totalOrders) ?? 0
        totalRevenue = try c.decodeIfPresent(Int.self, forKey: .totalRevenue) ?? 0
        completedOrders = try c.decodeIfPresent(Int.self, forKey: .completedOrders) ?? 0
        cancelledOrders = try c.decodeIfPresent(Int.self, forKey: .cancelledOrders) ?? 0
        avgOrderValue = try c.decodeIfPresent(Double.self, forKey: .avgOrderValue) ?? 0
        let fleetUtilization = try c.decodeIfPresent(FleetUtilization.self, forKey: .fleetUtilization) ?? .empty
        let fleetUtilizationPctLegacy = try c.decodeIfPresent(Double.self, forKey: .fleetUtilizationPct) ?? 0
        fleetUtilizationPct = fleetUtilization.utilizationPct > 0 ? fleetUtilization.utilizationPct : fleetUtilizationPctLegacy
        topProducts = try c.decodeIfPresent([TopProduct].self, forKey: .topProducts) ?? []
        dailyBreakdown = try c.decodeIfPresent([DailyMetric].self, forKey: .dailyBreakdown) ?? []
        daily = try c.decodeIfPresent([DailyMetric].self, forKey: .daily) ?? []
        importFreshness = try c.decodeIfPresent(ImportFreshness.self, forKey: .importFreshness) ?? .empty
        importAnomalyQueue = try c.decodeIfPresent(ImportAnomalyQueue.self, forKey: .importAnomalyQueue) ?? .empty
    }

    static let empty = AnalyticsData(
        period: "",
        totalOrders: 0,
        totalRevenue: 0,
        completedOrders: 0,
        cancelledOrders: 0,
        avgOrderValue: 0,
        fleetUtilizationPct: 0,
        topProducts: [],
        dailyBreakdown: [],
        daily: [],
        importFreshness: .empty,
        importAnomalyQueue: .empty
    )

    init(
        period: String,
        totalOrders: Int,
        totalRevenue: Int,
        completedOrders: Int,
        cancelledOrders: Int,
        avgOrderValue: Double,
        fleetUtilizationPct: Double,
        topProducts: [TopProduct],
        dailyBreakdown: [DailyMetric],
        daily: [DailyMetric],
        importFreshness: ImportFreshness,
        importAnomalyQueue: ImportAnomalyQueue
    ) {
        self.period = period
        self.totalOrders = totalOrders
        self.totalRevenue = totalRevenue
        self.completedOrders = completedOrders
        self.cancelledOrders = cancelledOrders
        self.avgOrderValue = avgOrderValue
        self.fleetUtilizationPct = fleetUtilizationPct
        self.topProducts = topProducts
        self.dailyBreakdown = dailyBreakdown
        self.daily = daily
        self.importFreshness = importFreshness
        self.importAnomalyQueue = importAnomalyQueue
    }
}

struct FleetUtilization: Decodable {
    let utilizationPct: Double

    enum CodingKeys: String, CodingKey {
        case utilizationPct = "utilization_pct"
    }

    static let empty = FleetUtilization(utilizationPct: 0)
}

struct ImportFreshness: Decodable {
    let appliedRows30d: Int
    let appliedSkus30d: Int
    let quantityDelta30d: Int
    let lastSessionId: String
    let lastAppliedAt: String

    enum CodingKeys: String, CodingKey {
        case appliedRows30d = "applied_rows_30d"
        case appliedSkus30d = "applied_skus_30d"
        case quantityDelta30d = "quantity_delta_30d"
        case lastSessionId = "last_session_id"
        case lastAppliedAt = "last_applied_at"
    }

    static let empty = ImportFreshness(
        appliedRows30d: 0,
        appliedSkus30d: 0,
        quantityDelta30d: 0,
        lastSessionId: "",
        lastAppliedAt: ""
    )
}

struct ImportAnomalyQueue: Decodable {
    let openRows30d: Int
    let affectedSessions30d: Int
    let lastSessionId: String
    let lastDetectedAt: String
    let lastDetail: String

    enum CodingKeys: String, CodingKey {
        case openRows30d = "open_rows_30d"
        case affectedSessions30d = "affected_sessions_30d"
        case lastSessionId = "last_session_id"
        case lastDetectedAt = "last_detected_at"
        case lastDetail = "last_detail"
    }

    static let empty = ImportAnomalyQueue(
        openRows30d: 0,
        affectedSessions30d: 0,
        lastSessionId: "",
        lastDetectedAt: "",
        lastDetail: ""
    )
}

struct TopProduct: Decodable, Identifiable {
    var id: String { productName }
    let productName: String
    let unitsSold: Int
    let revenue: Int

    enum CodingKeys: String, CodingKey {
        case productName = "product_name"
        case totalQty = "total_qty"
        case totalSold = "total_sold"
        case unitsSold = "units_sold"
        case revenue
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        productName = try c.decodeIfPresent(String.self, forKey: .productName) ?? ""
        unitsSold =
            try c.decodeIfPresent(Int.self, forKey: .totalQty) ??
            (try c.decodeIfPresent(Int.self, forKey: .totalSold)) ??
            (try c.decodeIfPresent(Int.self, forKey: .unitsSold)) ?? 0
        revenue = try c.decodeIfPresent(Int.self, forKey: .revenue) ?? 0
    }
}

// MARK: - Retailer (CRM)

struct Retailer: Decodable, Identifiable {
    var id: String { retailerId }
    let retailerId: String
    let name: String
    let totalOrders: Int
    let totalRevenue: Int
    let lastOrderDate: String

    enum CodingKeys: String, CodingKey {
        case retailerId = "retailer_id"
        case name
        case businessName = "business_name"
        case totalOrders = "total_orders"
        case orderCount = "order_count"
        case totalRevenue = "total_revenue"
        case revenueUzs = "revenue_uzs"
        case lastOrderDate = "last_order_date"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        retailerId = try c.decode(String.self, forKey: .retailerId)
        name = try c.decodeIfPresent(String.self, forKey: .businessName)
            ?? (try c.decodeIfPresent(String.self, forKey: .name)) ?? ""
        totalOrders = try c.decodeIfPresent(Int.self, forKey: .totalOrders)
            ?? (try c.decodeIfPresent(Int.self, forKey: .orderCount)) ?? 0
        totalRevenue = try c.decodeIfPresent(Int.self, forKey: .totalRevenue)
            ?? (try c.decodeIfPresent(Int.self, forKey: .revenueUzs)) ?? 0
        lastOrderDate = try c.decodeIfPresent(String.self, forKey: .lastOrderDate) ?? ""
    }
}

struct RetailerListResponse: Decodable {
    let retailers: [Retailer]
}

// MARK: - Return

struct ReturnItem: Decodable, Identifiable {
    var id: String { lineItemId }
    let lineItemId: String
    let orderId: String
    let productName: String
    let quantity: Int
    let status: String
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case lineItemId = "line_item_id"
        case returnId = "return_id"
        case orderId = "order_id"
        case productName = "product_name"
        case quantity, status
        case updatedAt = "updated_at"
        case createdAt = "created_at"
        case reason
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        lineItemId = try c.decodeIfPresent(String.self, forKey: .lineItemId)
            ?? (try c.decodeIfPresent(String.self, forKey: .returnId)) ?? ""
        orderId = try c.decodeIfPresent(String.self, forKey: .orderId) ?? ""
        productName = try c.decodeIfPresent(String.self, forKey: .productName) ?? ""
        quantity = try c.decodeIfPresent(Int.self, forKey: .quantity) ?? 0
        status = try c.decodeIfPresent(String.self, forKey: .status)
            ?? (try c.decodeIfPresent(String.self, forKey: .reason)) ?? ""
        updatedAt = try c.decodeIfPresent(String.self, forKey: .updatedAt)
            ?? (try c.decodeIfPresent(String.self, forKey: .createdAt)) ?? ""
    }
}

struct ReturnListResponse: Decodable {
    let items: [ReturnItem]

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        if let decoded = try c.decodeIfPresent([ReturnItem].self, forKey: .items) {
            items = decoded
            return
        }
        items = try c.decodeIfPresent([ReturnItem].self, forKey: .returns) ?? []
    }

    private enum CodingKeys: String, CodingKey {
        case items, returns
    }
}

struct InboundReturnRow: Decodable, Identifiable {
    var id: String { returnId }
    let returnId: String
    let orderId: String
    let productName: String
    let expectedQty: Int
    let receivedQty: Int
    let reason: String
    let physicalStatus: String
    let driverName: String
    let driverNotes: String
    let suggestedDisposition: String
    let barcode: String?

    var isClaimTicket: Bool {
        let notes = driverNotes.lowercased()
        return notes.contains("claim_id=")
            || notes.contains("source=retailer_claim")
            || notes.contains("source=claim")
    }

    enum CodingKeys: String, CodingKey {
        case returnId = "return_id"
        case orderId = "order_id"
        case productName = "product_name"
        case expectedQty = "expected_qty"
        case receivedQty = "received_qty"
        case reason
        case physicalStatus = "physical_status"
        case driverName = "driver_name"
        case driverNotes = "driver_notes"
        case suggestedDisposition = "suggested_disposition"
        case barcode
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        returnId = try c.decode(String.self, forKey: .returnId)
        orderId = try c.decodeIfPresent(String.self, forKey: .orderId) ?? ""
        productName = try c.decodeIfPresent(String.self, forKey: .productName) ?? ""
        expectedQty = try c.decodeIfPresent(Int.self, forKey: .expectedQty) ?? 0
        receivedQty = try c.decodeIfPresent(Int.self, forKey: .receivedQty) ?? 0
        reason = try c.decodeIfPresent(String.self, forKey: .reason) ?? ""
        physicalStatus = try c.decodeIfPresent(String.self, forKey: .physicalStatus) ?? ""
        driverName = try c.decodeIfPresent(String.self, forKey: .driverName) ?? ""
        driverNotes = try c.decodeIfPresent(String.self, forKey: .driverNotes) ?? ""
        suggestedDisposition = try c.decodeIfPresent(String.self, forKey: .suggestedDisposition) ?? ""
        barcode = try c.decodeIfPresent(String.self, forKey: .barcode)
    }
}

// MARK: - Credit-note reverse logistics

struct ReverseLogisticsTask: Decodable, Identifiable {
    var id: String { taskId }
    let taskId: String
    let orderId: String
    let status: String
    let warehouseId: String
    let expectedQtyJson: String?

    enum CodingKeys: String, CodingKey {
        case taskId = "task_id"
        case orderId = "order_id"
        case status
        case warehouseId = "warehouse_id"
        case expectedQtyJson = "expected_qty_json"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        taskId = try c.decodeIfPresent(String.self, forKey: .taskId) ?? ""
        orderId = try c.decodeIfPresent(String.self, forKey: .orderId) ?? ""
        status = try c.decodeIfPresent(String.self, forKey: .status) ?? ""
        warehouseId = try c.decodeIfPresent(String.self, forKey: .warehouseId) ?? ""
        expectedQtyJson = try c.decodeIfPresent(String.self, forKey: .expectedQtyJson)
    }
}

struct ReverseLogisticsListResponse: Decodable {
    let tasks: [ReverseLogisticsTask]

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        tasks = try c.decodeIfPresent([ReverseLogisticsTask].self, forKey: .tasks) ?? []
    }

    private enum CodingKeys: String, CodingKey { case tasks }
}

struct ReverseLogisticsReceiveRequest: Encodable {
    let warehouseId: String
    let receivedQty: [String: Int]

    enum CodingKeys: String, CodingKey {
        case warehouseId = "warehouse_id"
        case receivedQty = "received_qty"
    }
}

// MARK: - Ops exceptions triage

struct DeliveryExpectationWire: Decodable {
    let targetLabel: String

    enum CodingKeys: String, CodingKey {
        case targetLabel = "target_label"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        targetLabel = try c.decodeIfPresent(String.self, forKey: .targetLabel) ?? ""
    }
}

struct WarehouseOpsException: Decodable, Identifiable {
    var id: String {
        if !exceptionId.isEmpty { return exceptionId }
        return "\(kind)-\(orderId)-\(manifestId)-\(updatedAt)"
    }

    let exceptionId: String
    let kind: String
    let orderId: String
    let manifestId: String
    let reason: String
    let status: String
    let updatedAt: String
    let deliveryExpectation: DeliveryExpectationWire?

    enum CodingKeys: String, CodingKey {
        case exceptionId = "exception_id"
        case kind
        case orderId = "order_id"
        case manifestId = "manifest_id"
        case reason, status
        case updatedAt = "updated_at"
        case deliveryExpectation = "delivery_expectation"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        exceptionId = try c.decodeIfPresent(String.self, forKey: .exceptionId) ?? ""
        kind = try c.decodeIfPresent(String.self, forKey: .kind) ?? ""
        orderId = try c.decodeIfPresent(String.self, forKey: .orderId) ?? ""
        manifestId = try c.decodeIfPresent(String.self, forKey: .manifestId) ?? ""
        reason = try c.decodeIfPresent(String.self, forKey: .reason) ?? ""
        status = try c.decodeIfPresent(String.self, forKey: .status) ?? ""
        updatedAt = try c.decodeIfPresent(String.self, forKey: .updatedAt) ?? ""
        deliveryExpectation = try c.decodeIfPresent(DeliveryExpectationWire.self, forKey: .deliveryExpectation)
    }
}

struct WarehouseOpsExceptionsResponse: Decodable {
    let exceptions: [WarehouseOpsException]

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        exceptions = try c.decodeIfPresent([WarehouseOpsException].self, forKey: .exceptions) ?? []
    }

    private enum CodingKeys: String, CodingKey { case exceptions }
}

// MARK: - Claims (read-only)

struct WarehouseClaimLine: Decodable, Identifiable {
    var id: String { "\(sku)-\(quantity)" }
    let sku: String
    let quantity: Int
    let reason: String
    let amountMinor: Int

    enum CodingKeys: String, CodingKey {
        case sku, quantity, reason
        case amountMinor = "amount_minor"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        sku = try c.decodeIfPresent(String.self, forKey: .sku) ?? ""
        quantity = try c.decodeIfPresent(Int.self, forKey: .quantity) ?? 0
        reason = try c.decodeIfPresent(String.self, forKey: .reason) ?? ""
        amountMinor = try c.decodeIfPresent(Int.self, forKey: .amountMinor) ?? 0
    }
}

struct WarehouseClaim: Decodable, Identifiable {
    var id: String { claimId }
    let claimId: String
    let orderId: String
    let retailerId: String
    let claimType: String
    let status: String
    let amountMinor: Int
    let currency: String
    let description: String
    let lineItems: [WarehouseClaimLine]
    let createdAt: String

    enum CodingKeys: String, CodingKey {
        case claimId = "claim_id"
        case orderId = "order_id"
        case retailerId = "retailer_id"
        case claimType = "claim_type"
        case status
        case amountMinor = "amount_minor"
        case currency, description
        case lineItems = "line_items"
        case createdAt = "created_at"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        claimId = try c.decodeIfPresent(String.self, forKey: .claimId) ?? ""
        orderId = try c.decodeIfPresent(String.self, forKey: .orderId) ?? ""
        retailerId = try c.decodeIfPresent(String.self, forKey: .retailerId) ?? ""
        claimType = try c.decodeIfPresent(String.self, forKey: .claimType) ?? ""
        status = try c.decodeIfPresent(String.self, forKey: .status) ?? ""
        amountMinor = try c.decodeIfPresent(Int.self, forKey: .amountMinor) ?? 0
        currency = try c.decodeIfPresent(String.self, forKey: .currency) ?? packCurrency(MarketPackStore.pack)
        description = try c.decodeIfPresent(String.self, forKey: .description) ?? ""
        lineItems = try c.decodeIfPresent([WarehouseClaimLine].self, forKey: .lineItems) ?? []
        createdAt = try c.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
    }
}

struct WarehouseClaimsResponse: Decodable {
    let claims: [WarehouseClaim]

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        claims = try c.decodeIfPresent([WarehouseClaim].self, forKey: .claims) ?? []
    }

    private enum CodingKeys: String, CodingKey { case claims }
}

// MARK: - Fleet rescue

struct RescuePreviewRequest: Encodable {
    let brokenDriverId: String

    enum CodingKeys: String, CodingKey {
        case brokenDriverId = "broken_driver_id"
    }
}

struct RescueOption: Decodable, Identifiable {
    var id: String { driverId }
    let driverId: String
    let name: String
    let licensePlate: String
    let truckStatus: String
    let effectiveCapacityVu: Double
    let isCapacityExceeded: Bool

    enum CodingKeys: String, CodingKey {
        case driverId = "driver_id"
        case name
        case licensePlate = "license_plate"
        case truckStatus = "truck_status"
        case effectiveCapacityVu = "effective_capacity_vu"
        case isCapacityExceeded = "is_capacity_exceeded"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        driverId = try c.decodeIfPresent(String.self, forKey: .driverId) ?? ""
        name = try c.decodeIfPresent(String.self, forKey: .name) ?? ""
        licensePlate = try c.decodeIfPresent(String.self, forKey: .licensePlate) ?? ""
        truckStatus = try c.decodeIfPresent(String.self, forKey: .truckStatus) ?? ""
        effectiveCapacityVu = try c.decodeIfPresent(Double.self, forKey: .effectiveCapacityVu) ?? 0
        isCapacityExceeded = try c.decodeIfPresent(Bool.self, forKey: .isCapacityExceeded) ?? false
    }
}

struct RescuePreviewResponse: Decodable {
    let brokenDriverId: String
    let pendingVolumeVu: Double
    let rescueOptions: [RescueOption]

    enum CodingKeys: String, CodingKey {
        case brokenDriverId = "broken_driver_id"
        case pendingVolumeVu = "pending_volume_vu"
        case rescueOptions = "rescue_options"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        brokenDriverId = try c.decodeIfPresent(String.self, forKey: .brokenDriverId) ?? ""
        pendingVolumeVu = try c.decodeIfPresent(Double.self, forKey: .pendingVolumeVu) ?? 0
        rescueOptions = try c.decodeIfPresent([RescueOption].self, forKey: .rescueOptions) ?? []
    }
}

struct RescueProposeRequest: Encodable {
    let rescueId: String
    let brokenDriverId: String
    let rescueDriverId: String
    let forceCapacity: Bool

    enum CodingKeys: String, CodingKey {
        case rescueId = "rescue_id"
        case brokenDriverId = "broken_driver_id"
        case rescueDriverId = "rescue_driver_id"
        case forceCapacity = "force_capacity"
    }
}

struct InboundReturnListResponse: Decodable {
    let data: [InboundReturnRow]
}

struct InboundSessionResponse: Decodable {
    let sessionId: String

    enum CodingKeys: String, CodingKey {
        case sessionId = "session_id"
    }
}

struct InboundScanBody: Encodable {
    let barcode: String
    let qty: Int
    let sessionId: String

    enum CodingKeys: String, CodingKey {
        case barcode, qty
        case sessionId = "session_id"
    }
}

struct InboundConfirmLine: Encodable {
    let returnId: String
    let disposition: String

    enum CodingKeys: String, CodingKey {
        case returnId = "return_id"
        case disposition
    }
}

struct InboundConfirmBody: Encodable {
    let lines: [InboundConfirmLine]
    let sessionId: String

    enum CodingKeys: String, CodingKey {
        case lines
        case sessionId = "session_id"
    }
}

struct InboundConfirmResponse: Decodable {
    let status: String?
}

struct InboundScanResponse: Decodable {
    let matched: Bool
    let returnId: String?
    let variance: Bool
    let message: String?

    enum CodingKeys: String, CodingKey {
        case matched
        case returnId = "return_id"
        case variance, message
    }
}

// MARK: - Treasury

struct TreasuryOverview: Decodable {
    let balance: Int
    let totalReceivable: Int
    let totalCollected: Int
    let overdueAmount: Int

    enum CodingKeys: String, CodingKey {
        case balance
        case totalReceivable = "total_receivable"
        case totalCollected = "total_collected"
        case overdueAmount = "overdue_amount"
        case totalInvoiced = "total_invoiced"
        case totalPaid = "total_paid"
        case totalOutstanding = "total_outstanding"
        case invoicedUzs = "invoiced_uzs"
        case paidUzs = "paid_uzs"
        case outstandingUzs = "outstanding_uzs"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        let invoiced = try c.decodeIfPresent(Int.self, forKey: .totalInvoiced)
            ?? (try c.decodeIfPresent(Int.self, forKey: .invoicedUzs))
            ?? (try c.decodeIfPresent(Int.self, forKey: .totalReceivable)) ?? 0
        let paid = try c.decodeIfPresent(Int.self, forKey: .totalPaid)
            ?? (try c.decodeIfPresent(Int.self, forKey: .paidUzs))
            ?? (try c.decodeIfPresent(Int.self, forKey: .totalCollected)) ?? 0
        let outstanding = try c.decodeIfPresent(Int.self, forKey: .totalOutstanding)
            ?? (try c.decodeIfPresent(Int.self, forKey: .outstandingUzs))
            ?? (try c.decodeIfPresent(Int.self, forKey: .overdueAmount)) ?? 0
        balance = try c.decodeIfPresent(Int.self, forKey: .balance) ?? paid
        totalReceivable = invoiced
        totalCollected = paid
        overdueAmount = outstanding
    }

    static let empty = TreasuryOverview(balance: 0, totalReceivable: 0, totalCollected: 0, overdueAmount: 0)

    init(balance: Int, totalReceivable: Int, totalCollected: Int, overdueAmount: Int) {
        self.balance = balance
        self.totalReceivable = totalReceivable
        self.totalCollected = totalCollected
        self.overdueAmount = overdueAmount
    }
}

struct Invoice: Decodable, Identifiable {
    var id: String { invoiceId }
    let invoiceId: String
    let retailerName: String
    let amountUzs: Int
    let currency: String
    let status: String
    let dueDate: String
    let feeAmount: Int
    let netPayoutAmount: Int
    let payoutOwnerType: String
    let payoutOwnerId: String
    let feePolicyVersion: String
    let settlementTarget: String

    enum CodingKeys: String, CodingKey {
        case invoiceId = "invoice_id"
        case retailerName = "retailer_name"
        case amount = "amount"
        case amountUzs = "amount_uzs"
        case currency
        case status
        case dueDate = "due_date"
        case feeAmount = "fee_amount"
        case netPayoutAmount = "net_payout_amount"
        case payoutOwnerType = "payout_owner_type"
        case payoutOwnerId = "payout_owner_id"
        case feePolicyVersion = "fee_policy_version"
        case settlementTarget = "settlement_target"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        invoiceId = try c.decode(String.self, forKey: .invoiceId)
        retailerName = try c.decodeIfPresent(String.self, forKey: .retailerName) ?? ""
        let additiveAmount = try c.decodeIfPresent(Int.self, forKey: .amount)
        let legacyAmount = try c.decodeIfPresent(Int.self, forKey: .amountUzs)
        amountUzs = additiveAmount ?? legacyAmount ?? 0
        currency = displayPackCurrency(try c.decodeIfPresent(String.self, forKey: .currency))
        status = try c.decodeIfPresent(String.self, forKey: .status) ?? ""
        dueDate = try c.decodeIfPresent(String.self, forKey: .dueDate) ?? ""
        feeAmount = try c.decodeIfPresent(Int.self, forKey: .feeAmount) ?? 0
        netPayoutAmount = try c.decodeIfPresent(Int.self, forKey: .netPayoutAmount) ?? 0
        payoutOwnerType = try c.decodeIfPresent(String.self, forKey: .payoutOwnerType) ?? ""
        payoutOwnerId = try c.decodeIfPresent(String.self, forKey: .payoutOwnerId) ?? ""
        feePolicyVersion = try c.decodeIfPresent(String.self, forKey: .feePolicyVersion) ?? ""
        settlementTarget = try c.decodeIfPresent(String.self, forKey: .settlementTarget) ?? ""
    }
}

struct InvoiceListResponse: Decodable {
    let invoices: [Invoice]
}

// MARK: - Dispatch

struct DispatchPreview: Decodable {
    let undispatchedOrders: [DispatchOrder]
    let availableDrivers: [AvailableDriver]
    let unavailableDrivers: [AvailableDriver]
    let proposedRoutes: [DispatchProposedRoute]
    let optimizerSource: String?
    let optimizerWarnings: [String]
    let windowConstrainedCount: Int
    let fleetEffectiveCapacityVu: Double
    let planFingerprint: String?

    enum CodingKeys: String, CodingKey {
        case undispatchedOrders = "undispatched_orders"
        case availableDrivers = "available_drivers"
        case unavailableDrivers = "unavailable_drivers"
        case proposedRoutes = "proposed_routes"
        case optimizerSource = "optimizer_source"
        case optimizerWarnings = "optimizer_warnings"
        case windowConstrainedCount = "window_constrained_count"
        case fleetEffectiveCapacityVu = "fleet_effective_capacity_vu"
        case planFingerprint = "plan_fingerprint"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        undispatchedOrders = try c.decodeIfPresent([DispatchOrder].self, forKey: .undispatchedOrders) ?? []
        availableDrivers = try c.decodeIfPresent([AvailableDriver].self, forKey: .availableDrivers) ?? []
        unavailableDrivers = try c.decodeIfPresent([AvailableDriver].self, forKey: .unavailableDrivers) ?? []
        proposedRoutes = try c.decodeIfPresent([DispatchProposedRoute].self, forKey: .proposedRoutes) ?? []
        optimizerSource = try c.decodeIfPresent(String.self, forKey: .optimizerSource)
        optimizerWarnings = try c.decodeIfPresent([String].self, forKey: .optimizerWarnings) ?? []
        windowConstrainedCount = try c.decodeIfPresent(Int.self, forKey: .windowConstrainedCount) ?? 0
        fleetEffectiveCapacityVu = try c.decodeIfPresent(Double.self, forKey: .fleetEffectiveCapacityVu) ?? 0
        planFingerprint = try c.decodeIfPresent(String.self, forKey: .planFingerprint)
    }
}

struct DispatchProposedRoute: Decodable, Identifiable {
    var id: String { "\(driverId ?? "route")-\(orderIds.joined(separator: "-"))" }
    let driverId: String?
    let driverName: String?
    let vehicleId: String?
    let orderIds: [String]
    let stops: [DispatchProposedStop]
    let volumeVu: Double?
    let loadedVolume: Double?
    let maxVolumeVu: Double?
    let stopCount: Int?
    let routeGeometry: RouteGeometryWire?

    enum CodingKeys: String, CodingKey {
        case driverId = "driver_id"
        case driverName = "driver_name"
        case vehicleId = "vehicle_id"
        case orderIds = "order_ids"
        case stops
        case volumeVu = "volume_vu"
        case loadedVolume = "loaded_volume"
        case maxVolumeVu = "max_volume_vu"
        case stopCount = "stop_count"
        case routeGeometry = "route_geometry"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        driverId = try c.decodeIfPresent(String.self, forKey: .driverId)
        driverName = try c.decodeIfPresent(String.self, forKey: .driverName)
        vehicleId = try c.decodeIfPresent(String.self, forKey: .vehicleId)
        orderIds = try c.decodeIfPresent([String].self, forKey: .orderIds) ?? []
        stops = try c.decodeIfPresent([DispatchProposedStop].self, forKey: .stops) ?? []
        volumeVu = try c.decodeIfPresent(Double.self, forKey: .volumeVu)
        loadedVolume = try c.decodeIfPresent(Double.self, forKey: .loadedVolume)
        maxVolumeVu = try c.decodeIfPresent(Double.self, forKey: .maxVolumeVu)
        stopCount = try c.decodeIfPresent(Int.self, forKey: .stopCount)
        routeGeometry = try c.decodeIfPresent(RouteGeometryWire.self, forKey: .routeGeometry)
    }
}

struct DispatchProposedStop: Decodable {
    let orderId: String
    let retailerId: String?
    let retailerName: String?
    let lat: Double?
    let lng: Double?
    let volumeVu: Double?

    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
        case retailerId = "retailer_id"
        case retailerName = "retailer_name"
        case lat, lng
        case volumeVu = "volume_vu"
    }
}

struct DispatchOrder: Decodable, Identifiable {
    var id: String { orderId }
    let orderId: String
    let retailerName: String
    let totalUzs: Int
    let itemCount: Int
    let volumeVu: Double

    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
        case retailerName = "retailer_name"
        case totalUzs = "total_uzs"
        case itemCount = "item_count"
        case volumeVu = "volume_vu"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        orderId = try c.decode(String.self, forKey: .orderId)
        retailerName = try c.decodeIfPresent(String.self, forKey: .retailerName) ?? ""
        totalUzs = try c.decodeIfPresent(Int.self, forKey: .totalUzs) ?? 0
        itemCount = try c.decodeIfPresent(Int.self, forKey: .itemCount) ?? 0
        volumeVu = try c.decodeIfPresent(Double.self, forKey: .volumeVu) ?? 0
    }
}

struct AvailableDriver: Decodable, Identifiable {
    var id: String { driverId }
    let driverId: String
    let name: String
    let phone: String
    let vehicleLabel: String
    let truckStatus: String
    let maxVolumeVu: Double
    let usedVolumeVu: Double?
    let freeVolumeVu: Double?
    let activeManifestId: String?
    let unavailableReason: String?

    enum CodingKeys: String, CodingKey {
        case driverId = "driver_id"
        case name
        case phone
        case vehicleLabel = "vehicle_label"
        case truckStatus = "truck_status"
        case maxVolumeVu = "max_volume_vu"
        case usedVolumeVu = "used_volume_vu"
        case freeVolumeVu = "free_volume_vu"
        case activeManifestId = "active_manifest_id"
        case unavailableReason = "unavailable_reason"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        driverId = try c.decode(String.self, forKey: .driverId)
        name = try c.decodeIfPresent(String.self, forKey: .name) ?? ""
        phone = try c.decodeIfPresent(String.self, forKey: .phone) ?? ""
        vehicleLabel = try c.decodeIfPresent(String.self, forKey: .vehicleLabel) ?? ""
        truckStatus = try c.decodeIfPresent(String.self, forKey: .truckStatus) ?? ""
        maxVolumeVu = try c.decodeIfPresent(Double.self, forKey: .maxVolumeVu) ?? 0
        usedVolumeVu = try c.decodeIfPresent(Double.self, forKey: .usedVolumeVu)
        freeVolumeVu = try c.decodeIfPresent(Double.self, forKey: .freeVolumeVu)
        activeManifestId = try c.decodeIfPresent(String.self, forKey: .activeManifestId)
        unavailableReason = try c.decodeIfPresent(String.self, forKey: .unavailableReason)
    }
}

struct DispatchCapacityWarning: Decodable {
    let driverId: String
    let loadedVu: Double
    let maxVolumeVu: Double
    let effectiveMaxVu: Double
    let excessVu: Double
    let suggestedUnselectOrderIds: [String]
    let suggestedDeferOrderIds: [String]

    enum CodingKeys: String, CodingKey {
        case driverId = "driver_id"
        case loadedVu = "loaded_vu"
        case maxVolumeVu = "max_volume_vu"
        case effectiveMaxVu = "effective_max_vu"
        case excessVu = "excess_vu"
        case suggestedUnselectOrderIds = "suggested_unselect_order_ids"
        case suggestedDeferOrderIds = "suggested_defer_order_ids"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        driverId = try c.decodeIfPresent(String.self, forKey: .driverId) ?? ""
        loadedVu = try c.decodeIfPresent(Double.self, forKey: .loadedVu) ?? 0
        maxVolumeVu = try c.decodeIfPresent(Double.self, forKey: .maxVolumeVu) ?? 0
        effectiveMaxVu = try c.decodeIfPresent(Double.self, forKey: .effectiveMaxVu) ?? 0
        excessVu = try c.decodeIfPresent(Double.self, forKey: .excessVu) ?? 0
        suggestedUnselectOrderIds = try c.decodeIfPresent([String].self, forKey: .suggestedUnselectOrderIds) ?? []
        suggestedDeferOrderIds = try c.decodeIfPresent([String].self, forKey: .suggestedDeferOrderIds) ?? []
    }
}

struct DispatchExecuteResponse: Decodable {
    let status: String
    let ordersAssigned: Int
    let warnings: [String]
    let capacityWarnings: [DispatchCapacityWarning]
    let orphanOrderIds: [String]

    enum CodingKeys: String, CodingKey {
        case status
        case ordersAssigned = "orders_assigned"
        case warnings
        case capacityWarnings = "capacity_warnings"
        case orphanOrderIds = "orphan_order_ids"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        status = try c.decodeIfPresent(String.self, forKey: .status) ?? ""
        ordersAssigned = try c.decodeIfPresent(Int.self, forKey: .ordersAssigned) ?? 0
        warnings = try c.decodeIfPresent([String].self, forKey: .warnings) ?? []
        capacityWarnings = try c.decodeIfPresent([DispatchCapacityWarning].self, forKey: .capacityWarnings) ?? []
        orphanOrderIds = try c.decodeIfPresent([String].self, forKey: .orphanOrderIds) ?? []
    }
}

struct DispatchExecuteRouteRequest: Encodable {
    let driverId: String
    let orderIds: [String]

    enum CodingKeys: String, CodingKey {
        case driverId = "driver_id"
        case orderIds = "order_ids"
    }
}

struct DispatchExecuteRequest: Encodable {
    let mode: String
    let forceCapacity: Bool
    let acceptPartial: Bool?
    let orderIds: [String]?
    let planFingerprint: String?
    let routes: [DispatchExecuteRouteRequest]?

    enum CodingKeys: String, CodingKey {
        case mode
        case forceCapacity = "force_capacity"
        case acceptPartial = "accept_partial"
        case orderIds = "order_ids"
        case planFingerprint = "plan_fingerprint"
        case routes
    }
}

// MARK: - Warehouse Realtime

struct SupplyRequestListResponse: Decodable {
    let requests: [WarehouseSupplyRequest]

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        if let rows = try? c.decode([WarehouseSupplyRequest].self, forKey: .requests) {
            requests = rows
            return
        }
        if let rows = try? c.decode([WarehouseSupplyRequest].self, forKey: .supplyRequests) {
            requests = rows
            return
        }
        requests = []
    }

    enum CodingKeys: String, CodingKey {
        case requests
        case supplyRequests = "supply_requests"
    }
}

struct DemandForecastDay: Decodable, Identifiable {
    var id: String { date }
    let date: String
    let projectedUnits: Int64
    let projectedRevenue: Int64
    let committedUnits: Int64
    let pendingConfirmationUnits: Int64

    enum CodingKeys: String, CodingKey {
        case date
        case projectedUnits = "projected_units"
        case projectedRevenue = "projected_revenue"
        case committedUnits = "committed_units"
        case pendingConfirmationUnits = "pending_confirmation_units"
    }
}

// mirror of backend-go/warehouse demandForecastProduct (keep JSON tags aligned)
struct DemandForecastSources: Decodable {
    let incomingOrders: Int64
    let aiPrediction: Int64
    let preOrders: Int64
    let burnRate: Double

    enum CodingKeys: String, CodingKey {
        case incomingOrders = "incoming_orders"
        case aiPrediction = "ai_prediction"
        case preOrders = "pre_orders"
        case burnRate = "burn_rate"
    }

    init(incomingOrders: Int64 = 0, aiPrediction: Int64 = 0, preOrders: Int64 = 0, burnRate: Double = 0) {
        self.incomingOrders = incomingOrders
        self.aiPrediction = aiPrediction
        self.preOrders = preOrders
        self.burnRate = burnRate
    }
}

struct DemandForecastProduct: Decodable, Identifiable {
    var id: String { productId }
    let productId: String
    let productName: String
    let currentStock: Int64
    let recommendedQty: Int64
    let daysUntilStockout: Double
    let priority: String
    let unit: String
    let sources: DemandForecastSources
    let demandBreakdown: [String: AnyCodable]?

    enum CodingKeys: String, CodingKey {
        case productId = "product_id"
        case productName = "product_name"
        case currentStock = "current_stock"
        case recommendedQty = "recommended_qty"
        case daysUntilStockout = "days_until_stockout"
        case priority
        case unit
        case sources
        case demandBreakdown = "demand_breakdown"
    }

    init(
        productId: String,
        productName: String,
        currentStock: Int64,
        recommendedQty: Int64,
        daysUntilStockout: Double,
        priority: String,
        unit: String,
        sources: DemandForecastSources,
        demandBreakdown: [String: AnyCodable]? = nil
    ) {
        self.productId = productId
        self.productName = productName
        self.currentStock = currentStock
        self.recommendedQty = recommendedQty
        self.daysUntilStockout = daysUntilStockout
        self.priority = priority
        self.unit = unit
        self.sources = sources
        self.demandBreakdown = demandBreakdown
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        productId = try container.decodeIfPresent(String.self, forKey: .productId) ?? ""
        productName = try container.decodeIfPresent(String.self, forKey: .productName) ?? ""
        currentStock = try container.decodeIfPresent(Int64.self, forKey: .currentStock) ?? 0
        recommendedQty = try container.decodeIfPresent(Int64.self, forKey: .recommendedQty) ?? 0
        daysUntilStockout = try container.decodeIfPresent(Double.self, forKey: .daysUntilStockout) ?? 0
        priority = try container.decodeIfPresent(String.self, forKey: .priority) ?? ""
        unit = try container.decodeIfPresent(String.self, forKey: .unit) ?? ""
        sources = try container.decodeIfPresent(DemandForecastSources.self, forKey: .sources) ?? DemandForecastSources()
        demandBreakdown = try container.decodeIfPresent([String: AnyCodable].self, forKey: .demandBreakdown)
    }
}

struct DemandForecastResponse: Decodable {
    let warehouseId: String
    let forecastDays: Int
    let generatedAt: String?
    let series: [DemandForecastDay]
    let products: [DemandForecastProduct]
    let source: String

    enum CodingKeys: String, CodingKey {
        case warehouseId = "warehouse_id"
        case forecastDays = "forecast_days"
        case generatedAt = "generated_at"
        case series
        case products
        case source
    }

    init(
        warehouseId: String = "",
        forecastDays: Int = 7,
        generatedAt: String? = nil,
        series: [DemandForecastDay] = [],
        products: [DemandForecastProduct] = [],
        source: String = "empty"
    ) {
        self.warehouseId = warehouseId
        self.forecastDays = forecastDays
        self.generatedAt = generatedAt
        self.series = series
        self.products = products
        self.source = source
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        warehouseId = try container.decodeIfPresent(String.self, forKey: .warehouseId) ?? ""
        forecastDays = try container.decodeIfPresent(Int.self, forKey: .forecastDays) ?? 7
        generatedAt = try container.decodeIfPresent(String.self, forKey: .generatedAt)
        series = try container.decodeIfPresent([DemandForecastDay].self, forKey: .series) ?? []
        products = try container.decodeIfPresent([DemandForecastProduct].self, forKey: .products) ?? []
        source = try container.decodeIfPresent(String.self, forKey: .source) ?? "empty"
    }
}

struct WarehouseSupplyRequest: Decodable, Identifiable {
    var id: String { requestId }
    let requestId: String
    let warehouseId: String
    let factoryId: String
    let supplierId: String
    let state: String
    let priority: String
    let requestedDeliveryDate: String?
    let totalVolumeVu: Double
    let notes: String
    let transferOrderId: String?
    let createdBy: String
    let createdAt: String
    let updatedAt: String?

    enum CodingKeys: String, CodingKey {
        case requestId = "request_id"
        case warehouseId = "warehouse_id"
        case factoryId = "factory_id"
        case supplierId = "supplier_id"
        case state
        case priority
        case requestedDeliveryDate = "requested_delivery_date"
        case totalVolumeVu = "total_volume_vu"
        case notes
        case transferOrderId = "transfer_order_id"
        case createdBy = "created_by"
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        requestId = try c.decode(String.self, forKey: .requestId)
        warehouseId = try c.decodeIfPresent(String.self, forKey: .warehouseId) ?? ""
        factoryId = try c.decodeIfPresent(String.self, forKey: .factoryId) ?? ""
        supplierId = try c.decodeIfPresent(String.self, forKey: .supplierId) ?? ""
        state = try c.decodeIfPresent(String.self, forKey: .state) ?? ""
        priority = try c.decodeIfPresent(String.self, forKey: .priority) ?? ""
        requestedDeliveryDate = try c.decodeIfPresent(String.self, forKey: .requestedDeliveryDate)
        totalVolumeVu = try c.decodeIfPresent(Double.self, forKey: .totalVolumeVu) ?? 0
        notes = try c.decodeIfPresent(String.self, forKey: .notes) ?? ""
        transferOrderId = try c.decodeIfPresent(String.self, forKey: .transferOrderId)
        createdBy = try c.decodeIfPresent(String.self, forKey: .createdBy) ?? ""
        createdAt = try c.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
        updatedAt = try c.decodeIfPresent(String.self, forKey: .updatedAt)
    }
}

struct SupplyRequestQCResponse: Decodable {
    let requestId: String
    let result: String

    enum CodingKeys: String, CodingKey {
        case requestId = "request_id"
        case result
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        requestId = try c.decodeIfPresent(String.self, forKey: .requestId) ?? ""
        result = try c.decodeIfPresent(String.self, forKey: .result) ?? ""
    }
}

struct WarehouseDispatchLock: Decodable, Identifiable {
    var id: String { lockId }
    let lockId: String
    let supplierId: String
    let warehouseId: String
    let factoryId: String
    let lockType: String
    let lockedAt: String
    let unlockedAt: String?
    let lockedBy: String

    enum CodingKeys: String, CodingKey {
        case lockId = "lock_id"
        case supplierId = "supplier_id"
        case warehouseId = "warehouse_id"
        case factoryId = "factory_id"
        case lockType = "lock_type"
        case lockedAt = "locked_at"
        case unlockedAt = "unlocked_at"
        case lockedBy = "locked_by"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        lockId = try c.decode(String.self, forKey: .lockId)
        supplierId = try c.decodeIfPresent(String.self, forKey: .supplierId) ?? ""
        warehouseId = try c.decodeIfPresent(String.self, forKey: .warehouseId) ?? ""
        factoryId = try c.decodeIfPresent(String.self, forKey: .factoryId) ?? ""
        lockType = try c.decodeIfPresent(String.self, forKey: .lockType) ?? ""
        lockedAt = try c.decodeIfPresent(String.self, forKey: .lockedAt) ?? ""
        unlockedAt = try c.decodeIfPresent(String.self, forKey: .unlockedAt)
        lockedBy = try c.decodeIfPresent(String.self, forKey: .lockedBy) ?? ""
    }
}

struct WarehouseLiveEvent: Decodable {
    let type: String
    let warehouseId: String
    let requestId: String?
    let state: String?
    let lockId: String?
    let action: String?
    let timestamp: String?

    enum CodingKeys: String, CodingKey {
        case type
        case warehouseId = "warehouse_id"
        case requestId = "request_id"
        case state
        case lockId = "lock_id"
        case action
        case timestamp
    }
}

struct CreateWarehouseSupplyRequestItem: Encodable {
    let productId: String
    let requestedQuantity: Int
    let recommendedQty: Int
    let unitVolumeVu: Double

    enum CodingKeys: String, CodingKey {
        case productId = "product_id"
        case requestedQuantity = "requested_quantity"
        case recommendedQty = "recommended_qty"
        case unitVolumeVu = "unit_volume_vu"
    }
}

struct CreateWarehouseSupplyRequestRequest: Encodable {
    let factoryId: String
    let priority: String
    let notes: String
    let items: [CreateWarehouseSupplyRequestItem]
    let useDemandForecast: Bool
    let requestedDeliveryDate: String?

    enum CodingKeys: String, CodingKey {
        case factoryId = "factory_id"
        case priority
        case notes
        case items
        case useDemandForecast = "use_demand_forecast"
        case requestedDeliveryDate = "requested_delivery_date"
    }
}

struct CreateWarehouseSupplyRequestResponse: Decodable {
    let requestId: String
    let state: String
    let priority: String
    let totalVolumeVu: Double
    let itemsCount: Int

    enum CodingKeys: String, CodingKey {
        case requestId = "request_id"
        case state
        case priority
        case totalVolumeVu = "total_volume_vu"
        case itemsCount = "items_count"
    }
}

struct WarehouseSupplyRequestTransitionRequest: Encodable {
    let action: String
    let transferOrderId: String?

    enum CodingKeys: String, CodingKey {
        case action
        case transferOrderId = "transfer_order_id"
    }
}

struct WarehouseSupplyRequestTransitionResponse: Decodable {
    let requestId: String
    let state: String

    enum CodingKeys: String, CodingKey {
        case requestId = "request_id"
        case state
    }
}

struct CreateWarehouseDispatchLockRequest: Encodable {
    let lockType: String

    enum CodingKeys: String, CodingKey {
        case lockType = "lock_type"
    }
}

struct CreateWarehouseDispatchLockResponse: Decodable {
    let lockId: String
    let lockType: String
    let status: String

    enum CodingKeys: String, CodingKey {
        case lockId = "lock_id"
        case lockType = "lock_type"
        case status
    }
}

struct ReleaseWarehouseDispatchLockResponse: Decodable {
    let lockId: String
    let status: String

    enum CodingKeys: String, CodingKey {
        case lockId = "lock_id"
        case status
    }
}

// MARK: - Staff

struct StaffMember: Decodable, Identifiable {
    var id: String { workerId }
    let workerId: String
    let name: String
    let phone: String
    let role: String
    let isActive: Bool

    enum CodingKeys: String, CodingKey {
        case workerId = "worker_id"
        case name, phone, role
        case isActive = "is_active"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        workerId = try c.decode(String.self, forKey: .workerId)
        name = try c.decodeIfPresent(String.self, forKey: .name) ?? ""
        phone = try c.decodeIfPresent(String.self, forKey: .phone) ?? ""
        role = try c.decodeIfPresent(String.self, forKey: .role) ?? ""
        isActive = try c.decodeIfPresent(Bool.self, forKey: .isActive) ?? true
    }
}

struct StaffListResponse: Decodable {
    let staff: [StaffMember]
}

struct CreateStaffRequest: Encodable {
    let name: String
    let phone: String
    let role: String
}

struct CreateStaffResponse: Decodable {
    let workerId: String
    let pin: String

    enum CodingKeys: String, CodingKey {
        case workerId = "worker_id"
        case pin
    }
}

// MARK: - Payment Config

struct PaymentGateway: Decodable, Identifiable {
    var id: String { code }
    let code: String
    let name: String
    let provider: String
    let isActive: Bool
    let status: String
    let selectable: Bool

    enum CodingKeys: String, CodingKey {
        case gatewayName = "gateway_name"
        case code, name, provider, status, selectable
        case isActive = "is_active"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        let gatewayName = try c.decodeIfPresent(String.self, forKey: .gatewayName) ?? ""
        code = try c.decodeIfPresent(String.self, forKey: .code) ?? gatewayName
        name = try c.decodeIfPresent(String.self, forKey: .name) ?? gatewayName
        provider = try c.decodeIfPresent(String.self, forKey: .provider) ?? code
        isActive = try c.decodeIfPresent(Bool.self, forKey: .isActive) ?? false
        status = try c.decodeIfPresent(String.self, forKey: .status) ?? ""
        selectable = try c.decodeIfPresent(Bool.self, forKey: .selectable) ?? isActive
    }
}

struct PSPListing: Decodable, Identifiable {
    var id: String { code }
    let code: String
    let status: String
    let selectable: Bool
}

struct PaymentConfigResponse: Decodable {
    let gateways: [PaymentGateway]
    let catalog: [PSPListing]
    let currencyCode: String
    let marketCode: String

    enum CodingKeys: String, CodingKey {
        case gateways, catalog
        case currencyCode = "currency_code"
        case marketCode = "market_code"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        gateways = try c.decodeIfPresent([PaymentGateway].self, forKey: .gateways) ?? []
        catalog = try c.decodeIfPresent([PSPListing].self, forKey: .catalog) ?? []
        currencyCode = try c.decodeIfPresent(String.self, forKey: .currencyCode) ?? ""
        marketCode = try c.decodeIfPresent(String.self, forKey: .marketCode) ?? ""
    }
}

struct WarehouseCoverageResponse: Decodable {
    let warehouseId: String
    let mode: String
    let cities: [WarehouseCoverageCity]
    let pins: [WarehouseServicePin]
    let countryCode: String

    enum CodingKeys: String, CodingKey {
        case warehouseId = "warehouse_id"
        case mode, cities, pins
        case countryCode = "country_code"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        warehouseId = try c.decodeIfPresent(String.self, forKey: .warehouseId) ?? ""
        mode = try c.decodeIfPresent(String.self, forKey: .mode) ?? "COUNTRY_CLOSEST"
        cities = try c.decodeIfPresent([WarehouseCoverageCity].self, forKey: .cities) ?? []
        pins = try c.decodeIfPresent([WarehouseServicePin].self, forKey: .pins) ?? []
        countryCode = try c.decodeIfPresent(String.self, forKey: .countryCode) ?? ""
    }
}

struct WarehouseCoverageCity: Decodable, Identifiable {
    var id: String { "\(name):\(lat):\(lng)" }
    let name: String
    let lat: Double
    let lng: Double
}

struct WarehouseServicePin: Decodable, Identifiable {
    var id: String { "\(targetType):\(targetId)" }
    let targetType: String
    let targetId: String
    let priority: Int64

    enum CodingKeys: String, CodingKey {
        case targetType = "target_type"
        case targetId = "target_id"
        case priority
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        targetType = try c.decodeIfPresent(String.self, forKey: .targetType) ?? ""
        targetId = try c.decodeIfPresent(String.self, forKey: .targetId) ?? ""
        priority = try c.decodeIfPresent(Int64.self, forKey: .priority) ?? 0
    }
}

struct WarehouseSupplyFactoryResponse: Decodable {
    let warehouseId: String
    let factoryId: String
    let transferMode: String
    let source: String
    let countryCode: String

    enum CodingKeys: String, CodingKey {
        case warehouseId = "warehouse_id"
        case factoryId = "factory_id"
        case transferMode = "transfer_mode"
        case source
        case countryCode = "country_code"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        warehouseId = try c.decodeIfPresent(String.self, forKey: .warehouseId) ?? ""
        factoryId = try c.decodeIfPresent(String.self, forKey: .factoryId) ?? ""
        transferMode = try c.decodeIfPresent(String.self, forKey: .transferMode) ?? ""
        source = try c.decodeIfPresent(String.self, forKey: .source) ?? "engine"
        countryCode = try c.decodeIfPresent(String.self, forKey: .countryCode) ?? ""
    }
}

// MARK: - JSON helpers

struct AnyCodable: Codable {
    let value: Any

    init(_ value: Any) {
        self.value = value
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.singleValueContainer()
        if container.decodeNil() {
            value = NSNull()
        } else if let bool = try? container.decode(Bool.self) {
            value = bool
        } else if let int = try? container.decode(Int.self) {
            value = int
        } else if let double = try? container.decode(Double.self) {
            value = double
        } else if let string = try? container.decode(String.self) {
            value = string
        } else if let dict = try? container.decode([String: AnyCodable].self) {
            value = dict.mapValues { $0.value }
        } else if let array = try? container.decode([AnyCodable].self) {
            value = array.map { $0.value }
        } else {
            throw DecodingError.dataCorruptedError(in: container, debugDescription: "Unsupported JSON value")
        }
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.singleValueContainer()
        switch value {
        case is NSNull:
            try container.encodeNil()
        case let bool as Bool:
            try container.encode(bool)
        case let int as Int:
            try container.encode(int)
        case let double as Double:
            try container.encode(double)
        case let string as String:
            try container.encode(string)
        case let dict as [String: Any]:
            try container.encode(dict.mapValues { AnyCodable($0) })
        case let array as [Any]:
            try container.encode(array.map { AnyCodable($0) })
        default:
            throw EncodingError.invalidValue(
                value,
                EncodingError.Context(codingPath: container.codingPath, debugDescription: "Unsupported JSON value")
            )
        }
    }
}

extension Dictionary where Key == String, Value == AnyCodable {
    func prettyJSONString() -> String {
        let object = mapValues { $0.value }
        guard JSONSerialization.isValidJSONObject(object),
              let data = try? JSONSerialization.data(withJSONObject: object, options: [.prettyPrinted, .sortedKeys]),
              let string = String(data: data, encoding: .utf8) else {
            return "{\n  \"is_24h\": true\n}"
        }
        return string
    }
}

struct WarehouseOpsBoardDeliveryExpectation: Decodable {
    let targetLabel: String

    enum CodingKeys: String, CodingKey {
        case targetLabel = "target_label"
    }
}

struct WarehouseOpsBoardOrder: Decodable, Identifiable {
    var id: String { orderId }
    let orderId: String
    let status: String
    let retailerId: String?
    let totalMinor: Int64
    let deliveryExpectation: WarehouseOpsBoardDeliveryExpectation?

    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
        case status
        case retailerId = "retailer_id"
        case totalMinor = "total_minor"
        case deliveryExpectation = "delivery_expectation"
    }
}

struct WarehouseOpsBoardResponse: Decodable {
    let date: String
    let warehouseId: String
    let preorders: [WarehouseOpsBoardOrder]
    let deliverBefore: [WarehouseOpsBoardOrder]

    enum CodingKeys: String, CodingKey {
        case date
        case warehouseId = "warehouse_id"
        case preorders
        case deliverBefore = "deliver_before"
    }
}

// MARK: - Cold chain

struct TemperatureReading: Decodable, Identifiable {
    var id: String { readingId }
    let readingId: String
    let manifestId: String
    let sensorId: String?
    let recordedAt: String
    let tempC: Double
    let minC: Double?
    let maxC: Double?
    let excursion: Bool

    enum CodingKeys: String, CodingKey {
        case readingId = "reading_id"
        case manifestId = "manifest_id"
        case sensorId = "sensor_id"
        case recordedAt = "recorded_at"
        case tempC = "temp_c"
        case minC = "min_c"
        case maxC = "max_c"
        case excursion
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        readingId = try c.decodeIfPresent(String.self, forKey: .readingId) ?? ""
        manifestId = try c.decodeIfPresent(String.self, forKey: .manifestId) ?? ""
        sensorId = try c.decodeIfPresent(String.self, forKey: .sensorId)
        recordedAt = try c.decodeIfPresent(String.self, forKey: .recordedAt) ?? ""
        tempC = try c.decodeIfPresent(Double.self, forKey: .tempC) ?? 0
        minC = try c.decodeIfPresent(Double.self, forKey: .minC)
        maxC = try c.decodeIfPresent(Double.self, forKey: .maxC)
        excursion = try c.decodeIfPresent(Bool.self, forKey: .excursion) ?? false
    }
}

struct TemperatureReadingListResponse: Decodable {
    let readings: [TemperatureReading]

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        readings = try c.decodeIfPresent([TemperatureReading].self, forKey: .readings) ?? []
    }

    enum CodingKeys: String, CodingKey { case readings }
}

struct TemperatureReadingIngestRequest: Encodable {
    let manifestId: String
    let tempC: Double
    let sensorId: String?
    let minC: Double?
    let maxC: Double?

    enum CodingKeys: String, CodingKey {
        case manifestId = "manifest_id"
        case tempC = "temp_c"
        case sensorId = "sensor_id"
        case minC = "min_c"
        case maxC = "max_c"
    }

    func encode(to encoder: Encoder) throws {
        var c = encoder.container(keyedBy: CodingKeys.self)
        try c.encode(manifestId, forKey: .manifestId)
        try c.encode(tempC, forKey: .tempC)
        if let sensorId, !sensorId.isEmpty { try c.encode(sensorId, forKey: .sensorId) }
        if let minC { try c.encode(minC, forKey: .minC) }
        if let maxC { try c.encode(maxC, forKey: .maxC) }
    }
}

// MARK: - Labor capacity (camelCase API)

struct LaborZoneCapacity: Decodable, Identifiable {
    var id: String { "\(zoneH3)-\(date)" }
    let zoneH3: String
    let date: String
    let totalCapacity: Double
    let usedCapacity: Double
    let computedAt: String?

    enum CodingKeys: String, CodingKey {
        case zoneH3, date, totalCapacity, usedCapacity, computedAt
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        zoneH3 = try c.decodeIfPresent(String.self, forKey: .zoneH3) ?? ""
        if let s = try? c.decode(String.self, forKey: .date) {
            date = s
        } else {
            date = ""
        }
        totalCapacity = try c.decodeIfPresent(Double.self, forKey: .totalCapacity) ?? 0
        usedCapacity = try c.decodeIfPresent(Double.self, forKey: .usedCapacity) ?? 0
        if let s = try? c.decode(String.self, forKey: .computedAt) {
            computedAt = s
        } else {
            computedAt = nil
        }
    }

    init(zoneH3: String, date: String, totalCapacity: Double, usedCapacity: Double, computedAt: String?) {
        self.zoneH3 = zoneH3
        self.date = date
        self.totalCapacity = totalCapacity
        self.usedCapacity = usedCapacity
        self.computedAt = computedAt
    }
}

struct LaborZoneCapacityListResponse: Decodable {
    let zones: [LaborZoneCapacity]

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        if let list = try c.decodeIfPresent([LaborZoneCapacity].self, forKey: .zones), !list.isEmpty {
            zones = list
            return
        }
        if let h3 = try c.decodeIfPresent(String.self, forKey: .zoneH3), !h3.isEmpty {
            let date: String
            if let s = try? c.decode(String.self, forKey: .date) { date = s } else { date = "" }
            let total = try c.decodeIfPresent(Double.self, forKey: .totalCapacity) ?? 0
            let used = try c.decodeIfPresent(Double.self, forKey: .usedCapacity) ?? 0
            let computed: String?
            if let s = try? c.decode(String.self, forKey: .computedAt) { computed = s } else { computed = nil }
            zones = [LaborZoneCapacity(zoneH3: h3, date: date, totalCapacity: total, usedCapacity: used, computedAt: computed)]
            return
        }
        zones = []
    }

    enum CodingKeys: String, CodingKey {
        case zones, zoneH3, date, totalCapacity, usedCapacity, computedAt
    }
}

struct LaborDriverScore: Decodable {
    let driverId: String
    let score: Double
    let onTimeRate: Double
    let completionRate: Double
    let damageRate: Double
    let shopClosedRate: Double
    let feedbackScore: Double
    let stopsPerHour: Double

    enum CodingKeys: String, CodingKey {
        case driverId, score, onTimeRate, completionRate, damageRate, shopClosedRate, feedbackScore, stopsPerHour
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        driverId = try c.decodeIfPresent(String.self, forKey: .driverId) ?? ""
        score = try c.decodeIfPresent(Double.self, forKey: .score) ?? 0
        onTimeRate = try c.decodeIfPresent(Double.self, forKey: .onTimeRate) ?? 0
        completionRate = try c.decodeIfPresent(Double.self, forKey: .completionRate) ?? 0
        damageRate = try c.decodeIfPresent(Double.self, forKey: .damageRate) ?? 0
        shopClosedRate = try c.decodeIfPresent(Double.self, forKey: .shopClosedRate) ?? 0
        feedbackScore = try c.decodeIfPresent(Double.self, forKey: .feedbackScore) ?? 0
        stopsPerHour = try c.decodeIfPresent(Double.self, forKey: .stopsPerHour) ?? 0
    }
}

struct LaborDriverAvailabilityRequest: Encodable {
    let driverId: String
    let date: String
    let availableHours: Double
    let status: String
    let zoneH3: String?

    enum CodingKeys: String, CodingKey {
        case driverId, date, availableHours, status, zoneH3
    }

    func encode(to encoder: Encoder) throws {
        var c = encoder.container(keyedBy: CodingKeys.self)
        try c.encode(driverId, forKey: .driverId)
        try c.encode(date, forKey: .date)
        try c.encode(availableHours, forKey: .availableHours)
        try c.encode(status, forKey: .status)
        if let zoneH3, !zoneH3.isEmpty { try c.encode(zoneH3, forKey: .zoneH3) }
    }
}

struct LaborDriverAvailabilityResponse: Decodable {
    let status: String

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        status = try c.decodeIfPresent(String.self, forKey: .status) ?? ""
    }

    enum CodingKeys: String, CodingKey { case status }
}
