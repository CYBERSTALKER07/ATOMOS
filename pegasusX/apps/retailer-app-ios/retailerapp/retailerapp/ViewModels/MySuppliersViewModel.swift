import Foundation
import Observation

@Observable
@MainActor
final class MySuppliersViewModel {
    var suppliers: [Supplier] = []
    var isLoading = false
    var errorMessage: String?
    var autoOrderSettings = SimpleAutoOrderSettings.default
    var showConnectSheet = false
    var supplierPendingRemoval: Supplier?
    var showRemoveConfirmation = false

    private let api = APIClient.shared

    func loadSuppliers() async {
        isLoading = true
        errorMessage = nil
        do {
            let result: [Supplier] = try await api.get(path: "/v1/retailer/suppliers")
            suppliers = result
        } catch {
            suppliers = []
            errorMessage = error.localizedDescription
        }
        isLoading = false
    }

    func promptRemoval(for supplier: Supplier) {
        Haptics.warning()
        supplierPendingRemoval = supplier
        showRemoveConfirmation = true
    }

    func cancelRemoval() {
        supplierPendingRemoval = nil
        showRemoveConfirmation = false
    }

    func removeSupplier(_ supplierId: String) async {
        do {
            try await RetailerSupplierDiscoveryService.removeSupplier(api: api, supplierId: supplierId)
            Haptics.success()
            supplierPendingRemoval = nil
            showRemoveConfirmation = false
            await loadSuppliers()
        } catch {
            Haptics.error()
            errorMessage = error.localizedDescription
            supplierPendingRemoval = nil
            showRemoveConfirmation = false
        }
    }
}
