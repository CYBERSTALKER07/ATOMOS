import Foundation
import Observation

@Observable
@MainActor
final class AdjustInventoryViewModel {
    typealias AdjustStock = @MainActor (String, Int) async throws -> Void

    let item: InventoryItem
    var qty: String
    var reason = ""
    var showConfirm = false
    var submitting = false
    var error: String?

    private let adjustStock: AdjustStock

    init(
        item: InventoryItem,
        adjustStock: AdjustStock? = nil
    ) {
        self.item = item
        self.qty = "\(item.quantity)"
        self.adjustStock = adjustStock ?? { productId, quantity in
            try await WarehouseService.adjustInventory(productId: productId, quantity: quantity)
        }
    }

    var skuLabel: String {
        item.productId
    }

    var parsedQuantity: Int? {
        Int(qty.trimmingCharacters(in: .whitespacesAndNewlines))
    }

    var canReview: Bool {
        guard let parsed = parsedQuantity else { return false }
        return parsed >= 0 && parsed != item.quantity
    }

    func handleReconnect() {
        if submitting {
            submitting = false
            error = "Connection restored — verify status before retrying."
        }
    }

    func save() async -> Bool {
        guard let q = parsedQuantity else { return false }
        submitting = true
        error = nil
        defer { submitting = false }
        do {
            _ = reason.trimmingCharacters(in: .whitespacesAndNewlines)
            try await adjustStock(item.productId, q)
            PegasusHaptics.success()
            return true
        } catch {
            self.error = error.localizedDescription
            PegasusHaptics.error()
            return false
        }
    }
}
