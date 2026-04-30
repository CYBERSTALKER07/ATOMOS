import Foundation

// MARK: - API Client

@Observable
final class APIClient {
    static let shared = APIClient()

    private let session: URLSession
    private let decoder: JSONDecoder
    private let encoder: JSONEncoder

    #if DEBUG
    // Simulator: localhost. Physical device: set PEGASUS_DEV_HOST
    // scheme env variable to the Mac's LAN IP (e.g. 192.168.1.42)
    // for backend reachability over Wi-Fi.
    var baseURL: String = {
        let raw = (ProcessInfo.processInfo.environment["PEGASUS_DEV_HOST"] ?? "")
            .trimmingCharacters(in: .whitespaces)
        if raw.isEmpty { return "http://localhost:8080" }
        if raw.hasPrefix("http://") || raw.hasPrefix("https://") { return raw }
        return raw.contains(":") ? "http://\(raw)" : "http://\(raw):8080"
    }()
    #else
    var baseURL = "https://api.thelab.uz"
    #endif

    private init() {
        let config = URLSessionConfiguration.default
        config.timeoutIntervalForRequest = 30
        config.timeoutIntervalForResource = 60
        session = URLSession(configuration: config)

        decoder = JSONDecoder()
        encoder = JSONEncoder()
    }

    // MARK: - Token

    var authToken: String? {
        get { KeychainHelper.read(key: "auth_token") }
        set {
            if let newValue {
                KeychainHelper.save(key: "auth_token", value: newValue)
            } else {
                KeychainHelper.delete(key: "auth_token")
            }
        }
    }

    // MARK: - Generic Request

    /// Flag to prevent recursive refresh loops
    private var isRefreshing = false

    func request<T: Decodable>(
        method: String = "GET",
        path: String,
        body: (any Encodable)? = nil,
        isRetry: Bool = false
    ) async throws -> T {
        guard let url = URL(string: "\(baseURL)\(path)") else {
            throw APIError.invalidURL
        }

        var request = URLRequest(url: url)
        request.httpMethod = method
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.setValue(UUID().uuidString, forHTTPHeaderField: "X-Trace-Id")

        if let token = authToken {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }

        if let body {
            request.httpBody = try encoder.encode(AnyEncodable(body))
        }

        let (data, response) = try await session.data(for: request)

        guard let http = response as? HTTPURLResponse else {
            throw APIError.invalidResponse
        }

        if http.statusCode == 401 && !isRetry && !isRefreshing {
            if let _ = await attemptTokenRefresh() {
                return try await self.request(method: method, path: path, body: body, isRetry: true)
            }
            throw APIError.serverError(statusCode: 401, message: "Unauthorized")
        }

        guard (200...299).contains(http.statusCode) else {
            // Check for RFC 7807 structured error response
            let contentType = http.value(forHTTPHeaderField: "Content-Type") ?? ""
            if contentType.contains("application/problem+json"),
               let problem = try? decoder.decode(ProblemDetail.self, from: data) {
                throw APIError.problemDetail(problem)
            }
            let message = String(data: data, encoding: .utf8) ?? "Unknown error"
            throw APIError.serverError(statusCode: http.statusCode, message: message)
        }

        return try decoder.decode(T.self, from: data)
    }

    // MARK: - Token Refresh

    private func attemptTokenRefresh() async -> String? {
        guard let currentToken = authToken else { return nil }
        isRefreshing = true
        defer { isRefreshing = false }

        guard let url = URL(string: "\(baseURL)/v1/auth/refresh") else { return nil }
        var req = URLRequest(url: url)
        req.httpMethod = "POST"
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        req.setValue("Bearer \(currentToken)", forHTTPHeaderField: "Authorization")
        req.httpBody = "{}".data(using: .utf8)

        do {
            let (data, response) = try await session.data(for: req)
            guard let http = response as? HTTPURLResponse, (200...299).contains(http.statusCode) else {
                return nil
            }
            let json = try JSONSerialization.jsonObject(with: data) as? [String: Any]
            guard let newToken = json?["token"] as? String else { return nil }
            authToken = newToken
            return newToken
        } catch {
            return nil
        }
    }

    // MARK: - Convenience Methods

    func get<T: Decodable>(path: String) async throws -> T {
        try await request(method: "GET", path: path)
    }

    func post<T: Decodable>(path: String, body: (any Encodable)? = nil) async throws -> T {
        try await request(method: "POST", path: path, body: body)
    }

    func patch<T: Decodable>(path: String, body: (any Encodable)? = nil) async throws -> T {
        try await request(method: "PATCH", path: path, body: body)
    }

    // MARK: - Tracking

    func getTrackingOrders() async throws -> [TrackingOrder] {
        let response: TrackingResponse = try await get(path: "/v1/retailer/tracking")
        return response.orders
    }
}

// MARK: - API Error

enum APIError: LocalizedError {
    case invalidURL
    case invalidResponse
    case serverError(statusCode: Int, message: String)
    case problemDetail(ProblemDetail)
    case decodingError(Error)

    var errorDescription: String? {
        switch self {
        case .invalidURL: "Invalid URL"
        case .invalidResponse: "Invalid response from server"
        case .serverError(let code, let msg): "Server error \(code): \(msg)"
        case .problemDetail(let p): p.detail ?? p.title ?? "Server error \(p.status)"
        case .decodingError(let err): "Decoding error: \(err.localizedDescription)"
        }
    }

    var problem: ProblemDetail? {
        if case .problemDetail(let p) = self { return p }
        return nil
    }
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

// MARK: - Type Erased Encodable

private struct AnyEncodable: Encodable {
    private let _encode: (Encoder) throws -> Void

    init(_ wrapped: any Encodable) {
        _encode = wrapped.encode
    }

    func encode(to encoder: Encoder) throws {
        try _encode(encoder)
    }
}

// MARK: - Keychain Helper

enum KeychainHelper {
    static func save(key: String, value: String) {
        guard let data = value.data(using: .utf8) else { return }
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrAccount as String: key
        ]
        SecItemDelete(query as CFDictionary)

        let attributes: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrAccount as String: key,
            kSecValueData as String: data
        ]
        SecItemAdd(attributes as CFDictionary, nil)
    }

    static func read(key: String) -> String? {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrAccount as String: key,
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne
        ]
        var item: CFTypeRef?
        let status = SecItemCopyMatching(query as CFDictionary, &item)
        guard status == errSecSuccess, let data = item as? Data else { return nil }
        return String(data: data, encoding: .utf8)
    }

    static func delete(key: String) {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrAccount as String: key
        ]
        SecItemDelete(query as CFDictionary)
    }
}
