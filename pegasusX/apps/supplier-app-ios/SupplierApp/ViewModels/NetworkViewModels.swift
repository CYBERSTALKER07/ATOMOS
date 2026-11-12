import Foundation
import Observation

@Observable
@MainActor
final class CRMViewModel {
    var retailers: [SupplierCRMRetailer] = []
    var loading = true
    var error: String?
    var selected: SupplierCRMRetailerDetail?
    private var loadGeneration = 0
    private var detailGeneration = 0

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = !retailers.isEmpty || (error == nil && !loading)
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            let loaded = try await SupplierOperationsService.crmRetailers()
            guard generation == loadGeneration else { return }
            retailers = loaded
            SupplierScreenLoad.succeed(error: &error)
        } catch {
            guard generation == loadGeneration else { return }
            let message: String
            if let err = error as? APIError, case .httpError(503) = err {
                message = L10n.format("mobile_supplier.chrome.crm_unavailable")
            } else {
                message = error.localizedDescription
            }
            SupplierScreenLoad.fail(
                silent: silent,
                hasContent: hadContent,
                underlyingMessage: message,
                error: &self.error,
                noun: "data"
            )
        }
        if generation == loadGeneration {
            loading = false
        }
    }

    func loadDetail(_ id: String) async {
        detailGeneration += 1
        let generation = detailGeneration
        let previous = selected
        let loaded: Result<SupplierCRMRetailerDetail, Error>
        do {
            loaded = .success(try await SupplierOperationsService.crmRetailer(id))
        } catch {
            loaded = .failure(error)
        }
        guard generation == detailGeneration else { return }
        let outcome = CRMDetailLoad.selection(previous: previous, loaded: loaded)
        selected = outcome.value
        if let failure = outcome.failure {
            SupplierActionFailure.record(
                underlying: failure,
                hasContent: !retailers.isEmpty || outcome.value != nil,
                error: &self.error
            )
        } else {
            SupplierStatusCenter.shared.notice = nil
        }
    }
}

@Observable
@MainActor
final class TopologyNodesListViewModel {
    var warehouses: [SupplierTopologyWarehouse] = []
    var factories: [SupplierTopologyFactory] = []
    var loading = true
    var error: String?
    var showAdd = false
    private var loadGeneration = 0

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = !warehouses.isEmpty || !factories.isEmpty
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            let resp = try await SupplierOperationsService.topology()
            guard generation == loadGeneration else { return }
            warehouses = resp.warehouses
            factories = resp.factories
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
final class SupplyLanesViewModel {
    var lanes: [SupplierSupplyLaneRow] = []
    var loading = true
    var error: String?
    private var loadGeneration = 0

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = !lanes.isEmpty
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            let loadedLanes = try await SupplierOperationsService.supplyLanes()
            guard generation == loadGeneration else { return }
            lanes = loadedLanes
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
final class EntityResolutionViewModel {
    static let types = ["ANY", "ORDER", "RETAILER", "WAREHOUSE", "FACTORY", "DRIVER", "VEHICLE", "SUPPLIER"]

    var entityType = "ANY"
    var query = ""
    var entityId = ""
    var busy = false
    var error: String?
    var resolved: EntityResolutionResolveResponse?
    var explain: EntityResolutionExplainResponse?

    func runResolve() async {
        busy = true
        error = nil
        defer { busy = false }
        do {
            resolved = try await SupplierOperationsService.resolveEntity(
                EntityResolutionResolveRequest(
                    entityType: entityType,
                    query: query.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ? nil : query,
                    entityId: entityId.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ? nil : entityId
                )
            )
            explain = nil
            PegasusHaptics.success()
        } catch {
            PegasusHaptics.error()
            SupplierActionFailure.record(
                underlying: error,
                hasContent: resolved != nil || explain != nil,
                error: &self.error
            )
        }
    }

    func runExplain(type: String, id: String) async {
        busy = true
        error = nil
        defer { busy = false }
        do {
            explain = try await SupplierOperationsService.explainEntity(
                EntityResolutionExplainRequest(entityType: type, entityId: id)
            )
        } catch {
            SupplierActionFailure.record(
                underlying: error,
                hasContent: explain != nil || resolved != nil,
                error: &self.error
            )
        }
    }
}

@Observable
@MainActor
final class LoyaltyProgramViewModel {
    var program: LoyaltyProgram?
    var earnBps = "100"
    var reason = ""
    var status: String?
    var error: String?
    var loading = true
    var busy = false
    private var loadGeneration = 0

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = program != nil
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            let resp = try await SupplierOperationsService.loyaltyProgram()
            guard generation == loadGeneration else { return }
            program = resp
            earnBps = String(resp.earnBps)
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

    func save() async {
        let why = reason.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !why.isEmpty else {
            status = L10n.format("mobile_supplier.chrome.typed_reason_required")
            PegasusHaptics.error()
            return
        }
        busy = true
        defer { busy = false }
        do {
            let bps = Int64(earnBps) ?? 100
            let scope = SupplierIdempotencyKeys.supplierScopeId()
            var body = try await SupplierOperationsService.loyaltyProgram()
            body.earnBps = bps > 0 ? bps : 100
            body.reason = why
            let resp = try await SupplierOperationsService.patchLoyaltyProgram(
                body,
                idempotencyKey: SupplierIdempotencyKeys.loyaltyProgramPatch(scopeId: scope, reason: why)
            )
            program = resp
            reason = ""
            status = L10n.format("mobile_supplier.chrome.saved_source", resp.source ?? L10n.format("mobile_supplier.chrome.source_program"))
            PegasusHaptics.success()
        } catch {
            PegasusHaptics.error()
            status = error.localizedDescription
            SupplierActionFailure.record(underlying: error, hasContent: true, error: &self.error)
        }
    }
}

struct TopologyWarehouseDraft: Identifiable {
    let id: String
    var warehouseId: String?
    var name: String
    var location: AddressLocationValue
    var coverageRadiusKm: String
}

struct TopologyFactoryDraft: Identifiable {
    let id: String
    var factoryId: String?
    var name: String
    var location: AddressLocationValue
    var isActive: Bool = true
}

@Observable
@MainActor
final class TopologyViewModel {
    var loading = true
    var saving = false
    var editing = false
    var error: String?
    var warehouses: [SupplierTopologyWarehouse] = []
    var factories: [SupplierTopologyFactory] = []
    var warehouseDrafts: [TopologyWarehouseDraft] = []
    var factoryDrafts: [TopologyFactoryDraft] = []
    private var loadGeneration = 0

    func beginEditing() {
        applyDraftsFromLoaded()
        editing = true
    }

    func cancelEditing() {
        applyDraftsFromLoaded()
        editing = false
        error = nil
    }

    func addWarehouseDraft() {
        PegasusHaptics.light()
        warehouseDrafts.append(
            TopologyWarehouseDraft(
                id: "new-wh-\(warehouseDrafts.count)",
                warehouseId: nil,
                name: L10n.format("mobile_supplier.chrome.default_warehouse_name", "\(warehouseDrafts.count + 1)"),
                location: AddressLocationValue(lat: TopologyMutation.defaultLat, lng: TopologyMutation.defaultLng),
                coverageRadiusKm: "50"
            )
        )
    }

    func addFactoryDraft() {
        PegasusHaptics.light()
        factoryDrafts.append(
            TopologyFactoryDraft(
                id: "new-fc-\(factoryDrafts.count)",
                factoryId: nil,
                name: L10n.format("mobile_supplier.chrome.default_factory_name", "\(factoryDrafts.count + 1)"),
                location: AddressLocationValue(lat: TopologyMutation.defaultLat, lng: TopologyMutation.defaultLng)
            )
        )
    }

    func applyDraftsFromLoaded() {
        warehouseDrafts = warehouses.enumerated().map { index, node in
            TopologyWarehouseDraft(
                id: node.warehouseId.isEmpty ? "wh-\(index)" : node.warehouseId,
                warehouseId: node.warehouseId.isEmpty ? nil : node.warehouseId,
                name: node.name,
                location: AddressLocationValue(
                    address: node.address,
                    lat: node.lat,
                    lng: node.lng,
                    placeId: node.placeId
                ),
                coverageRadiusKm: String(node.coverageRadiusKm)
            )
        }
        factoryDrafts = factories.enumerated().map { index, node in
            TopologyFactoryDraft(
                id: node.factoryId.isEmpty ? "fc-\(index)" : node.factoryId,
                factoryId: node.factoryId.isEmpty ? nil : node.factoryId,
                name: node.name,
                location: AddressLocationValue(
                    address: node.address,
                    lat: node.lat,
                    lng: node.lng,
                    placeId: node.placeId
                )
            )
        }
    }

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = !warehouses.isEmpty || !factories.isEmpty
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            let resp = try await SupplierOperationsService.topology()
            guard generation == loadGeneration else { return }
            warehouses = resp.warehouses
            factories = resp.factories
            if editing {
                applyDraftsFromLoaded()
            }
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

    func save() async {
        saving = true
        error = nil
        defer { saving = false }
        do {
            guard !warehouseDrafts.isEmpty else {
                error = L10n.format("mobile_supplier.chrome.warehouse_required")
                PegasusHaptics.error()
                return
            }
            let request = SupplierTopologyUpdateRequest(
                warehouses: try warehouseDrafts.map { draft in
                    guard !draft.location.address.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty,
                          draft.location.lat != 0 || draft.location.lng != 0 else {
                        throw URLError(.badURL)
                    }
                    let existing = warehouses.first { $0.warehouseId == draft.warehouseId }
                    return TopologyMutation.warehouseInput(
                        from: existing,
                        warehouseId: draft.warehouseId,
                        name: draft.name.trimmingCharacters(in: .whitespacesAndNewlines),
                        location: draft.location,
                        coverageRadiusKm: Double(draft.coverageRadiusKm) ?? 50,
                        isActive: true,
                        isOnShift: true,
                        transferMode: "TRUCK"
                    )
                },
                factories: try factoryDrafts.map { draft in
                    guard !draft.location.address.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty,
                          draft.location.lat != 0 || draft.location.lng != 0 else {
                        throw URLError(.badURL)
                    }
                    let existing = factories.first { $0.factoryId == draft.factoryId }
                    return TopologyMutation.factoryInput(
                        from: existing,
                        factoryId: draft.factoryId,
                        name: draft.name.trimmingCharacters(in: .whitespacesAndNewlines),
                        location: draft.location,
                        isActive: draft.isActive
                    )
                }
            )
            let resp = try await SupplierOperationsService.updateTopology(request)
            warehouses = resp.warehouses
            factories = resp.factories
            editing = false
            PegasusHaptics.success()
        } catch {
            PegasusHaptics.error()
            self.error = L10n.format("mobile_supplier.chrome.node_needs_address")
        }
    }
}
