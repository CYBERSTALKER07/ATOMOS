import Foundation

@Observable
@MainActor
final class InventoryViewModel {
    var items: [InventoryItem] = []
    var loading = true
    private var loadGeneration = 0
    var error: String?
    var query = ""
    var adjustingSku: String?

    var filtered: [InventoryItem] {
        guard !query.isEmpty else { return items }
        return items.filter {
            $0.sku.localizedCaseInsensitiveContains(query)
                || $0.productName.localizedCaseInsensitiveContains(query)
        }
    }

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = !items.isEmpty
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            let loadedItems = try await SupplierService.inventory()
            guard generation == loadGeneration else { return }
            items = loadedItems
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


    func adjustQuantity(sku: String, delta: Int64) async {
        adjustingSku = sku
        defer { adjustingSku = nil }
        do {
            try await SupplierService.updateInventory(
                InventoryPatchRequest(
                    skuId: sku,
                    sku: sku,
                    quantityDelta: delta,
                    quantity: nil,
                    reason: "native_adjust"
                )
            )
            await load(silent: true)
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: !items.isEmpty, error: &self.error)
        }
    }
}
