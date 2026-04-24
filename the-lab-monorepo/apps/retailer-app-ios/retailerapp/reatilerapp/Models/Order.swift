import Foundation

// MARK: - Order Status

enum OrderStatus: String, Codable, CaseIterable {
    case pending = "PENDING"
    case pendingReview = "PENDING_REVIEW"
    case scheduled = "SCHEDULED"
    case autoAccepted = "AUTO_ACCEPTED"
    case loaded = "LOADED"
    case dispatched = "DISPATCHED"
    case inTransit = "IN_TRANSIT"
    case arriving = "ARRIVING"
    case arrived = "ARRIVED"
    case arrivedShopClosed = "ARRIVED_SHOP_CLOSED"
    case awaitingPayment = "AWAITING_GLOBAL_PAYNT"
    case pendingCashCollection = "PENDING_CASH_COLLECTION"
    case cancelRequested = "CANCEL_REQUESTED"
    case noCapacity = "NO_CAPACITY"
    case completed = "COMPLETED"
    case cancelled = "CANCELLED"
    case quarantine = "QUARANTINE"
    case deliveredOnCredit = "DELIVERED_ON_CREDIT"

    var displayName: String {
        switch self {
        case .pending: "Order Placed"
        case .pendingReview: "Under Review"
        case .scheduled: "Scheduled"
        case .autoAccepted: "Auto-Accepted"
        case .loaded: "Approved"
        case .dispatched: "Dispatched"
        case .inTransit: "Active"
        case .arriving: "Driver Nearby"
        case .arrived: "Driver Arrived"
        case .arrivedShopClosed: "Shop Closed"
        case .awaitingPayment: "Awaiting Payment"
        case .pendingCashCollection: "Cash Collection"
        case .cancelRequested: "Cancel Requested"
        case .noCapacity: "No Capacity"
        case .completed: "Delivered"
        case .cancelled: "Cancelled"
        case .quarantine: "On Hold"
        case .deliveredOnCredit: "Delivered (Credit)"
        }
    }

    var color: String {
        switch self {
        case .pending, .pendingReview, .scheduled: "systemOrange"
        case .autoAccepted: "systemOrange"
        case .loaded: "systemBlue"
        case .dispatched: "systemTeal"
        case .inTransit, .arriving: "systemGreen"
        case .arrived: "systemGreen"
        case .arrivedShopClosed: "systemOrange"
        case .awaitingPayment: "systemPurple"
        case .pendingCashCollection: "systemOrange"
        case .completed, .deliveredOnCredit: "systemGreen"
        case .cancelled, .cancelRequested, .noCapacity: "systemRed"
        case .quarantine: "systemYellow"
        }
    }

    var isActive: Bool {
        switch self {
        case .loaded, .dispatched, .inTransit, .arriving, .arrived, .arrivedShopClosed, .awaitingPayment, .pendingCashCollection, .autoAccepted:
            return true
        default:
            return false
        }
    }

    /// JIT: delivery token exists once payload terminal seals the order (DISPATCHED+)
    var hasDeliveryToken: Bool {
        self == .dispatched || self == .inTransit || self == .arriving || self == .arrived
    }

    /// Timeline step index for the 6-step retailer-facing tracking UI.
    var timelineStepIndex: Int {
        switch self {
        case .pending, .pendingReview, .scheduled, .autoAccepted: 0
        case .loaded: 1
        case .dispatched: 2
        case .inTransit: 3
        case .arriving, .arrived, .arrivedShopClosed: 4
        case .completed, .deliveredOnCredit: 5
        case .cancelled, .cancelRequested, .noCapacity, .quarantine: -1
        default: -1
        }
    }

    /// Ordered timeline labels.
    static let timelineSteps = ["Placed", "Approved", "Dispatched", "Active", "Arrived", "Delivered"]

    var canCancel: Bool {
        self == .pending || self == .pendingReview || self == .scheduled || self == .autoAccepted
    }
}

import Foundation

// MARK: - Order Line Item

struct OrderLineItem: Codable, Identifiable, Hashable {
    let id: String
    let productId: String
    let productName: String
    let variantId: String
    let variantSize: String
    let quantity: Int
    let unitPrice: Double
    let totalPrice: Double

    enum CodingKeys: String, CodingKey {
        case id = "line_item_id"
        case variantId = "sku_id"
        case productName = "sku_name"
        case quantity = "quantity"
        case unitPrice = "unit_price"
    }

    init(id: String, productId: String, productName: String, variantId: String, variantSize: String, quantity: Int, unitPrice: Double, totalPrice: Double) {
        self.id = id
        self.productId = productId
        self.productName = productName
        self.variantId = variantId
        self.variantSize = variantSize
        self.quantity = quantity
        self.unitPrice = unitPrice
        self.totalPrice = totalPrice
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        self.id = try container.decodeIfPresent(String.self, forKey: .id) ?? UUID().uuidString
        let parsedVariant = try container.decodeIfPresent(String.self, forKey: .variantId) ?? ""
        self.variantId = parsedVariant
        self.productId = parsedVariant
        self.productName = try container.decodeIfPresent(String.self, forKey: .productName) ?? "Unknown"
        self.variantSize = ""
        self.quantity = try container.decodeIfPresent(Int.self, forKey: .quantity) ?? 1
        let basePrice = try container.decodeIfPresent(Double.self, forKey: .unitPrice) ?? 0.0
        self.unitPrice = basePrice
        self.totalPrice = basePrice * Double(self.quantity)
    }
}

// MARK: - Order

struct Order: Codable, Identifiable, Hashable {
    let id: String
    let retailerId: String
    let supplierId: String?
    let supplierName: String?
    let status: OrderStatus
    let items: [OrderLineItem]
    let totalAmount: Double
    let orderSource: String?
    let createdAt: String
    let updatedAt: String
    let estimatedDelivery: String?
    let qrCode: String?

    enum CodingKeys: String, CodingKey {
        case id = "order_id"
        case retailerId = "retailer_id"
        case supplierId = "supplier_id"
        case supplierName = "supplier_name"
        case status = "state"
        case items = "items"
        case totalAmount = "amount"
        case orderSource = "order_source"
        case createdAt = "created_at"
        case qrCode = "delivery_token"
    }

    init(id: String, retailerId: String, supplierId: String?, supplierName: String?, status: OrderStatus, items: [OrderLineItem], totalAmount: Double, orderSource: String?, createdAt: String, updatedAt: String, estimatedDelivery: String?, qrCode: String?) {
        self.id = id
        self.retailerId = retailerId
        self.supplierId = supplierId
        self.supplierName = supplierName
        self.status = status
        self.items = items
        self.totalAmount = totalAmount
        self.orderSource = orderSource
        self.createdAt = createdAt
        self.updatedAt = updatedAt
        self.estimatedDelivery = estimatedDelivery
        self.qrCode = qrCode
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        self.id = try c.decode(String.self, forKey: .id)
        self.retailerId = try c.decodeIfPresent(String.self, forKey: .retailerId) ?? ""
        self.supplierId = try c.decodeIfPresent(String.self, forKey: .supplierId)
        self.supplierName = try c.decodeIfPresent(String.self, forKey: .supplierName)
        
        // Handle Status Safely
        if let rawStatus = try c.decodeIfPresent(String.self, forKey: .status), let parsedStatus = OrderStatus(rawValue: rawStatus) {
            self.status = parsedStatus
        } else {
            self.status = .pending
        }
        
        self.items = try c.decodeIfPresent([OrderLineItem].self, forKey: .items) ?? []
        self.totalAmount = try c.decodeIfPresent(Double.self, forKey: .totalAmount) ?? 0.0
        self.orderSource = try c.decodeIfPresent(String.self, forKey: .orderSource)
        self.createdAt = try c.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
        self.updatedAt = self.createdAt
        self.estimatedDelivery = nil
        self.qrCode = try c.decodeIfPresent(String.self, forKey: .qrCode)
    }

    var isAiGenerated: Bool {
        orderSource == "AI_PREDICTED"
    }

    var displayTotal: String {
        "\(Int(totalAmount).formatted())"
    }

    var itemCount: Int {
        items.reduce(0) { $0 + $1.quantity }
    }
}

// MARK: - Tracking Order (for delivery map)

struct TrackingOrderItem: Codable, Identifiable, Hashable {
    var id: String { productId }
    let productId: String
    let productName: String
    let quantity: Int
    let unitPrice: Double
    let lineTotal: Double

    enum CodingKeys: String, CodingKey {
        case productId = "product_id"
        case productName = "product_name"
        case quantity
        case unitPrice = "unit_price"
        case lineTotal = "line_total"
    }
}

struct TrackingOrder: Codable, Identifiable, Hashable {
    var id: String { orderId }
    let orderId: String
    let supplierId: String
    let supplierName: String
    let warehouseId: String?
    let warehouseName: String?
    let driverId: String
    let state: String
    let totalAmount: Int
    let orderSource: String
    let driverLatitude: Double?
    let driverLongitude: Double?
    let isApproaching: Bool
    let deliveryToken: String
    let createdAt: String
    let items: [TrackingOrderItem]

    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
        case supplierId = "supplier_id"
        case supplierName = "supplier_name"
        case warehouseId = "warehouse_id"
        case warehouseName = "warehouse_name"
        case driverId = "driver_id"
        case state
        case totalAmount = "total_amount"
        case orderSource = "order_source"
        case driverLatitude = "driver_latitude"
        case driverLongitude = "driver_longitude"
        case isApproaching = "is_approaching"
        case deliveryToken = "delivery_token"
        case createdAt = "created_at"
        case items
    }

    var displayTotal: String {
        "\(totalAmount.formatted())"
    }

    var isGreen: Bool {
        isApproaching || state == "ARRIVED"
    }

    var hasDriverLocation: Bool {
        driverLatitude != nil && driverLongitude != nil
    }
}

struct TrackingResponse: Codable {
    let orders: [TrackingOrder]
}

// MARK: - Sample Data

extension Order {
    static let samples: [Order] = [
        Order(
            id: "ord-001",
            retailerId: "retailer-123",
            supplierId: "sup-001",
            supplierName: "Fresh Farms Co.",
            status: .inTransit,
            items: [
                OrderLineItem(id: "li-001", productId: "prod-001", productName: "Organic Whole Milk", variantId: "v-001a", variantSize: "1L", quantity: 3, unitPrice: 3.49, totalPrice: 10.47),
                OrderLineItem(id: "li-002", productId: "prod-003", productName: "Free-Range Eggs", variantId: "v-003a", variantSize: "12 ct", quantity: 2, unitPrice: 5.99, totalPrice: 11.98)
            ],
            totalAmount: 22.45,
            orderSource: "MANUAL",
            createdAt: "2026-03-17T08:00:00Z",
            updatedAt: "2026-03-17T09:30:00Z",
            estimatedDelivery: "2026-03-17T14:00:00Z",
            qrCode: "ORD-001-QR-DATA"
        ),
        Order(
            id: "ord-002",
            retailerId: "retailer-123",
            supplierId: "sup-002",
            supplierName: "Bakery Express",
            status: .completed,
            items: [
                OrderLineItem(id: "li-003", productId: "prod-002", productName: "Sourdough Bread", variantId: "v-002b", variantSize: "500g", quantity: 1, unitPrice: 13.49, totalPrice: 13.49)
            ],
            totalAmount: 13.49,
            orderSource: "AI_PREDICTED",
            createdAt: "2026-03-16T10:00:00Z",
            updatedAt: "2026-03-16T15:00:00Z",
            estimatedDelivery: nil,
            qrCode: nil
        ),
        Order(
            id: "ord-003",
            retailerId: "retailer-123",
            supplierId: "sup-003",
            supplierName: "Mountain Spring Water",
            status: .pending,
            items: [
                OrderLineItem(id: "li-004", productId: "prod-005", productName: "Sparkling Water", variantId: "v-005b", variantSize: "500ml", quantity: 2, unitPrice: 14.99, totalPrice: 29.98)
            ],
            totalAmount: 29.98,
            orderSource: "B2B_CHECKOUT",
            createdAt: "2026-03-17T11:00:00Z",
            updatedAt: "2026-03-17T11:00:00Z",
            estimatedDelivery: "2026-03-18T10:00:00Z",
            qrCode: nil // JIT: no token until dispatched
        )
    ]
}
