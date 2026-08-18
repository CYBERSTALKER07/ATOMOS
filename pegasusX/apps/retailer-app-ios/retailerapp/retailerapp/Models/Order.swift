import Foundation

private func formatRetailerMoney(_ amount: Int64, currency: String) -> String {
    let formatter = NumberFormatter()
    formatter.numberStyle = .decimal
    formatter.groupingSeparator = " "
    formatter.maximumFractionDigits = 0
    let formatted = formatter.string(from: NSNumber(value: amount)) ?? "\(amount)"
    let normalizedCurrency = currency.isEmpty ? packCurrency(MarketPackStore.pack) : currency
    return "\(formatted) \(normalizedCurrency)"
}

private extension KeyedDecodingContainer {
    func decodeInt64Lossy(forKey key: Key) throws -> Int64 {
        if let value = try decodeIfPresent(Int64.self, forKey: key) {
            return value
        }
        if let value = try decodeIfPresent(Double.self, forKey: key) {
            return Int64(value.rounded())
        }
        return 0
    }

    func decodeDoubleLossy(forKey key: Key) throws -> Double {
        if let value = try decodeIfPresent(Double.self, forKey: key) {
            return value
        }
        if let value = try decodeIfPresent(Int64.self, forKey: key) {
            return Double(value)
        }
        return 0
    }
}

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
    case awaitingPayment = "AWAITING_PAYMENT"
    case pendingCashCollection = "PENDING_CASH_COLLECTION"
    case fiscalizing = "FISCALIZING"
    case fiscalFailed = "FISCAL_FAILED"
    case cancelRequested = "CANCEL_REQUESTED"
    case noCapacity = "NO_CAPACITY"
    case completed = "COMPLETED"
    case cancelled = "CANCELLED"
    case quarantine = "QUARANTINE"
    case deliveredOnCredit = "DELIVERED_ON_CREDIT"
    case reconciliationRequired = "RECONCILIATION_REQUIRED"
    case delayed = "DELAYED"

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
        case .fiscalizing: "Pending Fiscal"
        case .fiscalFailed: "Fiscal Failed"
        case .cancelRequested: "Cancel Requested"
        case .noCapacity: "No Capacity"
        case .completed: "Delivered"
        case .cancelled: "Cancelled"
        case .quarantine: "On Hold"
        case .deliveredOnCredit: "Delivered (Credit)"
        case .reconciliationRequired: "Reconciliation Required"
        case .delayed: "Delayed"
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
        case .pendingCashCollection, .fiscalizing: "systemOrange"
        case .fiscalFailed: "systemRed"
        case .completed, .deliveredOnCredit: "systemGreen"
        case .cancelled, .cancelRequested, .noCapacity: "systemRed"
        case .quarantine, .delayed, .reconciliationRequired: "systemYellow"
        }
    }

    var isActive: Bool {
        switch self {
        case .loaded, .dispatched, .inTransit, .arriving, .arrived, .arrivedShopClosed,
             .awaitingPayment, .pendingCashCollection, .fiscalizing, .fiscalFailed, .autoAccepted:
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
        case .arriving, .arrived, .arrivedShopClosed, .awaitingPayment, .pendingCashCollection, .fiscalizing, .fiscalFailed: 4
        case .completed, .deliveredOnCredit: 5
        case .cancelled, .cancelRequested, .noCapacity, .quarantine, .reconciliationRequired, .delayed: -1
        default: -1
        }
    }

    /// Ordered timeline labels.
    static let timelineSteps = ["Placed", "Approved", "Dispatched", "Active", "Arrived", "Delivered"]

    var canCancel: Bool {
        self == .pending || self == .pendingReview || self == .scheduled || self == .autoAccepted
    }
}

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
        case lineItemID = "line_item_id"
        case legacyID = "id"
        case productID = "product_id"
        case skuID = "sku_id"
        case productName = "product_name"
        case skuName = "sku_name"
        case variantId = "variant_id"
        case variantSize = "variant_size"
        case quantity = "quantity"
        case unitPrice = "unit_price"
        case totalPrice = "total_price"
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
        self.id = try container.decodeIfPresent(String.self, forKey: .lineItemID)
            ?? container.decodeIfPresent(String.self, forKey: .legacyID)
            ?? UUID().uuidString
        let decodedSku = try container.decodeIfPresent(String.self, forKey: .skuID) ?? ""
        self.variantId = try container.decodeIfPresent(String.self, forKey: .variantId) ?? decodedSku
        self.productId = try container.decodeIfPresent(String.self, forKey: .productID) ?? decodedSku
        self.productName = try container.decodeIfPresent(String.self, forKey: .skuName)
            ?? container.decodeIfPresent(String.self, forKey: .productName)
            ?? "Unknown"
        self.variantSize = try container.decodeIfPresent(String.self, forKey: .variantSize) ?? ""
        self.quantity = try container.decodeIfPresent(Int.self, forKey: .quantity) ?? 1
        let basePrice = try container.decodeDoubleLossy(forKey: .unitPrice)
        self.unitPrice = basePrice
        let decodedTotal = try container.decodeDoubleLossy(forKey: .totalPrice)
        self.totalPrice = decodedTotal > 0 ? decodedTotal : basePrice * Double(self.quantity)
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encode(id, forKey: .legacyID)
        try container.encode(productId, forKey: .productID)
        try container.encode(productName, forKey: .productName)
        try container.encode(variantId, forKey: .variantId)
        try container.encode(variantSize, forKey: .variantSize)
        try container.encode(quantity, forKey: .quantity)
        try container.encode(unitPrice, forKey: .unitPrice)
        try container.encode(totalPrice, forKey: .totalPrice)
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
    let totalAmount: Int64
    let currency: String
    let paymentGateway: String
    let paymentStatus: String?
    let routeId: String?
    let autoConfirmAt: String?
    let deliverBefore: String?
    let orderSource: String?
    let confirmationStatus: String?
    let deliveryPriority: String?
    let preorderBadge: String?
    let proposedDeliveryDate: String?
    let deliveryProposalReason: String?
    let createdAt: String
    let updatedAt: String
    let estimatedDelivery: String?
    let qrCode: String?
    let version: Int64

    enum CodingKeys: String, CodingKey {
        case id = "order_id"
        case retailerId = "retailer_id"
        case supplierId = "supplier_id"
        case supplierName = "supplier_name"
        case status = "state"
        case items = "items"
        case totalAmount = "amount"
        case currency = "currency"
        case paymentGateway = "payment_gateway"
        case paymentStatus = "payment_status"
        case routeId = "route_id"
        case autoConfirmAt = "auto_confirm_at"
        case deliverBefore = "deliver_before"
        case orderSource = "order_source"
        case confirmationStatus = "confirmation_status"
        case deliveryPriority = "delivery_priority"
        case preorderBadge = "preorder_badge"
        case proposedDeliveryDate = "proposed_delivery_date"
        case deliveryProposalReason = "delivery_proposal_reason"
        case createdAt = "created_at"
        case updatedAt = "updated_at"
        case estimatedDelivery = "estimated_delivery"
        case qrCode = "delivery_token"
        case version = "version"
    }

    init(id: String, retailerId: String, supplierId: String?, supplierName: String?, status: OrderStatus, items: [OrderLineItem], totalAmount: Int64, currency: String = packCurrency(MarketPackStore.pack), paymentGateway: String = "", paymentStatus: String? = nil, routeId: String? = nil, autoConfirmAt: String? = nil, deliverBefore: String? = nil, orderSource: String?, confirmationStatus: String? = nil, deliveryPriority: String? = nil, preorderBadge: String? = nil, proposedDeliveryDate: String? = nil, deliveryProposalReason: String? = nil, createdAt: String, updatedAt: String, estimatedDelivery: String?, qrCode: String?, version: Int64 = 0) {
        self.id = id
        self.retailerId = retailerId
        self.supplierId = supplierId
        self.supplierName = supplierName
        self.status = status
        self.items = items
        self.totalAmount = totalAmount
        self.currency = currency
        self.paymentGateway = paymentGateway
        self.paymentStatus = paymentStatus
        self.routeId = routeId
        self.autoConfirmAt = autoConfirmAt
        self.deliverBefore = deliverBefore
        self.orderSource = orderSource
        self.confirmationStatus = confirmationStatus
        self.deliveryPriority = deliveryPriority
        self.preorderBadge = preorderBadge
        self.proposedDeliveryDate = proposedDeliveryDate
        self.deliveryProposalReason = deliveryProposalReason
        self.createdAt = createdAt
        self.updatedAt = updatedAt
        self.estimatedDelivery = estimatedDelivery
        self.qrCode = qrCode
        self.version = version
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
                self.totalAmount = try c.decodeInt64Lossy(forKey: .totalAmount)
                self.currency = try c.decodeIfPresent(String.self, forKey: .currency) ?? packCurrency(MarketPackStore.pack)
                self.paymentGateway = try c.decodeIfPresent(String.self, forKey: .paymentGateway) ?? ""
                self.paymentStatus = try c.decodeIfPresent(String.self, forKey: .paymentStatus)
                self.routeId = try c.decodeIfPresent(String.self, forKey: .routeId)
                self.autoConfirmAt = try c.decodeIfPresent(String.self, forKey: .autoConfirmAt)
                self.deliverBefore = try c.decodeIfPresent(String.self, forKey: .deliverBefore)
        self.orderSource = try c.decodeIfPresent(String.self, forKey: .orderSource)
        self.confirmationStatus = try c.decodeIfPresent(String.self, forKey: .confirmationStatus)
        self.deliveryPriority = try c.decodeIfPresent(String.self, forKey: .deliveryPriority)
        self.preorderBadge = try c.decodeIfPresent(String.self, forKey: .preorderBadge)
        self.proposedDeliveryDate = try c.decodeIfPresent(String.self, forKey: .proposedDeliveryDate)
        self.deliveryProposalReason = try c.decodeIfPresent(String.self, forKey: .deliveryProposalReason)
        self.createdAt = try c.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
                self.updatedAt = try c.decodeIfPresent(String.self, forKey: .updatedAt) ?? self.createdAt
                self.estimatedDelivery = try c.decodeIfPresent(String.self, forKey: .estimatedDelivery) ?? self.deliverBefore
        self.qrCode = try c.decodeIfPresent(String.self, forKey: .qrCode)
                self.version = try c.decodeIfPresent(Int64.self, forKey: .version) ?? 0
    }

    var isAiGenerated: Bool {
        orderSource == "AI_PREDICTED"
    }

    var needsManualPreorderAction: Bool {
        orderSource == "MANUAL_PREORDER" && status == .scheduled &&
            (confirmationStatus == nil || confirmationStatus == "DRAFT")
    }

    var needsDeliveryProposalReview: Bool {
        confirmationStatus == "PENDING_WAREHOUSE" || preorderBadge == "REVIEW_DELIVERY"
    }

    var displayTotal: String {
        formatRetailerMoney(totalAmount, currency: currency)
    }

    var itemCount: Int {
        items.reduce(0) { $0 + $1.quantity }
    }

    var deliveryQRCodePayload: String? {
        guard let qrCode, !qrCode.isEmpty else { return nil }
        let payload = RetailerDeliveryQRPayload(order_id: id, token: qrCode)
        guard let data = try? JSONEncoder().encode(payload),
              let encoded = String(data: data, encoding: .utf8) else {
            return qrCode
        }
        return encoded
    }
}

private struct RetailerDeliveryQRPayload: Encodable {
    let order_id: String
    let token: String
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

struct RouteGeometryCoordinate: Codable, Hashable {
    let lat: Double
    let lng: Double
}

struct RouteGeometryWire: Codable, Hashable {
    let routeId: String?
    let encodedPolyline: String?
    let coordinates: [RouteGeometryCoordinate]
    let source: String
    let stopCount: Int?

    enum CodingKeys: String, CodingKey {
        case routeId = "route_id"
        case encodedPolyline = "encoded_polyline"
        case coordinates
        case source
        case stopCount = "stop_count"
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
    let liveLocationAvailable: Bool
    let isApproaching: Bool
    let deliveryToken: String
    let createdAt: String
    let fiscalStatus: String
    let fiscalQr: String
    let latestFiscalReceiptId: String
    let items: [TrackingOrderItem]
    let routeGeometry: RouteGeometryWire?

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
        case liveLocationAvailable = "live_location_available"
        case isApproaching = "is_approaching"
        case deliveryToken = "delivery_token"
        case createdAt = "created_at"
        case fiscalStatus = "fiscal_status"
        case fiscalQr = "fiscal_qr"
        case latestFiscalReceiptId = "latest_fiscal_receipt_id"
        case items
        case routeGeometry = "route_geometry"
    }

    init(
        orderId: String,
        supplierId: String,
        supplierName: String,
        warehouseId: String?,
        warehouseName: String?,
        driverId: String,
        state: String,
        totalAmount: Int,
        orderSource: String,
        driverLatitude: Double?,
        driverLongitude: Double?,
        liveLocationAvailable: Bool,
        isApproaching: Bool,
        deliveryToken: String,
        createdAt: String,
        fiscalStatus: String = "",
        fiscalQr: String = "",
        latestFiscalReceiptId: String = "",
        items: [TrackingOrderItem],
        routeGeometry: RouteGeometryWire? = nil
    ) {
        self.orderId = orderId
        self.supplierId = supplierId
        self.supplierName = supplierName
        self.warehouseId = warehouseId
        self.warehouseName = warehouseName
        self.driverId = driverId
        self.state = state
        self.totalAmount = totalAmount
        self.orderSource = orderSource
        self.driverLatitude = driverLatitude
        self.driverLongitude = driverLongitude
        self.liveLocationAvailable = liveLocationAvailable
        self.isApproaching = isApproaching
        self.deliveryToken = deliveryToken
        self.createdAt = createdAt
        self.fiscalStatus = fiscalStatus
        self.fiscalQr = fiscalQr
        self.latestFiscalReceiptId = latestFiscalReceiptId
        self.items = items
        self.routeGeometry = routeGeometry
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        orderId = try c.decode(String.self, forKey: .orderId)
        supplierId = try c.decodeIfPresent(String.self, forKey: .supplierId) ?? ""
        supplierName = try c.decodeIfPresent(String.self, forKey: .supplierName) ?? ""
        warehouseId = try c.decodeIfPresent(String.self, forKey: .warehouseId)
        warehouseName = try c.decodeIfPresent(String.self, forKey: .warehouseName)
        driverId = try c.decodeIfPresent(String.self, forKey: .driverId) ?? ""
        state = try c.decodeIfPresent(String.self, forKey: .state) ?? ""
        totalAmount = try c.decodeIfPresent(Int.self, forKey: .totalAmount) ?? 0
        orderSource = try c.decodeIfPresent(String.self, forKey: .orderSource) ?? ""
        driverLatitude = try c.decodeIfPresent(Double.self, forKey: .driverLatitude)
        driverLongitude = try c.decodeIfPresent(Double.self, forKey: .driverLongitude)
        liveLocationAvailable = try c.decodeIfPresent(Bool.self, forKey: .liveLocationAvailable) ?? false
        isApproaching = try c.decodeIfPresent(Bool.self, forKey: .isApproaching) ?? false
        deliveryToken = try c.decodeIfPresent(String.self, forKey: .deliveryToken) ?? ""
        createdAt = try c.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
        fiscalStatus = try c.decodeIfPresent(String.self, forKey: .fiscalStatus) ?? ""
        fiscalQr = try c.decodeIfPresent(String.self, forKey: .fiscalQr) ?? ""
        latestFiscalReceiptId = try c.decodeIfPresent(String.self, forKey: .latestFiscalReceiptId) ?? ""
        items = try c.decodeIfPresent([TrackingOrderItem].self, forKey: .items) ?? []
        routeGeometry = try c.decodeIfPresent(RouteGeometryWire.self, forKey: .routeGeometry)
    }

    var displayTotal: String {
        "\(totalAmount.formatted())"
    }

    /// Receipt row label for tracking / recent receipts.
    var fiscalReceiptLabel: String {
        let st = state.uppercased()
        let fs = fiscalStatus.uppercased()
        if fs == "SUCCESS" || st == "COMPLETED" {
            if !latestFiscalReceiptId.isEmpty {
                return "Fiscalized · \(latestFiscalReceiptId)"
            }
            return "Fiscalized"
        }
        if fs == "PENDING" || st == "FISCALIZING" { return "Pending fiscal" }
        if fs == "FAILED" || st == "FISCAL_FAILED" { return "Fiscal failed" }
        if fs == "FORCE_SKIPPED" { return "Fiscal exception" }
        return st.isEmpty ? "—" : st
    }

    var isGreen: Bool {
        isApproaching || state == "ARRIVED"
    }

    var hasDriverLocation: Bool {
        liveLocationAvailable && driverLatitude != nil && driverLongitude != nil
    }

    var deliveryQRCodePayload: String? {
        guard !deliveryToken.isEmpty else { return nil }
        let payload = RetailerDeliveryQRPayload(order_id: orderId, token: deliveryToken)
        guard let data = try? JSONEncoder().encode(payload),
              let encoded = String(data: data, encoding: .utf8) else {
            return deliveryToken
        }
        return encoded
    }
}

struct TrackingResponse: Codable {
    let status: String?
    let orders: [TrackingOrder]
    let recentReceipts: [TrackingOrder]?

    enum CodingKeys: String, CodingKey {
        case status
        case orders
        case recentReceipts = "recent_receipts"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        status = try c.decodeIfPresent(String.self, forKey: .status)
        orders = try c.decodeIfPresent([TrackingOrder].self, forKey: .orders) ?? []
        recentReceipts = try c.decodeIfPresent([TrackingOrder].self, forKey: .recentReceipts)
    }
}

struct ActiveFulfillmentItem: Codable, Identifiable, Hashable {
    var id: String { orderId }
    let orderId: String
    let supplierId: String
    let supplierName: String
    let state: String
    let adjustedAmount: Int
    let itemCount: Int
    let liveLocationAvailable: Bool?

    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
        case supplierId = "supplier_id"
        case supplierName = "supplier_name"
        case state
        case adjustedAmount = "adjusted_amount"
        case itemCount = "item_count"
        case liveLocationAvailable = "live_location_available"
    }
}

struct ActiveFulfillmentsResponse: Codable {
    let fulfillments: [ActiveFulfillmentItem]
    let count: Int
}

struct PendingPaymentSession: Codable, Identifiable, Hashable {
    var id: String { sessionId ?? orderId }
    let sessionId: String?
    let orderId: String
    let retailerId: String
    let supplierId: String
    let gateway: String?
    let lockedAmount: Int
    let currency: String
    let status: String
    let currentAttemptNo: Int
    let invoiceId: String?
    let redirectURL: String?
    let expiresAt: String?
    let createdAt: String
    let updatedAt: String?

    enum CodingKeys: String, CodingKey {
        case sessionId = "session_id"
        case orderId = "order_id"
        case retailerId = "retailer_id"
        case supplierId = "supplier_id"
        case gateway
        case lockedAmount = "locked_amount"
        case currency
        case status
        case currentAttemptNo = "current_attempt_no"
        case invoiceId = "invoice_id"
        case redirectURL = "redirect_url"
        case expiresAt = "expires_at"
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }
}

struct PendingPaymentsResponse: Codable {
    let pendingPayments: [PendingPaymentSession]
    let count: Int

    enum CodingKeys: String, CodingKey {
        case pendingPayments = "pending_payments"
        case count
    }
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
            totalAmount: 22_450,
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
            totalAmount: 13_490,
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
            totalAmount: 29_980,
            orderSource: "B2B_CHECKOUT",
            createdAt: "2026-03-17T11:00:00Z",
            updatedAt: "2026-03-17T11:00:00Z",
            estimatedDelivery: "2026-03-18T10:00:00Z",
            qrCode: nil // JIT: no token until dispatched
        )
    ]
}
