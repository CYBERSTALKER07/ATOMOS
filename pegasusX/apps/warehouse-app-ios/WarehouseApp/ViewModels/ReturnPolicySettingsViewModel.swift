import Foundation
import Observation

@Observable
@MainActor
final class ReturnPolicySettingsViewModel {
    typealias WarehouseIdProvider = @MainActor () -> String?
    typealias FetchPolicy = @MainActor (String?) async throws -> WarehouseReturnPolicy
    typealias SavePolicy = @MainActor (WarehouseReturnPolicy, String?) async throws -> WarehouseReturnPolicy

    var loading = true
    var saving = false
    var error: String?
    var saved = false
    var reverseSla = "24"
    var canOverride = false
    var retailerWindow = ""
    var supplierId = ""

    private let warehouseIdProvider: WarehouseIdProvider
    private let fetchPolicy: FetchPolicy
    private let savePolicy: SavePolicy

    init(
        warehouseIdProvider: WarehouseIdProvider? = nil,
        fetchPolicy: FetchPolicy? = nil,
        savePolicy: SavePolicy? = nil
    ) {
        self.warehouseIdProvider = warehouseIdProvider ?? {
            TokenStore.shared.warehouseId
        }
        self.fetchPolicy = fetchPolicy ?? { warehouseId in
            try await WarehouseService.getReturnPolicy(warehouseId: warehouseId)
        }
        self.savePolicy = savePolicy ?? { body, warehouseId in
            try await WarehouseService.putReturnPolicy(body, warehouseId: warehouseId)
        }
    }

    func sanitizeReverseSla(_ newValue: String) {
        let digits = newValue.filter(\.isNumber)
        if digits != newValue { reverseSla = digits }
    }

    func sanitizeRetailerWindow(_ newValue: String) {
        let digits = newValue.filter(\.isNumber)
        if digits != newValue { retailerWindow = digits }
    }

    func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let wh = warehouseIdProvider()
            let p = try await fetchPolicy(wh)
            supplierId = p.supplierId
            reverseSla = p.reverseDockSlaHours.map(String.init) ?? "24"
            canOverride = p.canOverrideRetailerWindow
            retailerWindow = p.retailerFileWindowHours.map(String.init) ?? ""
        } catch {
            self.error = error.localizedDescription
        }
    }

    func save() async {
        saving = true
        error = nil
        saved = false
        defer { saving = false }
        var body = WarehouseReturnPolicy(
            supplierId: supplierId,
            reverseDockSlaHours: Int64(reverseSla).flatMap { $0 > 0 ? $0 : nil },
            retailerFileWindowHours: nil,
            canOverrideRetailerWindow: canOverride
        )
        if canOverride {
            guard let hours = Int64(retailerWindow), hours >= 1 else {
                error = "Retailer file window hours required when override is enabled"
                PegasusHaptics.warning()
                return
            }
            body.retailerFileWindowHours = hours
        }
        do {
            let wh = warehouseIdProvider()
            _ = try await savePolicy(body, wh)
            saved = true
            PegasusHaptics.success()
        } catch {
            self.error = error.localizedDescription
            PegasusHaptics.error()
        }
    }
}
