import Foundation

final class APIClient: Sendable {
    static let shared = APIClient()

    #if DEBUG
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
    var cellBootstrap: String { bootstrapURL.absoluteString }

    private var session: URLSession
    private let decoder: JSONDecoder
    private let encoder: JSONEncoder
    private let coalescer = SupplierHTTPCoalescer()

    private init() {
        let config = URLSessionConfiguration.default
        config.timeoutIntervalForRequest = 30
        session = URLSession(configuration: config)
        decoder = JSONDecoder()
        encoder = JSONEncoder()
    }

#if DEBUG
    func setTestSession(_ session: URLSession) {
        self.session = session
        Task { await coalescer.invalidateCache() }
    }
#endif

    func registerDeviceToken(token: String, platform: String = "ios") async throws {
        let _: [String: String] = try await post(
            "v1/user/device-token",
            body: DeviceTokenRequest(token: token, platform: platform)
        )
    }

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

    func getJSONString(_ path: String, query: [String: String] = [:]) async throws -> String {
        var components = URLComponents(url: baseURL.appendingPathComponent(path), resolvingAgainstBaseURL: false)!
        if !query.isEmpty {
            components.queryItems = query.map { URLQueryItem(name: $0.key, value: $0.value) }
        }
        var request = URLRequest(url: components.url!)
        request.httpMethod = "GET"
        await attachToken(&request)
        let (data, response) = try await session.data(for: request)
        guard let http = response as? HTTPURLResponse else {
            throw APIError.invalidResponse
        }
        if http.statusCode == 401 {
            await MainActor.run { TokenStore.shared.clear() }
            throw APIError.unauthorized
        }
        guard (200...299).contains(http.statusCode) else {
            if let err = try? decoder.decode(APIErrorBody.self, from: data) {
                throw APIError.server(err.error)
            }
            throw APIError.httpError(http.statusCode)
        }
        return String(data: data, encoding: .utf8) ?? "{}"
    }

    func post<B: Encodable, T: Decodable>(
        _ path: String,
        body: B,
        idempotencyKey: String? = nil,
        authenticated: Bool = true
    ) async throws -> T {
        var request = URLRequest(url: baseURL.appendingPathComponent(path))
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        if let idempotencyKey {
            request.setValue(idempotencyKey, forHTTPHeaderField: "X-Idempotency-Key")
        }
        request.httpBody = try encoder.encode(body)
        if authenticated {
            await attachToken(&request)
        }
        return try await execute(request)
    }

    func postEmpty<T: Decodable>(_ path: String, authenticated: Bool = true) async throws -> T {
        var request = URLRequest(url: baseURL.appendingPathComponent(path))
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = Data("{}".utf8)
        if authenticated {
            await attachToken(&request)
        }
        return try await execute(request)
    }

    func postText(_ path: String, authenticated: Bool = true) async throws -> String {
        var request = URLRequest(url: baseURL.appendingPathComponent(path))
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = Data("{}".utf8)
        if authenticated {
            await attachToken(&request)
        }
        let (data, response) = try await session.data(for: request)
        guard let http = response as? HTTPURLResponse else {
            throw APIError.invalidResponse
        }
        if http.statusCode == 401 {
            await MainActor.run { TokenStore.shared.clear() }
            throw APIError.unauthorized
        }
        guard (200...299).contains(http.statusCode) else {
            if let err = try? decoder.decode(APIErrorBody.self, from: data) {
                throw APIError.server(err.error)
            }
            throw APIError.httpError(http.statusCode)
        }
        return String(data: data, encoding: .utf8) ?? ""
    }

    func put<B: Encodable, T: Decodable>(
        _ path: String,
        body: B,
        idempotencyKey: String? = nil,
        authenticated: Bool = true
    ) async throws -> T {
        var request = URLRequest(url: baseURL.appendingPathComponent(path))
        request.httpMethod = "PUT"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        if let idempotencyKey {
            request.setValue(idempotencyKey, forHTTPHeaderField: "X-Idempotency-Key")
        }
        request.httpBody = try encoder.encode(body)
        if authenticated {
            await attachToken(&request)
        }
        return try await execute(request)
    }

    func put<B: Encodable, T: Decodable>(_ path: String, body: B, authenticated: Bool = true) async throws -> T {
        try await put(path, body: body, idempotencyKey: nil, authenticated: authenticated)
    }

    func patch<B: Encodable, T: Decodable>(
        _ path: String,
        body: B,
        idempotencyKey: String? = nil,
        authenticated: Bool = true
    ) async throws -> T {
        var request = URLRequest(url: baseURL.appendingPathComponent(path))
        request.httpMethod = "PATCH"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        if let idempotencyKey {
            request.setValue(idempotencyKey, forHTTPHeaderField: "X-Idempotency-Key")
        }
        request.httpBody = try encoder.encode(body)
        if authenticated {
            await attachToken(&request)
        }
        return try await execute(request)
    }

    func patch<B: Encodable, T: Decodable>(_ path: String, body: B, authenticated: Bool = true) async throws -> T {
        try await patch(path, body: body, idempotencyKey: nil, authenticated: authenticated)
    }

    func postJSON<T: Decodable>(
        _ path: String,
        body: [String: Any],
        idempotencyKey: String? = nil,
        authenticated: Bool = true
    ) async throws -> T {
        var request = URLRequest(url: baseURL.appendingPathComponent(path))
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        if let idempotencyKey {
            request.setValue(idempotencyKey, forHTTPHeaderField: "X-Idempotency-Key")
        }
        request.httpBody = try JSONSerialization.data(withJSONObject: body)
        if authenticated {
            await attachToken(&request)
        }
        return try await execute(request)
    }

    func postRaw<T: Decodable>(
        _ path: String,
        body: Data,
        contentType: String,
        idempotencyKey: String? = nil,
        authenticated: Bool = true
    ) async throws -> T {
        var request = URLRequest(url: baseURL.appendingPathComponent(path))
        request.httpMethod = "POST"
        request.setValue(contentType, forHTTPHeaderField: "Content-Type")
        if let idempotencyKey {
            request.setValue(idempotencyKey, forHTTPHeaderField: "X-Idempotency-Key")
        }
        request.httpBody = body
        if authenticated {
            await attachToken(&request)
        }
        return try await execute(request)
    }

    func patchVoid<B: Encodable>(
        _ path: String,
        body: B,
        idempotencyKey: String? = nil,
        authenticated: Bool = true
    ) async throws {
        var request = URLRequest(url: baseURL.appendingPathComponent(path))
        request.httpMethod = "PATCH"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        if let idempotencyKey {
            request.setValue(idempotencyKey, forHTTPHeaderField: "X-Idempotency-Key")
        }
        request.httpBody = try encoder.encode(body)
        if authenticated {
            await attachToken(&request)
        }
        _ = try await executeVoid(request)
    }

    func postVoid<B: Encodable>(
        _ path: String,
        body: B,
        query: [String: String] = [:],
        idempotencyKey: String? = nil,
        authenticated: Bool = true
    ) async throws {
        var components = URLComponents(url: baseURL.appendingPathComponent(path), resolvingAgainstBaseURL: false)!
        if !query.isEmpty {
            components.queryItems = query.map { URLQueryItem(name: $0.key, value: $0.value) }
        }
        var request = URLRequest(url: components.url!)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        if let idempotencyKey {
            request.setValue(idempotencyKey, forHTTPHeaderField: "X-Idempotency-Key")
        }
        request.httpBody = try encoder.encode(body)
        if authenticated {
            await attachToken(&request)
        }
        _ = try await executeVoid(request)
    }

    func deleteVoid(
        _ path: String,
        idempotencyKey: String? = nil,
        authenticated: Bool = true
    ) async throws {
        var request = URLRequest(url: baseURL.appendingPathComponent(path))
        request.httpMethod = "DELETE"
        if let idempotencyKey {
            request.setValue(idempotencyKey, forHTTPHeaderField: "X-Idempotency-Key")
        }
        if authenticated {
            await attachToken(&request)
        }
        _ = try await executeVoid(request)
    }

    func delete<T: Decodable>(
        _ path: String,
        idempotencyKey: String? = nil,
        authenticated: Bool = true
    ) async throws -> T {
        var request = URLRequest(url: baseURL.appendingPathComponent(path))
        request.httpMethod = "DELETE"
        if let idempotencyKey {
            request.setValue(idempotencyKey, forHTTPHeaderField: "X-Idempotency-Key")
        }
        if authenticated {
            await attachToken(&request)
        }
        return try await execute(request)
    }

    private func execute<T: Decodable>(_ request: URLRequest) async throws -> T {
        let method = request.httpMethod ?? "GET"
        if method != "GET" {
            await coalescer.invalidateCache()
        }
        let (data, response): (Data, URLResponse)
        if method == "GET", let urlKey = request.url?.absoluteString {
            (data, response) = try await coalescer.fetchOrCoalesce(key: urlKey) { [self] in
                try await self.dataForRequestWithRetry(request)
            }
        } else {
            (data, response) = try await dataForRequestWithRetry(request)
        }
        guard let http = response as? HTTPURLResponse else {
            throw APIError.invalidResponse
        }
        if http.statusCode == 401 {
            let refreshed = try await coalescer.refreshToken { [self] in
                try await self.attemptRefresh()
            }
            if refreshed {
                var retry = request
                await attachToken(&retry)
                let (retryData, retryResp) = try await dataForRequestWithRetry(retry)
                guard let retryHttp = retryResp as? HTTPURLResponse, (200...299).contains(retryHttp.statusCode) else {
                    throw APIError.httpError((retryResp as? HTTPURLResponse)?.statusCode ?? 0)
                }
                return try decoder.decode(T.self, from: retryData)
            }
            await MainActor.run { TokenStore.shared.clear() }
            throw APIError.unauthorized
        }
        guard (200...299).contains(http.statusCode) else {
            if let err = try? decoder.decode(APIErrorBody.self, from: data) {
                throw APIError.server(err.error)
            }
            throw APIError.httpError(http.statusCode)
        }
        return try decoder.decode(T.self, from: data)
    }

    private func executeVoid(_ request: URLRequest) async throws {
        await coalescer.invalidateCache()
        let (data, response) = try await dataForRequestWithRetry(request)
        guard let http = response as? HTTPURLResponse else {
            throw APIError.invalidResponse
        }
        if http.statusCode == 401 {
            let refreshed = try await coalescer.refreshToken { [self] in
                try await self.attemptRefresh()
            }
            if refreshed {
                var retry = request
                await attachToken(&retry)
                let (retryData, retryResp) = try await dataForRequestWithRetry(retry)
                guard let retryHttp = retryResp as? HTTPURLResponse, (200...299).contains(retryHttp.statusCode) else {
                    if let err = try? decoder.decode(APIErrorBody.self, from: retryData) {
                        throw APIError.server(err.error)
                    }
                    throw APIError.httpError((retryResp as? HTTPURLResponse)?.statusCode ?? 0)
                }
                return
            }
            await MainActor.run { TokenStore.shared.clear() }
            throw APIError.unauthorized
        }
        guard (200...299).contains(http.statusCode) else {
            if let err = try? decoder.decode(APIErrorBody.self, from: data) {
                throw APIError.server(err.error)
            }
            throw APIError.httpError(http.statusCode)
        }
    }

    private func dataForRequestWithRetry(_ request: URLRequest) async throws -> (Data, URLResponse) {
        let isIdempotent = request.httpMethod == "GET" || request.value(forHTTPHeaderField: "X-Idempotency-Key") != nil
        let maxAttempts = isIdempotent ? 3 : 1
        var lastError: Error = URLError(.unknown)

        for attempt in 0..<maxAttempts {
            try Task.checkCancellation()
            do {
                let (data, response) = try await session.data(for: request)
                if let http = response as? HTTPURLResponse,
                   isIdempotent && (502...504).contains(http.statusCode) && attempt + 1 < maxAttempts {
                    let delayNs = UInt64(300_000_000 * (1 << attempt)) + UInt64.random(in: 0...150_000_000)
                    try await Task.sleep(nanoseconds: delayNs)
                    continue
                }
                return (data, response)
            } catch is CancellationError {
                throw CancellationError()
            } catch let urlErr as URLError where urlErr.code == .cancelled {
                throw CancellationError()
            } catch {
                lastError = error
                if isIdempotent && attempt + 1 < maxAttempts {
                    let delayNs = UInt64(300_000_000 * (1 << attempt)) + UInt64.random(in: 0...150_000_000)
                    try await Task.sleep(nanoseconds: delayNs)
                    continue
                }
                throw error
            }
        }
        throw lastError
    }

    private struct SupplierTokenRefreshResponse: Decodable {
        let token: String
        let refreshToken: String?
        enum CodingKeys: String, CodingKey {
            case token
            case refreshToken = "refresh_token"
        }
    }

    private func attemptRefresh() async throws -> Bool {
        let refresh = await MainActor.run { TokenStore.shared.refreshToken }
        guard let refresh, !refresh.isEmpty else { return false }
        var request = URLRequest(url: baseURL.appendingPathComponent("v1/auth/supplier/refresh"))
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try encoder.encode(["refresh_token": refresh])
        let (data, response) = try await session.data(for: request)
        guard let http = response as? HTTPURLResponse, (200...299).contains(http.statusCode),
              let auth = try? decoder.decode(SupplierTokenRefreshResponse.self, from: data) else {
            return false
        }
        await MainActor.run {
            TokenStore.shared.updateTokens(token: auth.token, refresh: auth.refreshToken ?? refresh)
        }
        return true
    }

    private func attachToken(_ request: inout URLRequest) async {
        request.setValue(UUID().uuidString, forHTTPHeaderField: "X-Trace-Id")
        let token = await MainActor.run { TokenStore.shared.token }
        if let token {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }
    }
}

private actor SupplierHTTPCoalescer {
    private var inFlightGETs: [String: Task<(Data, URLResponse), Error>] = [:]
    private var getCache: [String: (data: Data, response: URLResponse, expiresAt: Date)] = [:]
    private var refreshTask: Task<Bool, Error>?
    private let cacheTTL: TimeInterval = 2.0

    func fetchOrCoalesce(
        key: String,
        operation: @Sendable @escaping () async throws -> (Data, URLResponse)
    ) async throws -> (Data, URLResponse) {
        let now = Date()
        if let cached = getCache[key], cached.expiresAt > now {
            return (cached.data, cached.response)
        }
        if let existing = inFlightGETs[key] {
            return try await existing.value
        }
        let task = Task<(Data, URLResponse), Error> {
            try await operation()
        }
        inFlightGETs[key] = task
        do {
            let (data, response) = try await task.value
            inFlightGETs.removeValue(forKey: key)
            if let http = response as? HTTPURLResponse, (200...299).contains(http.statusCode) {
                getCache[key] = (data, response, Date().addingTimeInterval(cacheTTL))
            }
            return (data, response)
        } catch {
            inFlightGETs.removeValue(forKey: key)
            throw error
        }
    }

    func invalidateCache() {
        getCache.removeAll()
    }

    func refreshToken(using operation: @Sendable @escaping () async throws -> Bool) async throws -> Bool {
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
            return ok
        } catch {
            refreshTask = nil
            throw error
        }
    }
}

struct APIErrorBody: Decodable {
    let error: String
}

enum APIError: LocalizedError {
    case invalidResponse
    case httpError(Int)
    case unauthorized
    case server(String)

    var errorDescription: String? {
        switch self {
        case .invalidResponse: "Invalid response"
        case .httpError(let code): "HTTP \(code)"
        case .unauthorized: "Session expired — sign in again"
        case .server(let message): message.replacingOccurrences(of: "_", with: " ")
        }
    }
}
