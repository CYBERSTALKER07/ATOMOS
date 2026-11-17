import Foundation
import Observation

@Observable
@MainActor
final class CRMViewModel {
    typealias FetchRetailers = @MainActor () async throws -> RetailerListResponse

    var retailers: [Retailer] = []
    var loading = true
    var error: String?

    private let fetchRetailers: FetchRetailers

    init(fetchRetailers: FetchRetailers? = nil) {
        self.fetchRetailers = fetchRetailers ?? {
            try await WarehouseService.retailers()
        }
    }

    func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let resp = try await fetchRetailers()
            retailers = resp.retailers
        } catch {
            self.error = error.localizedDescription
        }
    }
}
