import Foundation
import Observation

@Observable
@MainActor
final class ClaimsViewModel {
    typealias FetchClaims = @MainActor (String?, Int) async throws -> WarehouseClaimsResponse

    let statusFilters = ["OPEN", "UNDER_REVIEW", "RESOLVED", "REJECTED", ""]

    var status = "OPEN"
    var claims: [WarehouseClaim] = []
    var loading = true
    var error: String?

    private let fetchClaims: FetchClaims

    init(fetchClaims: FetchClaims? = nil) {
        self.fetchClaims = fetchClaims ?? { status, limit in
            try await WarehouseService.supplierClaims(status: status, limit: limit)
        }
    }

    func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let resp = try await fetchClaims(status.isEmpty ? nil : status, 50)
            claims = resp.claims
        } catch {
            self.error = error.localizedDescription
        }
    }
}
