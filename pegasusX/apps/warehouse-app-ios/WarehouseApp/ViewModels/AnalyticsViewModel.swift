import Foundation
import Observation

@Observable
@MainActor
final class AnalyticsViewModel {
    typealias FetchAnalytics = @MainActor (String) async throws -> AnalyticsData

    var data = AnalyticsData.empty
    var loading = true
    var error: String?
    var period = "30d"

    private let fetchAnalytics: FetchAnalytics

    init(fetchAnalytics: FetchAnalytics? = nil) {
        self.fetchAnalytics = fetchAnalytics ?? { period in
            try await WarehouseService.analytics(period: period)
        }
    }

    func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            data = try await fetchAnalytics(period)
        } catch {
            self.error = error.localizedDescription
        }
    }
}
