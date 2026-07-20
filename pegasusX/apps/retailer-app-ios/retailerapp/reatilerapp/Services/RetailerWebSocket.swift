import Foundation

// MARK: - WebSocket Message Types

struct PaymentRequiredEvent: Decodable, Identifiable {
    var id: String { orderId }
    let type: String
    let orderId: String
    let invoiceId: String
    let sessionId: String
    let amountUzs: Int
    let originalAmountUzs: Int
    let availableCardGateways: [String]
    let message: String
    let paymentMethod: String

    enum CodingKeys: String, CodingKey {
        case type
        case orderId = "order_id"
        case invoiceId = "invoice_id"
        case sessionId = "session_id"
        case amountUzs = "amount"
        case originalAmountUzs = "original_amount"
        case availableCardGateways = "available_card_gateways"
        case message
        case paymentMethod = "payment_method"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        type = try c.decode(String.self, forKey: .type)
        orderId = try c.decode(String.self, forKey: .orderId)
        invoiceId = try c.decodeIfPresent(String.self, forKey: .invoiceId) ?? ""
        sessionId = try c.decodeIfPresent(String.self, forKey: .sessionId) ?? ""
        amountUzs = try c.decodeIfPresent(Int.self, forKey: .amountUzs) ?? 0
        originalAmountUzs = try c.decodeIfPresent(Int.self, forKey: .originalAmountUzs) ?? 0
        availableCardGateways = try c.decodeIfPresent([String].self, forKey: .availableCardGateways) ?? []
        message = try c.decodeIfPresent(String.self, forKey: .message) ?? ""
        paymentMethod = try c.decodeIfPresent(String.self, forKey: .paymentMethod) ?? ""
    }

    init(
        type: String,
        orderId: String,
        invoiceId: String,
        sessionId: String,
        amountUzs: Int,
        originalAmountUzs: Int,
        availableCardGateways: [String],
        message: String,
        paymentMethod: String
    ) {
        self.type = type
        self.orderId = orderId
        self.invoiceId = invoiceId
        self.sessionId = sessionId
        self.amountUzs = amountUzs
        self.originalAmountUzs = originalAmountUzs
        self.availableCardGateways = availableCardGateways
        self.message = message
        self.paymentMethod = paymentMethod
    }
}

struct OrderCompletedEvent: Decodable {
    let type: String
    let orderId: String
    let amountUzs: Int
    let message: String

    enum CodingKeys: String, CodingKey {
        case type
        case orderId = "order_id"
        case amountUzs = "amount"
        case amountMinor = "amount_minor"
        case message
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        type = try c.decodeIfPresent(String.self, forKey: .type) ?? ""
        orderId = try c.decode(String.self, forKey: .orderId)
        if let amount = try c.decodeIfPresent(Int.self, forKey: .amountUzs) {
            amountUzs = amount
        } else {
            amountUzs = try c.decodeIfPresent(Int.self, forKey: .amountMinor) ?? 0
        }
        message = try c.decodeIfPresent(String.self, forKey: .message) ?? ""
    }
}

struct PaymentFailureEvent: Decodable {
    let type: String
    let orderId: String
    let sessionId: String
    let gateway: String
    let message: String

    enum CodingKeys: String, CodingKey {
        case type
        case orderId = "order_id"
        case sessionId = "session_id"
        case gateway
        case message
    }
}

struct ShopClosedAlertEvent: Decodable, Identifiable {
    let type: String
    let orderId: String
    let driverName: String
    let options: [String]
    let attemptId: String

    var id: String { "\(orderId)-\(attemptId)" }

    enum CodingKeys: String, CodingKey {
        case type
        case orderId = "order_id"
        case driverName = "driver_name"
        case options
        case attemptId = "attempt_id"
    }
}

struct PromotionChangedEvent: Decodable {
    let type: String
    let supplierId: String

    enum CodingKeys: String, CodingKey {
        case type
        case supplierId = "supplier_id"
    }
}

struct CartSyncUpdatedEvent: Decodable {
    let type: String
    let retailerId: String
    let itemCount: Int?
    let updatedAt: String?

    enum CodingKeys: String, CodingKey {
        case type
        case retailerId = "retailer_id"
        case itemCount = "item_count"
        case updatedAt = "updated_at"
    }
}

enum RetailerWSEvent {
    case paymentRequired(PaymentRequiredEvent)
    case orderCompleted(OrderCompletedEvent)
    case paymentSettled(OrderCompletedEvent)
    case fiscalizing(orderId: String)
    case fiscalSucceeded(OrderCompletedEvent)
    case paymentFailed(PaymentFailureEvent)
    case paymentExpired(PaymentFailureEvent)
    case driverApproaching(orderId: String, deliveryToken: String, driverLatitude: Double?, driverLongitude: Double?, supplierId: String, supplierName: String)
    case orderStatusChanged(orderId: String, state: String)
    case orderReassigned(orderId: String, licensePlate: String)
    case preOrderAutoAccepted(orderId: String)
    case preOrderConfirmed(orderId: String)
    case preOrderEdited(orderId: String)
    case preOrderNudge(orderId: String)
    case preOrderConfirmationPush(orderId: String)
    case preOrderDateProposed(orderId: String)
    case preOrderDateAccepted(orderId: String)
    case preOrderDateRejected(orderId: String)
    case preOrderCancelled(orderId: String)
    case shopClosedAlert(ShopClosedAlertEvent)
    case cartSyncUpdated(CartSyncUpdatedEvent)
    case promotionChanged(supplierId: String)
    case transportReconnected
}

// MARK: - Retailer WebSocket

@Observable
final class RetailerWebSocket {
    static let shared = RetailerWebSocket()

    private(set) var isConnected = false
    private(set) var reconnectEpoch = 0
    private var task: URLSessionWebSocketTask?
    private var session: URLSession?
    private var retailerId: String?
    private var subscriberContinuations: [UUID: AsyncStream<RetailerWSEvent>.Continuation] = [:]
    private var shouldReconnect = false
    private var reconnectWorkItem: DispatchWorkItem?
    private let subscriberQueue = DispatchQueue(label: "RetailerWebSocket.subscribers")
    
    // Backoff tracking
    private var reconnectAttempts = 0
    private let maxReconnectDelay: TimeInterval = 60.0
    private let initialReconnectDelay: TimeInterval = 2.0

    private init() {}

    /// Returns a multicast event stream so each consumer receives every WebSocket event.
    func eventStream() -> AsyncStream<RetailerWSEvent> {
        AsyncStream { continuation in
            let id = UUID()
            subscriberQueue.sync {
                subscriberContinuations[id] = continuation
            }
            continuation.onTermination = { [weak self] _ in
                self?.subscriberQueue.async {
                    self?.subscriberContinuations.removeValue(forKey: id)
                }
            }
        }
    }

    // MARK: - Connect

    func connect(retailerId: String) {
        guard task == nil else { return }
        self.retailerId = retailerId
        shouldReconnect = true
        reconnectWorkItem?.cancel()
        reconnectWorkItem = nil

        let api = APIClient.shared
        let base = api.baseURL
            .replacingOccurrences(of: "https://", with: "wss://")
            .replacingOccurrences(of: "http://", with: "ws://")

        guard let url = URL(string: "\(base)/v1/ws") else { return }

        var request = URLRequest(url: url)
        if let token = api.authToken {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }
        
        let config = URLSessionConfiguration.default
        config.timeoutIntervalForRequest = 60
        config.timeoutIntervalForResource = 300
        
        // Use a delegate-free session for basic web socket, or handle ping/pong if necessary.
        let session = URLSession(configuration: config)

        self.session = session
        let wsTask = session.webSocketTask(with: request)
        self.task = wsTask
        wsTask.resume()
        let wasReconnect = reconnectAttempts > 0
        isConnected = true
        reconnectAttempts = 0
        if wasReconnect {
            reconnectEpoch += 1
            emit(.transportReconnected)
        }
        receiveNext()
    }

    // MARK: - Disconnect

    func disconnect() {
        shouldReconnect = false
        reconnectWorkItem?.cancel()
        reconnectWorkItem = nil
        task?.cancel(with: .goingAway, reason: nil)
        task = nil
        session?.invalidateAndCancel()
        session = nil
        isConnected = false
    }

    // MARK: - Read Loop

    private func receiveNext() {
        task?.receive { [weak self] result in
            guard let self else { return }
            switch result {
            case .success(let message):
                self.handleMessage(message)
                self.receiveNext()
            case .failure:
                self.isConnected = false
                self.scheduleReconnect()
            }
        }
    }

    // MARK: - Parse

    private func handleMessage(_ message: URLSessionWebSocketTask.Message) {
        let data: Data
        switch message {
        case .string(let text):
            guard let d = text.data(using: .utf8) else { return }
            data = d
        case .data(let d):
            data = d
        @unknown default:
            return
        }

        guard let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
              let type = json["type"] as? String else { return }

        let decoder = JSONDecoder()

        switch type {
        case "PAYMENT_REQUIRED", "GLOBAL_PAYNT_REQUIRED", "SETTLEMENT_REQUIRED":
            if let event = try? decoder.decode(PaymentRequiredEvent.self, from: data) {
                emit(.paymentRequired(event))
            }
        case "ORDER_COMPLETED", "ORDER_FINALIZED", "FISCAL_RECEIPT_SUCCEEDED":
            if let event = try? decoder.decode(OrderCompletedEvent.self, from: data) {
                if type == "FISCAL_RECEIPT_SUCCEEDED" {
                    emit(.fiscalSucceeded(event))
                } else {
                    emit(.orderCompleted(event))
                }
            }
        case "PAYMENT_SETTLED", "GLOBAL_PAYNT_SETTLED", "PAYMENT_CLEARED", "FISCAL_RECEIPT_REQUESTED":
            if type == "FISCAL_RECEIPT_REQUESTED" || type == "PAYMENT_CLEARED" {
                if let orderId = json["order_id"] as? String {
                    emit(.fiscalizing(orderId: orderId))
                }
            } else if let event = try? decoder.decode(OrderCompletedEvent.self, from: data) {
                // Money settled — UI still waits for fiscal success (ADR-009).
                emit(.paymentSettled(event))
                emit(.fiscalizing(orderId: event.orderId))
            }
        case "PAYMENT_FAILED", "GLOBAL_PAYNT_FAILED":
            if let event = try? decoder.decode(PaymentFailureEvent.self, from: data) {
                emit(.paymentFailed(event))
            }
        case "PAYMENT_EXPIRED", "GLOBAL_PAYNT_EXPIRED":
            if let event = try? decoder.decode(PaymentFailureEvent.self, from: data) {
                emit(.paymentExpired(event))
            }
        case "DRIVER_APPROACHING":
            if let orderId = json["order_id"] as? String,
               let token = json["delivery_token"] as? String {
                let driverLat = json["driver_latitude"] as? Double
                let driverLng = json["driver_longitude"] as? Double
                let supplierId = json["supplier_id"] as? String ?? ""
                let supplierName = json["supplier_name"] as? String ?? ""
                emit(.driverApproaching(orderId: orderId, deliveryToken: token, driverLatitude: driverLat, driverLongitude: driverLng, supplierId: supplierId, supplierName: supplierName))
            }
        case "ORDER_STATUS_CHANGED":
            if let orderId = json["order_id"] as? String {
                let state = (json["state"] as? String ?? json["status"] as? String ?? "").uppercased()
                emit(.orderStatusChanged(orderId: orderId, state: state))
                if state == "FISCALIZING" {
                    emit(.fiscalizing(orderId: orderId))
                } else if state == "COMPLETED" {
                    if let event = try? decoder.decode(OrderCompletedEvent.self, from: data) {
                        emit(.orderCompleted(event))
                    }
                }
            }
        case "DELIVERY_SESSION_UPDATED":
            if let orderId = json["order_id"] as? String {
                let state = json["state"] as? String ?? ""
                emit(.orderStatusChanged(orderId: orderId, state: state))
            }
        case "ORDER_REASSIGNED":
            let orderId = json["order_id"] as? String ?? ""
            let licensePlate = json["license_plate"] as? String ?? "Unknown"
            emit(.orderReassigned(orderId: orderId, licensePlate: licensePlate))
        case "PRE_ORDER_AUTO_ACCEPTED", "PRE_ORDER_CONFIRMED", "PRE_ORDER_EDITED", "PRE_ORDER_NUDGE", "PRE_ORDER_CONFIRMATION",
             "PRE_ORDER_DATE_PROPOSED", "PRE_ORDER_DATE_ACCEPTED", "PRE_ORDER_DATE_REJECTED", "PRE_ORDER_CANCELLED":
            if let orderId = json["order_id"] as? String {
                switch type {
                case "PRE_ORDER_AUTO_ACCEPTED": emit(.preOrderAutoAccepted(orderId: orderId))
                case "PRE_ORDER_CONFIRMED": emit(.preOrderConfirmed(orderId: orderId))
                case "PRE_ORDER_EDITED": emit(.preOrderEdited(orderId: orderId))
                case "PRE_ORDER_NUDGE": emit(.preOrderNudge(orderId: orderId))
                case "PRE_ORDER_CONFIRMATION": emit(.preOrderConfirmationPush(orderId: orderId))
                case "PRE_ORDER_DATE_PROPOSED": emit(.preOrderDateProposed(orderId: orderId))
                case "PRE_ORDER_DATE_ACCEPTED": emit(.preOrderDateAccepted(orderId: orderId))
                case "PRE_ORDER_DATE_REJECTED": emit(.preOrderDateRejected(orderId: orderId))
                case "PRE_ORDER_CANCELLED": emit(.preOrderCancelled(orderId: orderId))
                default: break
                }
            }
        case "SHOP_CLOSED", "SHOP_CLOSED_ALERT":
            if let event = try? decoder.decode(ShopClosedAlertEvent.self, from: data) {
                emit(.shopClosedAlert(event))
            } else if let orderId = json["order_id"] as? String, !orderId.isEmpty {
                let driverName = (json["driver_name"] as? String)
                    ?? (json["driver_id"] as? String)
                    ?? "Driver"
                let attemptId = json["attempt_id"] as? String ?? ""
                let options = json["options"] as? [String]
                    ?? ["OPEN_NOW", "5_MIN", "CALL_ME", "CLOSED_TODAY"]
                let event = ShopClosedAlertEvent(
                    type: type,
                    orderId: orderId,
                    driverName: driverName,
                    options: options,
                    attemptId: attemptId
                )
                emit(.shopClosedAlert(event))
            }
        case "CART_SYNC_UPDATED":
            if let event = try? decoder.decode(CartSyncUpdatedEvent.self, from: data) {
                emit(.cartSyncUpdated(event))
            }
        case "PROMOTION_CHANGED":
            if let payload = try? decoder.decode(PromotionChangedEvent.self, from: data) {
                emit(.promotionChanged(supplierId: payload.supplierId))
            }
        default:
            break
        }
    }

    // MARK: - Reconnect

    private func scheduleReconnect() {
        guard shouldReconnect, let retailerId else { return }
        task = nil
        session?.invalidateAndCancel()
        session = nil

        // Exponential backoff with full jitter (Desert Protocol)
        reconnectAttempts += 1
        let baseDelay = initialReconnectDelay * pow(2.0, Double(reconnectAttempts - 1))
        let maxDelay = min(baseDelay, maxReconnectDelay)
        let jitter = Double.random(in: 0...(maxDelay / 2))
        let finalDelay = min(maxDelay + jitter, maxReconnectDelay)
        
        let workItem = DispatchWorkItem { [weak self] in
            guard let self, self.shouldReconnect else { return }
            self.connect(retailerId: retailerId)
        }
        reconnectWorkItem = workItem
        DispatchQueue.main.asyncAfter(deadline: .now() + finalDelay, execute: workItem)
    }

    private func emit(_ event: RetailerWSEvent) {
        let continuations = subscriberQueue.sync {
            Array(subscriberContinuations.values)
        }
        for continuation in continuations {
            continuation.yield(event)
        }
    }
}
