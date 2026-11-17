import Foundation
import Observation

@Observable
@MainActor
final class DashboardViewModel {
    typealias FetchDashboard = @MainActor () async throws -> DashboardData

    var stats = DashboardData.empty
    var loading = true
    var hasData = false
    var error: String?
    var commandJump: CommandStatusJump?

    private let fetchDashboard: FetchDashboard

    init(fetchDashboard: FetchDashboard? = nil) {
        self.fetchDashboard = fetchDashboard ?? {
            try await WarehouseService.dashboard()
        }
    }

    func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        defer { if !silent { loading = false } }
        do {
            stats = try await fetchDashboard()
            hasData = true
        } catch {
            stats = .empty
            hasData = false
            self.error = error.localizedDescription
        }
    }
}
