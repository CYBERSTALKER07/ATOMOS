import Foundation

final class APIClient: Sendable {
    static let shared = APIClient()

    private struct TokenRefreshResponse: Decodable {
        let token: String
        let refreshToken: String?

        enum CodingKeys: String, CodingKey {
            case token
            case refreshToken = "refresh_token"
        }
    }

    private struct EmptyBody: Encodable {}

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

    private init() {
        let config = URLSessionConfiguration.default
        config.timeoutIntervalForRequest = 30
        session = URLSession(configuration: config)
        decoder = JSONDecoder()
        encoder = JSONEncoder()
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

    func getJSONString(_ path: String, query: [String: String] = [:]) async throws -> String {
        var components = URLComponents(url: baseURL.appendingPathComponent(path), resolvingAgainstBaseURL: false)!
        if !query.isEmpty {
            components.queryItems = query.map { URLQueryItem(name: $0.key, value: $0.value) }
        }
        var request = URLRequest(url: components.url!)
        request.httpMethod = "GET"
        await attachToken(&request)
        let (data, response) = try await dataForRequestWithFallback(request)
        guard let http = response as? HTTPURLResponse else {
            throw APIError.invalidResponse
        }
        if http.statusCode == 401 {
            throw APIError.unauthorized
        }
        guard (200...299).contains(http.statusCode) else {
            throw APIError.httpError(http.statusCode)
        }
        return String(data: data, encoding: .utf8) ?? "{}"
    }

    // MARK: - POST
    func post<B: Encodable, T: Decodable>(_ path: String, body: B, idempotencyKey: String? = nil) async throws -> T {
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

    func postEmpty<T: Decodable>(_ path: String, idempotencyKey: String? = nil) async throws -> T {
        try await post(path, body: EmptyBody(), idempotencyKey: idempotencyKey)
    }

    // MARK: - POST (no response body)
    func postVoid<B: Encodable>(_ path: String, body: B) async throws {
        var request = URLRequest(url: baseURL.appendingPathComponent(path))
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
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

    // MARK: - PATCH
    func patch<B: Encodable, T: Decodable>(_ path: String, body: B, idempotencyKey: String? = nil) async throws -> T {
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

    // MARK: - PUT
    func put<B: Encodable, T: Decodable>(
        _ path: String,
        body: B,
        query: [String: String] = [:],
        idempotencyKey: String? = nil
    ) async throws -> T {
        var components = URLComponents(url: baseURL.appendingPathComponent(path), resolvingAgainstBaseURL: false)!
        if !query.isEmpty {
            components.queryItems = query.map { URLQueryItem(name: $0.key, value: $0.value) }
        }
        var request = URLRequest(url: components.url!)
        request.httpMethod = "PUT"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        if let idempotencyKey {
            request.setValue(idempotencyKey, forHTTPHeaderField: "Idempotency-Key")
        }
        request.httpBody = try encoder.encode(body)
        await attachToken(&request)
        return try await execute(request)
    }

    func patchVoid<B: Encodable>(_ path: String, body: B, idempotencyKey: String? = nil) async throws {
        var request = URLRequest(url: baseURL.appendingPathComponent(path))
        request.httpMethod = "PATCH"
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

    // MARK: - DELETE
    func delete<T: Decodable>(_ path: String, query: [String: String] = [:], idempotencyKey: String? = nil) async throws -> T {
        var components = URLComponents(url: baseURL.appendingPathComponent(path), resolvingAgainstBaseURL: false)!
        if !query.isEmpty {
            components.queryItems = query.map { URLQueryItem(name: $0.key, value: $0.value) }
        }
        var request = URLRequest(url: components.url!)
        request.httpMethod = "DELETE"
        if let idempotencyKey {
            request.setValue(idempotencyKey, forHTTPHeaderField: "Idempotency-Key")
        }
        await attachToken(&request)
        return try await execute(request)
    }

    func registerDeviceToken(token: String, platform: String = "ios") async throws {
        let body = ["token": token, "platform": platform]
        let _: [String: String] = try await post("v1/user/device-token", body: body)
    }

    // MARK: - Execute with 401 retry
    private func execute<T: Decodable>(_ request: URLRequest) async throws -> T {
        let (data, response) = try await dataForRequestWithFallback(request)
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
        let refresh = await MainActor.run { TokenStore.shared.refreshToken }
        guard let refresh, !refresh.isEmpty else { return false }
        var request = URLRequest(url: baseURL.appendingPathComponent("v1/auth/warehouse/refresh"))
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try encoder.encode(RefreshTokenRequest(refreshToken: refresh))
        let (data, response) = try await dataForRequestWithFallback(request)
        guard let http = response as? HTTPURLResponse, (200...299).contains(http.statusCode) else {
            await MainActor.run { TokenStore.shared.clear() }
            return false
        }
        let auth = try decoder.decode(TokenRefreshResponse.self, from: data)
        let nextRefresh = auth.refreshToken ?? refresh
        await MainActor.run { TokenStore.shared.updateTokens(token: auth.token, refresh: nextRefresh) }
        return true
    }

    private func dataForRequestWithFallback(_ request: URLRequest) async throws -> (Data, URLResponse) {
        try await session.data(for: request)
    }

    func warehouseWebSocketURL(token: String) -> URL {
        var components = URLComponents(url: baseURL, resolvingAgainstBaseURL: false)!
        components.scheme = components.scheme == "https" ? "wss" : "ws"
        components.path = "/v1/ws"
        components.queryItems = [URLQueryItem(name: "token", value: token)]
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
