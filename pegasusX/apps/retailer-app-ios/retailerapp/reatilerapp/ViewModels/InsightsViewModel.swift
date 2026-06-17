import Foundation

@Observable
final class InsightsViewModel {
    var analytics: RetailerAnalytics?
    var detailed: RetailerDetailedAnalytics?
    var predictions: [DemandForecast] = []
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
        let rid = AuthManager.shared.currentUser?.id ?? ""
        do {
            predictions = try await api.get(path: "/v1/ai/predictions?retailer_id=\(rid)")
        } catch {
            predictions = []
        }
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

    func dismissPrediction(_ forecast: DemandForecast) async {
        correctingId = forecast.id
        defer { correctingId = nil }
        do {
            let body: [String: String] = ["status": "REJECTED"]
            let _: [String: String] = try await api.patch(
                path: "/v1/ai/predictions/correct?prediction_id=\(forecast.id)",
                body: body,
                headers: ["Idempotency-Key": "retailer-prediction-correct:\(forecast.id):rejected"]
            )
            Haptics.success()
            await loadAnalytics()
        } catch {
            Haptics.error()
        }
    }
}
