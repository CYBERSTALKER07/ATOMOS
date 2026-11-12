import Foundation
import SwiftUI
import Observation

@Observable
@MainActor
final class CategoryProductsViewModel {
    var products: [Product] = []
    var isLoading = false
    var selectedProduct: Product?
    var autoOrderSettings: [String: Bool] = [:]
    var togglingIds: Set<String> = []

    private let api = APIClient.shared

    var groupedProducts: [(supplier: String, products: [Product])] {
        let grouped = Dictionary(grouping: products) { product in
            if let supplierName = product.supplierName, !supplierName.isEmpty {
                return supplierName
            }
            return String(product.name.split(separator: " ").last ?? "Other")
        }
        return grouped.sorted { $0.key < $1.key }.map { (supplier: $0.key, products: $0.value) }
    }

    func loadProducts(categoryId: String) async {
        isLoading = true
        do {
            let result: [Product] = try await api.get(path: "/v1/catalog/products?category_id=\(categoryId)")
            products = result
        } catch {
            products = []
        }
        isLoading = false
    }

    func toggleProductAutoOrder(productId: String, enabled: Bool) async {
        guard !togglingIds.contains(productId) else { return }
        Haptics.selection()
        autoOrderSettings[productId] = enabled
        togglingIds.insert(productId)
        do {
            let _: [String: Bool] = try await api.patch(
                path: "/v1/retailer/settings/auto-order/product/\(productId)",
                body: ["auto_order_enabled": enabled]
            )
            Haptics.success()
        } catch {
            Haptics.error()
            withAnimation { autoOrderSettings[productId] = !enabled }
        }
        togglingIds.remove(productId)
    }
}
