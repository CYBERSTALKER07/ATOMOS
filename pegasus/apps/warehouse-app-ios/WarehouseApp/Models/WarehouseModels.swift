import Foundation

// MARK: - Auth

struct LoginRequest: Encodable {
    let phone: String
    let pin: String
}

struct AuthResponse: Decodable {
    let token: String
    let refreshToken: String
    let warehouseId: String

    enum CodingKeys: String, CodingKey {
        case token
        case refreshToken = "refresh_token"
        case warehouseId = "warehouse_id"
    }
}

// MARK: - Dashboard

struct DashboardData: Decodable {
    let activeOrders: Int
    let completedToday: Int
    let pendingDispatch: Int
    let todayRevenue: Int
    let driversOnRoute: Int
    let idleDrivers: Int
    let vehicles: Int
    let lowStockItems: Int
    let totalStaff: Int

    enum CodingKeys: String, CodingKey {
        case activeOrders = "active_orders"
        case completedToday = "completed_today"
        case pendingDispatch = "pending_dispatch"
        case todayRevenue = "today_revenue"
        case driversOnRoute = "drivers_on_route"
        case idleDrivers = "idle_drivers"
        case vehicles
        case lowStockItems = "low_stock_items"
        case totalStaff = "total_staff"
    }

    static let empty = DashboardData(
        activeOrders: 0, completedToday: 0, pendingDispatch: 0,
        todayRevenue: 0, driversOnRoute: 0, idleDrivers: 0,
        vehicles: 0, lowStockItems: 0, totalStaff: 0
    )
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

    enum CodingKeys: String, CodingKey {
        case driverId = "driver_id"
        case name, phone
        case truckStatus = "truck_status"
        case isActive = "is_active"
        case vehicleId = "vehicle_id"
        case vehicleClass = "vehicle_class"
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
        }
    }
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
        assignedDriverId = try c.decodeIfPresent(String.self, forKey: .assignedDriverId)
        assignedDriverName = try c.decodeIfPresent(String.self, forKey: .assignedDriverName)
    }
}

struct VehicleListResponse: Decodable {
    let vehicles: [Vehicle]
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

    enum CodingKeys: String, CodingKey {
        case isActive = "is_active"
        case unavailableReason = "unavailable_reason"
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

    enum CodingKeys: String, CodingKey {
        case productId = "product_id"
        case productName = "product_name"
        case quantity
        case reorderThreshold = "reorder_threshold"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        productId = try c.decode(String.self, forKey: .productId)
        productName = try c.decodeIfPresent(String.self, forKey: .productName) ?? ""
        quantity = try c.decodeIfPresent(Int.self, forKey: .quantity) ?? 0
        reorderThreshold = try c.decodeIfPresent(Int.self, forKey: .reorderThreshold) ?? 0
    }
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

struct AnalyticsData: Decodable {
    let totalOrders: Int
    let totalRevenue: Int
    let avgDeliveryMinutes: Int
    let completionRate: Int
    let topProducts: [TopProduct]

    enum CodingKeys: String, CodingKey {
        case totalOrders = "total_orders"
        case totalRevenue = "total_revenue"
        case avgDeliveryMinutes = "avg_delivery_minutes"
        case completionRate = "completion_rate"
        case topProducts = "top_products"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        totalOrders = try c.decodeIfPresent(Int.self, forKey: .totalOrders) ?? 0
        totalRevenue = try c.decodeIfPresent(Int.self, forKey: .totalRevenue) ?? 0
        avgDeliveryMinutes = try c.decodeIfPresent(Int.self, forKey: .avgDeliveryMinutes) ?? 0
        completionRate = try c.decodeIfPresent(Int.self, forKey: .completionRate) ?? 0
        topProducts = try c.decodeIfPresent([TopProduct].self, forKey: .topProducts) ?? []
    }

    static let empty = AnalyticsData(totalOrders: 0, totalRevenue: 0, avgDeliveryMinutes: 0, completionRate: 0, topProducts: [])

    init(totalOrders: Int, totalRevenue: Int, avgDeliveryMinutes: Int, completionRate: Int, topProducts: [TopProduct]) {
        self.totalOrders = totalOrders
        self.totalRevenue = totalRevenue
        self.avgDeliveryMinutes = avgDeliveryMinutes
        self.completionRate = completionRate
        self.topProducts = topProducts
    }
}

struct TopProduct: Decodable, Identifiable {
    var id: String { productName }
    let productName: String
    let unitsSold: Int
    let revenue: Int

    enum CodingKeys: String, CodingKey {
        case productName = "product_name"
        case unitsSold = "units_sold"
        case revenue
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        productName = try c.decodeIfPresent(String.self, forKey: .productName) ?? ""
        unitsSold = try c.decodeIfPresent(Int.self, forKey: .unitsSold) ?? 0
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

    enum CodingKeys: String, CodingKey {
        case retailerId = "retailer_id"
        case name
        case totalOrders = "total_orders"
        case totalRevenue = "total_revenue"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        retailerId = try c.decode(String.self, forKey: .retailerId)
        name = try c.decodeIfPresent(String.self, forKey: .name) ?? ""
        totalOrders = try c.decodeIfPresent(Int.self, forKey: .totalOrders) ?? 0
        totalRevenue = try c.decodeIfPresent(Int.self, forKey: .totalRevenue) ?? 0
    }
}

struct RetailerListResponse: Decodable {
    let retailers: [Retailer]
}

// MARK: - Return

struct ReturnItem: Decodable, Identifiable {
    var id: String { returnId }
    let returnId: String
    let orderId: String
    let productName: String
    let quantity: Int
    let reason: String
    let createdAt: String

    enum CodingKeys: String, CodingKey {
        case returnId = "return_id"
        case orderId = "order_id"
        case productName = "product_name"
        case quantity, reason
        case createdAt = "created_at"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        returnId = try c.decode(String.self, forKey: .returnId)
        orderId = try c.decodeIfPresent(String.self, forKey: .orderId) ?? ""
        productName = try c.decodeIfPresent(String.self, forKey: .productName) ?? ""
        quantity = try c.decodeIfPresent(Int.self, forKey: .quantity) ?? 0
        reason = try c.decodeIfPresent(String.self, forKey: .reason) ?? ""
        createdAt = try c.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
    }
}

struct ReturnListResponse: Decodable {
    let returns: [ReturnItem]
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
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        balance = try c.decodeIfPresent(Int.self, forKey: .balance) ?? 0
        totalReceivable = try c.decodeIfPresent(Int.self, forKey: .totalReceivable) ?? 0
        totalCollected = try c.decodeIfPresent(Int.self, forKey: .totalCollected) ?? 0
        overdueAmount = try c.decodeIfPresent(Int.self, forKey: .overdueAmount) ?? 0
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
    let status: String
    let dueDate: String

    enum CodingKeys: String, CodingKey {
        case invoiceId = "invoice_id"
        case retailerName = "retailer_name"
        case amountUzs = "amount_uzs"
        case status
        case dueDate = "due_date"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        invoiceId = try c.decode(String.self, forKey: .invoiceId)
        retailerName = try c.decodeIfPresent(String.self, forKey: .retailerName) ?? ""
        amountUzs = try c.decodeIfPresent(Int.self, forKey: .amountUzs) ?? 0
        status = try c.decodeIfPresent(String.self, forKey: .status) ?? ""
        dueDate = try c.decodeIfPresent(String.self, forKey: .dueDate) ?? ""
    }
}

struct InvoiceListResponse: Decodable {
    let invoices: [Invoice]
}

// MARK: - Dispatch

struct DispatchPreview: Decodable {
    let undispatchedOrders: [DispatchOrder]
    let availableDrivers: [AvailableDriver]

    enum CodingKeys: String, CodingKey {
        case undispatchedOrders = "undispatched_orders"
        case availableDrivers = "available_drivers"
    }
}

struct DispatchOrder: Decodable, Identifiable {
    var id: String { orderId }
    let orderId: String
    let retailerName: String
    let totalUzs: Int
    let itemCount: Int

    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
        case retailerName = "retailer_name"
        case totalUzs = "total_uzs"
        case itemCount = "item_count"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        orderId = try c.decode(String.self, forKey: .orderId)
        retailerName = try c.decodeIfPresent(String.self, forKey: .retailerName) ?? ""
        totalUzs = try c.decodeIfPresent(Int.self, forKey: .totalUzs) ?? 0
        itemCount = try c.decodeIfPresent(Int.self, forKey: .itemCount) ?? 0
    }
}

struct AvailableDriver: Decodable, Identifiable {
    var id: String { driverId }
    let driverId: String
    let name: String
    let vehicleLabel: String

    enum CodingKeys: String, CodingKey {
        case driverId = "driver_id"
        case name
        case vehicleLabel = "vehicle_label"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        driverId = try c.decode(String.self, forKey: .driverId)
        name = try c.decodeIfPresent(String.self, forKey: .name) ?? ""
        vehicleLabel = try c.decodeIfPresent(String.self, forKey: .vehicleLabel) ?? ""
    }
}

// MARK: - Warehouse Realtime

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

    enum CodingKeys: String, CodingKey {
        case factoryId = "factory_id"
        case priority
        case notes
        case items
        case useDemandForecast = "use_demand_forecast"
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
    var id: String { gatewayId }
    let gatewayId: String
    let name: String
    let provider: String
    let isActive: Bool

    enum CodingKeys: String, CodingKey {
        case gatewayId = "gateway_id"
        case name, provider
        case isActive = "is_active"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        gatewayId = try c.decode(String.self, forKey: .gatewayId)
        name = try c.decodeIfPresent(String.self, forKey: .name) ?? ""
        provider = try c.decodeIfPresent(String.self, forKey: .provider) ?? ""
        isActive = try c.decodeIfPresent(Bool.self, forKey: .isActive) ?? false
    }
}

struct PaymentConfigResponse: Decodable {
    let gateways: [PaymentGateway]
}
