import Foundation
import SwiftUI
import Observation

@Observable
@MainActor
final class CategorySuppliersViewModel {
    var suppliers: [Supplier] = []
    var isLoading = false
    var errorMessage: String?
    var mySupplierIds: Set<String> = []
    var togglingIds: Set<String> = []

    private let api = APIClient.shared

    func loadSuppliers(categoryId: String) async {
        isLoading = true
        errorMessage = nil
        do {
            let result: [Supplier] = try await api.get(path: "/v1/catalog/categories/\(categoryId)/suppliers")
            suppliers = result
        } catch {
            suppliers = []
            errorMessage = error.localizedDescription
        }
        isLoading = false
    }

    func toggleMySupplier(_ supplier: Supplier) async {
        guard !togglingIds.contains(supplier.id) else { return }
        Haptics.medium()
        withAnimation(AnimationConstants.bouncy) {
            if mySupplierIds.contains(supplier.id) {
                mySupplierIds.remove(supplier.id)
            } else {
                mySupplierIds.insert(supplier.id)
            }
        }
        togglingIds.insert(supplier.id)
        let wasAdded = mySupplierIds.contains(supplier.id)
        do {
            let _: [String: Bool] = try await api.post(
                path: "/v1/retailer/suppliers/\(supplier.id)/\(wasAdded ? "add" : "remove")",
                body: ["supplier_id": supplier.id]
            )
            Haptics.success()
            errorMessage = nil
        } catch {
            Haptics.error()
            withAnimation(AnimationConstants.express) {
                if wasAdded {
                    mySupplierIds.remove(supplier.id)
                } else {
                    mySupplierIds.insert(supplier.id)
                }
            }
            errorMessage = RetailerErrorSupport.message(
                for: error,
                restricted: "Category supplier favorites access is restricted for this account.",
                offline: "Offline mode active. Reconnect and retry category supplier favorites.",
                fallback: "Category supplier favorites update failed. Please try again."
            )
        }
        togglingIds.remove(supplier.id)
    }
}
