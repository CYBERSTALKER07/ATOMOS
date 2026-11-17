import Foundation
import SwiftUI
import Observation

@Observable
@MainActor
final class SupplierProductsViewModel {
    var products: [Product] = []
    var isLoading = false
    var errorMessage: String?
    var selectedProduct: Product?
    var autoOrderSettings: [String: Bool] = [:]
    var supplierAutoOrder = false
    var isMySupplier = false
    var isTogglingMySupplier = false
    var isTogglingSupplierAutoOrder = false
    var togglingProductIds: Set<String> = []

    private let api = APIClient.shared

    func groupedProducts(for supplier: Supplier) -> [(key: String, value: [Product])] {
        Dictionary(grouping: products) { product in
            product.categoryName ?? supplier.displayCategory ?? "Other"
        }
        .sorted { $0.key.localizedCaseInsensitiveCompare($1.key) == .orderedAscending }
    }

    func loadProducts(supplierId: String) async {
        isLoading = true
        errorMessage = nil
        do {
            let _: [String: String] = try await api.post(
                path: "/v1/retailer/promotions/watch",
                body: ["supplier_id": supplierId]
            )
            let result: [Product] = try await api.get(path: "/v1/catalog/products?supplier_id=\(supplierId)")
            products = result
        } catch {
            products = []
        }
        isLoading = false
    }

    func loadAutoOrderState(supplierId: String) async {
        do {
            let s: AutoOrderSettings = try await api.get(path: "/v1/retailer/settings/auto-order")
            if let sov = s.supplierOverrides.first(where: { $0.supplierID == supplierId }) {
                supplierAutoOrder = sov.enabled
            }
            for pov in s.productOverrides {
                autoOrderSettings[pov.productID] = pov.enabled
            }
            errorMessage = nil
        } catch {
            supplierAutoOrder = false
        }
    }

    func toggleMySupplier(supplierId: String) async {
        guard !isTogglingMySupplier else { return }
        Haptics.medium()
        withAnimation(AnimationConstants.bouncy) {
            isMySupplier.toggle()
        }
        isTogglingMySupplier = true
        do {
            if isMySupplier {
                try await api.favoriteSupplier(supplierId: supplierId)
            } else {
                try await api.unfavoriteSupplier(supplierId: supplierId)
            }
            Haptics.success()
            errorMessage = nil
        } catch {
            Haptics.error()
            withAnimation(AnimationConstants.express) { isMySupplier.toggle() }
            errorMessage = RetailerErrorSupport.message(
                for: error,
                restricted: "Supplier favorites access is restricted for this account.",
                offline: "Offline mode active. Reconnect and retry supplier favorites.",
                fallback: "Supplier favorites update failed. Please try again."
            )
        }
        isTogglingMySupplier = false
    }

    func toggleSupplierAutoOrder(supplierId: String) async {
        isTogglingSupplierAutoOrder = true
        do {
            let _: [String: Bool] = try await api.patch(
                path: "/v1/retailer/settings/auto-order/supplier/\(supplierId)",
                body: ["auto_order_enabled": supplierAutoOrder]
            )
            Haptics.success()
            errorMessage = nil
        } catch {
            Haptics.error()
            withAnimation { supplierAutoOrder.toggle() }
            errorMessage = RetailerErrorSupport.message(
                for: error,
                restricted: "Supplier auto-order access is restricted for this account.",
                offline: "Offline mode active. Reconnect and retry supplier auto-order update.",
                fallback: "Supplier auto-order update failed. Please try again."
            )
        }
        isTogglingSupplierAutoOrder = false
    }

    func toggleProductAutoOrder(productId: String, enabled: Bool) async {
        togglingProductIds.insert(productId)
        do {
            let _: [String: Bool] = try await api.patch(
                path: "/v1/retailer/settings/auto-order/product/\(productId)",
                body: ["auto_order_enabled": enabled]
            )
            Haptics.success()
            errorMessage = nil
        } catch {
            Haptics.error()
            withAnimation { autoOrderSettings[productId] = !enabled }
            errorMessage = RetailerErrorSupport.message(
                for: error,
                restricted: "Product auto-order access is restricted for this account.",
                offline: "Offline mode active. Reconnect and retry product auto-order update.",
                fallback: "Product auto-order update failed. Please try again."
            )
        }
        togglingProductIds.remove(productId)
    }
}
