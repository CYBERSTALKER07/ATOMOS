import SwiftUI

struct DemandForecastView: View {
    @State private var horizon = 7
    @State private var forecast = DemandForecastResponse()
    @State private var loading = true
    @State private var error: String?
    @State private var selectedSegment = 0

    private let summaryColumns = [
        GridItem(.flexible(), spacing: LabTheme.spacingSM),
        GridItem(.flexible(), spacing: LabTheme.spacingSM),
        GridItem(.flexible(), spacing: LabTheme.spacingSM),
    ]

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

                    LazyVGrid(columns: summaryColumns, spacing: LabTheme.spacingSM) {
                        ForecastSummaryCard(
                            title: "Critical",
                            count: criticalCount,
                            subtitle: "< 2 days",
                            tint: LabTheme.destructive,
                            index: 0
                        )
                        ForecastSummaryCard(
                            title: "Urgent",
                            count: urgentCount,
                            subtitle: "< 5 days",
                            tint: LabTheme.warning,
                            index: 1
                        )
                        ForecastSummaryCard(
                            title: "Healthy",
                            count: normalCount,
                            subtitle: "5+ days",
                            tint: LabTheme.success,
                            index: 2
                        )
                    }

                    ForEach(Array(forecast.products.enumerated()), id: \.element.id) { index, product in
                        ForecastProductRow(product: product)
                            .staggeredAppear(index: index + 3)
                    }

                    if let generatedAt = forecast.generatedAt, !generatedAt.isEmpty {
                        Text("Generated \(formattedGeneratedAt(generatedAt)) · \(forecast.forecastDays)-day window")
                            .font(.caption2)
                            .foregroundStyle(.tertiary)
                    }
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
            List(forecast.series) { day in
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
            .listStyle(.insetGrouped)
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

    private func formattedGeneratedAt(_ raw: String) -> String {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        if let date = formatter.date(from: raw) ?? ISO8601DateFormatter().date(from: raw) {
            return date.formatted(date: .abbreviated, time: .shortened)
        }
        return raw
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
            sources: DemandForecastSources(burnRate: insight.avgDailyVelocity)
        )
    }
    return DemandForecastResponse(
        warehouseId: insights.first?.warehouseId ?? "",
        forecastDays: horizon,
        generatedAt: insights.first?.createdAt,
        products: products
    )
}

private struct ForecastSummaryCard: View {
    let title: String
    let count: Int
    let subtitle: String
    let tint: Color
    let index: Int

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
            Text(title)
                .font(.caption2)
                .foregroundStyle(.secondary)
            Text("\(count)")
                .font(.title2.bold())
                .foregroundStyle(tint)
            Text(subtitle)
                .font(.caption2)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .labCard()
        .staggeredAppear(index: index)
    }
}

private struct ForecastProductRow: View {
    let product: DemandForecastProduct

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
            HStack(alignment: .firstTextBaseline) {
                Text(product.productName.isEmpty ? String(product.productId.prefix(8)) : product.productName)
                    .font(.headline)
                Spacer()
                Text(product.priority)
                    .font(.caption.bold())
                    .foregroundStyle(priorityColor(product.priority))
            }

            HStack {
                metricColumn(title: "Stock", value: "\(product.currentStock)")
                metricColumn(title: "Rec.", value: "\(product.recommendedQty)")
                metricColumn(
                    title: "Stockout",
                    value: String(format: "%.1fd", product.daysUntilStockout),
                    valueColor: stockoutColor(product.daysUntilStockout)
                )
            }

            HStack(spacing: LabTheme.spacingMD) {
                sourceChip(label: "In", value: "\(product.sources.incomingOrders)")
                sourceChip(label: "AI", value: "\(product.sources.aiPrediction)")
                sourceChip(label: "Pre", value: "\(product.sources.preOrders)")
                sourceChip(label: "Burn", value: String(format: "%.1f", product.sources.burnRate))
            }
        }
        .labCard()
    }

    private func metricColumn(title: String, value: String, valueColor: Color = LabTheme.label) -> some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(title)
                .font(.caption2)
                .foregroundStyle(.secondary)
            Text(value)
                .font(.subheadline.monospacedDigit())
                .foregroundStyle(valueColor)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    private func sourceChip(label: String, value: String) -> some View {
        VStack(spacing: 2) {
            Text(label)
                .font(.caption2)
                .foregroundStyle(.secondary)
            Text(value)
                .font(.caption.monospacedDigit())
        }
        .frame(maxWidth: .infinity)
    }

    private func priorityColor(_ priority: String) -> Color {
        switch priority.uppercased() {
        case "CRITICAL": return LabTheme.destructive
        case "URGENT": return LabTheme.warning
        default: return LabTheme.secondaryLabel
        }
    }

    private func stockoutColor(_ days: Double) -> Color {
        if days < 2 { return LabTheme.destructive }
        if days < 5 { return LabTheme.warning }
        return LabTheme.label
    }
}
