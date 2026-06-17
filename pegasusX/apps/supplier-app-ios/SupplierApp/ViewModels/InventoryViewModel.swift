import Foundation

@Observable
@MainActor
final class InventoryViewModel {
    var items: [InventoryItem] = []
    var loading = true
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
        if !silent { loading = true }
        error = nil
        defer { loading = false }
        do {
            items = try await SupplierService.inventory()
        } catch {
            if !silent { self.error = error.localizedDescription }
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
            self.error = error.localizedDescription
        }
    }
}
