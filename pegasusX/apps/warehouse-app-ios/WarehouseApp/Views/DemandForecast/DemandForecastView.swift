import SwiftUI

struct DemandForecastView: View {
    @State private var horizon = 7
    @State private var forecast = DemandForecastResponse()
    @State private var loading = true
    @State private var error: String?
    @State private var selectedSegment = 0



    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                Picker("View", selection: $selectedSegment) {
                    Text("portal.nav.products").tag(0)
                    Text("mobile_warehouse.ui.series").tag(1)
                }
                .pickerStyle(.segmented)
                .padding()

                forecastBody
            }
            .navigationTitle("portal.nav.demand_forecast")
            .toolbar { forecastToolbar }
            .onChange(of: horizon) { _, _ in load() }
            .task { load() }
            .refreshable { load() }
        }
    }

    @ToolbarContentBuilder
    private var forecastToolbar: some ToolbarContent {
        ToolbarItem(placement: .topBarTrailing) {
            Picker("Days", selection: $horizon) {
                Text("mobile_warehouse.ui.7d").tag(7)
                Text("mobile_warehouse.ui.14d").tag(14)
                Text("mobile_warehouse.ui.30d").tag(30)
            }
            .pickerStyle(.menu)
        }
        ToolbarItem(placement: .topBarTrailing) {
            Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") { load() }
        }
    }

    @ViewBuilder
    private var forecastBody: some View {
        if loading {
            ProgressView()
                .frame(maxWidth: .infinity, maxHeight: .infinity)
        } else if let error {
            ContentUnavailableView(
                "Forecast unavailable",
                systemImage: "exclamationmark.triangle",
                description: Text(error)
            )
        } else if selectedSegment == 0 {
            productsSegment
        } else {
            seriesSegment
        }
    }

    @ViewBuilder
    private var productsSegment: some View {
        if forecast.products.isEmpty {
            ContentUnavailableView(
                "No product recommendations",
                systemImage: "cube.box",
                description: Text("mobile_warehouse.ui.try_another_horizon_or_switch_to_series_for_daily_projection")
            )
        } else {
            ScrollView {
                VStack(alignment: .leading, spacing: LabTheme.spacingLG) {
                    Text("mobile_warehouse.ui.ai_powered_stock_recommendations_from_4_data_sources")
                        .font(.caption)
                        .foregroundStyle(.secondary)

                    ForecastChartPanel(
                        criticalCount: criticalCount,
                        urgentCount: urgentCount,
                        normalCount: normalCount
                    )

                    ForecastSkuTable(
                        products: forecast.products,
                        generatedAt: forecast.generatedAt,
                        forecastDays: forecast.forecastDays
                    )
                }
                .padding()
            }
        }
    }

    @ViewBuilder
    private var seriesSegment: some View {
        if forecast.series.isEmpty {
            ContentUnavailableView(
                "No series data",
                systemImage: "chart.line.uptrend.xyaxis",
                description: Text("mobile_warehouse.ui.no_daily_demand_projection_for_this_window")
            )
        } else {
            ResponsiveGridContentWrapper {
                ForEach(forecast.series) { day in
                    VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                        Text(day.date)
                            .font(.headline)
                        Text(L10n.format("mobile_warehouse.ui.projected_units_projectedunits", "\(day.projectedUnits)"))
                            .font(.subheadline)
                        Text(L10n.format("mobile_warehouse.ui.committed_committedunits_pending_pendingconfirmationunits", "\(day.committedUnits)", "\(day.pendingConfirmationUnits)"))
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                    .padding(.vertical, LabTheme.spacingXS)
                }
            }
        }
    }

    private var criticalCount: Int {
        forecast.products.filter { $0.priority.uppercased() == "CRITICAL" }.count
    }

    private var urgentCount: Int {
        forecast.products.filter { $0.priority.uppercased() == "URGENT" }.count
    }

    private var normalCount: Int {
        forecast.products.filter { $0.priority.uppercased() == "NORMAL" }.count
    }



    private func load() {
        loading = true
        error = nil
        Task {
            do {
                var body = try await WarehouseService.demandForecast(days: horizon)
                if body.products.isEmpty {
                    let insights = try await WarehouseService.replenishmentInsights()
                    let rows = insights.rows
                    if !rows.isEmpty {
                        body = demandForecastFromInsights(rows, horizon: horizon)
                    }
                }
                forecast = body
            } catch {
                self.error = error.localizedDescription
            }
            loading = false
        }
    }
}

private func demandForecastFromInsights(_ insights: [ReplenishmentInsight], horizon: Int) -> DemandForecastResponse {
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


