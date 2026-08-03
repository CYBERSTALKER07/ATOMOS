//
//  APIClient.swift
//  driverappios
//

import CoreLocation
import Foundation

// MARK: - API Errors

enum APIError: Error {
    case unauthorized       // 401
    case forbidden          // 403
    case httpError(Int)     // other HTTP errors
    case problemDetail(ProblemDetail) // RFC 7807 structured error
    case explainError(message: String, explain: StatusExplain?)
    case networkError       // connectivity
    case decodingError      // JSON parse failure
    case invalidURL
}

// MARK: - RFC 7807 Problem Detail

struct ProblemDetail: Codable {
    let type: String?
    let title: String?
    let status: Int
    let detail: String?
    let traceId: String?
    let instance: String?
    let code: String?
    let messageKey: String?
    let retryable: Bool?
    let action: String?

    enum CodingKeys: String, CodingKey {
        case type, title, status, detail, instance, code, retryable, action
        case traceId = "trace_id"
        case messageKey = "message_key"
    }
}

// MARK: - API Client

final class APIClient: @unchecked Sendable {
    static let shared = APIClient()

    #if DEBUG
    // Simulator: resolves to localhost. Physical device: set PEGASUS_DEV_HOST
    // scheme env variable (Edit Scheme → Run → Arguments →
    // Environment Variables) to the Mac's LAN IP (e.g. 192.168.1.42). Supports
    // bare host, host:port, or full scheme URL.
    let apiBaseURL: String = {
        let raw = (ProcessInfo.processInfo.environment["PEGASUS_DEV_HOST"] ?? "")
            .trimmingCharacters(in: .whitespaces)
        if raw.isEmpty { return "http://localhost:8180" }
        if raw.hasPrefix("http://") || raw.hasPrefix("https://") { return raw }
        return raw.contains(":") ? "http://\(raw)" : "http://\(raw):8180"
    }()
    #else
    let apiBaseURL = "https://api.pegasus.uz"
    #endif

    private var baseURL: String { apiBaseURL }

    private let session: URLSession
    private let decoder: JSONDecoder

    private init() {
        let config = URLSessionConfiguration.default
        config.timeoutIntervalForRequest = 15
        config.timeoutIntervalForResource = 30
        session = URLSession(configuration: config)

        decoder = JSONDecoder()
        decoder.keyDecodingStrategy = .convertFromSnakeCase
    }

    // MARK: - Auth

    func login(phone: String, pin: String) async throws -> AuthResponse {
        let body = LoginRequest(phone: phone, pin: pin)
        return try await post("v1/auth/driver/login", body: body, authenticated: false)
    }

    func login(idToken: String) async throws -> AuthResponse {
        var body = LoginRequest()
        body.idToken = idToken
        return try await post("v1/auth/driver/login", body: body, authenticated: false)
    }

    func registerDeviceToken(token: String, platform: String = "ios") async throws {
        let body = ["token": token, "platform": platform]
        let _: [String: String] = try await post("v1/user/device-token", body: body)
    }

    // MARK: - Fleet

    func getAssignedOrders() async throws -> [Order] {
        try await get("v1/fleet/orders")
    }

    func getRouteGeometry(
        routeId: String,
        includeSteps: Bool = true,
        from coordinate: CLLocationCoordinate2D? = nil,
        reroute: Bool = false
    ) async throws -> RouteGeometryResponse {
        var query = "include_steps=\(includeSteps ? "true" : "false")"
        if reroute, let coordinate {
            query += "&reroute=true&from_lat=\(coordinate.latitude)&from_lng=\(coordinate.longitude)"
        }
        return try await get("v1/fleet/route/\(routeId)/geometry?\(query)")
    }

    func getManifest(date: String) async throws -> RouteManifest {
        try await get("v1/driver/manifest?date=\(date)")
    }

    // MARK: - Driver Earnings & Cash

    /// GET /v1/driver/earnings — lifetime totals + per-day breakdown for last 30 days
    func getEarnings() async throws -> DriverEarningsResponse {
        try await get("v1/driver/earnings")
    }

    /// GET /v1/driver/pending-collections — outstanding cash collections (cash shadow recovery)
    func getPendingCollections() async throws -> [PendingCollection] {
        try await get("v1/driver/pending-collections")
    }

    // MARK: - Orders

    func getOrder(id: String) async throws -> Order {
        try await get("v1/orders/\(id)")
    }

    func submitDelivery(orderId: String, qrToken: String, latitude: Double, longitude: Double) async throws -> DeliverySubmitResponse {
        let body = DeliverySubmitRequest(
            orderId: orderId,
            qrToken: qrToken,
            latitude: latitude,
            longitude: longitude
        )
        return try await post(
            "v1/order/deliver",
            body: body,
            headers: ["Idempotency-Key": DriverIdempotency.deliver(orderId: orderId)]
        )
    }

    func amendOrder(request: AmendOrderRequest) async throws -> AmendOrderResponse {
        try await post(
            "v1/order/amend",
            body: request,
            headers: ["Idempotency-Key": DriverIdempotency.amendOrder(orderId: request.orderId, items: request.items)]
        )
    }

    func validateQR(orderId: String, scannedToken: String) async throws -> ValidateQRResponse {
        let body = ["order_id": orderId, "scanned_token": scannedToken]
        return try await post("v1/order/validate-qr", body: body)
    }

    func confirmOffload(orderId: String) async throws -> ConfirmOffloadResponse {
        let body = ["order_id": orderId]
        return try await post(
            "v1/order/confirm-offload",
            body: body,
            headers: ["Idempotency-Key": DriverIdempotency.offload(orderId: orderId)]
        )
    }

    /// POST /v1/delivery/scan-qr — ARRIVED → AWAITING_PAYMENT (canonical doorstep transition)
    func scanDeliveryQR(orderId: String, qrToken: String) async throws -> DeliveryScanQRResponse {
        struct Req: Encodable {
            let order_id: String
            let qr_token: String
        }
        return try await post(
            "v1/delivery/scan-qr",
            body: Req(order_id: orderId, qr_token: qrToken),
            headers: ["Idempotency-Key": DriverIdempotency.offload(orderId: orderId)]
        )
    }

    func completeOrder(orderId: String) async throws {
        struct Resp: Decodable { let status: String }
        let body = ["order_id": orderId]
        let _: Resp = try await post(
            "v1/order/complete",
            body: body,
            headers: ["Idempotency-Key": DriverIdempotency.complete(orderId: orderId)]
        )
    }

    func collectCash(orderId: String, latitude: Double, longitude: Double, amountReceivedMinor: Int64? = nil) async throws -> CollectCashResponse {
        let body = CollectCashRequest(
            orderId: orderId,
            latitude: latitude,
            longitude: longitude,
            amountReceivedMinor: amountReceivedMinor
        )
        return try await post(
            "v1/order/collect-cash",
            body: body,
            headers: ["Idempotency-Key": DriverIdempotency.collectCash(orderId: orderId)]
        )
    }

    /// POST /v1/order/{id}/fiscal/retry — ADR-009
    func retryFiscal(orderId: String) async throws -> CollectCashResponse {
        struct Empty: Encodable {}
        return try await post(
            "v1/order/\(orderId)/fiscal/retry",
            body: Empty(),
            headers: ["Idempotency-Key": DriverIdempotency.fiscalRetry(orderId: orderId)]
        )
    }

    func transitionState(orderId: String, newState: String) async throws -> Order {
        let body = ["state": newState]
        return try await patch(
            "v1/orders/\(orderId)/state",
            body: body,
            headers: ["Idempotency-Key": DriverIdempotency.transitionState(orderId: orderId, newState: newState)]
        )
    }

    // MARK: - Dynamic Delivery Handshake

    func verifyHandshake(orderId: String, token: String, latitude: Double, longitude: Double) async throws -> VerifyHandshakeResponse {
        let body = VerifyHandshakeRequest(orderId: orderId, token: token, latitude: latitude, longitude: longitude)
        return try await post("v1/delivery/verify-handshake", body: body)
    }

    func updateOrderDuringDelivery(orderId: String, latitude: Double, longitude: Double) async throws -> UpdateOrderDuringDeliveryResponse {
        let body = UpdateOrderDuringDeliveryRequest(orderId: orderId, latitude: latitude, longitude: longitude)
        return try await post("v1/delivery/update-order-during-delivery", body: body)
    }

    /// Mark arrived — driver enters 500m geofence (IN_TRANSIT → ARRIVED)
    func markArrived(orderId: String) async throws {
        struct Resp: Decodable { let status: String; let orderId: String }
        let body = ["order_id": orderId]
        let _: Resp = try await post(
            "v1/delivery/arrive",
            body: body,
            headers: ["Idempotency-Key": DriverIdempotency.markArrived(orderId: orderId)]
        )
    }

    // MARK: - Shop Closed / Proximity / Partial

    func reportShopClosed(
        orderId: String,
        latitude: Double? = nil,
        longitude: Double? = nil,
        reason: String? = nil,
        photoURL: String? = nil,
        clientTimestamp: String? = nil
    ) async throws -> [String: String] {
        struct Req: Encodable {
            let order_id: String
            let latitude: Double?
            let longitude: Double?
            let reason: String?
            let photo_url: String?
            let client_timestamp: String?
        }
        return try await post(
            "v1/delivery/shop-closed",
            body: Req(
                order_id: orderId,
                latitude: latitude,
                longitude: longitude,
                reason: reason,
                photo_url: photoURL,
                client_timestamp: clientTimestamp
            ),
            headers: ["Idempotency-Key": DriverIdempotency.reportShopClosed(orderId: orderId)]
        )
    }

    /// Unlock payment modes at the stop (H3 or ≤100 m). Required before cash/credit.
    func proximityUnlock(
        orderId: String,
        latitude: Double,
        longitude: Double,
        clientTimestamp: String? = nil,
        forceBypassToken: String? = nil
    ) async throws -> [String: String] {
        struct Req: Encodable {
            let order_id: String
            let latitude: Double
            let longitude: Double
            let client_timestamp: String?
            let force_bypass_token: String?
        }
        return try await post(
            "v1/delivery/proximity-unlock",
            body: Req(
                order_id: orderId,
                latitude: latitude,
                longitude: longitude,
                client_timestamp: clientTimestamp,
                force_bypass_token: forceBypassToken
            ),
            headers: ["Idempotency-Key": DriverIdempotency.proximityUnlock(orderId: orderId)]
        )
    }

    /// Line-level partial offload. Each line: delivered_qty + remaining_qty == original qty.
    func partialOffload(
        orderId: String,
        lines: [PartialOffloadLineRequest],
        clientTimestamp: String? = nil,
        signedNonce: String? = nil,
        note: String? = nil
    ) async throws -> [String: String] {
        struct Req: Encodable {
            let order_id: String
            let lines: [PartialOffloadLineRequest]
            let client_timestamp: String?
            let signed_nonce: String?
            let note: String?
        }
        let fingerprint = lines
            .map { "\($0.sku):\($0.delivered_qty):\($0.remaining_qty)" }
            .sorted()
            .joined(separator: "|")
        return try await post(
            "v1/delivery/partial-offload",
            body: Req(
                order_id: orderId,
                lines: lines,
                client_timestamp: clientTimestamp,
                signed_nonce: signedNonce,
                note: note
            ),
            headers: ["Idempotency-Key": DriverIdempotency.partialOffload(orderId: orderId, fingerprint: fingerprint)]
        )
    }

    func bypassOffload(orderId: String, token: String) async throws -> [String: String] {
        let body = ["order_id": orderId, "bypass_token": token]
        return try await post(
            "v1/delivery/bypass-offload",
            body: body,
            headers: ["Idempotency-Key": DriverIdempotency.bypassOffload(orderId: orderId)]
        )
    }

    func confirmPaymentBypass(orderId: String, token: String) async throws -> [String: String] {
        let body = ["order_id": orderId, "bypass_token": token]
        return try await post(
            "v1/delivery/confirm-payment-bypass",
            body: body,
            headers: ["Idempotency-Key": DriverIdempotency.confirmPaymentBypass(orderId: orderId)]
        )
    }

    // MARK: - WebSocket Command Handshake

    /// POST /v1/ws/ack — confirms native receipt/settlement of a verified command.
    func ackWebSocketCommand(commandId: String, traceId: String?, eventType: String) async throws {
        struct AckRequest: Encodable {
            let commandId: String
            let traceId: String?
            let eventType: String

            enum CodingKeys: String, CodingKey {
                case commandId = "command_id"
                case traceId = "trace_id"
                case eventType = "event_type"
            }
        }
        struct AckResponse: Decodable {
            let status: String
            let commandId: String

            enum CodingKeys: String, CodingKey {
                case status
                case commandId = "command_id"
            }
        }

        let _: AckResponse = try await post(
            "v1/ws/ack",
            body: AckRequest(commandId: commandId, traceId: traceId, eventType: eventType)
        )
    }

    // MARK: - Fleet Dispatch

    func depart(truckId: String) async throws -> [String: String] {
        let body = DepartRequest(truckId: truckId)
        return try await post(
            "v1/fleet/driver/depart",
            body: body,
            headers: ["Idempotency-Key": DriverIdempotency.depart(truckId: truckId)]
        )
    }

    /// LEO: Ghost Stop Prevention — check if manifest is sealed before depart
    func checkManifestGate(manifestId: String) async throws -> ManifestGateResult {
        let request = try buildRequest(path: "v1/driver/manifest-gate?manifest_id=\(manifestId)", method: "GET")
        let (data, response) = try await dataWithFallback(for: request)
        guard let http = response as? HTTPURLResponse else { throw APIError.networkError }
        if http.statusCode == 200 || http.statusCode == 403 {
            let gate = try decoder.decode(ManifestGateResponse.self, from: data)
            let parsed = ApiExplainParser.parse(from: data)
            return ManifestGateResult(gate: gate, explain: gate.explain ?? parsed?.explain)
        }
        if let parsed = ApiExplainParser.parse(from: data) {
            throw APIError.explainError(message: parsed.message, explain: parsed.explain)
        }
        if let problem = Self.parseProblemDetail(data: data, response: http) {
            throw APIError.problemDetail(problem)
        }
        throw APIError.httpError(http.statusCode)
    }

    func getFleetManifest() async throws -> [String: AnyDecodable] {
        return try await get("v1/fleet/manifest")
    }

    struct OpenFiscalResponse: Decodable {
        let openFiscalCount: Int64
        let orderIds: [String]?
        let cashBagFrozen: Bool

        enum CodingKeys: String, CodingKey {
            case openFiscalCount = "open_fiscal_count"
            case orderIds = "order_ids"
            case cashBagFrozen = "cash_bag_frozen"
        }
    }

    func getOpenFiscal() async throws -> OpenFiscalResponse {
        try await get("v1/driver/open-fiscal")
    }

    func returnComplete(truckId: String) async throws -> [String: String] {
        let body = ReturnCompleteRequest(truckId: truckId)
        return try await post(
            "v1/fleet/driver/return-complete",
            body: body,
            headers: ["Idempotency-Key": DriverIdempotency.returnComplete(truckId: truckId)]
        )
    }

    struct CashReconciliationRow: Decodable {
        let reconciliationId: String
        let expectedCashMinor: Int64
        let declaredCashMinor: Int64
        let differenceMinor: Int64
        let status: String

        enum CodingKeys: String, CodingKey {
            case reconciliationId = "reconciliation_id"
            case expectedCashMinor = "expected_cash_minor"
            case declaredCashMinor = "declared_cash_minor"
            case differenceMinor = "difference_minor"
            case status
        }
    }

    struct CashReconciliationsResponse: Decodable {
        let reconciliations: [CashReconciliationRow]
    }

    func listCashReconciliations() async throws -> CashReconciliationsResponse {
        try await get("v1/driver/cash-reconciliations")
    }

    func submitCashReconciliation(declaredCashMinor: Int64, driverNote: String?) async throws -> CashReconciliationRow {
        struct Body: Encodable {
            let declaredCashMinor: Int64
            let driverNote: String?

            enum CodingKeys: String, CodingKey {
                case declaredCashMinor = "declared_cash_minor"
                case driverNote = "driver_note"
            }
        }
        let body = Body(declaredCashMinor: declaredCashMinor, driverNote: driverNote)
        return try await post(
            "v1/driver/cash-reconciliations",
            body: body,
            headers: ["Idempotency-Key": DriverIdempotency.cashReconciliation(declaredMinor: declaredCashMinor)]
        )
    }

    func getReturnGoods() async throws -> ReturnGoodsResponse {
        try await get("v1/driver/return-goods")
    }

    // MARK: - Driver Session

    func setAvailability(available: Bool, reason: String? = nil, note: String? = nil) async throws {
        struct Req: Encodable { let available: Bool; let reason: String?; let note: String? }
        struct Resp: Decodable { let status: String }
        let _: Resp = try await post(
            "v1/driver/availability",
            body: Req(available: available, reason: reason, note: note),
            headers: ["Idempotency-Key": DriverIdempotency.availability(onShift: available, reason: reason ?? "", note: note)]
        )
    }

    func getAvailability() async throws -> [String: AnyDecodable] {
        return try await get("v1/driver/availability")
    }

    func updateAvailability(payload: [String: AnyEncodable]) async throws -> [String: String] {
        let onShift = (payload["on_shift"]?.value as? Bool) ?? false
        let reason = (payload["reason"]?.value as? String) ?? ""
        let note = (payload["note"]?.value as? String) ?? ""
        return try await patch(
            "v1/driver/availability",
            body: payload,
            headers: ["Idempotency-Key": DriverIdempotency.availability(onShift: onShift, reason: reason, note: note)]
        )
    }

    func reorderStops(routeId: String, orderSequence: [String]) async throws -> RouteReorderResponse {
        let body = ReorderStopsRequest(routeId: routeId, orderSequence: orderSequence)
        return try await post(
            "v1/fleet/route/reorder",
            body: body,
            headers: ["Idempotency-Key": DriverIdempotency.routeReorder(routeId: routeId, orderSequence: orderSequence)]
        )
    }

    // MARK: - v3.1 Human-Centric Edges

    /// Edge 27: Request early route completion (fatigue/issue)
    func requestEarlyComplete(reason: String, note: String) async throws -> EarlyCompleteRequestResponse {
        struct Req: Encodable { let reason: String; let note: String }
        return try await post(
            "v1/fleet/route/request-early-complete",
            body: Req(reason: reason, note: note),
            headers: ["Idempotency-Key": DriverIdempotency.requestEarlyComplete(reason: reason)]
        )
    }

    // MARK: - Rescue Operations

    func requestRescue(reason: String, note: String) async throws -> [String: String] {
        struct Req: Encodable { let reason: String; let note: String }
        return try await post(
            "v1/driver/ops/rescue/request",
            body: Req(reason: reason, note: note)
        )
    }

    func respondRescue(rescueId: String, accept: Bool) async throws -> [String: String] {
        struct Req: Encodable { let rescue_id: String; let accept: Bool }
        return try await post(
            "v1/driver/ops/rescue/respond",
            body: Req(rescue_id: rescueId, accept: accept)
        )
    }

    func reassignHandshake(orderId: String) async throws -> [String: String] {
        return try await post(
            "v1/fleet/orders/\(orderId)/reassign-handshake",
            body: EmptyBody(),
            headers: ["Idempotency-Key": UUID().uuidString]
        )
    }

    // Quantity negotiation product-disabled — backend returns 410 feature_disabled.
    // Not a substitute for missing-items / credit-leave / shop-closed.
    // func proposeNegotiation(orderId: String, items: [NegotiationItemRequest]) async throws -> NegotiationProposalResponse { ... }

    /// Edge 32: Mark order as delivered on credit (requires proximity unlock or force_bypass_token).
    func markCreditDelivery(
        orderId: String,
        photoProofUrl: String? = nil,
        forceBypassToken: String? = nil
    ) async throws -> [String: String] {
        var body: [String: String] = ["order_id": orderId]
        if let url = photoProofUrl { body["photo_proof_url"] = url }
        if let token = forceBypassToken { body["force_bypass_token"] = token }
        return try await post(
            "v1/delivery/credit-delivery",
            body: body,
            headers: ["Idempotency-Key": DriverIdempotency.creditDelivery(orderId: orderId)]
        )
    }

    /// Edge 33: Report missing/damaged items (exception-report alias). DAMAGED needs photo_url.
    func reportMissingItems(
        orderId: String,
        missingItems: [MissingItemRequest],
        photoURL: String? = nil,
        note: String? = nil
    ) async throws -> MissingItemsResponse {
        struct Req: Encodable {
            let order_id: String
            let missing_items: [MissingItemRequest]
            let photo_url: String?
            let note: String?
        }
        return try await post(
            "v1/delivery/missing-items",
            body: Req(
                order_id: orderId,
                missing_items: missingItems,
                photo_url: photoURL,
                note: note
            ),
            headers: ["Idempotency-Key": DriverIdempotency.missingItems(orderId: orderId)]
        )
    }

    /// Edge 35: Create split payment
    func splitPayment(orderId: String, cashMinor: Int64, cardMinor: Int64, currency: String? = nil) async throws -> SplitPaymentResponse {
        struct Req: Encodable {
            let order_id: String
            let cash_minor: Int64
            let card_minor: Int64
            let currency: String?
        }
        return try await post(
            "v1/delivery/split-payment",
            body: Req(order_id: orderId, cash_minor: cashMinor, card_minor: cardMinor, currency: currency),
            headers: ["Idempotency-Key": DriverIdempotency.splitPayment(orderId: orderId, cashMinor: cashMinor, cardMinor: cardMinor)]
        )
    }

    // MARK: - Factory Supply Transfers

    func getSupplyTransfers() async throws -> SupplyTransfersResponse {
        try await get("v1/driver/supply-transfers")
    }

    func arriveSupplyTransfer(transferId: String, latitude: Double, longitude: Double) async throws -> ArriveSupplyTransferResponse {
        struct Body: Encodable {
            let latitude: Double
            let longitude: Double
        }
        return try await post(
            "v1/driver/supply-transfers/\(transferId)/arrive",
            body: Body(latitude: latitude, longitude: longitude),
            headers: ["Idempotency-Key": DriverIdempotency.supplyTransferArrive(transferId: transferId)]
        )
    }

    // MARK: - Driver Profile

    func getDriverProfile() async throws -> DriverProfileResponse {
        try await get("v1/driver/profile")
    }

    func getDriverHistory() async throws -> DriverHistoryResponse {
        return try await get("v1/driver/history")
    }

    func getPulse() async throws -> PulseResponse {
        try await get("v1/driver/pulse")
    }

    // MARK: - Generic HTTP

    private struct EmptyBody: Encodable {}

    func get<T: Decodable>(_ path: String) async throws -> T {
        let request = try buildRequest(path: path, method: "GET")
        return try await execute(request)
    }

    func post<B: Encodable, T: Decodable>(
        _ path: String,
        body: B,
        authenticated: Bool = true,
        headers: [String: String] = [:]
    ) async throws -> T {
        var request = try buildRequest(path: path, method: "POST", authenticated: authenticated)
        for (name, value) in headers {
            request.setValue(value, forHTTPHeaderField: name)
        }
        request.httpBody = try JSONEncoder().encode(body)
        return try await execute(request)
    }

    private func patch<B: Encodable, T: Decodable>(
        _ path: String,
        body: B,
        headers: [String: String] = [:]
    ) async throws -> T {
        var request = try buildRequest(path: path, method: "PATCH")
        for (name, value) in headers {
            request.setValue(value, forHTTPHeaderField: name)
        }
        request.httpBody = try JSONEncoder().encode(body)
        return try await execute(request)
    }

    private func deterministicIdempotencyKey(action: String, orderId: String) -> String {
        "driver-\(action)-\(orderId)"
    }

    /// Replay a queued offline action. Returns (statusCode, raw bytes).
    func rawRequest(
        endpoint: String,
        method: String,
        body: String,
        idempotencyKey: String? = nil
    ) async throws -> (Int, Data) {
        let path = endpoint.hasPrefix("/") ? String(endpoint.dropFirst()) : endpoint
        var req = try buildRequest(path: path, method: method)
        if let idempotencyKey, !idempotencyKey.isEmpty {
            req.setValue(idempotencyKey, forHTTPHeaderField: "Idempotency-Key")
        }
        if !body.isEmpty {
            req.httpBody = body.data(using: .utf8)
        }
        let (data, response) = try await dataWithFallback(for: req)
        guard let http = response as? HTTPURLResponse else { throw APIError.networkError }
        return (http.statusCode, data)
    }

    private func buildRequest(path: String, method: String, authenticated: Bool = true) throws -> URLRequest {
        guard let url = URL(string: "\(baseURL)/\(path)") else {
            throw APIError.invalidURL
        }
        var request = URLRequest(url: url)
        request.httpMethod = method
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.setValue(UUID().uuidString, forHTTPHeaderField: "X-Trace-Id")

        if authenticated, let token = TokenStore.shared.token {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }
        return request
    }

    private func dataWithFallback(for request: URLRequest) async throws -> (Data, URLResponse) {
        try await session.data(for: request)
    }

    /// Flag to prevent recursive refresh loops
    private var isRefreshing = false

    private func execute<T: Decodable>(_ request: URLRequest, isRetry: Bool = false) async throws -> T {
        let data: Data
        let response: URLResponse

        do {
            (data, response) = try await dataWithFallback(for: request)
        } catch {
            throw APIError.networkError
        }

        guard let http = response as? HTTPURLResponse else {
            throw APIError.networkError
        }

        switch http.statusCode {
        case 200...299:
            break
        case 401:
            // On first 401, attempt silent token refresh before giving up
            if !isRetry && !isRefreshing {
                if let newToken = await attemptTokenRefresh() {
                    // Re-build the request with fresh token and retry once
                    var retryRequest = request
                    retryRequest.setValue("Bearer \(newToken)", forHTTPHeaderField: "Authorization")
                    return try await execute(retryRequest, isRetry: true)
                }
            }
            // Refresh failed or already retried — surface structured error if available
            if let problem = Self.parseProblemDetail(data: data, response: http) {
                throw APIError.problemDetail(problem)
            }
            await MainActor.run { TokenStore.shared.logout() }
            throw APIError.unauthorized
        case 403:
            if let problem = Self.parseProblemDetail(data: data, response: http) {
                throw APIError.problemDetail(problem)
            }
            throw APIError.forbidden
        default:
            if let problem = Self.parseProblemDetail(data: data, response: http) {
                throw APIError.problemDetail(problem)
            }
            if http.statusCode == 429, let dict = try? JSONSerialization.jsonObject(with: data) as? [String: Any], let errStr = dict["error"] as? String, errStr == "rate_limit_exceeded" {
                let problem = ProblemDetail(type: "about:blank", title: "Too many requests", status: 429, detail: "Too many requests. Please try again later.", traceId: nil, instance: nil, code: "rate_limit_exceeded", messageKey: nil, retryable: true, action: nil)
                throw APIError.problemDetail(problem)
            }
            throw APIError.httpError(http.statusCode)
        }

        do {
            return try decoder.decode(T.self, from: data)
        } catch {
            throw APIError.decodingError
        }
    }

    // MARK: - Problem Detail

    private static func parseProblemDetail(data: Data, response: HTTPURLResponse) -> ProblemDetail? {
        let contentType = response.value(forHTTPHeaderField: "Content-Type") ?? ""
        guard contentType.contains("application/problem+json") else { return nil }
        return try? JSONDecoder().decode(ProblemDetail.self, from: data)
    }

    // MARK: - Token Refresh

    private func attemptTokenRefresh() async -> String? {
        guard let currentToken = await MainActor.run(body: { TokenStore.shared.token }) else {
            return nil
        }
        isRefreshing = true
        defer { isRefreshing = false }

        guard let url = URL(string: "\(baseURL)/v1/auth/refresh") else { return nil }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.setValue("Bearer \(currentToken)", forHTTPHeaderField: "Authorization")

        guard let (data, response) = try? await session.data(for: request),
              let http = response as? HTTPURLResponse, http.statusCode == 200 else {
            return nil
        }

        struct RefreshResponse: Decodable { let token: String }
        guard let refreshed = try? JSONDecoder().decode(RefreshResponse.self, from: data) else {
            return nil
        }

        // Persist the new token
        await MainActor.run { TokenStore.shared.updateToken(refreshed.token) }
        return refreshed.token
    }
}

// MARK: - Type Erased Codable

struct AnyEncodable: Encodable {
    private let _encode: (Encoder) throws -> Void
    let value: Any

    init(_ wrapped: any Encodable) {
        _encode = wrapped.encode
        value = wrapped
    }

    func encode(to encoder: Encoder) throws {
        try _encode(encoder)
    }
}

struct AnyDecodable: Decodable {
    let value: Any

    init(from decoder: Decoder) throws {
        let container = try decoder.singleValueContainer()
        if let bool = try? container.decode(Bool.self) {
            value = bool
        } else if let int = try? container.decode(Int.self) {
            value = int
        } else if let double = try? container.decode(Double.self) {
            value = double
        } else if let string = try? container.decode(String.self) {
            value = string
        } else if let array = try? container.decode([AnyDecodable].self) {
            value = array.map(\.value)
        } else if let dict = try? container.decode([String: AnyDecodable].self) {
            value = dict.mapValues(\.value)
        } else if container.decodeNil() {
            value = NSNull()
        } else {
            throw DecodingError.dataCorruptedError(in: container, debugDescription: "Unsupported JSON value")
        }
    }
}

// MARK: - Fleet Dispatch Request DTOs

/// LEO: Ghost Stop Prevention gate response
struct ManifestGateResponse: Decodable {
    let cleared: Bool
    let state: String?
    let manifestId: String?
    let error: String?
    let message: String?
    let explain: StatusExplain?

    enum CodingKeys: String, CodingKey {
        case cleared, state, error, message, explain
        case manifestId = "manifest_id"
    }
}

struct ManifestGateResult {
    let gate: ManifestGateResponse
    let explain: StatusExplain?
}

/// Edge 27 response payload for /v1/fleet/route/request-early-complete.
struct EarlyCompleteRequestResponse: Decodable {
    let status: String
    let orderCount: Int
    let orderIds: [String]

    enum CodingKeys: String, CodingKey {
        case status
        case orderCount = "order_count"
        case orderIds = "order_ids"
    }
}

/// Edge 28 request item for POST /v1/delivery/negotiate.
struct NegotiationItemRequest: Encodable {
    let skuId: String
    let originalQty: Int64
    let proposedQty: Int64

    enum CodingKeys: String, CodingKey {
        case skuId = "sku_id"
        case originalQty = "original_qty"
        case proposedQty = "proposed_qty"
    }
}

/// Edge 28 response for POST /v1/delivery/negotiate.
struct NegotiationProposalResponse: Decodable {
    let status: String
    let proposalId: String

    enum CodingKeys: String, CodingKey {
        case status
        case proposalId = "proposal_id"
    }
}

/// Response payload for /v1/fleet/route/reorder.
struct RouteReorderResponse: Decodable {
    let status: String
    let routeId: String
    let stopCount: Int

    enum CodingKeys: String, CodingKey {
        case status
        case routeId = "route_id"
        case stopCount = "stop_count"
    }
}

/// One line for POST /v1/delivery/partial-offload.
struct PartialOffloadLineRequest: Encodable {
    let sku: String
    let delivered_qty: Int64
    let remaining_qty: Int64
    let reason: String?
}

private struct DepartRequest: Encodable {
    let truckId: String

    enum CodingKeys: String, CodingKey {
        case truckId = "truck_id"
    }
}

private struct ReturnCompleteRequest: Encodable {
    let truckId: String

    enum CodingKeys: String, CodingKey {
        case truckId = "truck_id"
    }
}

struct SupplyTransferRow: Decodable, Identifiable {
    let transferId: String
    let warehouseId: String
    let supplyRequestId: String?
    let state: String
    let totalVolumeVu: Double

    var id: String { transferId }

    enum CodingKeys: String, CodingKey {
        case transferId = "transfer_id"
        case warehouseId = "warehouse_id"
        case supplyRequestId = "supply_request_id"
        case state
        case totalVolumeVu = "total_volume_vu"
    }
}

struct SupplyTransfersResponse: Decodable {
    let transfers: [SupplyTransferRow]
}

struct ArriveSupplyTransferResponse: Decodable {
    let transferId: String
    let state: String
    let eventType: String?

    enum CodingKeys: String, CodingKey {
        case transferId = "transfer_id"
        case state
        case eventType = "event_type"
    }
}

struct ReturnGoodsLine: Decodable, Identifiable {
    var id: String { returnId }
    let returnId: String
    let orderId: String
    let skuId: String
    let productName: String
    let quantity: Int64
    let reason: String

    enum CodingKeys: String, CodingKey {
        case returnId = "return_id"
        case orderId = "order_id"
        case skuId = "sku_id"
        case productName = "product_name"
        case quantity, reason
    }
}

struct ReturnGoodsResponse: Decodable {
    let items: [ReturnGoodsLine]
    let totalUnits: Int64
    let lineCount: Int

    enum CodingKeys: String, CodingKey {
        case items
        case totalUnits = "total_units"
        case lineCount = "line_count"
    }
}
