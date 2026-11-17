import Foundation
import Observation

@Observable
@MainActor
final class InventoryViewModel {
    typealias FetchInventory = @MainActor (Bool) async throws -> InventoryListResponse
    typealias PatchPolicy = @MainActor (String, String) async throws -> Void
    typealias AdjustStock = @MainActor (String, Int) async throws -> Void

    var items: [InventoryItem] = []
    var loading = true
    var error: String?
    var lowOnly = false
    var adjustItem: InventoryItem?
    var policySavingId: String?

    let policies = ["INHERIT", "REJECT", "ACCEPT_BACKORDER"]

    private let fetchInventory: FetchInventory
    private let patchPolicy: PatchPolicy
    private let adjustStock: AdjustStock

    init(
        fetchInventory: FetchInventory? = nil,
        patchPolicy: PatchPolicy? = nil,
        adjustStock: AdjustStock? = nil
    ) {
        self.fetchInventory = fetchInventory ?? { lowStock in
            try await WarehouseService.inventory(lowStock: lowStock)
        }
        self.patchPolicy = patchPolicy ?? { productId, policy in
            try await WarehouseService.patchInventoryPolicy(productId: productId, policy: policy)
        }
        self.adjustStock = adjustStock ?? { productId, quantity in
            try await WarehouseService.adjustInventory(productId: productId, quantity: quantity)
        }
    }

    func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        do {
            let resp = try await fetchInventory(lowOnly)
            items = resp.items
        } catch {
            if !silent { self.error = error.localizedDescription }
        }
        if !silent { loading = false }
    }

    func updatePolicy(item: InventoryItem, policy: String) async {
        let current = (item.outOfStockPolicy?.isEmpty == false ? item.outOfStockPolicy : nil) ?? "INHERIT"
        guard policy != current else { return }
        policySavingId = item.productId
        defer { policySavingId = nil }
        do {
            try await patchPolicy(item.productId, policy)
            PegasusHaptics.success()
            await load(silent: true)
        } catch {
            self.error = error.localizedDescription
            PegasusHaptics.error()
        }
    }

    func adjustInventory(productId: String, quantity: Int) async throws {
        try await adjustStock(productId, quantity)
        await load(silent: true)
    }
}
