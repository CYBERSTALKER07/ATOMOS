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
        case message
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

enum RetailerWSEvent {
    case paymentRequired(PaymentRequiredEvent)
    case orderCompleted(OrderCompletedEvent)
    case paymentSettled(OrderCompletedEvent)
    case paymentFailed(PaymentFailureEvent)
    case paymentExpired(PaymentFailureEvent)
    case driverApproaching(orderId: String, deliveryToken: String, driverLatitude: Double?, driverLongitude: Double?, supplierId: String, supplierName: String)
    case orderStatusChanged(orderId: String, state: String)
    case preOrderAutoAccepted(orderId: String)
    case preOrderConfirmed(orderId: String)
    case preOrderEdited(orderId: String)
}

// MARK: - Retailer WebSocket

    @Observable
final class RetailerWebSocket {
    static let shared = RetailerWebSocket()

    private(set) var isConnected = false
    private var task: URLSessionWebSocketTask?
    private var session: URLSession?
    private var retailerId: String?
    private var eventContinuation: AsyncStream<RetailerWSEvent>.Continuation?
    
    // Backoff tracking
    private var reconnectAttempts = 0
    private let maxReconnectDelay: TimeInterval = 60.0
    private let initialReconnectDelay: TimeInterval = 2.0

    /// AsyncStream that views can iterate to receive real-time events.
    private(set) var events: AsyncStream<RetailerWSEvent>!

    private init() {
        resetStream()
    }

    private func resetStream() {
        events = AsyncStream { continuation in
            self.eventContinuation = continuation
        }
    }

    // MARK: - Connect

    func connect(retailerId: String) {
        guard task == nil else { return }
        self.retailerId = retailerId

        let api = APIClient.shared
        let base = api.baseURL
            .replacingOccurrences(of: "https://", with: "wss://")
            .replacingOccurrences(of: "http://", with: "ws://")

        guard let url = URL(string: "\(base)/v1/ws/retailer?retailer_id=\(retailerId)") else { return }

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
        isConnected = true
        reconnectAttempts = 0
        receiveNext()
    }

    // MARK: - Disconnect

    func disconnect() {
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
        case "PAYMENT_REQUIRED":
            if let event = try? decoder.decode(PaymentRequiredEvent.self, from: data) {
                eventContinuation?.yield(.paymentRequired(event))
            }
        case "ORDER_COMPLETED":
            if let event = try? decoder.decode(OrderCompletedEvent.self, from: data) {
                eventContinuation?.yield(.orderCompleted(event))
            }
        case "PAYMENT_SETTLED":
            if let event = try? decoder.decode(OrderCompletedEvent.self, from: data) {
                eventContinuation?.yield(.paymentSettled(event))
            }
        case "PAYMENT_FAILED":
            if let event = try? decoder.decode(PaymentFailureEvent.self, from: data) {
                eventContinuation?.yield(.paymentFailed(event))
            }
        case "PAYMENT_EXPIRED":
            if let event = try? decoder.decode(PaymentFailureEvent.self, from: data) {
                eventContinuation?.yield(.paymentExpired(event))
            }
        case "DRIVER_APPROACHING":
            if let orderId = json["order_id"] as? String,
               let token = json["delivery_token"] as? String {
                let driverLat = json["driver_latitude"] as? Double
                let driverLng = json["driver_longitude"] as? Double
                let supplierId = json["supplier_id"] as? String ?? ""
                let supplierName = json["supplier_name"] as? String ?? ""
                eventContinuation?.yield(.driverApproaching(orderId: orderId, deliveryToken: token, driverLatitude: driverLat, driverLongitude: driverLng, supplierId: supplierId, supplierName: supplierName))
            }
        case "ORDER_STATUS_CHANGED":
            if let orderId = json["order_id"] as? String {
                let state = json["state"] as? String ?? ""
                eventContinuation?.yield(.orderStatusChanged(orderId: orderId, state: state))
            }
        case "PRE_ORDER_AUTO_ACCEPTED", "PRE_ORDER_CONFIRMED", "PRE_ORDER_EDITED":
            if let orderId = json["order_id"] as? String {
                switch type {
                case "PRE_ORDER_AUTO_ACCEPTED": eventContinuation?.yield(.preOrderAutoAccepted(orderId: orderId))
                case "PRE_ORDER_CONFIRMED": eventContinuation?.yield(.preOrderConfirmed(orderId: orderId))
                case "PRE_ORDER_EDITED": eventContinuation?.yield(.preOrderEdited(orderId: orderId))
                default: break
                }
            }
        default:
            break
        }
    }

    // MARK: - Reconnect

    private func scheduleReconnect() {
        guard let retailerId else { return }
        task = nil
        session?.invalidateAndCancel()
        session = nil

        // Exponential backoff with jitter
        reconnectAttempts += 1
        let baseDelay = initialReconnectDelay * pow(2.0, Double(reconnectAttempts - 1))
        let maxDelay = min(baseDelay, maxReconnectDelay)
        
        // Add random jitter (-10% to +10%)
        let jitter = Double.random(in: -0.1...0.1) * maxDelay
        let delayWithJitter = maxDelay + jitter
        
        let finalDelay = max(initialReconnectDelay, min(delayWithJitter, maxReconnectDelay))
        
        print("WebSocket disconnected. Scheduled reconnect in \(String(format: "%.2f", finalDelay)) seconds")

        DispatchQueue.main.asyncAfter(deadline: .now() + finalDelay) { [weak self] in
            // Connect will bypass guard if task == nil, which it is
            self?.connect(retailerId: retailerId)
        }
    }
}
