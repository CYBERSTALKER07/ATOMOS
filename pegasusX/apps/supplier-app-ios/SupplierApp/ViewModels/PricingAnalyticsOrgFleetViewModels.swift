import Foundation
import Observation

@Observable
@MainActor
final class PricingViewModel {
    var products: [CatalogProduct] = []
    var loading = true
    private var loadGeneration = 0
    var error: String?
    var query = ""

    var filtered: [CatalogProduct] {
        guard !query.isEmpty else { return products }
        return products.filter { $0.name.localizedCaseInsensitiveContains(query) }
    }

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = !products.isEmpty
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            let loadedProducts = try await SupplierService.catalogProducts()
            guard generation == loadGeneration else { return }
            products = loadedProducts
            guard generation == loadGeneration else { return }
            SupplierScreenLoad.succeed(error: &error)
        } catch {
            guard generation == loadGeneration else { return }
            SupplierScreenLoad.fail(
                silent: silent,
                hasContent: hadContent,
                underlying: error,
                error: &self.error,
                noun: "data"
            )
        }
        if generation == loadGeneration {
            loading = false
        }
    }
}

@Observable
@MainActor
final class ProductPricingDetailViewModel {
    var priceMajor = ""
    var saleDiscountBps = ""
    var saleEnabled = false
    var saving = false
    var error: String?
    var activeSale: SupplierPromotion?

    func bootstrap(product: CatalogProduct) async {
        priceMajor = formatMinorUnits(product.priceMinor, fractionDigits: 2, trimTrailingZeros: false)
        do {
            let promotions = try await SupplierService.promotions()
            activeSale = promotions.first {
                $0.isActive && $0.scopeType == "PRODUCT" && $0.scopeProductId == product.productId
            }
            if let activeSale {
                saleEnabled = true
                saleDiscountBps = String(activeSale.discountBps)
            }
            error = nil
        } catch let failure {
            if activeSale == nil {
                error = failure.localizedDescription
            } else {
                SupplierStatusCenter.shared.notice = failure.localizedDescription
            }
        }
    }

    func save(product: CatalogProduct, onUpdated: () -> Void) async {
        guard let priceMinor = ProductPricingDetailView.parsePriceMinor(priceMajor) else {
            error = L10n.format("mobile_supplier.chrome.enter_valid_list_price")
            PegasusHaptics.warning()
            return
        }
        saving = true
        error = nil
        defer { saving = false }
        do {
            let request = CatalogProductUpdateRequest(
                name: product.name,
                priceMinor: priceMinor,
                currency: product.currency,
                unit: product.unit,
                unitVolumeVu: product.unitVolumeVu,
                imageUrl: product.imageUrl,
                barcode: product.barcode,
                isActive: product.isActive,
                version: product.version
            )
            let catalogKey = SupplierIdempotencyKeys.catalogUpdate(
                productId: product.productId,
                priceMinor: priceMinor,
                unitVolumeVu: product.unitVolumeVu,
                barcode: product.barcode
            )
            _ = try await SupplierService.updateCatalogProduct(
                productId: product.productId,
                request: request,
                idempotencyKey: catalogKey
            )

            if saleEnabled {
                let bps = Int64(saleDiscountBps) ?? 0
                guard bps > 0 else {
                    error = L10n.format("mobile_supplier.chrome.sale_discount_positive")
                    PegasusHaptics.warning()
                    return
                }
                let scope = SupplierIdempotencyKeys.supplierScopeId()
                let promoName = "Sale · \(product.name) · \(product.productId)"
                let upsert = SupplierPromotionUpsertRequest(
                    name: promoName,
                    description: "Product sale pricing",
                    discountBps: bps,
                    scopeType: "PRODUCT",
                    retailerScope: "ALL",
                    scopeProductId: product.productId
                )
                if let activeSale {
                    _ = try await SupplierService.updatePromotion(
                        promotionId: activeSale.promotionId,
                        upsert,
                        idempotencyKey: SupplierIdempotencyKeys.promotionUpdate(
                            scopeId: scope,
                            promotionId: activeSale.promotionId,
                            payloadFingerprint: "\(promoName):\(bps):PRODUCT"
                        )
                    )
                } else {
                    _ = try await SupplierService.createPromotion(
                        upsert,
                        idempotencyKey: SupplierIdempotencyKeys.promotionCreate(
                            scopeId: scope,
                            payloadFingerprint: "\(promoName):\(bps):PRODUCT"
                        )
                    )
                }
            } else if let activeSale {
                let scope = SupplierIdempotencyKeys.supplierScopeId()
                try await SupplierService.deactivatePromotion(
                    promotionId: activeSale.promotionId,
                    idempotencyKey: SupplierIdempotencyKeys.promotionDeactivate(
                        scopeId: scope,
                        promotionId: activeSale.promotionId
                    )
                )
            }

            PegasusHaptics.success()
            onUpdated()
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: false, error: &self.error)
            PegasusHaptics.error()
        }
    }
}

@Observable
@MainActor
final class RetailerOverridesViewModel {
    var overrides: [RetailerPriceOverride] = []
    var loading = true
    private var loadGeneration = 0
    var error: String?
    var showCreate = false
    var retailerId = ""
    var productId = ""
    var price = ""
    var creating = false
    var preview: RetailerOverridePreview?
    var previewLoading = false

    var previewKey: String {
        "\(retailerId)|\(productId)|\(price)"
    }

    func refreshPreview() async {
        guard showCreate else {
            preview = nil
            previewLoading = false
            return
        }
        guard let priceMinor = Int(price), priceMinor > 0, !productId.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else {
            preview = nil
            previewLoading = false
            return
        }
        previewLoading = true
        try? await Task.sleep(for: .milliseconds(400))
        do {
            preview = try await SupplierOperationsService.previewRetailerPriceOverride(
                RetailerOverridePreviewRequest(
                    retailerId: retailerId.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ? nil : retailerId,
                    productId: productId.trimmingCharacters(in: .whitespacesAndNewlines),
                    skuId: nil,
                    proposedPrice: Int64(priceMinor)
                )
            )
        } catch {
            preview = nil
        }
        previewLoading = false
    }

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = !overrides.isEmpty || (error == nil && !loading)
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            let resp = try await SupplierOperationsService.listRetailerPriceOverrides()
            guard generation == loadGeneration else { return }
            overrides = resp.overrides
            SupplierScreenLoad.succeed(error: &error)
        } catch {
            guard generation == loadGeneration else { return }
            SupplierScreenLoad.fail(
                silent: silent,
                hasContent: hadContent,
                underlying: error,
                error: &self.error,
                noun: "data"
            )
        }
        if generation == loadGeneration {
            loading = false
        }
    }

    func createOverride() async {
        guard let priceMinor = Int(price) else { return }
        creating = true
        defer { creating = false }
        do {
            _ = try await SupplierOperationsService.createRetailerPriceOverride(
                CreateRetailerPriceOverrideRequest(
                    retailerId: retailerId,
                    productId: productId,
                    price: priceMinor,
                    notes: nil,
                    expiresAt: nil
                )
            )
            showCreate = false
            retailerId = ""
            productId = ""
            price = ""
            preview = nil
            PegasusHaptics.success()
            await load(silent: true)
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: !overrides.isEmpty, error: &self.error)
            PegasusHaptics.error()
        }
    }

    func deleteOverride(_ id: String) async {
        do {
            try await SupplierOperationsService.deleteRetailerPriceOverride(overrideId: id)
            PegasusHaptics.warning()
            await load(silent: true)
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: !overrides.isEmpty, error: &self.error)
            PegasusHaptics.error()
        }
    }
}

@Observable
@MainActor
final class AnalyticsViewModel {
    var loading = true
    private var loadGeneration = 0
    var error: String?
    var hasSnapshot = false
    var pendingOrders: Int?
    var inventorySKUs: Int?
    var revenueLabel = "—"
    var predictionCount = 0
    var forecastUnits = 0
    var velocityCreated = 0
    var demandGeneratedAt: String?
    var demandConfidence: ForecastConfidence?

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = hasSnapshot || (error == nil && !loading)
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            async let dash = SupplierService.dashboard()
            async let revenue = SupplierOperationsService.analyticsRevenue()
            async let demand = SupplierOperationsService.demandToday()
            async let velocity = SupplierOperationsService.analyticsVelocity()

            let dashValue = try await dash
            let revenueValue = try await revenue
            let demandValue = try await demand
            let velocityValue = try await velocity
            guard generation == loadGeneration else { return }

            pendingOrders = dashValue.pendingOrders
            inventorySKUs = dashValue.inventorySKUs
            revenueLabel = MoneyFormat.minor(revenueValue.totalMinor, currency: revenueValue.currency)
            predictionCount = demandValue.predictionCount
            forecastUnits = demandValue.totalPallets
            demandGeneratedAt = demandValue.generatedAt
            demandConfidence = ForecastConfidenceSupport.fromDemand(demandValue)
            velocityCreated = velocityValue.points.reduce(0) { $0 + $1.ordersCreated }
            hasSnapshot = true
            SupplierScreenLoad.succeed(error: &error)
        } catch {
            guard generation == loadGeneration else { return }
            SupplierScreenLoad.fail(
                silent: silent,
                hasContent: hadContent,
                underlying: error,
                error: &self.error,
                noun: "data"
            )
        }
        if generation == loadGeneration {
            loading = false
        }
    }
}

@Observable
@MainActor
final class PlanningBrainSectionViewModel {
    var sandop: PlanningSAndOPSnapshot?
    var scenario: PlanningScenarioResult?
    var downtimeHours = 8.0
    var demandDeltaPct = 10.0
    var loading = true
    private var loadGeneration = 0
    var running = false
    var error: String?

    func loadSandop(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = sandop != nil || (error == nil && !loading)
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            let loaded = try await SupplierOperationsService.planningSAndOP()
            guard generation == loadGeneration else { return }
            sandop = loaded
            SupplierScreenLoad.succeed(error: &error)
        } catch {
            guard generation == loadGeneration else { return }
            SupplierScreenLoad.fail(
                silent: silent,
                hasContent: hadContent,
                underlying: error,
                error: &self.error,
                noun: "data"
            )
        }
        if generation == loadGeneration {
            loading = false
        }
    }

    func runScenario() async {
        running = true
        error = nil
        defer { running = false }
        do {
            let scope = SupplierIdempotencyKeys.supplierScopeId()
            let key = SupplierIdempotencyKeys.planningScenario(
                scopeId: scope,
                factoryDowntimeHours: Int(downtimeHours),
                demandDeltaPct: demandDeltaPct
            )
            scenario = try await SupplierOperationsService.runPlanningScenario(
                factoryDowntimeHours: Int(downtimeHours),
                demandDeltaPct: demandDeltaPct,
                idempotencyKey: key
            )
            PegasusHaptics.success()
        } catch {
            SupplierActionFailure.record(
                underlying: error,
                hasContent: sandop != nil || scenario != nil,
                error: &self.error
            )
            PegasusHaptics.error()
        }
    }
}

@Observable
@MainActor
final class DemandHistoryViewModel {
    var history: SupplierDemandHistoryResponse?
    var demandConfidence: ForecastConfidence?
    var demandGeneratedAt: String?
    var loading = true
    private var loadGeneration = 0
    var error: String?

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = history != nil || (error == nil && !loading)
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            async let historyValue = SupplierService.getDemandHistory()
            async let demandValue = SupplierOperationsService.demandToday()
            let loadedHistory = try await historyValue
            let demand = try await demandValue
            guard generation == loadGeneration else { return }
            history = loadedHistory
            demandGeneratedAt = demand.generatedAt
            demandConfidence = ForecastConfidenceSupport.fromDemand(demand)
            SupplierScreenLoad.succeed(error: &error)
        } catch {
            guard generation == loadGeneration else { return }
            SupplierScreenLoad.fail(
                silent: silent,
                hasContent: hadContent,
                underlying: error,
                error: &self.error,
                noun: "data"
            )
        }
        if generation == loadGeneration {
            loading = false
        }
    }
}

@Observable
@MainActor
final class CreateDriverViewModel {
    var name = ""
    var phone = ""
    var pin = ""
    var nodeType = "WAREHOUSE"
    var nodeId = ""
    var vehicleId = ""
    var busy = false
    var error: String?

    func handleReconnect() {
        if busy {
            busy = false
            error = L10n.format("mobile_supplier.chrome.connection_restored_status")
        }
    }

    func resetNodeSelection() {
        nodeId = ""
        vehicleId = ""
    }

    func nodeOptions(topology: SupplierTopologyResponse) -> [(String, String)] {
        if nodeType == "FACTORY" {
            return topology.factories.map { ($0.factoryId, $0.name) }
        }
        return topology.warehouses.map { ($0.warehouseId, $0.name) }
    }

    func filteredVehicles(from vehicles: [FleetVehicle]) -> [FleetVehicle] {
        vehicles.filter { $0.homeNodeType == nodeType && $0.homeNodeId == nodeId }
    }

    func create(onSuccess: () -> Void) async {
        busy = true
        error = nil
        defer { busy = false }
        do {
            let request = FleetDriverCreateRequest(
                name: name.trimmingCharacters(in: .whitespaces),
                phone: phone.trimmingCharacters(in: .whitespaces),
                pin: pin,
                homeNodeType: nodeType,
                homeNodeId: nodeId,
                vehicleId: vehicleId.isEmpty ? nil : vehicleId,
                isActive: nil
            )
            _ = try await SupplierOperationsService.createFleetDriver(
                request,
                idempotencyKey: SupplierIdempotencyKeys.fleetDriverCreate(scopeId: SupplierIdempotencyKeys.supplierScopeId(), phone: request.phone)
            )
            PegasusHaptics.success()
            onSuccess()
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: false, error: &self.error)
            PegasusHaptics.error()
        }
    }
}

@Observable
@MainActor
final class CreateOrgMemberViewModel {
    var name = ""
    var email = ""
    var phone = ""
    var password = ""
    var role = "WAREHOUSE_ADMIN"
    var nodeType = "WAREHOUSE"
    var nodeId = ""
    var busy = false
    var error: String?

    func handleReconnect() {
        if busy {
            busy = false
            error = L10n.format("mobile_supplier.chrome.connection_restored_status")
        }
    }

    func nodeOptions(topology: SupplierTopologyResponse) -> [(String, String)] {
        switch role {
        case "FACTORY_ADMIN":
            return topology.factories.map { ($0.factoryId, $0.name) }
        case "ADMIN":
            return []
        default:
            if nodeType == "FACTORY" {
                return topology.factories.map { ($0.factoryId, $0.name) }
            }
            return topology.warehouses.map { ($0.warehouseId, $0.name) }
        }
    }

    func create(onSuccess: () -> Void) async {
        busy = true
        error = nil
        defer { busy = false }
        let warehouseId: String? = {
            if role == "WAREHOUSE_ADMIN" { return nodeId }
            if role == "PAYLOAD" && nodeType == "WAREHOUSE" { return nodeId }
            return nil
        }()
        let factoryId: String? = {
            if role == "FACTORY_ADMIN" { return nodeId }
            if role == "PAYLOAD" && nodeType == "FACTORY" { return nodeId }
            return nil
        }()
        do {
            let request = SupplierOrgMemberCreateRequest(
                name: name.trimmingCharacters(in: .whitespaces),
                email: email.isEmpty ? nil : email,
                phone: phone.trimmingCharacters(in: .whitespaces),
                password: password,
                supplierRole: role,
                assignedWarehouseId: warehouseId,
                assignedFactoryId: factoryId
            )
            _ = try await SupplierOperationsService.createOrgMember(
                request,
                idempotencyKey: SupplierIdempotencyKeys.orgMemberCreate(scopeId: SupplierIdempotencyKeys.supplierScopeId(), phone: request.phone)
            )
            PegasusHaptics.success()
            onSuccess()
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: false, error: &self.error)
            PegasusHaptics.error()
        }
    }
}

@Observable
@MainActor
final class CreateVehicleViewModel {
    var label = ""
    var plate = ""
    var nodeType = "WAREHOUSE"
    var nodeId = ""
    var busy = false
    var error: String?

    func handleReconnect() {
        if busy {
            busy = false
            error = L10n.format("mobile_supplier.chrome.connection_restored_status")
        }
    }

    func nodeOptions(topology: SupplierTopologyResponse) -> [(String, String)] {
        if nodeType == "FACTORY" {
            return topology.factories.map { ($0.factoryId, $0.name) }
        }
        return topology.warehouses.map { ($0.warehouseId, $0.name) }
    }

    func create(onSuccess: () -> Void) async {
        busy = true
        error = nil
        defer { busy = false }
        do {
            let request = FleetVehicleCreateRequest(
                label: label.isEmpty ? nil : label,
                licensePlate: plate.uppercased(),
                homeNodeType: nodeType,
                homeNodeId: nodeId,
                isActive: nil
            )
            _ = try await SupplierOperationsService.createFleetVehicle(
                request,
                idempotencyKey: SupplierIdempotencyKeys.fleetVehicleCreate(scopeId: SupplierIdempotencyKeys.supplierScopeId(), licensePlate: request.licensePlate)
            )
            PegasusHaptics.success()
            onSuccess()
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: false, error: &self.error)
            PegasusHaptics.error()
        }
    }
}

@Observable
@MainActor
final class EditOrgMemberViewModel {
    let member: SupplierOrgMember
    var name: String
    var role: String
    var isActive: Bool
    var busy = false
    var error: String?

    init(member: SupplierOrgMember) {
        self.member = member
        self.name = member.name
        self.role = member.supplierRole
        self.isActive = member.isActive
    }

    func save(onSuccess: () -> Void) async {
        busy = true
        error = nil
        defer { busy = false }
        do {
            let revision = "\(name):\(role):\(isActive)"
            _ = try await SupplierOperationsService.updateOrgMember(
                member.userId,
                request: SupplierOrgMemberUpdateRequest(
                    name: name.trimmingCharacters(in: .whitespaces),
                    supplierRole: role,
                    assignedWarehouseId: member.assignedWarehouseId,
                    assignedFactoryId: member.assignedFactoryId,
                    isActive: isActive
                ),
                idempotencyKey: SupplierIdempotencyKeys.orgMemberUpdate(scopeId: SupplierIdempotencyKeys.supplierScopeId(), userId: member.userId, revision: revision)
            )
            PegasusHaptics.success()
            onSuccess()
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: false, error: &self.error)
            PegasusHaptics.error()
        }
    }
}

@Observable
@MainActor
final class OrgFleetViewModel {
    var tab = 0
    var loading = true
    private var loadGeneration = 0
    var error: String?
    var topology: SupplierTopologyResponse?
    var drivers: [FleetDriver] = []
    var vehicles: [FleetVehicle] = []
    var orgMembers: [SupplierOrgMember] = []
    var showDriverSheet = false
    var showVehicleSheet = false
    var showOrgSheet = false
    var showEditMemberSheet = false
    var editingMember: SupplierOrgMember?
    var memberActionId: String?

    func openCreateForCurrentTab() {
        PegasusHaptics.light()
        switch tab {
        case 0: showDriverSheet = true
        case 1: showVehicleSheet = true
        default: showOrgSheet = true
        }
    }

    func reload(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = topology != nil || !drivers.isEmpty || !vehicles.isEmpty || !orgMembers.isEmpty
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            async let topo = SupplierOperationsService.topology()
            async let driverResp = SupplierService.fleetDrivers()
            async let vehicleResp = SupplierService.fleetVehicles()
            async let orgResp = SupplierOperationsService.orgMembers()
            let loadedTopology = try await topo
            let loadedDrivers = try await driverResp
            let loadedVehicles = try await vehicleResp
            let loadedMembers = try await orgResp
            guard generation == loadGeneration else { return }
            topology = loadedTopology
            drivers = loadedDrivers
            vehicles = loadedVehicles
            orgMembers = loadedMembers
            SupplierScreenLoad.succeed(error: &error)
        } catch {
            guard generation == loadGeneration else { return }
            SupplierScreenLoad.fail(
                silent: silent,
                hasContent: hadContent,
                underlying: error,
                error: &self.error,
                noun: "data"
            )
        }
        if generation == loadGeneration {
            loading = false
        }
    }

    func deactivateMember(_ userId: String) async {
        memberActionId = userId
        defer { memberActionId = nil }
        do {
            orgMembers = try await SupplierOperationsService.deactivateOrgMember(
                userId,
                idempotencyKey: SupplierIdempotencyKeys.orgMemberDeactivate(scopeId: SupplierIdempotencyKeys.supplierScopeId(), userId: userId)
            )
            PegasusHaptics.warning()
        } catch {
            SupplierActionFailure.record(
                underlying: error,
                hasContent: topology != nil || !drivers.isEmpty || !vehicles.isEmpty || !orgMembers.isEmpty,
                error: &self.error
            )
            PegasusHaptics.error()
        }
    }
}
