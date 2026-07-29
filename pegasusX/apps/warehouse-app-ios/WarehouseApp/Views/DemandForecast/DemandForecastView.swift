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
                    Text("Products").tag(0)
                    Text("Series").tag(1)
                }
                .pickerStyle(.segmented)
                .padding()

                forecastBody
            }
            .navigationTitle("Demand Forecast")
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
                Text("7d").tag(7)
                Text("14d").tag(14)
                Text("30d").tag(30)
            }
            .pickerStyle(.menu)
        }
        ToolbarItem(placement: .topBarTrailing) {
            Button("Refresh", systemImage: "arrow.clockwise") { load() }
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
                description: Text("Try another horizon or switch to Series for daily projection.")
            )
        } else {
            ScrollView {
                VStack(alignment: .leading, spacing: LabTheme.spacingLG) {
                    Text("AI-powered stock recommendations from 4 data sources")
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
                description: Text("No daily demand projection for this window.")
            )
        } else {
            ResponsiveGridContentWrapper {
                ForEach(forecast.series) { day in
                VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                    Text(day.date)
                        .font(.headline)
                    Text("Projected units: \(day.projectedUnits)")
                        .font(.subheadline)
                    Text("Committed: \(day.committedUnits) · Pending: \(day.pendingConfirmationUnits)")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                .padding(.vertical, LabTheme.spacingXS)
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


