//
//  Order.swift
//  driverappios
//
//  Matches the Android Order model returned by the Go backend.
//

import Foundation

// MARK: - Order State (matches Android OrderState)

enum OrderState: String, Codable, CaseIterable {
    case PENDING
    case PENDING_REVIEW
    case SCHEDULED
    case LOADED
    case DISPATCHED
    case IN_TRANSIT
    case ARRIVING
    case ARRIVED
    case ARRIVED_SHOP_CLOSED
    case AWAITING_PAYMENT
    case PENDING_CASH_COLLECTION
    case CANCEL_REQUESTED
    case NO_CAPACITY
    case COMPLETED
    case CANCELLED
    case QUARANTINE
    case DELIVERED_ON_CREDIT

    var label: String {
        switch self {
        case .PENDING:                  return "Pending"
        case .PENDING_REVIEW:           return "Under Review"
        case .SCHEDULED:                return "Scheduled"
        case .LOADED:                   return "Loaded"
        case .DISPATCHED:               return "Dispatched"
        case .IN_TRANSIT:               return "In Transit"
        case .ARRIVING:                 return "Arriving"
        case .ARRIVED:                  return "Arrived"
        case .ARRIVED_SHOP_CLOSED:      return "Shop Closed"
        case .AWAITING_PAYMENT:         return "Awaiting Payment"
        case .PENDING_CASH_COLLECTION:  return "Cash Collection"
        case .CANCEL_REQUESTED:         return "Cancel Requested"
        case .NO_CAPACITY:              return "No Capacity"
        case .COMPLETED:                return "Completed"
        case .CANCELLED:                return "Cancelled"
        case .QUARANTINE:               return "Quarantine"
        case .DELIVERED_ON_CREDIT:       return "On Credit"
        }
    }

    var isActive: Bool {
        switch self {
        case .LOADED, .DISPATCHED, .IN_TRANSIT, .ARRIVING, .ARRIVED, .ARRIVED_SHOP_CLOSED, .AWAITING_PAYMENT, .PENDING_CASH_COLLECTION, .DELIVERED_ON_CREDIT:
            return true
        default:
            return false
        }
    }
}

// MARK: - Order Line Item

struct OrderLineItem: Codable, Identifiable, Hashable {
    let productId: String
    let productName: String
    let quantity: Int
    let unitPrice: Int

    var id: String { productId }
    var lineTotal: Int { quantity * unitPrice }

    enum CodingKeys: String, CodingKey {
        case productId = "product_id"
        case productName = "product_name"
        case quantity
        case unitPrice = "unit_price"
    }
}

// MARK: - Order

struct Order: Codable, Identifiable, Hashable {
    let id: String
    let retailerId: String
    let retailerName: String
    let state: OrderState
    let totalAmount: Int
    let deliveryAddress: String
    let latitude: Double
    let longitude: Double
    let qrToken: String
    let paymentGateway: String?
    let createdAt: String
    let updatedAt: String
    let items: [OrderLineItem]
    let estimatedArrivalAt: String?
    let etaDurationSec: Int?
    let etaDistanceM: Int?
    let routeId: String?
    let sequenceIndex: Int?

    enum CodingKeys: String, CodingKey {
        case id
        case retailerId = "retailer_id"
        case retailerName = "retailer_name"
        case state
        case totalAmount = "total_amount"
        case deliveryAddress = "delivery_address"
        case latitude
        case longitude
        case qrToken = "qr_token"
        case paymentGateway = "payment_gateway"
        case createdAt = "created_at"
        case updatedAt = "updated_at"
        case items
        case estimatedArrivalAt = "estimated_arrival_at"
        case etaDurationSec = "eta_duration_sec"
        case etaDistanceM = "eta_distance_m"
        case routeId = "route_id"
        case sequenceIndex = "sequence_index"
    }

    /// Formatted amount in
    var displayTotal: String { totalAmount.formattedAmount }
}

// MARK: - Delivery Submit

struct DeliverySubmitRequest: Codable {
    let orderId: String
    let qrToken: String
    let latitude: Double
    let longitude: Double

    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
        case qrToken = "qr_token"
        case latitude
        case longitude
    }
}

// MARK: - Reorder Stops

struct ReorderStopsRequest: Codable {
    let routeId: String
    let orderSequence: [String]

    enum CodingKeys: String, CodingKey {
        case routeId = "route_id"
        case orderSequence = "order_sequence"
    }
}

struct CollectCashRequest: Codable {
    let orderId: String
    let latitude: Double
    let longitude: Double

    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
        case latitude
        case longitude
    }
}

struct DeliverySubmitResponse: Codable {
    let success: Bool
    let message: String
    let newState: OrderState?

    enum CodingKeys: String, CodingKey {
        case success
        case message
        case newState = "new_state"
    }
}

// MARK: - Amend Order

enum RejectionReason: String, Codable, CaseIterable {
    case DAMAGED
    case MISSING
    case WRONG_ITEM
    case OTHER
}

struct AmendItemPayload: Codable {
    let productId: String
    let acceptedQty: Int
    let rejectedQty: Int
    let reason: String

    enum CodingKeys: String, CodingKey {
        case productId = "product_id"
        case acceptedQty = "accepted_qty"
        case rejectedQty = "rejected_qty"
        case reason
    }
}

struct AmendOrderRequest: Codable {
    let orderId: String
    let items: [AmendItemPayload]
    let driverNotes: String

    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
        case items
        case driverNotes = "driver_notes"
    }
}

struct AmendOrderResponse: Codable {
    let success: Bool
    let message: String
    let adjustedTotal: Int

    enum CodingKeys: String, CodingKey {
        case success
        case message
        case adjustedTotal = "adjusted_total"
    }
}

// MARK: - Validate QR Response

struct ValidateQRResponse: Codable {
    let orderId: String
    let retailerName: String
    let totalAmount: Int
    let state: String
    let items: [OrderLineItem]

    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
        case retailerName = "retailer_name"
        case totalAmount = "total_amount"
        case state
        case items
    }
}

// MARK: - Confirm Offload Response

struct ConfirmOffloadResponse: Codable {
    let orderId: String
    let state: String
    let paymentMethod: String
    let amount: Int
    let invoiceId: String?
    let retailerId: String
    let message: String

    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
        case state
        case paymentMethod = "payment_method"
        case amount = "amount"
        case invoiceId = "invoice_id"
        case retailerId = "retailer_id"
        case message
    }
}

// MARK: - Collect Cash Response

struct CollectCashResponse: Codable {
    let orderId: String
    let state: String
    let amount: Int
    let distanceM: Double
    let message: String

    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
        case state
        case amount = "amount"
        case distanceM = "distance_m"
        case message
    }
}

// MARK: - v3.1 Edge Request/Response Models

struct MissingItemRequest: Codable {
    let skuId: String
    let missingQty: Int

    enum CodingKeys: String, CodingKey {
        case skuId = "sku_id"
        case missingQty = "missing_qty"
    }
}

struct MissingItemsResponse: Codable {
    let status: String
    let orderId: String
    let adjustedTotal: Int64

    enum CodingKeys: String, CodingKey {
        case status
        case orderId = "order_id"
        case adjustedTotal = "adjusted_total"
    }
}

struct SplitPaymentResponse: Codable {
    let status: String
    let orderId: String
    let firstAmount: Int64
    let secondAmount: Int64

    enum CodingKeys: String, CodingKey {
        case status
        case orderId = "order_id"
        case firstAmount = "first_amount"
        case secondAmount = "second_amount"
    }
}

// MARK: - Mock Orders

extension Order {
    static let mockOrders: [Order] = [
        Order(
            id: "ORD-TASH-0056",
            retailerId: "RET-001",
            retailerName: "Korzinka Chilanzar",
            state: .LOADED,
            totalAmount: 1_247_000,
            deliveryAddress: "Chilanzar 9, Block 3",
            latitude: 41.3111,
            longitude: 69.2797,
            qrToken: "ORD-TASH-0056:abc123",
            paymentGateway: "GLOBAL_PAY",
            createdAt: "2026-03-19T08:00:00Z",
            updatedAt: "2026-03-19T08:30:00Z",
            items: [
                OrderLineItem(productId: "SKU-COKE-500", productName: "Coca-Cola 500ml", quantity: 4, unitPrice: 11_980),
                OrderLineItem(productId: "SKU-FANTA-15L", productName: "Fanta 1.5L", quantity: 2, unitPrice: 23_970),
            ],
            estimatedArrivalAt: nil,
            etaDurationSec: nil,
            etaDistanceM: nil,
            routeId: nil,
            sequenceIndex: nil
        ),
        Order(
            id: "ORD-TASH-0057",
            retailerId: "RET-002",
            retailerName: "Makro Sergeli",
            state: .LOADED,
            totalAmount: 856_400,
            deliveryAddress: "Sergeli 7, Market Row B",
            latitude: 41.2887,
            longitude: 69.2044,
            qrToken: "ORD-TASH-0057:def456",
            paymentGateway: "CASH",
            createdAt: "2026-03-19T08:15:00Z",
            updatedAt: "2026-03-19T08:45:00Z",
            items: [
                OrderLineItem(productId: "SKU-SPRITE-2L", productName: "Sprite 2L", quantity: 2, unitPrice: 19_980),
            ],
            estimatedArrivalAt: nil,
            etaDurationSec: nil,
            etaDistanceM: nil,
            routeId: nil,
            sequenceIndex: nil
        ),
        Order(
            id: "ORD-TASH-0058",
            retailerId: "RET-003",
            retailerName: "Havas Yunusabad",
            state: .LOADED,
            totalAmount: 2_100_000,
            deliveryAddress: "Yunusabad 4, Block 12",
            latitude: 41.3275,
            longitude: 69.3341,
            qrToken: "ORD-TASH-0058:ghi789",
            paymentGateway: "GLOBAL_PAY",
            createdAt: "2026-03-19T09:00:00Z",
            updatedAt: "2026-03-19T09:30:00Z",
            items: [
                OrderLineItem(productId: "SKU-TEA-100", productName: "Ahmad Tea 100p", quantity: 10, unitPrice: 45_000),
                OrderLineItem(productId: "SKU-SUGAR-1KG", productName: "Sugar 1kg", quantity: 20, unitPrice: 12_500),
            ],
            estimatedArrivalAt: nil,
            etaDurationSec: nil,
            etaDistanceM: nil,
            routeId: nil,
            sequenceIndex: nil
        ),
    ]
}
