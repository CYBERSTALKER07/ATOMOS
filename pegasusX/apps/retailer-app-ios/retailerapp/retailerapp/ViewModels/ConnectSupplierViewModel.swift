import Foundation
import Observation

@Observable
@MainActor
final class ConnectSupplierViewModel {
    var query = ""
    var results: [Supplier] = []
    var isSearching = false
    var connectingSupplierId: String?
    var statusMessage: String?

    private let api = APIClient.shared

    var hasMinQueryLength: Bool {
        query.trimmingCharacters(in: .whitespacesAndNewlines).count >= 2
    }

    func search(_ raw: String, existingSuppliers: [Supplier]) async {
        let trimmed = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        guard trimmed.count >= 2 else {
            results = []
            return
        }
        isSearching = true
        statusMessage = nil
        defer { isSearching = false }
        do {
            let found = try await RetailerSupplierDiscoveryService.searchSuppliers(api: api, query: trimmed)
            results = RetailerSupplierDiscoveryService.filterNewCandidates(found, existing: existingSuppliers)
        } catch {
            statusMessage = error.localizedDescription
            results = []
        }
    }

    func connect(_ supplierId: String, onConnected: () async -> Void) async -> Bool {
        statusMessage = nil
        connectingSupplierId = supplierId
        defer { connectingSupplierId = nil }
        do {
            try await RetailerSupplierDiscoveryService.addSupplier(api: api, supplierId: supplierId)
            Haptics.success()
            await onConnected()
            return true
        } catch {
            Haptics.error()
            statusMessage = error.localizedDescription
            return false
        }
    }
}
