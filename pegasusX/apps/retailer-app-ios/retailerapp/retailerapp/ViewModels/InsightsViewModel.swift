import Foundation

@Observable
final class InsightsViewModel {
    var analytics: RetailerAnalytics?
    var detailed: RetailerDetailedAnalytics?
    var predictions: [RetailerAIPrediction] = []
    var isLoading = false
    var selectedRange: DateRange = .month
    var correctingId: String?

    private let api = APIClient.shared

    var delta: Int {
        guard let a = analytics else { return 0 }
        guard a.totalLastMonth > 0 else { return 0 }
        return Int(Double(a.totalThisMonth - a.totalLastMonth) / Double(a.totalLastMonth) * 100)
    }

    func loadAnalytics() async {
        isLoading = true
        defer { isLoading = false }
        do {
            analytics = try await api.get(path: "/v1/retailer/analytics/expenses")
        } catch {
            analytics = nil
        }
        predictions = []
    }

    func loadDetailedAnalytics() async {
        let fmt = ISO8601DateFormatter()
        fmt.formatOptions = [.withFullDate]
        let to = Date()
        let from = Calendar.current.date(byAdding: .day, value: -selectedRange.days, to: to) ?? to
        let fromStr = fmt.string(from: from)
        let toStr = fmt.string(from: to)
        do {
            detailed = try await api.get(path: "/v1/retailer/analytics/detailed?from=\(fromStr)&to=\(toStr)")
        } catch {
            detailed = nil
        }
    }
}
