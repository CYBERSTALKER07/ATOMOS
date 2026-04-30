//
//  APIClient.swift
//  payload-app-ios
//
//  Single networking surface for every endpoint the Expo payload-terminal calls.
//  Backend route paths verified against authroutes/, payloaderroutes/,
//  adminroutes/, deliveryroutes/, fleetroutes/, userroutes/. No backend changes.
//

import Foundation

enum APIError: Error {
    case unauthorized
    case forbidden
    case httpError(Int)
    case problemDetail(ProblemDetail)
    case networkError
    case decodingError
    case invalidURL
}

struct ProblemDetail: Decodable {
    let type: String?
    let title: String?
    let status: Int
    let detail: String?
    let traceId: String?
    let code: String?
    let retryable: Bool?
}

final class APIClient: @unchecked Sendable {
    static let shared = APIClient()

    #if DEBUG
    /// Simulator: localhost. Physical iPad: set `PEGASUS_DEV_HOST`
    /// env var (Edit Scheme → Run → Environment Variables)
    /// to your Mac's LAN IP.
    let baseURL: String = {
        let raw = (ProcessInfo.processInfo.environment["PEGASUS_DEV_HOST"] ?? "")
            .trimmingCharacters(in: .whitespaces)
        if raw.isEmpty { return "http://localhost:8080" }
        if raw.hasPrefix("http://") || raw.hasPrefix("https://") { return raw }
        return raw.contains(":") ? "http://\(raw)" : "http://\(raw):8080"
    }()
    #else
    let baseURL = "https://api.thelab.uz"
    #endif

    /// WebSocket origin derived from baseURL: http → ws, https → wss.
    var wsBaseURL: String {
        if baseURL.hasPrefix("https://") {
            return "wss://" + baseURL.dropFirst("https://".count)
        }
        if baseURL.hasPrefix("http://") {
            return "ws://" + baseURL.dropFirst("http://".count)
        }
        return baseURL
    }

    private let session: URLSession
    private let decoder: JSONDecoder
    private let encoder: JSONEncoder

    private init() {
        let cfg = URLSessionConfiguration.default
        cfg.timeoutIntervalForRequest = 15
        cfg.timeoutIntervalForResource = 30
        session = URLSession(configuration: cfg)
        decoder = JSONDecoder()
        decoder.keyDecodingStrategy = .convertFromSnakeCase
        encoder = JSONEncoder()
    }

    // MARK: - Auth
    func login(phone: String, pin: String) async throws -> LoginResponse {
        try await post("v1/auth/payloader/login", body: LoginRequest(phone: phone, pin: pin), authenticated: false)
    }

    // MARK: - Trucks / Orders
    func trucks() async throws -> [Truck] { try await get("v1/payloader/trucks") }
    func orders(vehicleId: String?, state: String? = nil) async throws -> [LiveOrder] {
        var params: [String] = []
        if let v = vehicleId { params.append("vehicle_id=\(v)") }
        if let s = state { params.append("state=\(s)") }
        let q = params.isEmpty ? "" : "?" + params.joined(separator: "&")
        return try await get("v1/payloader/orders\(q)")
    }
    func recommendReassign(orderId: String) async throws -> RecommendReassignResponse {
        try await post("v1/payloader/recommend-reassign", body: RecommendReassignRequest(orderId: orderId))
    }

    // MARK: - Manifest lifecycle
    func draftManifests(truckId: String?) async throws -> ManifestsResponse {
        try await manifests(state: "DRAFT", truckId: truckId)
    }
    func manifests(state: String, truckId: String?) async throws -> ManifestsResponse {
        var q = "?state=\(state)"
        if let t = truckId { q += "&truck_id=\(t)" }
        return try await get("v1/supplier/manifests\(q)")
    }
    func manifestDetail(_ manifestId: String) async throws -> Manifest {
        try await get("v1/supplier/manifests/\(manifestId)")
    }
    func startLoading(manifestId: String) async throws -> StatusResponse {
        try await post("v1/supplier/manifests/\(manifestId)/start-loading", body: EmptyBody())
    }
    func sealManifest(manifestId: String) async throws -> SealManifestResponse {
        try await post("v1/supplier/manifests/\(manifestId)/seal", body: EmptyBody())
    }
    func injectOrder(manifestId: String, orderId: String) async throws -> StatusResponse {
        try await post("v1/supplier/manifests/\(manifestId)/inject-order", body: InjectOrderRequest(orderId: orderId))
    }

    // MARK: - Per-order seal / exception
    /// Backend wants {order_id, terminal_id, manifest_cleared}. Per Expo,
    /// terminal_id is the active vehicle/truck id.
    func sealOrder(orderId: String, terminalId: String) async throws -> SealOrderResponse {
        try await post("v1/payload/seal",
                       body: SealOrderRequest(orderId: orderId, terminalId: terminalId, manifestCleared: true))
    }
    func manifestException(manifestId: String, orderId: String, reason: String, metadata: String = "") async throws -> ManifestExceptionResponse {
        try await post("v1/payload/manifest-exception",
                       body: ManifestExceptionRequest(manifestId: manifestId, orderId: orderId, reason: reason, metadata: metadata))
    }
    func reportMissingItems(orderId: String, items: [MissingItemEntry]) async throws -> StatusResponse {
        try await post("v1/delivery/missing-items", body: MissingItemsRequest(orderId: orderId, missingItems: items))
    }

    // MARK: - Fleet reassign
    func fleetReassign(orderIds: [String], newRouteId: String) async throws -> FleetReassignResponse {
        try await post("v1/fleet/reassign", body: FleetReassignRequest(orderIds: orderIds, newRouteId: newRouteId))
    }

    // MARK: - Notifications
    func notifications(limit: Int = 50) async throws -> NotificationsResponse {
        try await get("v1/user/notifications?limit=\(limit)")
    }
    func markRead(ids: [String]?, all: Bool? = nil) async throws -> StatusResponse {
        try await post("v1/user/notifications/read", body: MarkReadRequest(notificationIds: ids, markAll: all))
    }

    // MARK: - FCM
    func registerDeviceToken(_ token: String) async throws -> StatusResponse {
        try await post("v1/user/device-token", body: DeviceTokenRequest(token: token, platform: "IOS"))
    }
    func unregisterDeviceToken(_ token: String) async throws -> StatusResponse {
        var req = try buildRequest(path: "v1/user/device-token", method: "DELETE")
        req.httpBody = try encoder.encode(DeviceTokenRequest(token: token, platform: "IOS"))
        return try await execute(req)
    }

    // MARK: - Raw replay (offline queue)
    /// Replay a queued action with arbitrary endpoint/method/body. Returns
    /// (statusCode, raw bytes) so the caller can decide retention vs drop.
    func rawRequest(endpoint: String, method: String, body: String) async throws -> (Int, Data) {
        let path = endpoint.hasPrefix("/") ? String(endpoint.dropFirst()) : endpoint
        var req = try buildRequest(path: path, method: method)
        if !body.isEmpty { req.httpBody = body.data(using: .utf8) }
        let (data, response) = try await session.data(for: req)
        guard let http = response as? HTTPURLResponse else { throw APIError.networkError }
        return (http.statusCode, data)
    }

    // MARK: - Generic plumbing

    private struct EmptyBody: Encodable {}

    private func get<T: Decodable>(_ path: String) async throws -> T {
        let req = try buildRequest(path: path, method: "GET")
        return try await execute(req)
    }

    private func post<B: Encodable, T: Decodable>(_ path: String, body: B, authenticated: Bool = true) async throws -> T {
        var req = try buildRequest(path: path, method: "POST", authenticated: authenticated)
        req.httpBody = try encoder.encode(body)
        return try await execute(req)
    }

    private func buildRequest(path: String, method: String, authenticated: Bool = true) throws -> URLRequest {
        guard let url = URL(string: "\(baseURL)/\(path)") else { throw APIError.invalidURL }
        var req = URLRequest(url: url)
        req.httpMethod = method
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        req.setValue(UUID().uuidString, forHTTPHeaderField: "X-Trace-Id")
        if authenticated, let token = TokenStore.shared.token {
            req.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }
        return req
    }

    private func execute<T: Decodable>(_ request: URLRequest) async throws -> T {
        let data: Data; let response: URLResponse
        do { (data, response) = try await session.data(for: request) }
        catch { throw APIError.networkError }

        guard let http = response as? HTTPURLResponse else { throw APIError.networkError }
        switch http.statusCode {
        case 200...299:
            if T.self == StatusResponse.self, data.isEmpty {
                // Backend may return empty body on no-content success.
                return StatusResponse(status: "ok") as! T
            }
            do { return try decoder.decode(T.self, from: data) }
            catch { throw APIError.decodingError }
        case 401:
            await MainActor.run { TokenStore.shared.logout() }
            throw APIError.unauthorized
        case 403:
            if let p = parseProblem(data) { throw APIError.problemDetail(p) }
            throw APIError.forbidden
        default:
            if let p = parseProblem(data) { throw APIError.problemDetail(p) }
            throw APIError.httpError(http.statusCode)
        }
    }

    private func parseProblem(_ data: Data) -> ProblemDetail? {
        try? decoder.decode(ProblemDetail.self, from: data)
    }
}
