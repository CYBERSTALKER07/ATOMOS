import Foundation

final class APIClient: Sendable {
    static let shared = APIClient()

    #if DEBUG
    // Simulator: localhost. Physical device: set LAB_DEV_HOST scheme env variable
    // to the Mac's LAN IP (e.g. 192.168.1.42) for backend reachability over Wi-Fi.
    private let baseURL: URL = {
        let raw = ProcessInfo.processInfo.environment["LAB_DEV_HOST"]?
            .trimmingCharacters(in: .whitespaces) ?? ""
        let s: String
        if raw.isEmpty { s = "http://localhost:8080/" }
        else if raw.hasPrefix("http://") || raw.hasPrefix("https://") {
            s = raw.hasSuffix("/") ? raw : raw + "/"
        } else if raw.contains(":") { s = "http://\(raw)/" }
        else { s = "http://\(raw):8080/" }
        return URL(string: s)!
    }()
    #else
    private let baseURL = URL(string: "https://api.thelab.uz/")!
    #endif

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

    // MARK: - POST
    func post<B: Encodable, T: Decodable>(_ path: String, body: B) async throws -> T {
        var request = URLRequest(url: baseURL.appendingPathComponent(path))
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try encoder.encode(body)
        await attachToken(&request)
        return try await execute(request)
    }

    // MARK: - POST (no response body)
    func postVoid<B: Encodable>(_ path: String, body: B) async throws {
        var request = URLRequest(url: baseURL.appendingPathComponent(path))
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try encoder.encode(body)
        await attachToken(&request)
        let (_, response) = try await session.data(for: request)
        guard let http = response as? HTTPURLResponse else {
            throw APIError.invalidResponse
        }
        if http.statusCode == 401 {
            let refreshed = try await attemptRefresh()
            if refreshed {
                var retry = request
                await attachToken(&retry)
                let (_, retryResp) = try await session.data(for: retry)
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
    func patch<B: Encodable, T: Decodable>(_ path: String, body: B) async throws -> T {
        var request = URLRequest(url: baseURL.appendingPathComponent(path))
        request.httpMethod = "PATCH"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try encoder.encode(body)
        await attachToken(&request)
        return try await execute(request)
    }

    func patchVoid<B: Encodable>(_ path: String, body: B) async throws {
        var request = URLRequest(url: baseURL.appendingPathComponent(path))
        request.httpMethod = "PATCH"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try encoder.encode(body)
        await attachToken(&request)
        let (_, response) = try await session.data(for: request)
        guard let http = response as? HTTPURLResponse else {
            throw APIError.invalidResponse
        }
        if http.statusCode == 401 {
            let refreshed = try await attemptRefresh()
            if refreshed {
                var retry = request
                await attachToken(&retry)
                let (_, retryResp) = try await session.data(for: retry)
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
        let (data, response) = try await session.data(for: request)
        guard let http = response as? HTTPURLResponse else {
            throw APIError.invalidResponse
        }
        if http.statusCode == 401 {
            let refreshed = try await attemptRefresh()
            if refreshed {
                var retry = request
                await attachToken(&retry)
                let (retryData, retryResp) = try await session.data(for: retry)
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
        guard let refresh = await MainActor.run(body: { TokenStore.shared.refreshToken }) else { return false }
        var request = URLRequest(url: baseURL.appendingPathComponent("v1/auth/warehouse/refresh"))
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try encoder.encode(["refresh_token": refresh])
        let (data, response) = try await session.data(for: request)
        guard let http = response as? HTTPURLResponse, (200...299).contains(http.statusCode) else {
            await MainActor.run { TokenStore.shared.clear() }
            return false
        }
        let auth = try decoder.decode(AuthResponse.self, from: data)
        await MainActor.run { TokenStore.shared.updateTokens(token: auth.token, refresh: auth.refreshToken) }
        return true
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
