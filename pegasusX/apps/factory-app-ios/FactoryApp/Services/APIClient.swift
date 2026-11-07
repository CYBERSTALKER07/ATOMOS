import Foundation

private actor FactoryHTTPCoalescer {
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

final class APIClient: Sendable {
    static let shared = APIClient()

    #if DEBUG
    // Simulator: localhost. Physical device: set PEGASUS_DEV_HOST
    // scheme env variable to the Mac's LAN IP (e.g. 192.168.1.42)
    // for backend reachability over Wi-Fi.
    private let bootstrapURL: URL = {
        let raw = (ProcessInfo.processInfo.environment["PEGASUS_DEV_HOST"] ?? "")
            .trimmingCharacters(in: .whitespaces)
        let s: String
        if raw.isEmpty { s = "http://localhost:8180/" }
        else if raw.hasPrefix("http://") || raw.hasPrefix("https://") {
            s = raw.hasSuffix("/") ? raw : raw + "/"
        } else if raw.contains(":") { s = "http://\(raw)/" }
        else { s = "http://\(raw):8180/" }
        return URL(string: s)!
    }()
    #else
    private let bootstrapURL: URL = {
        let raw = (ProcessInfo.processInfo.environment["PEGASUSX_API_BASE_URL"] ?? "")
            .trimmingCharacters(in: .whitespaces)
        let s: String
        if raw.isEmpty { s = "https://api.pegasusx.app/" }
        else if raw.hasSuffix("/") { s = raw }
        else { s = raw + "/" }
        return URL(string: s)!
    }()
    #endif

    private var baseURL: URL { pinnedAPIBaseURL(bootstrap: bootstrapURL) }

    private let session: URLSession
    private let decoder: JSONDecoder
    private let encoder: JSONEncoder
    private let coalescer = FactoryHTTPCoalescer()

    private init() {
        let config = URLSessionConfiguration.default
        config.timeoutIntervalForRequest = 30
        session = URLSession(configuration: config)
        decoder = JSONDecoder()
        encoder = JSONEncoder()
    }

    func registerDeviceToken(token: String, platform: String = "ios") async throws {
        let _: [String: String] = try await post(
            "v1/user/device-token",
            body: DeviceTokenRequest(token: token, platform: platform)
        )
    }

    // MARK: - GET
    func get<T: Decodable>(_ path: String, query: [String: String] = [:]) async throws -> T {
        var components = URLComponents(url: baseURL.appendingPathComponent(path), resolvingAgainstBaseURL: false)!
        if !query.isEmpty {
            components.queryItems = query.map { URLQueryItem(name: $0.key, value: $0.value) }
        }
        var request = URLRequest(url: components.url!)
        request.httpMethod = "GET"
        await attachToken(&request)
        return try await execute(request)
    }

    // MARK: - POST
    func post<B: Encodable, T: Decodable>(_ path: String, body: B, idempotencyKey: String? = nil) async throws -> T {
        await coalescer.invalidateCache()
        var request = URLRequest(url: baseURL.appendingPathComponent(path))
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        if let idempotencyKey {
            request.setValue(idempotencyKey, forHTTPHeaderField: "Idempotency-Key")
        }
        request.httpBody = try encoder.encode(body)
        await attachToken(&request)
        return try await execute(request)
    }

    // MARK: - PATCH
    func patch<B: Encodable, T: Decodable>(_ path: String, body: B, idempotencyKey: String? = nil) async throws -> T {
        await coalescer.invalidateCache()
        var request = URLRequest(url: baseURL.appendingPathComponent(path))
        request.httpMethod = "PATCH"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        if let idempotencyKey {
            request.setValue(idempotencyKey, forHTTPHeaderField: "Idempotency-Key")
        }
        request.httpBody = try encoder.encode(body)
        await attachToken(&request)
        return try await execute(request)
    }

    // MARK: - POST (no response body)
    func postVoid<B: Encodable>(_ path: String, body: B, idempotencyKey: String? = nil) async throws {
        await coalescer.invalidateCache()
        var request = URLRequest(url: baseURL.appendingPathComponent(path))
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        if let idempotencyKey {
            request.setValue(idempotencyKey, forHTTPHeaderField: "Idempotency-Key")
        }
        request.httpBody = try encoder.encode(body)
        await attachToken(&request)
        let (_, response) = try await dataForRequestWithFallback(request)
        guard let http = response as? HTTPURLResponse else {
            throw APIError.invalidResponse
        }
        if http.statusCode == 401 {
            let refreshed = try await attemptRefresh()
            if refreshed {
                var retry = request
                await attachToken(&retry)
                let (_, retryResp) = try await dataForRequestWithFallback(retry)
                guard let retryHttp = retryResp as? HTTPURLResponse, (200...299).contains(retryHttp.statusCode) else {
                    throw APIError.httpError((retryResp as? HTTPURLResponse)?.statusCode ?? 0)
                }
                return
            }
            throw APIError.unauthorized
        }
        guard (200...299).contains(http.statusCode) else {
            throw APIError.httpError(http.statusCode)
        }
    }

    // MARK: - Execute with 401 retry
    private func execute<T: Decodable>(_ request: URLRequest) async throws -> T {
        let method = (request.httpMethod ?? "GET").uppercased()
        let (data, response): (Data, URLResponse)
        if method == "GET", let key = request.url?.absoluteString {
            (data, response) = try await coalescer.fetchGet(key: key) { [self] in
                try await self.dataForRequestWithFallback(request)
            }
        } else {
            (data, response) = try await dataForRequestWithFallback(request)
        }
        guard let http = response as? HTTPURLResponse else {
            throw APIError.invalidResponse
        }
        if http.statusCode == 401 {
            let refreshed = try await attemptRefresh()
            if refreshed {
                var retry = request
                await attachToken(&retry)
                let (retryData, retryResp) = try await dataForRequestWithFallback(retry)
                guard let retryHttp = retryResp as? HTTPURLResponse, (200...299).contains(retryHttp.statusCode) else {
                    throw APIError.httpError((retryResp as? HTTPURLResponse)?.statusCode ?? 0)
                }
                return try decoder.decode(T.self, from: retryData)
            }
            throw APIError.unauthorized
        }
        guard (200...299).contains(http.statusCode) else {
            throw APIError.httpError(http.statusCode)
        }
        return try decoder.decode(T.self, from: data)
    }

    // MARK: - Token
    private func attachToken(_ request: inout URLRequest) async {
        request.setValue(UUID().uuidString, forHTTPHeaderField: "X-Trace-Id")
        let token = await MainActor.run { TokenStore.shared.token }
        if let token {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }
    }

    private func attemptRefresh() async throws -> Bool {
        try await coalescer.refreshSingleFlight { [self] in
            guard let refresh = await MainActor.run(body: { TokenStore.shared.refreshToken }) else { return false }
            var request = URLRequest(url: self.baseURL.appendingPathComponent("v1/auth/factory/refresh"))
            request.httpMethod = "POST"
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
            request.httpBody = try self.encoder.encode(["refresh_token": refresh])
            let (data, response) = try await self.session.data(for: request)
            guard let http = response as? HTTPURLResponse, (200...299).contains(http.statusCode) else {
                await MainActor.run { TokenStore.shared.clear() }
                return false
            }
            let auth = try self.decoder.decode(AuthResponse.self, from: data)
            await MainActor.run { TokenStore.shared.updateTokens(token: auth.token, refresh: auth.refreshToken) }
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
        throw lastError ?? APIError.invalidResponse
    }

    func factoryWebSocketURL(token: String, lastEventId: String? = nil) -> URL {
        var components = URLComponents(url: baseURL, resolvingAgainstBaseURL: false)!
        components.scheme = components.scheme == "https" ? "wss" : "ws"
        components.path = "/v1/ws"
        var items = [URLQueryItem(name: "token", value: token)]
        if let lastEventId, !lastEventId.isEmpty {
            items.append(URLQueryItem(name: "last_event_id", value: lastEventId))
        }
        components.queryItems = items
        return components.url!
    }
}

// MARK: - Errors
enum APIError: LocalizedError {
    case invalidResponse
    case httpError(Int)
    case unauthorized

    var errorDescription: String? {
        switch self {
        case .invalidResponse: "Invalid response"
        case .httpError(let code): "HTTP \(code)"
        case .unauthorized: "Session expired"
        }
    }
}
