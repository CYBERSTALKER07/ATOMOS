import SwiftUI

struct DemandHistoryView: View {
    @State private var history: SupplierDemandHistoryResponse?
    @State private var demandConfidence: ForecastConfidence?
    @State private var demandGeneratedAt: String?
    @State private var loading = true
    @State private var error: String?

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading demand forecast…")
            } else if let error {
                SupplierErrorView(message: error) { Task { await load() } }
            } else if let history {
                ResponsiveGridContentWrapper {
                    if let demandConfidence {
                        Section {
                            ForecastConfidenceView(
                                confidence: demandConfidence,
                                updatedAt: ForecastConfidenceSupport.formatForecastUpdatedAt(generatedAt: demandGeneratedAt),
                                stale: ForecastConfidenceSupport.isForecastStale(generatedAt: demandGeneratedAt)
                            )
                        }
                    }
                    if !history.timeSeries.isEmpty {
                        Section("Baseline vs actual") {
                            ForEach(history.timeSeries) { point in
                                VStack(alignment: .leading, spacing: SupplierTheme.spacingXS) {
                                    Text(point.date).font(.headline)
                                    Text(L10n.format("mobile_supplier.ui.baseline_predictedqty_actual_actualqty", "\(Int(point.predictedQty))", "\(Int(point.actualQty))"))
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                }
                            }
                        }
                    }
                    if !history.upcoming.isEmpty {
                        Section("Upcoming demand") {
                            ForEach(history.upcoming) { row in
                                VStack(alignment: .leading, spacing: SupplierTheme.spacingXS) {
                                    Text(row.productName).font(.headline)
                                    Text(L10n.format("mobile_supplier.ui.retailername_date_qty_predictedqty", "\(row.retailerName)", "\(row.date)", "\(Int(row.predictedQty))"))
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                }
                            }
                        }
                    }
                    if history.timeSeries.isEmpty && history.upcoming.isEmpty {
                        SupplierEmptyView(title: "No forecast data", message: "Demand predictions will appear when analytics is active.")
                    }
                }
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("supplier_portal.analytics.demand.text.demand_forecast")
        .task { await load() }
        .refreshable { await load(silent: true) }
    }

    @MainActor
    private func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        defer { loading = false }
        do {
            async let historyValue = SupplierService.getDemandHistory()
            async let demandValue = SupplierOperationsService.demandToday()
            history = try await historyValue
            let demand = try await demandValue
            demandGeneratedAt = demand.generatedAt
            demandConfidence = ForecastConfidenceSupport.fromDemand(demand)
        } catch {
            if !silent { self.error = error.localizedDescription }
        }
    }
}
