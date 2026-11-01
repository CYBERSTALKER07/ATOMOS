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
    case explainError(message: String, explain: StatusExplain?)
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

private actor PayloadHTTPCoalescer {
    private struct CacheEntry {
        let data: Data
        let response: URLResponse
        let timestamp: Date
    }

    private var inFlightGets: [String: Task<(Data, URLResponse), Error>] = [:]
    private var microCache: [String: CacheEntry] = [:]
    private let microCacheTTL: TimeInterval = 2.0
    private var refreshTask: Task<Bool, Error>?

    func fetchGet(
        key: String,
        operation: @Sendable @escaping () async throws -> (Data, URLResponse)
    ) async throws -> (Data, URLResponse) {
        let now = Date()
        if let cached = microCache[key], now.timeIntervalSince(cached.timestamp) < microCacheTTL {
            return (cached.data, cached.response)
        }
        if let existing = inFlightGets[key] {
            return try await existing.value
        }
        let task = Task<(Data, URLResponse), Error> {
            try await operation()
        }
        inFlightGets[key] = task
        do {
            let result = try await task.value
            inFlightGets.removeValue(forKey: key)
            if let http = result.1 as? HTTPURLResponse, (200...299).contains(http.statusCode) {
                microCache[key] = CacheEntry(data: result.0, response: result.1, timestamp: Date())
            }
            return result
        } catch {
            inFlightGets.removeValue(forKey: key)
            throw error
        }
    }

    func invalidateCache() {
        microCache.removeAll()
    }

    func refreshSingleFlight(
        operation: @Sendable @escaping () async throws -> Bool
    ) async throws -> Bool {
        if let existing = refreshTask {
            return try await existing.value
        }
        let task = Task<Bool, Error> {
            try await operation()
        }
        refreshTask = task
        do {
            let ok = try await task.value
            refreshTask = nil
            if ok {
                microCache.removeAll()
            }
            return ok
        } catch {
            refreshTask = nil
            throw error
        }
    }
}

final class APIClient: @unchecked Sendable {
    static let shared = APIClient()

    #if DEBUG
    /// Simulator: localhost. Physical iPad: set `PEGASUS_DEV_HOST`
    /// env var (Edit Scheme → Run → Environment Variables)
    /// to your Mac's LAN IP.
    let bootstrapURL: String = {
        let raw = (ProcessInfo.processInfo.environment["PEGASUS_DEV_HOST"] ?? "")
            .trimmingCharacters(in: .whitespaces)
        if raw.isEmpty { return "http://localhost:8180" }
        if raw.hasPrefix("http://") || raw.hasPrefix("https://") { return raw }
        return raw.contains(":") ? "http://\(raw)" : "http://\(raw):8180"
    }()
    #else
    let bootstrapURL: String = {
        let raw = (ProcessInfo.processInfo.environment["PEGASUSX_API_BASE_URL"] ?? "")
            .trimmingCharacters(in: .whitespaces)
        if raw.isEmpty { return "https://api.pegasusx.app" }
        return raw.hasSuffix("/") ? String(raw.dropLast()) : raw
    }()
    #endif

    var baseURL: String {
        CellApi.pinApiBaseUrl(
            bootstrap: bootstrapURL,
            homeCell: CellApi.homeCellFromJwt(CellTokenCache.token),
            sessionApiUrl: MarketPackStore.sessionApiUrl
        )
    }

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
    private let coalescer = PayloadHTTPCoalescer()

    private init() {
        let cfg = URLSessionConfiguration.default
        cfg.timeoutIntervalForRequest = 15
        cfg.timeoutIntervalForResource = 30
        session = URLSession(configuration: cfg)
        decoder = JSONDecoder()
        // Explicit CodingKeys use snake_case wire names; do not convertFromSnakeCase.
        encoder = JSONEncoder()
    }

    // MARK: - Auth
    func login(phone: String, pin: String) async throws -> LoginResponse {
        try await post("v1/auth/payloader/login", body: LoginRequest(phone: phone, pin: pin), authenticated: false)
    }

    func login(idToken: String) async throws -> LoginResponse {
        var body = LoginRequest()
        body.idToken = idToken
        return try await post("v1/auth/payloader/login", body: body, authenticated: false)
    }

    func lookupBarcode(ean: String) async throws -> CatalogBarcodeLookup {
        let encoded = ean.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? ean
        return try await get("v1/catalog/barcode/\(encoded)")
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
        let payload = ["order_id": orderId]
        return try await post(
            "v1/payloader/recommend-reassign",
            body: payload,
            headers: ["Idempotency-Key": PayloadIdempotency.recommendReassign(orderId: orderId)]
        )
    }

    func reassignOrder(orderId: String, toDriverId: String, isPartial: Bool = false, reason: String = "payload-redispatch") async throws -> StatusResponse {
        let payload: [String: Any] = ["order_id": orderId, "to_driver_id": toDriverId, "reason": reason, "is_partial": isPartial]
        // Convert to data
        let bodyData = try JSONSerialization.data(withJSONObject: payload)
        
        var req = try buildRequest(path: "v1/payloader/reassign-order", method: "POST", authenticated: true)
        req.setValue(PayloadIdempotency.applyReassign(orderId: orderId, toDriverId: toDriverId), forHTTPHeaderField: "Idempotency-Key")
        req.httpBody = bodyData
        
        return try await execute(req)
    }

    // MARK: - Manifest lifecycle
    func draftManifests(truckId: String?) async throws -> ManifestsResponse {
        try await manifests(state: "DRAFT", truckId: truckId)
    }
    func manifests(state: String, truckId: String?) async throws -> ManifestsResponse {
        var q = "?state=\(state)"
        if let t = truckId { q += "&truck_id=\(t)" }
        return try await get("v1/payloader/manifests\(q)")
    }
    func manifestDetail(_ manifestId: String) async throws -> Manifest {
        try await get("v1/payloader/manifests/\(manifestId)")
    }
    func startLoading(manifestId: String) async throws -> StatusResponse {
        try await post(
            "v1/payloader/manifests/\(manifestId)/start-loading",
            body: EmptyBody(),
            headers: ["Idempotency-Key": deterministicIdempotencyKey(action: "start-loading", entityId: manifestId)]
        )
    }
    func sealManifest(manifestId: String) async throws -> SealManifestResponse {
        try await post(
            "v1/payloader/manifests/\(manifestId)/seal",
            body: EmptyBody(),
            headers: ["Idempotency-Key": deterministicIdempotencyKey(action: "seal-manifest", entityId: manifestId)]
        )
    }
    func injectOrder(manifestId: String, orderId: String) async throws -> StatusResponse {
        let payload = ["order_id": orderId]
        let key = deterministicIdempotencyKey(action: "payloader_inject", entityId: "\(manifestId)_\(orderId)")
        return try await post("v1/payloader/manifests/\(manifestId)/inject-order", body: payload, idempotencyKey: key)
    }

    // MARK: - Supplier Manifests
    func supplierManifests(state: String = "DRAFT") async throws -> ManifestsResponse {
        return try await get("v1/payloader/manifests?state=\(state)")
    }

    func supplierManifestDetail(_ manifestId: String) async throws -> Manifest {
        return try await get("v1/payloader/manifests/\(manifestId)")
    }

    func supplierStartLoading(manifestId: String) async throws -> StatusResponse {
        try await post(
            "v1/payloader/manifests/\(manifestId)/start-loading",
            body: [String: String](),
            idempotencyKey: PayloadIdempotency.supplierStartLoading(manifestId: manifestId)
        )
    }

    func supplierSealManifest(manifestId: String) async throws -> SealManifestResponse {
        let key = PayloadIdempotency.key(action: "supplier-seal-manifest", entityId: manifestId)
        return try await post("v1/payloader/manifests/\(manifestId)/seal", body: [String: String](), idempotencyKey: key)
    }

    // MARK: - Factory loading-bay (P1-18 / P2-25 parity with Expo terminal)
    func factoryManifests(state: String = "DRAFT") async throws -> ManifestsResponse {
        try await get("v1/factory/manifests?state=\(state)")
    }

    func factoryManifestDetail(_ manifestId: String) async throws -> Manifest {
        try await get("v1/factory/manifests/\(manifestId)")
    }

    func factoryStartLoading(manifestId: String) async throws -> StatusResponse {
        try await post(
            "v1/factory/manifests/\(manifestId)/start-loading",
            body: [String: String](),
            idempotencyKey: PayloadIdempotency.supplierStartLoading(manifestId: manifestId)
        )
    }

    func factorySealManifest(manifestId: String) async throws -> SealManifestResponse {
        try await post(
            "v1/factory/manifests/\(manifestId)/seal",
            body: [String: String](),
            idempotencyKey: PayloadIdempotency.sealCompleted(manifestIds: [manifestId])
        )
    }

    /// Payloader + factory manifests; payloader wins on id collision.
    func listLoadingBayManifests(state: String = "DRAFT") async throws -> [Manifest] {
        async let primaryTask: ManifestsResponse? = try? get("v1/payloader/manifests?state=\(state)")
        async let factoryTask: ManifestsResponse? = try? factoryManifests(state: state)
        let (primaryOpt, factoryOpt) = await (primaryTask, factoryTask)

        var out: [String: Manifest] = [:]
        let payloaderManifests: [Manifest]
        if let primary = primaryOpt {
            payloaderManifests = primary.manifests
        } else {
            payloaderManifests = (try? await supplierManifests(state: state))?.manifests ?? []
        }
        for m in payloaderManifests {
            out[m.manifestId] = m.withSource("payloader")
        }
        if let factory = factoryOpt {
            for m in factory.manifests where out[m.manifestId] == nil {
                out[m.manifestId] = m.withSource("factory")
            }
        }
        return Array(out.values)
    }

    func sealCompletedManifests(manifestIds: [String]) async throws -> SealCompletedManifestsResponse {
        let ids = Array(Set(manifestIds.map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }.filter { !$0.isEmpty }))
        guard !ids.isEmpty else { throw APIError.httpError(400) }
        return try await post(
            "v1/payloader/manifests/seal-completed",
            body: SealCompletedManifestsRequest(manifestIds: ids),
            idempotencyKey: PayloadIdempotency.sealCompleted(manifestIds: ids)
        )
    }

    func sealAllManifests() async throws -> SealCompletedManifestsResponse {
        try await post(
            "v1/payloader/manifests/seal-all",
            body: EmptyBody(),
            idempotencyKey: PayloadIdempotency.key(action: "seal-all", entityId: "payloader")
        )
    }

    func loadingManifests() async throws -> ManifestsResponse {
        let manifests = try await listLoadingBayManifests(state: "LOADING")
        return ManifestsResponse(manifests: manifests)
    }

    func supplierInjectOrder(manifestId: String, orderId: String) async throws -> StatusResponse {
        let payload = ["order_id": orderId]
        return try await post(
            "v1/payloader/manifests/\(manifestId)/inject-order",
            body: payload,
            idempotencyKey: PayloadIdempotency.supplierInjectOrder(manifestId: manifestId, orderId: orderId)
        )
    }

    // MARK: - Per-order seal / exception
    /// Backend wants {order_id, terminal_id, manifest_cleared}. Per Expo,
    /// terminal_id is the active vehicle/truck id.
    func sealOrder(manifestId: String, orderId: String, terminalId: String) async throws -> SealOrderResponse {
        try await post(
            "v1/payload/seal",
            body: SealOrderRequest(manifestId: manifestId, orderId: orderId, terminalId: terminalId, manifestCleared: true),
            headers: ["Idempotency-Key": PayloadIdempotency.orderSeal(orderId: orderId)]
        )
    }
    func manifestException(manifestId: String, orderId: String, reason: String, metadata: String = "") async throws -> ManifestExceptionResponse {
        let payload = ["manifest_id": manifestId, "order_id": orderId, "reason": reason, "metadata": metadata]
        return try await post(
            "v1/payload/manifest-exception",
            body: payload,
            idempotencyKey: PayloadIdempotency.manifestException(manifestId: manifestId, orderId: orderId)
        )
    }

    func manifestExceptionsList(limit: Int = 50, offset: Int = 0) async throws -> ManifestExceptionsResponse {
        try await get("v1/payloader/manifest-exceptions?limit=\(limit)&offset=\(offset)")
    }

    func reportMissingItems(orderId: String, items: [MissingItemEntry]) async throws -> StatusResponse {
        try await post(
            "v1/delivery/missing-items",
            body: MissingItemsRequest(orderId: orderId, missingItems: items),
            headers: ["Idempotency-Key": PayloadIdempotency.missingItems(orderId: orderId)]
        )
    }

    /// Best-effort GET drain after WS reconnect — matches Android/Expo session reconcile.
    func reconcileSession() async {
        async let t = try? trucks()
        async let m: ManifestsResponse? = try? get("v1/payloader/manifests")
        _ = await (t, m)
    }

    // MARK: - Fleet reassign
    func fleetReassign(orderIds: [String], newRouteId: String) async throws -> FleetReassignResponse {
        try await post(
            "v1/fleet/reassign",
            body: FleetReassignRequest(orderIds: orderIds, newRouteId: newRouteId),
            headers: ["Idempotency-Key": PayloadIdempotency.fleetReassign(orderIds: orderIds)]
        )
    }

    // MARK: - Notifications
    func notifications(limit: Int = 100, offset: Int = 0) async throws -> NotificationsResponse {
        try await get("v1/user/notifications?limit=\(limit)&offset=\(offset)")
    }

    func pulse() async throws -> PulseResponse {
        try await get("v1/payloader/pulse")
    }
    func markRead(ids: [String]?, all: Bool? = nil) async throws -> StatusResponse {
        try await post("v1/user/notifications/read", body: MarkReadRequest(notificationIds: ids, markAll: all))
    }

    func clientPolicy(platform: String, version: String) async throws -> ClientPolicyResponse {
        let role = EnterpriseUpdateConfig.policyRole
        let channel = EnterpriseUpdateConfig.channel
        return try await get(
            "v1/platform/client-policy?role=\(role)&platform=\(platform)&version=\(version)&channel=\(channel)"
        )
    }

    // MARK: - Inbound returns gate
    func inboundReturns(physicalStatus: String = "ARRIVED", limit: Int = 100) async throws -> InboundReturnListResponse {
        try await get("v1/returns/inbound?physical_status=\(physicalStatus)&limit=\(limit)")
    }

    func createInboundSession() async throws -> InboundSessionResponse {
        try await post("v1/returns/inbound/sessions", body: EmptyBody())
    }

    func scanInboundBarcode(barcode: String, qty: Int = 1, sessionId: String) async throws -> InboundScanResponse {
        struct ScanBody: Encodable {
            let barcode: String
            let qty: Int
            let sessionId: String
            enum CodingKeys: String, CodingKey {
                case barcode, qty
                case sessionId = "session_id"
            }
        }
        let key = PayloadIdempotency.inboundScan(barcode: barcode, sessionId: sessionId)
        return try await post(
            "v1/returns/inbound/scan",
            body: ScanBody(barcode: barcode, qty: qty, sessionId: sessionId),
            headers: ["Idempotency-Key": key]
        )
    }

    func confirmInboundReturns(
        returnIds: [String],
        disposition: String,
        sessionId: String,
        quantities: [String: Int] = [:]
    ) async throws -> StatusResponse {
        struct Line: Encodable {
            let returnId: String
            let disposition: String
            let qty: Int?
            enum CodingKeys: String, CodingKey {
                case returnId = "return_id"
                case disposition, qty
            }
        }
        struct ConfirmBody: Encodable {
            let lines: [Line]
            let sessionId: String
            enum CodingKeys: String, CodingKey {
                case lines
                case sessionId = "session_id"
            }
        }
        let key = PayloadIdempotency.inboundConfirm(returnIds: returnIds, disposition: disposition)
        return try await post(
            "v1/returns/inbound/confirm",
            body: ConfirmBody(
                lines: returnIds.map { rid in
                    Line(returnId: rid, disposition: disposition, qty: quantities[rid])
                },
                sessionId: sessionId
            ),
            headers: ["Idempotency-Key": key]
        )
    }

    func returnsHistory(limit: Int = 50) async throws -> InboundHistoryResponse {
        try await get("v1/returns/history?limit=\(limit)")
    }

    // MARK: - FCM
    func registerDeviceToken(_ token: String) async throws -> StatusResponse {
        try await post("v1/user/device-token", body: DeviceTokenRequest(token: token, platform: "ios"))
    }
    func unregisterDeviceToken(_ token: String) async throws -> StatusResponse {
        var req = try buildRequest(path: "v1/user/device-token", method: "DELETE")
        req.httpBody = try encoder.encode(DeviceTokenRequest(token: token, platform: "ios"))
        return try await execute(req)
    }

    // MARK: - Raw replay (offline queue)
    /// Replay a queued action with arbitrary endpoint/method/body. Returns
    /// (statusCode, raw bytes) so the caller can decide retention vs drop.
    func rawRequest(endpoint: String, method: String, body: String, idempotencyKey: String? = nil) async throws -> (Int, Data) {
        let path = endpoint.hasPrefix("/") ? String(endpoint.dropFirst()) : endpoint
        var req = try buildRequest(path: path, method: method)
        if let idempotencyKey, !idempotencyKey.isEmpty {
            req.setValue(idempotencyKey, forHTTPHeaderField: "Idempotency-Key")
        }
        if !body.isEmpty { req.httpBody = body.data(using: .utf8) }
        let (data, response) = try await dataForRequestWithFallback(req)
        guard let http = response as? HTTPURLResponse else { throw APIError.networkError }
        return (http.statusCode, data)
    }

    // MARK: - Generic plumbing

    private struct EmptyBody: Encodable {}

    private func get<T: Decodable>(_ path: String) async throws -> T {
        let req = try buildRequest(path: path, method: "GET")
        return try await execute(req)
    }

    private func post<B: Encodable, T: Decodable>(
        _ path: String,
        body: B,
        authenticated: Bool = true,
        headers: [String: String] = [:],
        idempotencyKey: String? = nil
    ) async throws -> T {
        var req = try buildRequest(path: path, method: "POST", authenticated: authenticated)
        for (name, value) in headers {
            req.setValue(value, forHTTPHeaderField: name)
        }
        if let idempotencyKey, !idempotencyKey.isEmpty {
            req.setValue(idempotencyKey, forHTTPHeaderField: "Idempotency-Key")
        }
        req.httpBody = try encoder.encode(body)
        return try await execute(req)
    }

    private func deterministicIdempotencyKey(action: String, entityId: String) -> String {
        "payload-\(action)-\(entityId)"
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
        let method = (request.httpMethod ?? "GET").uppercased()
        if method != "GET" {
            await coalescer.invalidateCache()
        }
        let data: Data
        let response: URLResponse
        do {
            if method == "GET", let key = request.url?.absoluteString {
                (data, response) = try await coalescer.fetchGet(key: key) { [self] in
                    try await self.dataForRequestWithFallback(request)
                }
            } else {
                (data, response) = try await dataForRequestWithFallback(request)
            }
        } catch {
            throw APIError.networkError
        }

        guard let http = response as? HTTPURLResponse else { throw APIError.networkError }
        switch http.statusCode {
        case 200...299:
            if T.self == StatusResponse.self, data.isEmpty {
                // Backend may return empty body on no-content success.
                if let status = StatusResponse(status: "ok") as? T {
                    return status
                }
            }
            do { return try decoder.decode(T.self, from: data) }
            catch { throw APIError.decodingError }
        case 401:
            let refreshed = try await attemptRefresh()
            if refreshed {
                var retry = request
                if let token = await MainActor.run(body: { TokenStore.shared.token }) {
                    retry.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
                }
                let (retryData, retryResponse) = try await dataForRequestWithFallback(retry)
                guard let retryHttp = retryResponse as? HTTPURLResponse else { throw APIError.networkError }
                if (200...299).contains(retryHttp.statusCode) {
                    if T.self == StatusResponse.self, retryData.isEmpty {
                        if let status = StatusResponse(status: "ok") as? T {
                            return status
                        }
                    }
                    do { return try decoder.decode(T.self, from: retryData) }
                    catch { throw APIError.decodingError }
                }
            }
            await MainActor.run { TokenStore.shared.logout() }
            throw APIError.unauthorized
        case 403:
            if let p = parseProblem(data) { throw APIError.problemDetail(p) }
            throw APIError.forbidden
        default:
            if let parsed = ApiExplainParser.parse(from: data) {
                throw APIError.explainError(message: parsed.message, explain: parsed.explain)
            }
            if let p = parseProblem(data) { throw APIError.problemDetail(p) }
            throw APIError.httpError(http.statusCode)
        }
    }

    private func parseProblem(_ data: Data) -> ProblemDetail? {
        try? decoder.decode(ProblemDetail.self, from: data)
    }

    private func attemptRefresh() async throws -> Bool {
        try await coalescer.refreshSingleFlight { [self] in
            let refresh = await MainActor.run { TokenStore.shared.refreshToken }
            guard let refresh, !refresh.isEmpty else { return false }
            guard let url = URL(string: "\(self.baseURL)/v1/auth/payloader/refresh") else { return false }
            var request = URLRequest(url: url)
            request.httpMethod = "POST"
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
            request.httpBody = try self.encoder.encode(RefreshTokenRequest(refreshToken: refresh))
            let (data, response) = try await self.session.data(for: request)
            guard let http = response as? HTTPURLResponse, (200...299).contains(http.statusCode) else {
                await MainActor.run { TokenStore.shared.logout() }
                return false
            }
            let auth = try self.decoder.decode(RefreshTokenResponse.self, from: data)
            let nextRefresh = auth.refreshToken ?? refresh
            await MainActor.run { TokenStore.shared.updateTokens(token: auth.token, refresh: nextRefresh) }
            return true
        }
    }

    private func dataForRequestWithFallback(_ request: URLRequest) async throws -> (Data, URLResponse) {
        let method = (request.httpMethod ?? "GET").uppercased()
        let isIdempotent = method == "GET"
            || request.value(forHTTPHeaderField: "Idempotency-Key") != nil
            || request.value(forHTTPHeaderField: "X-Idempotency-Key") != nil
        let maxAttempts = isIdempotent ? 3 : 1
        var lastError: Error?

        for attempt in 0..<maxAttempts {
            try Task.checkCancellation()
            do {
                let (data, response) = try await session.data(for: request)
                if let http = response as? HTTPURLResponse,
                   (http.statusCode == 502 || http.statusCode == 503 || http.statusCode == 504 || http.statusCode == 429),
                   attempt + 1 < maxAttempts {
                    let delayMs = UInt64(300 * (1 << attempt) + Int.random(in: 0...120))
                    try await Task.sleep(nanoseconds: delayMs * 1_000_000)
                    continue
                }
                return (data, response)
            } catch is CancellationError {
                throw CancellationError()
            } catch let urlError as URLError where urlError.code == .cancelled {
                throw CancellationError()
            } catch {
                lastError = error
                if attempt + 1 < maxAttempts {
                    let delayMs = UInt64(300 * (1 << attempt) + Int.random(in: 0...120))
                    try? await Task.sleep(nanoseconds: delayMs * 1_000_000)
                    continue
                }
            }
        }
        throw lastError ?? APIError.networkError
    }
}
