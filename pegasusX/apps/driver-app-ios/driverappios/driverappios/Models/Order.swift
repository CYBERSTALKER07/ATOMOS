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
    case AUTO_ACCEPTED
    case BACKORDERED
    case DELAYED
    case LOADED
    case DISPATCHED
    case IN_TRANSIT
    case ARRIVING
    case ARRIVED
    case SHOP_CLOSED_PENDING
    case ARRIVED_SHOP_CLOSED
    case AWAITING_PAYMENT
    case PENDING_CASH_COLLECTION
    case FISCALIZING
    case FISCAL_FAILED
    case CANCEL_REQUESTED
    case NO_CAPACITY
    case COMPLETED
    case CANCELLED
    case QUARANTINE
    case DELIVERED_ON_CREDIT
    case RECONCILIATION_REQUIRED

    init(from decoder: Decoder) throws {
        let container = try decoder.singleValueContainer()
        let raw = try container.decode(String.self)
        self = OrderState(rawValue: raw) ?? .PENDING
    }

    var label: String {
        switch self {
        case .PENDING:                  return "Pending"
        case .PENDING_REVIEW:           return "Under Review"
        case .SCHEDULED:                return "Scheduled"
        case .AUTO_ACCEPTED:            return "Auto-Accepted"
        case .BACKORDERED:              return "Backordered"
        case .DELAYED:                  return "Delayed"
        case .LOADED:                   return "Loaded"
        case .DISPATCHED:               return "Dispatched"
        case .IN_TRANSIT:               return "In Transit"
        case .ARRIVING:                 return "Arriving"
        case .ARRIVED:                  return "Arrived"
        case .SHOP_CLOSED_PENDING:      return "Shop Closed"
        case .ARRIVED_SHOP_CLOSED:      return "Shop Closed"
        case .AWAITING_PAYMENT:         return "Awaiting Payment"
        case .PENDING_CASH_COLLECTION:  return "Cash Collection"
        case .FISCALIZING:              return "Fiscalizing"
        case .FISCAL_FAILED:            return "Fiscal Failed"
        case .CANCEL_REQUESTED:         return "Cancel Requested"
        case .NO_CAPACITY:              return "No Capacity"
        case .COMPLETED:                return "Completed"
        case .CANCELLED:                return "Cancelled"
        case .QUARANTINE:               return "Quarantine"
        case .DELIVERED_ON_CREDIT:       return "On Credit"
        case .RECONCILIATION_REQUIRED:  return "Reconciliation Required"
        }
    }

    var customerStage: CustomerStage {
        switch self {
        case .SCHEDULED, .AUTO_ACCEPTED, .BACKORDERED, .PENDING_REVIEW:
            return .PLACED
        case .PENDING, .DELAYED, .LOADED, .DISPATCHED:
            return .PROCESSING
        case .IN_TRANSIT, .ARRIVING, .ARRIVED,
             .AWAITING_PAYMENT, .PENDING_CASH_COLLECTION:
            return .OUT_FOR_DELIVERY
        case .COMPLETED, .DELIVERED_ON_CREDIT, .FISCALIZING:
            return .DELIVERED
        case .CANCELLED, .NO_CAPACITY:
            return .CANCELLED
        case .SHOP_CLOSED_PENDING, .ARRIVED_SHOP_CLOSED, .CANCEL_REQUESTED,
             .FISCAL_FAILED, .RECONCILIATION_REQUIRED, .QUARANTINE:
            return .EXCEPTION_ACTION_REQUIRED
        }
    }

    var isActive: Bool {
        switch self {
        case .LOADED, .DISPATCHED, .IN_TRANSIT, .ARRIVING, .ARRIVED, .ARRIVED_SHOP_CLOSED,
             .AWAITING_PAYMENT, .PENDING_CASH_COLLECTION, .FISCALIZING, .FISCAL_FAILED, .DELIVERED_ON_CREDIT:
            return true
        default:
            return false
        }
    }
}

// MARK: - Customer Stage (High-level 4-step progress facade matching backend)

enum CustomerStage: String, Codable, CaseIterable {
    case PLACED
    case PROCESSING
    case OUT_FOR_DELIVERY
    case DELIVERED
    case CANCELLED
    case EXCEPTION_ACTION_REQUIRED

    var displayName: String {
        switch self {
        case .PLACED:                     return "Order Placed"
        case .PROCESSING:                 return "Preparing Order"
        case .OUT_FOR_DELIVERY:           return "Out for Delivery"
        case .DELIVERED:                  return "Delivered"
        case .CANCELLED:                  return "Cancelled"
        case .EXCEPTION_ACTION_REQUIRED:  return "Action Required"
        }
    }

    var progressPercent: Int {
        switch self {
        case .PLACED:                     return 20
        case .PROCESSING:                 return 50
        case .OUT_FOR_DELIVERY:           return 80
        case .DELIVERED:                  return 100
        case .CANCELLED:                  return 0
        case .EXCEPTION_ACTION_REQUIRED:  return 75
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
    let deliveryFeeMinor: Int?
    let deliveryDistanceKm: Double?
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
    let isPartial: Bool?
    let splitGroupId: String?

    enum CodingKeys: String, CodingKey {
        case id
        case retailerId = "retailer_id"
        case retailerName = "retailer_name"
        case state
        case totalAmount = "total_amount"
        case deliveryFeeMinor = "delivery_fee_minor"
        case deliveryDistanceKm = "delivery_distance_km"
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
        case isPartial = "is_partial"
        case splitGroupId = "split_group_id"
    }

    /// Formatted amount in
    var displayTotal: String { totalAmount.formattedAmount }

    var deliveryFeeLabel: String? {
        guard let fee = deliveryFeeMinor, fee > 0 else { return nil }
        return "+\(fee.formattedAmount) delivery fee"
    }

    static let samples: [Order] = [
        Order(
            id: "ord-tash-101",
            retailerId: "ret-chilonzor-4",
            retailerName: "Chilonzor Supermarket #4",
            state: .IN_TRANSIT,
            totalAmount: 1_450_000,
            deliveryFeeMinor: 25_000,
            deliveryDistanceKm: 4.2,
            deliveryAddress: "Bunyodkor Ave 15, Chilonzor, Tashkent",
            latitude: 41.2858,
            longitude: 69.2035,
            qrToken: "qr-ord-tash-101-token",
            paymentGateway: "CASH",
            createdAt: "2026-11-01T08:00:00Z",
            updatedAt: "2026-11-01T08:30:00Z",
            items: [
                OrderLineItem(productId: "prod-cola-15", productName: "Coca-Cola Classic 1.5L (Pack of 6)", quantity: 10, unitPrice: 72000),
                OrderLineItem(productId: "prod-water-5", productName: "Nestle Pure Life 5L", quantity: 20, unitPrice: 18000),
                OrderLineItem(productId: "prod-oil-1", productName: "O'Ble 1L Sunflower Oil", quantity: 15, unitPrice: 24000)
            ],
            estimatedArrivalAt: "10:45 AM",
            etaDurationSec: 900,
            etaDistanceM: 4200,
            routeId: "route-tashkent-south-1",
            sequenceIndex: 1,
            isPartial: false,
            splitGroupId: nil
        ),
        Order(
            id: "ord-tash-102",
            retailerId: "ret-yunusobod-12",
            retailerName: "Yunusobod Eco Market",
            state: .DISPATCHED,
            totalAmount: 2_890_000,
            deliveryFeeMinor: 30_000,
            deliveryDistanceKm: 7.8,
            deliveryAddress: "Amir Temur Street 108, Yunusobod, Tashkent",
            latitude: 41.3456,
            longitude: 69.2891,
            qrToken: "qr-ord-tash-102-token",
            paymentGateway: "PAYME",
            createdAt: "2026-11-01T08:15:00Z",
            updatedAt: "2026-11-01T08:45:00Z",
            items: [
                OrderLineItem(productId: "prod-sugar-50", productName: "Khorazm Sugar 50kg Sack", quantity: 2, unitPrice: 650000),
                OrderLineItem(productId: "prod-flour-50", productName: "Qozogiston Un 1-nav 50kg", quantity: 2, unitPrice: 420000),
                OrderLineItem(productId: "prod-rice-25", productName: "Alanga Rice 25kg", quantity: 3, unitPrice: 250000)
            ],
            estimatedArrivalAt: "11:30 AM",
            etaDurationSec: 1800,
            etaDistanceM: 7800,
            routeId: "route-tashkent-south-1",
            sequenceIndex: 2,
            isPartial: false,
            splitGroupId: nil
        )
    ]
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
    /// Cash actually taken (Tiyin). Fiscal hard-gate uses this amount (ADR-009 Phase 3).
    let amountReceivedMinor: Int64?
    let note: String?

    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
        case latitude
        case longitude
        case amountReceivedMinor = "amount_received_minor"
        case note
    }

    init(orderId: String, latitude: Double, longitude: Double, amountReceivedMinor: Int64? = nil, note: String? = nil) {
        self.orderId = orderId
        self.latitude = latitude
        self.longitude = longitude
        self.amountReceivedMinor = amountReceivedMinor
        self.note = note
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
    let customReason: String?

    enum CodingKeys: String, CodingKey {
        case productId = "product_id"
        case acceptedQty = "accepted_qty"
        case rejectedQty = "rejected_qty"
        case reason
        case customReason = "custom_reason"
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

// MARK: - Delivery Scan QR Response

struct DeliveryScanQRResponse: Codable {
    let valid: Bool
    let orderId: String
    let state: String

    enum CodingKeys: String, CodingKey {
        case valid
        case orderId = "order_id"
        case state
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        valid = (try? c.decode(Bool.self, forKey: .valid)) ?? false
        orderId = (try? c.decode(String.self, forKey: .orderId)) ?? ""
        state = (try? c.decode(String.self, forKey: .state)) ?? "AWAITING_PAYMENT"
    }
}

// MARK: - Collect Cash Response

struct CollectCashResponse: Codable {
    let orderId: String
    let state: String
    let amount: Int
    let amountReceivedMinor: Int?
    let shortfallMinor: Int?
    let overageMinor: Int?
    let distanceM: Double
    let message: String
    let attemptId: String?
    let fiscalStatus: String?
    let currency: String?

    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
        case state
        case amount = "amount"
        case amountReceivedMinor = "amount_received_minor"
        case shortfallMinor = "shortfall_minor"
        case overageMinor = "overage_minor"
        case distanceM = "distance_m"
        case message
        case attemptId = "attempt_id"
        case fiscalStatus = "fiscal_status"
        case currency
    }
}

// MARK: - v3.1 Edge Request/Response Models

struct VerifyHandshakeRequest: Codable {
    let orderId: String
    let token: String
    let latitude: Double
    let longitude: Double

    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
        case token
        case latitude
        case longitude
    }
}

struct VerifyHandshakeResponse: Codable {
    let success: Bool
    let message: String
}

struct UpdateOrderDuringDeliveryRequest: Codable {
    let orderId: String
    let latitude: Double
    let longitude: Double

    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
        case latitude
        case longitude
    }
}

struct UpdateOrderDuringDeliveryResponse: Codable {
    let success: Bool
    let message: String
}

struct MissingItemRequest: Codable {
    let skuId: String
    let missingQty: Int
    let reason: String?
    let photoURL: String?

    enum CodingKeys: String, CodingKey {
        case skuId = "sku_id"
        case missingQty = "missing_qty"
        case reason
        case photoURL = "photo_url"
    }

    init(skuId: String, missingQty: Int, reason: String? = nil, photoURL: String? = nil) {
        self.skuId = skuId
        self.missingQty = missingQty
        self.reason = reason
        self.photoURL = photoURL
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
}

