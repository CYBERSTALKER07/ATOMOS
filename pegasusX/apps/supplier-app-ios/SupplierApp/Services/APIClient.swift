import Foundation

final class APIClient: Sendable {
    static let shared = APIClient()

    #if DEBUG
    private let baseURL: URL = {
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
    private let baseURL = URL(string: "https://api.pegasus.uz/")!
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

    func post<B: Encodable, T: Decodable>(_ path: String, body: B, authenticated: Bool = true) async throws -> T {
        var request = URLRequest(url: baseURL.appendingPathComponent(path))
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
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

    func put<B: Encodable, T: Decodable>(_ path: String, body: B, authenticated: Bool = true) async throws -> T {
        var request = URLRequest(url: baseURL.appendingPathComponent(path))
        request.httpMethod = "PUT"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try encoder.encode(body)
        if authenticated {
            await attachToken(&request)
        }
        return try await execute(request)
    }

    func patch<B: Encodable, T: Decodable>(_ path: String, body: B, authenticated: Bool = true) async throws -> T {
        var request = URLRequest(url: baseURL.appendingPathComponent(path))
        request.httpMethod = "PATCH"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try encoder.encode(body)
        if authenticated {
            await attachToken(&request)
        }
        return try await execute(request)
    }

    func postVoid<B: Encodable>(_ path: String, body: B, authenticated: Bool = true) async throws {
        var request = URLRequest(url: baseURL.appendingPathComponent(path))
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try encoder.encode(body)
        if authenticated {
            await attachToken(&request)
        }
        _ = try await executeVoid(request)
    }

    func deleteVoid(_ path: String, authenticated: Bool = true) async throws {
        var request = URLRequest(url: baseURL.appendingPathComponent(path))
        request.httpMethod = "DELETE"
        if authenticated {
            await attachToken(&request)
        }
        _ = try await executeVoid(request)
    }

    private func execute<T: Decodable>(_ request: URLRequest) async throws -> T {
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
        return try decoder.decode(T.self, from: data)
    }

    private func executeVoid(_ request: URLRequest) async throws {
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
    }

    private func attachToken(_ request: inout URLRequest) async {
        request.setValue(UUID().uuidString, forHTTPHeaderField: "X-Trace-Id")
        let token = await MainActor.run { TokenStore.shared.token }
        if let token {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
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
