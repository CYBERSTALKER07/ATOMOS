//
//  APIClient.swift
//  driverappios
//

import Foundation

// MARK: - API Errors

enum APIError: Error {
    case unauthorized       // 401
    case forbidden          // 403
    case httpError(Int)     // other HTTP errors
    case problemDetail(ProblemDetail) // RFC 7807 structured error
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
        if raw.isEmpty { return "http://localhost:8080" }
        if raw.hasPrefix("http://") || raw.hasPrefix("https://") { return raw }
        return raw.contains(":") ? "http://\(raw)" : "http://\(raw):8080"
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

    // MARK: - Fleet

    func getAssignedOrders() async throws -> [Order] {
        try await get("v1/fleet/orders")
    }

    func getManifest(date: String) async throws -> RouteManifest {
        try await get("v1/fleet/manifest?date=\(date)")
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
        return try await post("v1/order/deliver", body: body)
    }

    func amendOrder(request: AmendOrderRequest) async throws -> AmendOrderResponse {
        try await post("v1/order/amend", body: request)
    }

    func validateQR(orderId: String, scannedToken: String) async throws -> ValidateQRResponse {
        let body = ["order_id": orderId, "scanned_token": scannedToken]
        return try await post("v1/order/validate-qr", body: body)
    }

    func confirmOffload(orderId: String) async throws -> ConfirmOffloadResponse {
        let body = ["order_id": orderId]
        return try await post("v1/order/confirm-offload", body: body)
    }

    func completeOrder(orderId: String) async throws {
        struct Resp: Decodable { let status: String }
        let body = ["order_id": orderId]
        let _: Resp = try await post("v1/order/complete", body: body)
    }

    func collectCash(orderId: String, latitude: Double, longitude: Double) async throws -> CollectCashResponse {
        let body = CollectCashRequest(orderId: orderId, latitude: latitude, longitude: longitude)
        return try await post("v1/order/collect-cash", body: body)
    }

    func transitionState(orderId: String, newState: String) async throws -> Order {
        let body = ["state": newState]
        return try await patch("v1/orders/\(orderId)/state", body: body)
    }

    /// Mark arrived — driver enters 100m geofence (IN_TRANSIT → ARRIVED)
    func markArrived(orderId: String) async throws {
        struct Resp: Decodable { let status: String; let orderId: String }
        let body = ["order_id": orderId]
        let _: Resp = try await post("v1/delivery/arrive", body: body)
    }

    // MARK: - Shop Closed

    func reportShopClosed(orderId: String) async throws -> [String: String] {
        let body = ["order_id": orderId]
        return try await post("v1/delivery/shop-closed", body: body)
    }

    func bypassOffload(orderId: String, token: String) async throws -> [String: String] {
        let body = ["order_id": orderId, "bypass_token": token]
        return try await post("v1/delivery/bypass-offload", body: body)
    }

    func confirmPaymentBypass(orderId: String, token: String) async throws -> [String: String] {
        let body = ["order_id": orderId, "bypass_token": token]
        return try await post("v1/delivery/confirm-payment-bypass", body: body)
    }

    // MARK: - Fleet Dispatch

    func depart(truckId: String) async throws -> [String: String] {
        let body = DepartRequest(truckId: truckId)
        return try await post("v1/fleet/driver/depart", body: body)
    }

    /// LEO: Ghost Stop Prevention — check if manifest is sealed before depart
    func checkManifestGate(manifestId: String) async throws -> ManifestGateResponse {
        return try await get("v1/driver/manifest-gate?manifest_id=\(manifestId)")
    }

    func returnComplete(truckId: String) async throws -> [String: String] {
        let body = ReturnCompleteRequest(truckId: truckId)
        return try await post("v1/fleet/driver/return-complete", body: body)
    }

    // MARK: - Driver Session

    func setAvailability(available: Bool, reason: String? = nil, note: String? = nil) async throws {
        struct Req: Encodable { let available: Bool; let reason: String?; let note: String? }
        struct Resp: Decodable { let status: String }
        let _: Resp = try await post("v1/driver/availability", body: Req(available: available, reason: reason, note: note))
    }

    func reorderStops(routeId: String, orderSequence: [String]) async throws -> RouteReorderResponse {
        let body = ReorderStopsRequest(routeId: routeId, orderSequence: orderSequence)
        return try await post("v1/fleet/route/reorder", body: body)
    }

    // MARK: - v3.1 Human-Centric Edges

    /// Edge 27: Request early route completion (fatigue/issue)
    func requestEarlyComplete(reason: String, note: String) async throws -> EarlyCompleteRequestResponse {
        struct Req: Encodable { let reason: String; let note: String }
        return try await post("v1/fleet/route/request-early-complete", body: Req(reason: reason, note: note))
    }

    /// Edge 28: Propose quantity negotiation to supplier
    func proposeNegotiation(orderId: String, items: [NegotiationItemRequest]) async throws -> NegotiationProposalResponse {
        struct Req: Encodable {
            let orderId: String
            let items: [NegotiationItemRequest]

            enum CodingKeys: String, CodingKey {
                case orderId = "order_id"
                case items
            }
        }
        return try await post("v1/delivery/negotiate", body: Req(orderId: orderId, items: items))
    }

    /// Edge 32: Mark order as delivered on credit
    func markCreditDelivery(orderId: String, photoProofUrl: String? = nil) async throws -> [String: String] {
        var body: [String: String] = ["order_id": orderId]
        if let url = photoProofUrl { body["photo_proof_url"] = url }
        return try await post("v1/delivery/credit-delivery", body: body)
    }

    /// Edge 33: Report missing items after seal
    func reportMissingItems(orderId: String, missingItems: [MissingItemRequest]) async throws -> MissingItemsResponse {
        struct Req: Encodable { let order_id: String; let missing_items: [MissingItemRequest] }
        return try await post("v1/delivery/missing-items", body: Req(order_id: orderId, missing_items: missingItems))
    }

    /// Edge 35: Create split payment
    func splitPayment(orderId: String, firstAmount: Int64, secondAmount: Int64) async throws -> SplitPaymentResponse {
        struct Req: Encodable { let order_id: String; let first_amount: Int64; let second_amount: Int64 }
        return try await post("v1/delivery/split-payment", body: Req(order_id: orderId, first_amount: firstAmount, second_amount: secondAmount))
    }

    // MARK: - Driver Profile

    func getDriverProfile() async throws -> DriverProfileResponse {
        try await get("v1/driver/profile")
    }

    // MARK: - Generic HTTP

    func get<T: Decodable>(_ path: String) async throws -> T {
        let request = try buildRequest(path: path, method: "GET")
        return try await execute(request)
    }

    func post<B: Encodable, T: Decodable>(_ path: String, body: B, authenticated: Bool = true) async throws -> T {
        var request = try buildRequest(path: path, method: "POST", authenticated: authenticated)
        request.httpBody = try JSONEncoder().encode(body)
        return try await execute(request)
    }

    private func patch<B: Encodable, T: Decodable>(_ path: String, body: B) async throws -> T {
        var request = try buildRequest(path: path, method: "PATCH")
        request.httpBody = try JSONEncoder().encode(body)
        return try await execute(request)
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

// MARK: - Fleet Dispatch Request DTOs

/// LEO: Ghost Stop Prevention gate response
struct ManifestGateResponse: Decodable {
    let cleared: Bool
    let state: String?
    let manifestId: String?

    enum CodingKeys: String, CodingKey {
        case cleared, state
        case manifestId = "manifest_id"
    }
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

/// Edge 28 request item payload for /v1/delivery/negotiate.
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

/// Edge 28 response payload for /v1/delivery/negotiate.
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
