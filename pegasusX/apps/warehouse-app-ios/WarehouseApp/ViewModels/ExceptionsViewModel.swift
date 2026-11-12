import Foundation
import Observation

@Observable
@MainActor
final class ExceptionsViewModel {
    typealias FetchExceptions = @MainActor () async throws -> WarehouseOpsExceptionsResponse

    var rows: [WarehouseOpsException] = []
    var loading = true
    var error: String?

    private let fetchExceptions: FetchExceptions

    init(fetchExceptions: FetchExceptions? = nil) {
        self.fetchExceptions = fetchExceptions ?? {
            try await WarehouseService.opsExceptions()
        }
    }

    func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let resp = try await fetchExceptions()
            rows = resp.exceptions
        } catch {
            self.error = error.localizedDescription
        }
    }
}
