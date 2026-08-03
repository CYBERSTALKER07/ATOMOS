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
        if raw.isEmpty { return "http://localhost:8180" }
        if raw.hasPrefix("http://") || raw.hasPrefix("https://") { return raw }
        return raw.contains(":") ? "http://\(raw)" : "http://\(raw):8180"
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

        guard let url = URL(string: "\(baseURL)/v1/auth/retailer/refresh") else { return nil }
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
    
    // MARK: - Logistics claims (post-delivery)

    func listOrderClaims(orderId: String) async throws -> [RetailerClaim] {
        let response: RetailerClaimsListResponse = try await get(path: "/v1/orders/\(orderId)/claims")
        return response.claims
    }

    func fileOrderClaim(
        orderId: String,
        claimType: String,
        description: String,
        lines: [FileClaimLineBody],
        photoURL: String?
    ) async throws -> RetailerClaim {
        var evidences: [FileClaimEvidenceBody] = []
        if let photoURL, !photoURL.isEmpty {
            evidences.append(FileClaimEvidenceBody(
                evidenceType: "PHOTO",
                uri: photoURL,
                mimeType: "image/jpeg"
            ))
        }
        let body = FileClaimRequestBody(
            claimType: claimType,
            description: description,
            lineItems: lines,
            evidences: evidences
        )
        return try await post(path: "/v1/orders/\(orderId)/claims", body: body)
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
        let _: APIResponse<String> = try await post(
            path: "/v1/retailer/shop-closed-response",
            body: ShopClosedRequest(orderId: orderId, response: response),
            headers: ["Idempotency-Key": RetailerIdempotency.shopClosedResponse(orderId: orderId, response: response)]
        )
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

    struct AcceptDeliveryProposalRequest: Encodable {
        let orderId: String
        enum CodingKeys: String, CodingKey { case orderId = "order_id" }
    }

    struct RejectDeliveryProposalRequest: Encodable {
        let orderId: String
        let reason: String?
        enum CodingKeys: String, CodingKey { case orderId = "order_id"; case reason }
    }

    struct RejectPreorderRequest: Encodable {
        let orderId: String
        let reason: String?
        enum CodingKeys: String, CodingKey { case orderId = "order_id"; case reason }
    }
    
    struct EditPreorderItem: Encodable {
        let sku: String
        let name: String
        let quantity: Int64
        let unitPriceMinor: Int64

        enum CodingKeys: String, CodingKey {
            case sku
            case name
            case quantity
            case unitPriceMinor = "unit_price_minor"
        }
    }

    struct EditPreorderRequest: Encodable {
        let orderId: String
        let requestedDeliveryDate: String
        let lineItems: [EditPreorderItem]

        enum CodingKeys: String, CodingKey {
            case orderId = "order_id"
            case requestedDeliveryDate = "requested_delivery_date"
            case lineItems = "line_items"
        }
    }

    func confirmAiOrder(orderId: String) async throws {
        let _: APIResponse<String> = try await post(
            path: "/v1/retailer/orders/confirm-ai",
            body: ConfirmAiOrderRequest(orderId: orderId),
            headers: ["Idempotency-Key": RetailerIdempotency.confirmAI(orderId: orderId)]
        )
    }

    func rejectAiOrder(orderId: String, reason: String) async throws {
        let _: APIResponse<String> = try await post(
            path: "/v1/retailer/orders/reject-ai",
            body: RejectAiOrderRequest(orderId: orderId, reason: reason),
            headers: ["Idempotency-Key": RetailerIdempotency.rejectAI(orderId: orderId, reason: reason)]
        )
    }

    func confirmPreorder(orderId: String) async throws {
        let _: APIResponse<String> = try await post(
            path: "/v1/orders/confirm-preorder",
            body: ConfirmPreorderRequest(orderId: orderId),
            headers: ["Idempotency-Key": RetailerIdempotency.confirmPreorder(orderId: orderId)]
        )
    }

    func acceptDeliveryProposal(orderId: String) async throws {
        let _: APIResponse<String> = try await post(
            path: "/v1/orders/accept-delivery-proposal",
            body: AcceptDeliveryProposalRequest(orderId: orderId),
            headers: ["Idempotency-Key": RetailerIdempotency.acceptDeliveryProposal(orderId: orderId)]
        )
    }

    func rejectDeliveryProposal(orderId: String, reason: String? = nil) async throws {
        let _: APIResponse<String> = try await post(
            path: "/v1/orders/reject-delivery-proposal",
            body: RejectDeliveryProposalRequest(orderId: orderId, reason: reason),
            headers: ["Idempotency-Key": RetailerIdempotency.rejectDeliveryProposal(orderId: orderId, reason: reason ?? "")]
        )
    }

    func rejectPreorder(orderId: String, reason: String? = nil) async throws {
        let _: APIResponse<String> = try await post(
            path: "/v1/orders/reject-preorder",
            body: RejectPreorderRequest(orderId: orderId, reason: reason),
            headers: ["Idempotency-Key": RetailerIdempotency.rejectPreorder(orderId: orderId, reason: reason ?? "")]
        )
    }

    func editPreorder(orderId: String, deliveryDate: String, items: [EditPreorderItem]) async throws {
        let _: APIResponse<String> = try await post(
            path: "/v1/orders/edit-preorder",
            body: EditPreorderRequest(orderId: orderId, requestedDeliveryDate: deliveryDate, lineItems: items),
            headers: ["Idempotency-Key": RetailerIdempotency.editPreorder(orderId: orderId)]
        )
    }

    // MARK: - Tracking
    // MARK: - Phase 4: Retailer Ecosystem
    
    func setupRetailer(payload: [String: AnyEncodable], idempotencyKey: String? = nil) async throws {
        var headers: [String: String] = [:]
        if let idempotencyKey {
            headers["Idempotency-Key"] = idempotencyKey
        }
        let _: APIResponse<String> = try await post(path: "/v1/retailer/setup", body: payload, headers: headers)
    }

    func requestCancelOrder(orderId: String, retailerId: String, reason: String = "Retailer requested cancellation") async throws {
        let _: APIResponse<String> = try await post(
            path: "/v1/orders/request-cancel",
            body: [
                "order_id": AnyEncodable(orderId),
                "retailer_id": AnyEncodable(retailerId),
                "reason": AnyEncodable(reason),
            ],
            headers: ["Idempotency-Key": RetailerIdempotency.requestCancel(orderId: orderId)]
        )
    }

    func getPricingRules() async throws -> [String: AnyDecodable] {
        return try await get(path: "/v1/retailer/pricing/rules")
    }

    func getProfile() async throws -> RetailerProfileResponse {
        return try await get(path: "/v1/retailer/profile")
    }

    func getCreditProfile() async throws -> CreditProfile {
        try await get(path: "/v1/retailer/credit-profile")
    }
    
    func updateProfile(request: RetailerProfileRequest, idempotencyKey: String) async throws {
        let _: APIResponse<String> = try await put(
            path: "/v1/retailer/profile",
            body: request,
            headers: ["Idempotency-Key": idempotencyKey]
        )
    }

    // Retail OS Phase 0
    func getRetailerMe() async throws -> RetailerMeResponse {
        try await get(path: "/v1/retailer/me")
    }

    func getCapabilities() async throws -> RetailerCapabilitiesResponse {
        try await get(path: "/v1/retailer/capabilities")
    }

    func enableCapability(packId: String, acceptSoft: Bool = true, enableDeps: Bool = true) async throws -> RetailerCapabilityMutationResponse {
        struct Body: Encodable {
            let accept_soft_deps: Bool
            let enable_deps: Bool
        }
        return try await post(
            path: "/v1/retailer/capabilities/\(packId)/enable",
            body: Body(accept_soft_deps: acceptSoft, enable_deps: enableDeps),
            headers: ["Idempotency-Key": "cap-en-\(packId)-\(Int(Date().timeIntervalSince1970 * 1000))"]
        )
    }

    func disableCapability(packId: String) async throws -> RetailerCapabilityMutationResponse {
        struct Empty: Encodable {}
        return try await post(
            path: "/v1/retailer/capabilities/\(packId)/disable",
            body: Empty(),
            headers: ["Idempotency-Key": "cap-dis-\(packId)-\(Int(Date().timeIntervalSince1970 * 1000))"]
        )
    }

    // Retail OS Phase 2 locations
    func getLocations() async throws -> RetailerLocationsResponse {
        try await get(path: "/v1/retailer/locations")
    }

    func createLocation(name: String, address: String, lat: Double = 0, lng: Double = 0) async throws -> RetailerLocationsResponse {
        struct Body: Encodable {
            let name: String
            let delivery_address: String
            let lat: Double
            let lng: Double
        }
        return try await post(
            path: "/v1/retailer/locations",
            body: Body(name: name, delivery_address: address, lat: lat, lng: lng),
            headers: ["Idempotency-Key": "loc-\(Int(Date().timeIntervalSince1970 * 1000))"]
        )
    }

    func setPrimaryLocation(locationId: String) async throws -> RetailerLocationsResponse {
        struct Empty: Encodable {}
        return try await post(
            path: "/v1/retailer/locations/\(locationId)/set-primary",
            body: Empty(),
            headers: ["Idempotency-Key": "prim-\(locationId)-\(Int(Date().timeIntervalSince1970 * 1000))"]
        )
    }

    func switchLocation(locationId: String) async throws -> [String: AnyDecodable] {
        struct Body: Encodable { let location_id: String }
        return try await post(
            path: "/v1/auth/retailer/switch-location",
            body: Body(location_id: locationId)
        )
    }

    // Retail OS Phase 3 store stock
    func getStoreStock(locationId: String?) async throws -> StoreStockListWire {
        var path = "/v1/retailer/stock"
        if let locationId, !locationId.isEmpty {
            path += "?location_id=\(locationId.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? locationId)"
        }
        return try await get(path: path)
    }

    func receiveStoreStock(orderId: String, locationId: String) async throws -> [String: AnyDecodable] {
        struct Body: Encodable {
            let order_id: String
            let location_id: String
            let confirm: Bool
            let stock_bin: String
        }
        return try await post(
            path: "/v1/retailer/stock/receive-sessions",
            body: Body(order_id: orderId, location_id: locationId, confirm: true, stock_bin: "BACKROOM"),
            headers: ["Idempotency-Key": "recv-\(Int(Date().timeIntervalSince1970 * 1000))"]
        )
    }

    func transferStoreStock(locationId: String, sku: String, qty: Int64) async throws -> [String: AnyDecodable] {
        struct Body: Encodable {
            let location_id: String
            let sku: String
            let qty: Int64
            let from_bin: String
            let to_bin: String
        }
        return try await post(
            path: "/v1/retailer/stock/transfer",
            body: Body(location_id: locationId, sku: sku, qty: qty, from_bin: "BACKROOM", to_bin: "FLOOR"),
            headers: ["Idempotency-Key": "xfer-\(Int(Date().timeIntervalSince1970 * 1000))"]
        )
    }

    func adjustStoreStock(locationId: String, sku: String, qtyDelta: Int64) async throws -> [String: AnyDecodable] {
        struct Body: Encodable {
            let location_id: String
            let sku: String
            let qty_delta: Int64
            let stock_bin: String
            let note: String
        }
        return try await post(
            path: "/v1/retailer/stock/adjust",
            body: Body(location_id: locationId, sku: sku, qty_delta: qtyDelta, stock_bin: "BACKROOM", note: "ios_adjust"),
            headers: ["Idempotency-Key": "adj-\(Int(Date().timeIntervalSince1970 * 1000))"]
        )
    }

    // L6 local / manual POS SKUs
    func getLocalSkus() async throws -> LocalSkuListWire {
        try await get(path: "/v1/retailer/local-skus")
    }

    func createLocalSku(name: String, barcode: String, priceMinor: Int64) async throws -> LocalSkuWire {
        struct Body: Encodable {
            let name: String
            let barcode: String
            let default_price_minor: Int64
        }
        return try await post(
            path: "/v1/retailer/local-skus",
            body: Body(name: name, barcode: barcode, default_price_minor: priceMinor),
            headers: ["Idempotency-Key": "local-\(Int(Date().timeIntervalSince1970 * 1000))"]
        )
    }

    func patchLocalSku(id: String, isActive: Bool) async throws -> LocalSkuWire {
        struct Body: Encodable { let is_active: Bool }
        return try await patch(path: "/v1/retailer/local-skus/\(id)", body: Body(is_active: isActive))
    }

    // Retail OS Phase 4 POS
    func getRegisters() async throws -> RetailerRegistersWire {
        try await get(path: "/v1/retailer/registers")
    }

    func createRegister(label: String) async throws -> RetailerRegisterWire {
        struct Body: Encodable { let label: String }
        return try await post(
            path: "/v1/retailer/registers",
            body: Body(label: label),
            headers: ["Idempotency-Key": "reg-\(Int(Date().timeIntervalSince1970 * 1000))"]
        )
    }

    func openPosSession(registerId: String) async throws -> PosSessionWire {
        struct Body: Encodable {
            let register_id: String
            let opening_float_minor: Int64
            let currency: String
        }
        return try await post(
            path: "/v1/retailer/pos/sessions/open",
            body: Body(register_id: registerId, opening_float_minor: 0, currency: "UZS"),
            headers: ["Idempotency-Key": "open-\(Int(Date().timeIntervalSince1970 * 1000))"]
        )
    }

    func closePosSession(sessionId: String, closingCashMinor: Int64) async throws -> PosSessionWire {
        struct Body: Encodable { let closing_cash_minor: Int64 }
        return try await post(
            path: "/v1/retailer/pos/sessions/\(sessionId)/close",
            body: Body(closing_cash_minor: closingCashMinor)
        )
    }

    func createPosSale(
        sessionId: String,
        lines: [PosSaleLineWire],
        totalMinor: Int64,
        clientSaleId: String = UUID().uuidString,
        origin: String = "online",
        clientCreatedAt: String? = nil
    ) async throws -> PosSaleWire {
        struct Body: Encodable {
            let session_id: String
            let stock_bin: String
            let lines: [PosSaleLineWire]
            let tenders: [PosTenderWire]
            let client_sale_id: String
            let origin: String
            let client_created_at: String?
        }
        return try await post(
            path: "/v1/retailer/pos/sales",
            body: Body(
                session_id: sessionId,
                stock_bin: "FLOOR",
                lines: lines,
                tenders: [PosTenderWire(method: "CASH", amount_minor: totalMinor)],
                client_sale_id: clientSaleId,
                origin: origin,
                client_created_at: clientCreatedAt ?? ISO8601DateFormatter().string(from: Date())
            ),
            headers: ["Idempotency-Key": "pos-sale:\(clientSaleId)"]
        )
    }

    func voidPosSale(saleId: String) async throws -> PosSaleWire {
        struct Body: Encodable { let reason: String }
        return try await post(
            path: "/v1/retailer/pos/sales/\(saleId)/void",
            body: Body(reason: "ios_void")
        )
    }

    // Retail OS Phase 5 shifts & time
    func clockIn(locationId: String? = nil) async throws -> TimeEntryWire {
        struct Body: Encodable { let location_id: String? }
        return try await post(
            path: "/v1/retailer/time/clock-in",
            body: Body(location_id: locationId)
        )
    }

    func clockOut() async throws -> TimeEntryWire {
        struct Empty: Encodable {}
        return try await post(path: "/v1/retailer/time/clock-out", body: Empty())
    }

    func getTimeEntries() async throws -> TimeEntriesWire {
        try await get(path: "/v1/retailer/time/entries")
    }

    func getShifts(locationId: String? = nil) async throws -> ShiftsListWire {
        if let locationId, !locationId.isEmpty {
            return try await get(path: "/v1/retailer/shifts?location_id=\(locationId)")
        }
        return try await get(path: "/v1/retailer/shifts")
    }

    func openShift(registerId: String?, openingFloatMinor: Int64) async throws -> ShiftWire {
        struct Body: Encodable {
            let register_id: String?
            let opening_float_minor: Int64
            let currency: String
        }
        return try await post(
            path: "/v1/retailer/shifts",
            body: Body(register_id: registerId, opening_float_minor: openingFloatMinor, currency: "UZS"),
            headers: ["Idempotency-Key": "shift-\(Int(Date().timeIntervalSince1970 * 1000))"]
        )
    }

    func closeShift(shiftId: String, closingCashMinor: Int64) async throws -> ShiftWire {
        struct Body: Encodable { let closing_cash_minor: Int64 }
        return try await post(
            path: "/v1/retailer/shifts/\(shiftId)/close",
            body: Body(closing_cash_minor: closingCashMinor)
        )
    }

    // Retail OS Phase 6
    func getSections(locationId: String? = nil) async throws -> SectionsListWire {
        if let locationId, !locationId.isEmpty {
            return try await get(path: "/v1/retailer/sections?location_id=\(locationId)")
        }
        return try await get(path: "/v1/retailer/sections")
    }

    func createSection(name: String, aisleTag: String?) async throws -> SectionWire {
        struct Body: Encodable {
            let name: String
            let aisle_tag: String?
        }
        return try await post(
            path: "/v1/retailer/sections",
            body: Body(name: name, aisle_tag: aisleTag),
            headers: ["Idempotency-Key": "sec-\(Int(Date().timeIntervalSince1970 * 1000))"]
        )
    }

    func putSectionSkus(sectionId: String, skus: [String]) async throws -> [String: AnyDecodable] {
        struct Body: Encodable { let skus: [String] }
        return try await put(path: "/v1/retailer/sections/\(sectionId)/skus", body: Body(skus: skus))
    }

    func getReportsSummary() async throws -> ReportsSummaryWire {
        try await get(path: "/v1/retailer/reports/summary")
    }

    func getControlTowerPulse() async throws -> ControlTowerPulseWire {
        try await get(path: "/v1/retailer/control-tower/pulse")
    }

    func getAssistTickets() async throws -> AssistTicketsWire {
        try await get(path: "/v1/retailer/assist/tickets")
    }

    func createAssistTicket(sectionId: String, note: String) async throws -> AssistTicketWire {
        struct Body: Encodable {
            let section_id: String
            let note: String
        }
        return try await post(
            path: "/v1/retailer/assist/tickets",
            body: Body(section_id: sectionId, note: note),
            headers: ["Idempotency-Key": "assist-\(Int(Date().timeIntervalSince1970 * 1000))"]
        )
    }

    func claimAssistTicket(ticketId: String) async throws -> AssistTicketWire {
        struct Empty: Encodable {}
        return try await post(path: "/v1/retailer/assist/tickets/\(ticketId)/claim", body: Empty())
    }

    func completeAssistTicket(ticketId: String) async throws -> AssistTicketWire {
        struct Empty: Encodable {}
        return try await post(path: "/v1/retailer/assist/tickets/\(ticketId)/complete", body: Empty())
    }

    // Retail OS Phase 1 team
    func getOrgMembers() async throws -> RetailerOrgMembersResponse {
        try await get(path: "/v1/retailer/org/members")
    }

    func createOrgMember(name: String, phone: String, password: String, role: String) async throws -> RetailerOrgMembersResponse {
        struct Body: Encodable {
            let name: String
            let phone: String
            let password: String
            let retailer_role: String
        }
        return try await post(
            path: "/v1/retailer/org/members",
            body: Body(name: name, phone: phone, password: password, retailer_role: role),
            headers: ["Idempotency-Key": "team-\(Int(Date().timeIntervalSince1970 * 1000))"]
        )
    }

    func deactivateOrgMember(userId: String) async throws -> RetailerOrgMembersResponse {
        struct Empty: Encodable {}
        // DELETE via request helper
        let components = URLComponents(string: baseURL + "/v1/retailer/org/members/\(userId)")!
        var request = URLRequest(url: components.url!)
        request.httpMethod = "DELETE"
        request.setValue("team-deact-\(userId)-\(Int(Date().timeIntervalSince1970 * 1000))", forHTTPHeaderField: "Idempotency-Key")
        if let token = authToken {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }
        let data = try await dataForRequestWithFallback(request)
        return try JSONDecoder().decode(RetailerOrgMembersResponse.self, from: data)
    }

    func getFamilyMembersList() async throws -> FamilyMembersListResponse {
        try await get(path: "/v1/retailer/family-members")
    }

    func getFamilyMembers() async throws -> [FamilyMemberResponse] {
        let list = try await getFamilyMembersList()
        return list.members
    }

    func addFamilyMember(request: FamilyMemberRequest) async throws {
        struct Empty: Decodable {}
        let _: Empty = try await post(
            path: "/v1/retailer/family-members",
            body: request,
            headers: ["Idempotency-Key": "fam-add-\(Int(Date().timeIntervalSince1970 * 1000))"]
        )
    }

    func migrateFamilyToTeam(role: String = "RECEIVER") async throws -> FamilyMigrateResult {
        struct Body: Encodable {
            let retailerRole: String
            enum CodingKeys: String, CodingKey { case retailerRole = "retailer_role" }
        }
        return try await post(
            path: "/v1/retailer/family-members/migrate-to-team",
            body: Body(retailerRole: role),
            headers: ["Idempotency-Key": "fam-migrate-\(Int(Date().timeIntervalSince1970 * 1000))"]
        )
    }

    func removeFamilyMember(memberId: String) async throws {
        // DELETE requires custom wrapper or generic empty request. Reusing existing request helper:
        let components = URLComponents(string: baseURL + "/v1/retailer/family-members/\(memberId)")!
        var request = URLRequest(url: components.url!)
        request.httpMethod = "DELETE"
        if let token = authToken {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }
        _ = try await dataForRequestWithFallback(request)
    }

    func getCartSync() async throws -> CartSyncResponse {
        try await get(path: "/v1/retailer/cart/sync")
    }

    func syncCart(request: CartSyncRequest) async throws -> CartSyncResponse {
        return try await post(path: "/v1/retailer/cart/sync", body: request)
    }

    func checkoutQuote(supplierID: String, lines: [CheckoutQuoteLine]) async throws -> CheckoutQuoteResponse {
        try await post(
            path: "/v1/retailer/checkout/quote",
            body: CheckoutQuoteRequest(supplierID: supplierID, lines: lines)
        )
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

    /// Auto-order worker tick. mode=draft|place (place requires flag + role + geo).
    func runAutoOrder(mode: String = "draft") async throws -> AutoOrderRun {
        let encoded = mode.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? mode
        return try await post(path: "/v1/retailer/settings/auto-order/run?mode=\(encoded)")
    }

    func getAutoOrderRuns() async throws -> AutoOrderRunsResponse {
        try await get(path: "/v1/retailer/settings/auto-order/runs")
    }

    /// OPEN reorder suggestions with sources[] (STORE_POS / WHOLESALE_HISTORY).
    func getReorderSuggestions(source: String? = nil) async throws -> RetailerReorderSuggestionsResponse {
        var path = "/v1/retailer/reorder-suggestions"
        if let source, !source.isEmpty {
            let enc = source.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? source
            path += "?source=\(enc)"
        }
        return try await get(path: path)
    }

    func getTracking() async throws -> TrackingResponse {
        try await get(path: "/v1/retailer/tracking")
    }

    func getTrackingOrders() async throws -> [TrackingOrder] {
        let response = try await getTracking()
        return response.orders
    }

    func getPendingPayments() async throws -> PendingPaymentsResponse {
        try await get(path: "/v1/retailer/pending-payments")
    }

    func getActiveFulfillments() async throws -> ActiveFulfillmentsResponse {
        try await get(path: "/v1/retailer/active-fulfillment")
    }

    func getOrderTimeline(orderId: String) async throws -> OrderTimelineResponse {
        try await get(path: "/v1/order/\(orderId)/timeline")
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

// MARK: - Type Erased Codable

struct AnyEncodable: Encodable {
    private let _encode: (Encoder) throws -> Void

    init(_ wrapped: any Encodable) {
        _encode = wrapped.encode
    }

    func encode(to encoder: Encoder) throws {
        try _encode(encoder)
    }
}

struct RetailerRegistersWire: Codable {
    let items: [RetailerRegisterWire]
}

struct RetailerRegisterWire: Codable {
    let registerId: String
    let label: String?
    enum CodingKeys: String, CodingKey {
        case label
        case registerId = "register_id"
    }
}

struct PosSessionWire: Codable {
    let sessionId: String
    let status: String?
    enum CodingKeys: String, CodingKey {
        case status
        case sessionId = "session_id"
    }
}

struct PosSaleLineWire: Encodable {
    let sku: String
    let name: String
    let qty: Int64
    let unit_price_minor: Int64
    init(sku: String, name: String, qty: Int64, unitPriceMinor: Int64) {
        self.sku = sku
        self.name = name
        self.qty = qty
        self.unit_price_minor = unitPriceMinor
    }
}

struct PosTenderWire: Encodable {
    let method: String
    let amount_minor: Int64
}

struct PosSaleWire: Codable {
    let saleId: String
    let receiptNumber: String
    let totalMinor: Int64?
    let status: String?
    enum CodingKeys: String, CodingKey {
        case status
        case saleId = "sale_id"
        case receiptNumber = "receipt_number"
        case totalMinor = "total_minor"
    }
}

struct TimeEntriesWire: Codable {
    let items: [TimeEntryWire]?
    let clockedIn: Bool
    let openEntry: TimeEntryWire?

    enum CodingKeys: String, CodingKey {
        case items
        case clockedIn = "clocked_in"
        case openEntry = "open_entry"
    }
}

struct TimeEntryWire: Codable {
    let entryId: String?
    let userId: String?
    let locationId: String?
    let status: String?
    let clockInAt: String?
    let clockOutAt: String?

    enum CodingKeys: String, CodingKey {
        case status
        case entryId = "entry_id"
        case userId = "user_id"
        case locationId = "location_id"
        case clockInAt = "clock_in_at"
        case clockOutAt = "clock_out_at"
    }
}

struct ShiftsListWire: Codable {
    let items: [ShiftWire]
}

struct ShiftWire: Codable {
    let shiftId: String
    let status: String
    let openingFloatMinor: Int64
    let varianceMinor: Int64?
    let currency: String?

    enum CodingKeys: String, CodingKey {
        case status
        case currency
        case shiftId = "shift_id"
        case openingFloatMinor = "opening_float_minor"
        case varianceMinor = "variance_minor"
    }
}

struct SectionsListWire: Codable {
    let items: [SectionWire]
}

struct SectionWire: Codable {
    let sectionId: String
    let name: String
    let aisleTag: String?

    enum CodingKeys: String, CodingKey {
        case name
        case sectionId = "section_id"
        case aisleTag = "aisle_tag"
    }
}

struct ReportsSummaryWire: Codable {
    let salesMinor: Int64?
    let saleCount: Int?
    let onHandSkuCount: Int?
    let lowStockCount: Int?
    let topSkus: [ReportsTopSkuWire]?

    enum CodingKeys: String, CodingKey {
        case salesMinor = "sales_minor"
        case saleCount = "sale_count"
        case onHandSkuCount = "on_hand_sku_count"
        case lowStockCount = "low_stock_count"
        case topSkus = "top_skus"
    }
}

struct ControlTowerPulseWire: Codable {
    let retailerId: String?
    let generatedAt: String?
    let openOrders: Int
    let activeFulfillments: Int
    let dockPending: Int
    let posOpenSessions: Int
    let openShifts: Int
    let openAssistTickets: Int
    let lowStockSkuBins: Int
    let shiftVariances7d: Int
    let salesMinor7d: Int64
    let capabilities: [String]?
    let empty: Bool

    enum CodingKeys: String, CodingKey {
        case empty
        case capabilities
        case retailerId = "retailer_id"
        case generatedAt = "generated_at"
        case openOrders = "open_orders"
        case activeFulfillments = "active_fulfillments"
        case dockPending = "dock_pending"
        case posOpenSessions = "pos_open_sessions"
        case openShifts = "open_shifts"
        case openAssistTickets = "open_assist_tickets"
        case lowStockSkuBins = "low_stock_sku_bins"
        case shiftVariances7d = "shift_variances_7d"
        case salesMinor7d = "sales_minor_7d"
    }
}

struct ReportsTopSkuWire: Codable {
    let sku: String?
    let salesMinor: Int64?
    let units: Int64?

    enum CodingKeys: String, CodingKey {
        case sku
        case salesMinor = "sales_minor"
        case units
    }
}

struct AssistTicketsWire: Codable {
    let items: [AssistTicketWire]
}

struct AssistTicketWire: Codable {
    let ticketId: String
    let note: String
    let status: String

    enum CodingKeys: String, CodingKey {
        case note
        case status
        case ticketId = "ticket_id"
    }
}

struct LocalSkuListWire: Codable {
    let items: [LocalSkuWire]
}

struct LocalSkuWire: Codable {
    let localSkuId: String
    let name: String
    let barcode: String?
    let defaultPriceMinor: Int64?
    let isActive: Bool?

    enum CodingKeys: String, CodingKey {
        case name
        case barcode
        case localSkuId = "local_sku_id"
        case defaultPriceMinor = "default_price_minor"
        case isActive = "is_active"
    }
}

struct StoreStockListWire: Codable {
    let items: [StoreStockBalanceWire]
}

struct StoreStockBalanceWire: Codable {
    let locationId: String?
    let stockBin: String
    let sku: String
    let onHand: Int64
    let available: Int64?

    enum CodingKeys: String, CodingKey {
        case sku
        case locationId = "location_id"
        case stockBin = "stock_bin"
        case onHand = "on_hand"
        case available
    }
}

struct AnyDecodable: Decodable {
    let value: Any

    init(from decoder: Decoder) throws {
        let container = try decoder.singleValueContainer()
        if let bool = try? container.decode(Bool.self) {
            value = bool
        } else if let int = try? container.decode(Int.self) {
            value = int
        } else if let double = try? container.decode(Double.self) {
            value = double
        } else if let string = try? container.decode(String.self) {
            value = string
        } else if let array = try? container.decode([AnyDecodable].self) {
            value = array.map(\.value)
        } else if let dict = try? container.decode([String: AnyDecodable].self) {
            value = dict.mapValues(\.value)
        } else if container.decodeNil() {
            value = NSNull()
        } else {
            throw DecodingError.dataCorruptedError(in: container, debugDescription: "Unsupported JSON value")
        }
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
