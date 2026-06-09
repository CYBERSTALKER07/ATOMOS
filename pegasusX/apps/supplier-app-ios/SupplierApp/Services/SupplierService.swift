import Foundation

enum SupplierService {
    static func login(phone: String, password: String) async throws -> LoginResponse {
        try await APIClient.shared.post(
            "v1/auth/supplier/login",
            body: LoginRequest(phone: phone, password: password),
            authenticated: false
        )
    }

    static func register(body: [String: String]) async throws {
        try await APIClient.shared.postVoid("v1/auth/supplier/register", body: body)
    }

    static func configure(body: [String: String]) async throws {
        try await APIClient.shared.postVoid("v1/supplier/configure", body: body)
    }

    static func setupBusiness(body: [String: String]) async throws {
        try await APIClient.shared.postVoid("v1/supplier/business/setup", body: body)
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

    static func updatePricingRules(body: [String: String]) async throws -> SupplierPricingRule {
        try await APIClient.shared.patch("v1/supplier/pricing/rules", body: body)
    }

    static func updateTopology(body: [String: String]) async throws -> SupplierTopologyResponse {
        try await APIClient.shared.put("v1/supplier/topology", body: body)
    }

    static func orgMembers() async throws -> [String: String] { // placeholder return type
        try await APIClient.shared.get("v1/supplier/org/members")
    }

    static func createOrgMember(body: [String: String]) async throws {
        try await APIClient.shared.postVoid("v1/supplier/org/members", body: body)
    }
}
