import Foundation

// MARK: - API Client

@Observable
final class APIClient {
    static let shared = APIClient()

    private let session: URLSession
    private let decoder: JSONDecoder
    private let encoder: JSONEncoder

    #if DEBUG
    // Simulator: localhost. Physical device: set LAB_DEV_HOST
    // scheme env variable to the Mac's LAN IP (e.g. 192.168.1.42).
    // PEGASUS_DEV_HOST remains as a legacy fallback for existing schemes.
    // for backend reachability over Wi-Fi.
    var baseURL: String = {
        let env = ProcessInfo.processInfo.environment
        let raw = (env["LAB_DEV_HOST"] ?? env["PEGASUS_DEV_HOST"] ?? "")
            .trimmingCharacters(in: .whitespaces)
        if raw.isEmpty { return "http://localhost:8080" }
        if raw.hasPrefix("http://") || raw.hasPrefix("https://") { return raw }
        return raw.contains(":") ? "http://\(raw)" : "http://\(raw):8080"
    }()
    #else
    var baseURL = "https://api.pegasus.uz"
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
        headers: [String: String] = [:],
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

        headers.forEach { key, value in
            request.setValue(value, forHTTPHeaderField: key)
        }

        if let body {
            request.httpBody = try encoder.encode(AnyEncodable(body))
        }

        let (data, response) = try await dataForRequestWithFallback(request)

        guard let http = response as? HTTPURLResponse else {
            throw APIError.invalidResponse
        }

        if http.statusCode == 401 && !isRetry && !isRefreshing {
            if let _ = await attemptTokenRefresh() {
                return try await self.request(method: method, path: path, body: body, headers: headers, isRetry: true)
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

    private func dataForRequestWithFallback(_ request: URLRequest) async throws -> (Data, URLResponse) {
        try await session.data(for: request)
    }

    // MARK: - Convenience Methods

    func get<T: Decodable>(path: String) async throws -> T {
        try await request(method: "GET", path: path)
    }

    func post<T: Decodable>(path: String, body: (any Encodable)? = nil, headers: [String: String] = [:]) async throws -> T {
        try await request(method: "POST", path: path, body: body, headers: headers)
    }

    func patch<T: Decodable>(path: String, body: (any Encodable)? = nil, headers: [String: String] = [:]) async throws -> T {
        try await request(method: "PATCH", path: path, body: body, headers: headers)
    }

    func put<T: Decodable>(path: String, body: (any Encodable)? = nil, headers: [String: String] = [:]) async throws -> T {
        try await request(method: "PUT", path: path, body: body, headers: headers)
    }
    
    // MARK: - Card Management
    
    func getCards() async throws -> [RetailerCardToken] {
        let response: RetailerCardsResponse = try await get(path: "/v1/retailer/cards")
        return response.cards
    }
    
    func initiateCardSave(gateway: String = "GLOBAL_PAY") async throws -> CardInitiateResponse {
        return try await post(path: "/v1/retailer/card/initiate", body: CardInitiateRequest(gateway: gateway))
    }
    
    func confirmCardSave(cardToken: String, otpCode: String) async throws -> CardConfirmResponse {
        return try await post(path: "/v1/retailer/card/confirm", body: CardConfirmRequest(cardToken: cardToken, otpCode: otpCode))
    }
    
    func deactivateCard(tokenId: String) async throws {
        let _: APIResponse<String> = try await post(path: "/v1/retailer/card/deactivate", body: CardIdRequest(tokenId: tokenId))
    }
    
    func setDefaultCard(tokenId: String) async throws {
        let _: APIResponse<String> = try await post(path: "/v1/retailer/card/default", body: CardIdRequest(tokenId: tokenId))
    }

    // MARK: - Shop Closed

    func respondToShopClosed(orderId: String, response: String) async throws {
        struct ShopClosedRequest: Encodable {
            let orderId: String
            let response: String
            enum CodingKeys: String, CodingKey {
                case orderId = "order_id"
                case response
            }
        }
        let _: APIResponse<String> = try await post(path: "/v1/retailer/shop-closed-response", body: ShopClosedRequest(orderId: orderId, response: response))
    }

    // MARK: - AI & Preorder Integrations
    
    struct ConfirmAiOrderRequest: Encodable {
        let orderId: String
        enum CodingKeys: String, CodingKey { case orderId = "order_id" }
    }
    
    struct RejectAiOrderRequest: Encodable {
        let orderId: String
        let reason: String
        enum CodingKeys: String, CodingKey { case orderId = "order_id"; case reason }
    }
    
    struct ConfirmPreorderRequest: Encodable {
        let orderId: String
        enum CodingKeys: String, CodingKey { case orderId = "order_id" }
    }
    
    struct EditPreorderItem: Encodable {
        let lineItemId: String
        let quantity: Int
        enum CodingKeys: String, CodingKey { case lineItemId = "line_item_id"; case quantity }
    }
    
    struct EditPreorderRequest: Encodable {
        let orderId: String
        let requestedDeliveryDate: String?
        let items: [EditPreorderItem]?
        enum CodingKeys: String, CodingKey { case orderId = "order_id"; case requestedDeliveryDate = "requested_delivery_date"; case items }
    }

    func confirmAiOrder(orderId: String) async throws {
        let _: APIResponse<String> = try await post(path: "/v1/retailer/orders/confirm-ai", body: ConfirmAiOrderRequest(orderId: orderId))
    }

    func rejectAiOrder(orderId: String, reason: String) async throws {
        let _: APIResponse<String> = try await post(path: "/v1/retailer/orders/reject-ai", body: RejectAiOrderRequest(orderId: orderId, reason: reason))
    }

    func confirmPreorder(orderId: String) async throws {
        let _: APIResponse<String> = try await post(path: "/v1/orders/confirm-preorder", body: ConfirmPreorderRequest(orderId: orderId))
    }

    func editPreorder(orderId: String, deliveryDate: String?, items: [EditPreorderItem]?) async throws {
        let _: APIResponse<String> = try await post(path: "/v1/orders/edit-preorder", body: EditPreorderRequest(orderId: orderId, requestedDeliveryDate: deliveryDate, items: items))
    }

    // MARK: - Tracking
    // MARK: - Phase 4: Retailer Ecosystem
    
    func getProfile() async throws -> RetailerProfileResponse {
        return try await get(path: "/v1/retailer/profile")
    }
    
    func updateProfile(request: RetailerProfileRequest) async throws {
        let _: APIResponse<String> = try await put(path: "/v1/retailer/profile", body: request)
    }

    func getFamilyMembers() async throws -> [FamilyMemberResponse] {
        return try await get(path: "/v1/retailer/family-members")
    }
    
    func addFamilyMember(request: FamilyMemberRequest) async throws {
        let _: APIResponse<String> = try await post(path: "/v1/retailer/family-members", body: request)
    }
    
    func removeFamilyMember(memberId: String) async throws {
        // DELETE requires custom wrapper or generic empty request. Reusing existing request helper:
        var components = URLComponents(string: baseURL + "/v1/retailer/family-members/\(memberId)")!
        var request = URLRequest(url: components.url!)
        request.httpMethod = "DELETE"
        if let token = UserDefaults.standard.string(forKey: "auth_token") {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }
        _ = try await dataForRequestWithFallback(request)
    }

    func syncCart(request: CartSyncRequest) async throws -> CartSyncResponse {
        return try await post(path: "/v1/retailer/cart/sync", body: request)
    }

    func getSuppliers() async throws -> [RetailerSupplierResponse] {
        return try await get(path: "/v1/retailer/suppliers")
    }
    
    func favoriteSupplier(supplierId: String) async throws {
        let _: APIResponse<String> = try await post(path: "/v1/retailer/suppliers/\(supplierId)/add")
    }
    
    func unfavoriteSupplier(supplierId: String) async throws {
        let _: APIResponse<String> = try await post(path: "/v1/retailer/suppliers/\(supplierId)/remove")
    }

    struct GlobalAutoOrderRequest: Encodable {
        let globalAutoOrderEnabled: Bool
        let useHistory: Bool?

        enum CodingKeys: String, CodingKey {
            case globalAutoOrderEnabled = "global_auto_order_enabled"
            case useHistory = "use_history"
        }

        func encode(to encoder: Encoder) throws {
            var container = encoder.container(keyedBy: CodingKeys.self)
            try container.encode(globalAutoOrderEnabled, forKey: .globalAutoOrderEnabled)
            try container.encodeIfPresent(useHistory, forKey: .useHistory)
        }
    }

    struct ScopedAutoOrderRequest: Encodable {
        let autoOrderEnabled: Bool
        let useHistory: Bool?

        enum CodingKeys: String, CodingKey {
            case autoOrderEnabled = "auto_order_enabled"
            case useHistory = "use_history"
        }

        func encode(to encoder: Encoder) throws {
            var container = encoder.container(keyedBy: CodingKeys.self)
            try container.encode(autoOrderEnabled, forKey: .autoOrderEnabled)
            try container.encodeIfPresent(useHistory, forKey: .useHistory)
        }
    }

    func getAutoOrderSettings() async throws -> AutoOrderSettings {
        try await get(path: "/v1/retailer/settings/auto-order")
    }

    func setGlobalAutoOrder(enabled: Bool, useHistory: Bool? = nil) async throws -> [String: Bool] {
        try await patch(
            path: "/v1/retailer/settings/auto-order/global",
            body: GlobalAutoOrderRequest(globalAutoOrderEnabled: enabled, useHistory: useHistory)
        )
    }

    func setSupplierAutoOrder(supplierId: String, enabled: Bool, useHistory: Bool? = nil) async throws -> [String: Bool] {
        try await patch(
            path: "/v1/retailer/settings/auto-order/supplier/\(supplierId)",
            body: ScopedAutoOrderRequest(autoOrderEnabled: enabled, useHistory: useHistory)
        )
    }

    func setCategoryAutoOrder(categoryId: String, enabled: Bool, useHistory: Bool? = nil) async throws -> [String: Bool] {
        try await patch(
            path: "/v1/retailer/settings/auto-order/category/\(categoryId)",
            body: ScopedAutoOrderRequest(autoOrderEnabled: enabled, useHistory: useHistory)
        )
    }

    func setProductAutoOrder(productId: String, enabled: Bool, useHistory: Bool? = nil) async throws -> [String: Bool] {
        try await patch(
            path: "/v1/retailer/settings/auto-order/product/\(productId)",
            body: ScopedAutoOrderRequest(autoOrderEnabled: enabled, useHistory: useHistory)
        )
    }

    func setVariantAutoOrder(skuId: String, enabled: Bool, useHistory: Bool? = nil) async throws -> [String: Bool] {
        try await patch(
            path: "/v1/retailer/settings/auto-order/variant/\(skuId)",
            body: ScopedAutoOrderRequest(autoOrderEnabled: enabled, useHistory: useHistory)
        )
    }

    func getTrackingOrders() async throws -> [TrackingOrder] {
        let response: TrackingResponse = try await get(path: "/v1/retailer/tracking")
        return response.orders
    }

    func getPendingPayments() async throws -> PendingPaymentsResponse {
        try await get(path: "/v1/retailer/pending-payments")
    }

    func getActiveFulfillments() async throws -> ActiveFulfillmentsResponse {
        try await get(path: "/v1/retailer/active-fulfillment")
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

private enum AuthNamespace {
    static let primaryService = "com.pegasus.retailerapp"
}

enum KeychainHelper {
    static func save(key: String, value: String) {
        guard let data = value.data(using: .utf8) else { return }
        deleteFromService(AuthNamespace.primaryService, key: key)

        let attributes: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrAccount as String: key,
            kSecAttrService as String: AuthNamespace.primaryService,
            kSecValueData as String: data
        ]
        SecItemAdd(attributes as CFDictionary, nil)
    }

    static func read(key: String) -> String? {
        readFromService(AuthNamespace.primaryService, key: key)
    }

    static func delete(key: String) {
        deleteFromService(AuthNamespace.primaryService, key: key)
    }

    private static func readFromService(_ service: String, key: String) -> String? {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrAccount as String: key,
            kSecAttrService as String: service,
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne
        ]
        var item: CFTypeRef?
        let status = SecItemCopyMatching(query as CFDictionary, &item)
        guard status == errSecSuccess, let data = item as? Data else { return nil }
        return String(data: data, encoding: .utf8)
    }

    private static func deleteFromService(_ service: String, key: String) {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrAccount as String: key,
            kSecAttrService as String: service,
        ]
        SecItemDelete(query as CFDictionary)
    }
}
