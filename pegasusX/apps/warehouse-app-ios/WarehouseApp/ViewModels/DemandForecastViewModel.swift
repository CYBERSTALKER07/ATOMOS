import Foundation
import Observation

@Observable
@MainActor
final class DemandForecastViewModel {
    typealias FetchForecast = @MainActor (Int) async throws -> DemandForecastResponse
    typealias FetchInsights = @MainActor () async throws -> ReplenishmentInsightsResponse

    var horizon = 7
    var forecast = DemandForecastResponse()
    var loading = true
    var error: String?
    var selectedSegment = 0

    var criticalCount: Int {
        forecast.products.filter { $0.priority.uppercased() == "CRITICAL" }.count
    }

    var urgentCount: Int {
        forecast.products.filter { $0.priority.uppercased() == "URGENT" }.count
    }

    var normalCount: Int {
        forecast.products.filter { $0.priority.uppercased() == "NORMAL" }.count
    }

    private let fetchForecast: FetchForecast
    private let fetchInsights: FetchInsights

    init(
        fetchForecast: FetchForecast? = nil,
        fetchInsights: FetchInsights? = nil
    ) {
        self.fetchForecast = fetchForecast ?? { days in
            try await WarehouseService.demandForecast(days: days)
        }
        self.fetchInsights = fetchInsights ?? {
            try await WarehouseService.replenishmentInsights()
        }
    }

    func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            var body = try await fetchForecast(horizon)
            if body.products.isEmpty {
                let insights = try await fetchInsights()
                let rows = insights.rows
                if !rows.isEmpty {
                    body = Self.demandForecastFromInsights(rows, horizon: horizon)
                }
            }
            forecast = body
        } catch {
            self.error = error.localizedDescription
        }
    }

    static func demandForecastFromInsights(_ insights: [ReplenishmentInsight], horizon: Int) -> DemandForecastResponse {
        let products = insights.map { insight in
            let urgency = insight.urgency.uppercased()
            let priority: String
            if urgency == "CRITICAL" || urgency == "HIGH" {
                priority = "CRITICAL"
            } else if urgency == "URGENT" || urgency == "MEDIUM" {
                priority = "URGENT"
            } else {
                priority = "NORMAL"
            }
            return DemandForecastProduct(
                productId: insight.productId,
                productName: insight.productName,
                currentStock: insight.currentStock,
                recommendedQty: insight.reorderQuantity,
                daysUntilStockout: Double(insight.daysUntilStockout),
                priority: priority,
                unit: "VU",
                sources: DemandForecastSources(burnRate: insight.avgDailyVelocity),
                demandBreakdown: insight.demandBreakdown
            )
        }
        return DemandForecastResponse(
            warehouseId: insights.first?.warehouseId ?? "",
            forecastDays: horizon,
            generatedAt: insights.first?.createdAt,
            products: products
        )
    }
}
