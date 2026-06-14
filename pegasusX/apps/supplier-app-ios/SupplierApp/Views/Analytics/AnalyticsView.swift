import SwiftUI

struct AnalyticsView: View {
    @State private var loading = true
    @State private var error: String?
    @State private var pendingOrders = 0
    @State private var inventorySKUs = 0
    @State private var revenueLabel = "—"
    @State private var predictionCount = 0
    @State private var forecastUnits = 0
    @State private var velocityCreated = 0

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading analytics…")
            } else if let error {
                SupplierErrorView(message: error) { Task { await load() } }
            } else {
                List {
                    Section("Intelligence") {
                        AnalyticsKpiRow(label: "30-day revenue", value: revenueLabel)
                        AnalyticsKpiRow(label: "Demand predictions", value: "\(predictionCount)")
                        AnalyticsKpiRow(label: "Forecast units (24h)", value: "\(forecastUnits)")
                        AnalyticsKpiRow(label: "Orders created (velocity window)", value: "\(velocityCreated)")
                    }
                    Section("Operational snapshot") {
                        AnalyticsKpiRow(label: "Pending orders", value: "\(pendingOrders)")
                        AnalyticsKpiRow(label: "Inventory SKUs", value: "\(inventorySKUs)")
                    }
                }
                .listStyle(.insetGrouped)
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("Analytics")
        .task { await load() }
        .refreshable { await load(silent: true) }
    }

    @MainActor
    private func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        defer { if !silent { loading = false } }
        do {
            async let dash = SupplierService.dashboard()
            async let revenue = SupplierOperationsService.analyticsRevenue()
            async let demand = SupplierOperationsService.demandToday()
            async let velocity = SupplierOperationsService.analyticsVelocity()

            let dashValue = try await dash
            let revenueValue = try await revenue
            let demandValue = try await demand
            let velocityValue = try await velocity

            pendingOrders = dashValue.pendingOrders
            inventorySKUs = dashValue.inventorySKUs
            revenueLabel = MoneyFormat.minor(revenueValue.totalMinor, currency: revenueValue.currency)
            predictionCount = demandValue.predictionCount
            forecastUnits = demandValue.totalPallets
            velocityCreated = velocityValue.points.reduce(0) { $0 + $1.ordersCreated }
        } catch {
            self.error = error.localizedDescription
        }
    }
}

private struct AnalyticsKpiRow: View {
    let label: String
    let value: String

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(label).font(.caption).foregroundStyle(.secondary)
            Text(value).font(.title2.bold())
        }
    }
}
