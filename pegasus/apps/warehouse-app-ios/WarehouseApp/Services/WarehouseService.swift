import Foundation

enum WarehouseService {
    private static let api = APIClient.shared

    // MARK: - Auth
    static func login(phone: String, pin: String) async throws -> AuthResponse {
        try await api.post("v1/auth/warehouse/login", body: LoginRequest(phone: phone, pin: pin))
    }

    // MARK: - Dashboard
    static func dashboard() async throws -> DashboardData {
        try await api.get("v1/warehouse/ops/dashboard")
    }

    // MARK: - Orders
    static func orders(state: String? = nil) async throws -> OrderListResponse {
        var query: [String: String] = [:]
        if let state { query["state"] = state }
        return try await api.get("v1/warehouse/ops/orders", query: query)
    }

    static func order(id: String) async throws -> Order {
        try await api.get("v1/warehouse/ops/orders/\(id)")
    }

    // MARK: - Drivers
    static func drivers() async throws -> DriverListResponse {
        try await api.get("v1/warehouse/ops/drivers")
    }

    static func createDriver(name: String, phone: String) async throws -> CreateDriverResponse {
        try await api.post("v1/warehouse/ops/drivers", body: CreateDriverRequest(name: name, phone: phone))
    }

    // MARK: - Vehicles
    static func vehicles() async throws -> VehicleListResponse {
        try await api.get("v1/warehouse/ops/vehicles")
    }

    static func createVehicle(label: String, licensePlate: String, vehicleClass: String) async throws -> Vehicle {
        try await api.post("v1/warehouse/ops/vehicles", body: CreateVehicleRequest(label: label, licensePlate: licensePlate, vehicleClass: vehicleClass))
    }

    // MARK: - Inventory
    static func inventory(lowStock: Bool = false) async throws -> InventoryListResponse {
        var query: [String: String] = [:]
        if lowStock { query["low_stock"] = "true" }
        return try await api.get("v1/warehouse/ops/inventory", query: query)
    }

    static func adjustInventory(productId: String, quantity: Int) async throws {
        try await api.patchVoid("v1/warehouse/ops/inventory", body: InventoryAdjustRequest(productId: productId, quantity: quantity))
    }

    // MARK: - Products
    static func products() async throws -> ProductListResponse {
        try await api.get("v1/warehouse/ops/products")
    }

    // MARK: - Manifests
    static func manifests() async throws -> ManifestListResponse {
        try await api.get("v1/warehouse/ops/manifests")
    }

    // MARK: - Analytics
    static func analytics(period: String = "7d") async throws -> AnalyticsData {
        try await api.get("v1/warehouse/ops/analytics", query: ["period": period])
    }

    // MARK: - CRM
    static func retailers() async throws -> RetailerListResponse {
        try await api.get("v1/warehouse/ops/crm")
    }

    // MARK: - Returns
    static func returns() async throws -> ReturnListResponse {
        try await api.get("v1/warehouse/ops/returns")
    }

    // MARK: - Treasury
    static func treasuryOverview() async throws -> TreasuryOverview {
        try await api.get("v1/warehouse/ops/treasury", query: ["view": "overview"])
    }

    static func treasuryInvoices() async throws -> InvoiceListResponse {
        try await api.get("v1/warehouse/ops/treasury", query: ["view": "invoices"])
    }

    // MARK: - Dispatch
    static func dispatchPreview() async throws -> DispatchPreview {
        try await api.get("v1/warehouse/ops/dispatch/preview")
    }

    // MARK: - Staff
    static func staff() async throws -> StaffListResponse {
        try await api.get("v1/warehouse/ops/staff")
    }

    static func createStaff(name: String, phone: String, role: String) async throws -> CreateStaffResponse {
        try await api.post("v1/warehouse/ops/staff", body: CreateStaffRequest(name: name, phone: phone, role: role))
    }

    // MARK: - Payment Config
    static func paymentConfig() async throws -> PaymentConfigResponse {
        try await api.get("v1/warehouse/ops/payment-config")
    }
}
