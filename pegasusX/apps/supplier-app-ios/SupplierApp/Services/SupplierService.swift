import Foundation

enum SupplierService {
    static func login(phone: String, password: String) async throws -> LoginResponse {
        try await APIClient.shared.post(
            "v1/auth/supplier/login",
            body: LoginRequest(phone: phone, password: password),
            authenticated: false
        )
    }

    static func register(body: [String: Any]) async throws -> RegisterResponse {
        try await APIClient.shared.postJSON("v1/auth/supplier/register", body: body, authenticated: false)
    }

    static func setupBusiness(_ request: BusinessSetupRequest) async throws -> BusinessSetupResponse {
        try await APIClient.shared.post("v1/supplier/business/setup", body: request)
    }

    static func configure(body: [String: String]) async throws {
        try await APIClient.shared.postVoid("v1/supplier/configure", body: body)
    }

    static func updateInventory(_ request: InventoryPatchRequest) async throws {
        try await SupplierOperationsService.updateInventory(request)
    }

    static func getDemandHistory() async throws -> SupplierDemandHistoryResponse {
        try await SupplierOperationsService.demandHistory()
    }

    static func importInventoryCSV(_ csv: String, idempotencyKey: String) async throws -> SupplierInventoryImportResult {
        try await SupplierOperationsService.importInventoryCSV(csv, idempotencyKey: idempotencyKey)
    }

    static func recordChargeback(_ request: PaymentChargebackRequest, idempotencyKey: String) async throws -> PaymentChargebackResponse {
        try await SupplierOperationsService.recordChargeback(request, idempotencyKey: idempotencyKey)
    }

    static func recordChargebackReversal(_ request: PaymentChargebackReversalRequest, idempotencyKey: String) async throws -> PaymentChargebackReversalResponse {
        try await SupplierOperationsService.recordChargebackReversal(request, idempotencyKey: idempotencyKey)
    }

    static func dashboard() async throws -> SupplierDashboard {
        try await APIClient.shared.get("v1/supplier/dashboard")
    }

    static func profile() async throws -> SupplierProfile {
        try await APIClient.shared.get("v1/supplier/profile")
    }

    static func orders(
        status: String? = nil,
        filter: String? = nil,
        limit: Int? = nil,
        offset: Int? = nil
    ) async throws -> SupplierOrdersResponse {
        try await SupplierOperationsService.orders(
            status: status,
            filter: filter,
            limit: limit,
            offset: offset
        )
    }

    static func fleetDrivers() async throws -> [FleetDriver] {
        let resp: FleetDriversResponse = try await APIClient.shared.get("v1/supplier/fleet/drivers")
        return resp.items
    }

    static func fleetVehicles() async throws -> [FleetVehicle] {
        let resp: FleetVehiclesResponse = try await APIClient.shared.get("v1/supplier/fleet/vehicles")
        return resp.items
    }

    static func inventory() async throws -> [InventoryItem] {
        let resp: InventoryListResponse = try await APIClient.shared.get("v1/supplier/inventory")
        return resp.items
    }

    static func catalogProducts() async throws -> [CatalogProduct] {
        try await APIClient.shared.get("v1/catalog/products")
    }

    static func catalogCategories(supplierId: String? = nil) async throws -> [CatalogCategory] {
        var query: [String: String] = [:]
        if let supplierId, !supplierId.isEmpty {
            query["supplier_id"] = supplierId
        }
        return try await APIClient.shared.get("v1/catalog/categories", query: query)
    }

    static func catalogUploadTicket(ext: String) async throws -> CatalogUploadTicket {
        try await APIClient.shared.get("v1/catalog/products/upload-ticket", query: ["ext": ext])
    }

    static func uploadCatalogImage(data: Data, ext: String) async throws -> String {
        let ticket = try await catalogUploadTicket(ext: ext)
        if !ticket.uploadUrl.contains("placehold.co") {
            var request = URLRequest(url: URL(string: ticket.uploadUrl)!)
            request.httpMethod = "PUT"
            request.httpBody = data
            request.setValue(mimeType(for: ext), forHTTPHeaderField: "Content-Type")
            let (_, response) = try await URLSession.shared.data(for: request)
            guard let http = response as? HTTPURLResponse, (200...299).contains(http.statusCode) else {
                throw URLError(.badServerResponse)
            }
        }
        return ticket.imageUrl
    }

    static func createCatalogProduct(_ request: CatalogProductCreateRequest) async throws -> CatalogProduct {
        try await APIClient.shared.post("v1/catalog/products", body: request)
    }

    static func updateCatalogProduct(
        productId: String,
        request: CatalogProductUpdateRequest
    ) async throws -> CatalogProduct {
        try await APIClient.shared.put("v1/catalog/products/\(productId)", body: request)
    }

    private static func mimeType(for ext: String) -> String {
        switch ext.lowercased() {
        case "png": return "image/png"
        case "webp": return "image/webp"
        default: return "image/jpeg"
        }
    }

    static func promotions() async throws -> [SupplierPromotion] {
        let resp: SupplierPromotionsResponse = try await APIClient.shared.get("v1/supplier/promotions")
        return resp.promotions
    }

    static func createPromotion(_ request: SupplierPromotionUpsertRequest) async throws -> SupplierPromotion {
        try await APIClient.shared.post("v1/supplier/promotions", body: request)
    }

    static func updatePromotion(
        promotionId: String,
        _ request: SupplierPromotionUpsertRequest
    ) async throws -> SupplierPromotion {
        try await APIClient.shared.patch("v1/supplier/promotions/\(promotionId)", body: request)
    }

    static func deactivatePromotion(promotionId: String) async throws {
        try await APIClient.shared.deleteVoid("v1/supplier/promotions/\(promotionId)")
    }

    static func earnings() async throws -> SupplierEarnings {
        do {
            return try await APIClient.shared.get("v1/supplier/earnings")
        } catch {
            let ledger = try await SupplierOperationsService.paymentLedger()
            let total = ledger.items.reduce(Int64(0)) { $0 + $1.amountMinor }
            let currency = ledger.items.first?.currency ?? "UZS"
            return SupplierEarnings(
                currency: currency,
                todayMinor: 0,
                weekMinor: total,
                monthMinor: total,
                authoritative: false,
                updatedAt: nil
            )
        }
    }

    static func configureBilling(_ request: BillingSetupRequest) async throws -> BillingSetupResponse {
        try await APIClient.shared.post("v1/supplier/billing/setup", body: request)
    }

    static func updatePricingRules(_ request: SupplierPricingRuleUpdateRequest) async throws -> SupplierPricingRule {
        try await APIClient.shared.patch("v1/supplier/pricing/rules", body: request)
    }

    static func updateTopology(body: [String: String]) async throws -> SupplierTopologyResponse {
        try await APIClient.shared.put("v1/supplier/topology", body: body)
    }
}
