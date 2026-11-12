import Foundation
import Observation

@Observable
@MainActor
final class StockCommitmentsViewModel {
    typealias FetchCommitments = @MainActor () async throws -> StockCommitmentsResponse

    var commitments: [StockCommitmentRow] = []
    var loading = true
    var error: String?

    private let fetchCommitments: FetchCommitments

    init(fetchCommitments: FetchCommitments? = nil) {
        self.fetchCommitments = fetchCommitments ?? {
            try await WarehouseService.stockCommitments()
        }
    }

    func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        defer { if !silent { loading = false } }
        do {
            let response = try await fetchCommitments()
            commitments = response.items.isEmpty ? response.skus : response.items
        } catch {
            self.error = error.localizedDescription
        }
    }
}
