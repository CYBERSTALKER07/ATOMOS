import Foundation

public actor ControlTowerAPIClient {
    public static let shared = ControlTowerAPIClient()

    public static var defaultBaseURL: URL {
        var components = URLComponents()
        components.scheme = "http"
        components.host = "127.0.0.1"
        components.port = 8180
        return components.url ?? URL(fileURLWithPath: "/")
    }

    public nonisolated let baseURL: URL
    private let session: URLSession
    private var authToken: String?

    public init(baseURL: URL? = nil, session: URLSession? = nil) {
        if let session {
            self.session = session
        } else {
            let config = URLSessionConfiguration.default
            config.timeoutIntervalForRequest = 8.0
            config.timeoutIntervalForResource = 15.0
            self.session = URLSession(configuration: config)
        }
        self.baseURL = baseURL ?? Self.defaultBaseURL
    }

    public static func isAllowedEndpointURL(_ url: URL) -> Bool {
        guard let scheme = url.scheme?.lowercased(), let host = url.host?.lowercased(), !host.isEmpty else {
            return false
        }
        if scheme == "https" {
            return true
        }
        return scheme == "http" && (host == "127.0.0.1" || host == "localhost" || host == "::1")
    }

    public func setAuthToken(_ token: String?) {
        let trimmed = token?.trimmingCharacters(in: .whitespacesAndNewlines)
        self.authToken = (trimmed?.isEmpty == false) ? trimmed : nil
    }

    public func currentAuthToken() -> String? {
        authToken
    }

    public func clearSession() {
        authToken = nil
        session.configuration.urlCache?.removeAllCachedResponses()
    }

    private func getJSON<T: Decodable>(_ path: String, queryItems: [URLQueryItem] = []) async throws -> T {
        guard Self.isAllowedEndpointURL(baseURL) else {
            throw URLError(.unsupportedURL)
        }
        var normalizedPath = path
        if !normalizedPath.hasPrefix("/") {
            normalizedPath = "/" + normalizedPath
        }
        guard var components = URLComponents(url: baseURL, resolvingAgainstBaseURL: true) else {
            throw URLError(.badURL)
        }
        let basePath = components.path.hasSuffix("/") ? String(components.path.dropLast()) : components.path
        components.path = basePath + normalizedPath
        if !queryItems.isEmpty {
            components.queryItems = queryItems
        }
        guard let url = components.url else {
            throw URLError(.badURL)
        }

        var request = URLRequest(url: url)
        request.httpMethod = "GET"
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        if let token = authToken, !token.isEmpty {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }

        let (data, response) = try await session.data(for: request)
        guard let httpResponse = response as? HTTPURLResponse else {
            throw URLError(.badServerResponse)
        }
        guard (200...299).contains(httpResponse.statusCode) else {
            throw URLError(URLError.Code(rawValue: httpResponse.statusCode))
        }

        let decoder = JSONDecoder()
        return try decoder.decode(T.self, from: data)
    }

    public func fetchCellHealth() async throws -> [CellHealth] {
        try await getJSON("/v1/control-tower/cells")
    }

    public func fetchSpannerMetrics() async throws -> SpannerClusterMetrics {
        try await getJSON("/v1/control-tower/metrics/spanner")
    }

    public func fetchKafkaLags() async throws -> [KafkaPartitionLag] {
        try await getJSON("/v1/control-tower/metrics/kafka")
    }

    public func fetchRedisPools() async throws -> [RedisPermitPool] {
        try await getJSON("/v1/control-tower/metrics/redis")
    }

    public func fetchWaterfall(for role: RolePerspective) async throws -> [TenderWaterfallSlice] {
        try await getJSON("/v1/control-tower/waterfall", queryItems: [URLQueryItem(name: "role", value: role.rawValue)])
    }

    public func fetchAttributedCosts() async throws -> [AttributedCostItem] {
        try await getJSON("/v1/control-tower/costs/attributed")
    }

    public func fetchScorecards() async throws -> [RolePerformanceScorecard] {
        try await getJSON("/v1/control-tower/scorecards")
    }
}
