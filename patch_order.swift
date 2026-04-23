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
