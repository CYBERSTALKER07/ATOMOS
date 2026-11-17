import Foundation
import Observation

enum DispatchMode: String, CaseIterable {
    case manual = "Manual truck"
    case auto = "Smart assign"
}

@Observable
@MainActor
final class DispatchPreviewViewModel {
    static let tetrisBuffer = 0.95

    var preview: SupplierDispatchPreview?
    var warehouses: [SupplierTopologyWarehouse] = []
    var selectedWarehouseId: String?
    var loading = true
    var executing = false
    var error: String?
    var executeMessage: String?
    var showAutoConfirm = false
    var showCapacityOverride = false
    var dispatchMode: DispatchMode = .manual
    var selectedDriverId = ""
    var selectedOrderIds = Set<String>()
    private var loadGeneration = 0

    func toggleWarehouse(_ warehouseId: String) async {
        PegasusHaptics.selection()
        selectedWarehouseId = selectedWarehouseId == warehouseId ? nil : warehouseId
        await load()
    }

    func toggleOrderSelection(_ orderId: String) {
        PegasusHaptics.selection()
        if selectedOrderIds.contains(orderId) {
            selectedOrderIds.remove(orderId)
        } else {
            selectedOrderIds.insert(orderId)
        }
    }

    func selectAllUndispatched() {
        guard let preview else { return }
        PegasusHaptics.selection()
        selectedOrderIds = Set(preview.undispatchedOrders.map(\.orderId))
    }

    func handleReconnect() {
        if executing {
            executing = false
            error = L10n.format("mobile_supplier.chrome.connection_restored_dispatch")
        }
    }

    func pruneSelectedOrders(validIds: [String]?) {
        guard let validIds else { return }
        selectedOrderIds = selectedOrderIds.intersection(Set(validIds))
    }

    func syncDriverSelection() {
        guard let preview else { return }
        let drivers = preview.availableDrivers
        if selectedDriverId.isEmpty, let first = drivers.first {
            selectedDriverId = first.driverId
        } else if !drivers.contains(where: { $0.driverId == selectedDriverId }) {
            selectedDriverId = drivers.first?.driverId ?? ""
        }
    }

    func executeAuto() async {
        executing = true
        defer { executing = false }
        do {
            let routeFingerprint: String
            if let preview {
                routeFingerprint = """
                {"pending":\(preview.pendingCount.map(String.init) ?? "null"),"drivers":\(preview.availableDriverCount.map(String.init) ?? "null"),"undispatched":\(preview.undispatchedOrders.count)}
                """
            } else {
                routeFingerprint = "[]"
            }
            let supplierId = TokenStore.shared.supplierId ?? "supplier"
            let warehouseId = selectedWarehouseId ?? "default"
            let key = SupplierIdempotency.dispatch(
                supplierId: supplierId,
                warehouseId: warehouseId,
                mode: "AUTO",
                routeFingerprint: routeFingerprint
            )
            _ = try await SupplierOperationsService.executeDispatch(
                warehouseId: selectedWarehouseId,
                idempotencyKey: key,
                mode: "AUTO"
            )
            PegasusHaptics.success()
            executeMessage = L10n.format("mobile_supplier.chrome.auto_dispatch_executed")
            await load()
        } catch {
            PegasusHaptics.error()
            SupplierActionFailure.record(underlying: error, hasContent: preview != nil, error: &self.error)
        }
    }

    func executeManual(forceCapacity: Bool) async {
        let orderIds = selectedOrderIds.sorted()
        guard !selectedDriverId.isEmpty, !orderIds.isEmpty else { return }
        executing = true
        defer { executing = false }
        do {
            let routeFingerprint = """
            {"mode":"MANUAL","force_capacity":\(forceCapacity),"routes":[{"driver_id":"\(selectedDriverId)","order_ids":\(orderIds)}]}
            """
            let supplierId = TokenStore.shared.supplierId ?? "supplier"
            let warehouseId = selectedWarehouseId ?? "default"
            let key = SupplierIdempotency.dispatch(
                supplierId: supplierId,
                warehouseId: warehouseId,
                mode: "MANUAL",
                routeFingerprint: routeFingerprint
            )
            let result = try await SupplierOperationsService.executeDispatch(
                warehouseId: selectedWarehouseId,
                idempotencyKey: key,
                mode: "MANUAL",
                forceCapacity: forceCapacity,
                routes: [SupplierDispatchManualRoutePayload(driverId: selectedDriverId, orderIds: orderIds)]
            )
            if result.status == "capacity_exceeded" {
                PegasusHaptics.warning()
                executeMessage = L10n.format("mobile_supplier.chrome.truck_capacity_exceeded")
                showCapacityOverride = true
            } else {
                PegasusHaptics.success()
                executeMessage = L10n.format("mobile_supplier.chrome.manual_dispatch_committed")
                selectedOrderIds = []
                showCapacityOverride = false
                await load()
            }
        } catch {
            PegasusHaptics.error()
            SupplierActionFailure.record(underlying: error, hasContent: preview != nil, error: &self.error)
        }
    }

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = !warehouses.isEmpty
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            let topology = try await SupplierOperationsService.topology()
            guard generation == loadGeneration else { return }
            warehouses = topology.warehouses
            preview = try await SupplierOperationsService.dispatchPreview(warehouseId: selectedWarehouseId)
            guard generation == loadGeneration else { return }
            syncDriverSelection()
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
