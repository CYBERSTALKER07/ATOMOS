import SwiftUI

struct DemandHistoryView: View {
    @State private var viewModel = DemandHistoryViewModel()

    var body: some View {
        Group {
            if viewModel.loading && viewModel.history == nil {
                SupplierLoadingView(title: "mobile_supplier.loading.demand_forecast")
            } else if let error = viewModel.error, viewModel.history == nil {
                SupplierErrorView(message: error) { Task { await viewModel.load() } }
            } else if let history = viewModel.history {
                ResponsiveGridContentWrapper {
                    if let demandConfidence = viewModel.demandConfidence {
                        Section {
                            ForecastConfidenceView(
                                confidence: demandConfidence,
                                updatedAt: ForecastConfidenceSupport.formatForecastUpdatedAt(generatedAt: viewModel.demandGeneratedAt),
                                stale: ForecastConfidenceSupport.isForecastStale(generatedAt: viewModel.demandGeneratedAt)
                            )
                        }
                    }
                    if !history.timeSeries.isEmpty {
                        Section("mobile_supplier.chrome.baseline_vs_actual") {
                            ForEach(history.timeSeries) { point in
                                VStack(alignment: .leading, spacing: SupplierTheme.spacingXS) {
                                    Text(point.date).font(.headline)
                                    Text(L10n.format("mobile_supplier.ui.baseline_predictedqty_actual_actualqty", "\(Int(point.predictedQty))", "\(Int(point.actualQty))"))
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                }
                                .accessibilityElement(children: .combine)
                            }
                        }
                    }
                    if !history.upcoming.isEmpty {
                        Section("mobile_supplier.chrome.upcoming_demand") {
                            ForEach(history.upcoming) { row in
                                VStack(alignment: .leading, spacing: SupplierTheme.spacingXS) {
                                    Text(row.productName).font(.headline)
                                    Text(L10n.format("mobile_supplier.ui.retailername_date_qty_predictedqty", "\(row.retailerName)", "\(row.date)", "\(Int(row.predictedQty))"))
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                }
                                .accessibilityElement(children: .combine)
                            }
                        }
                    }
                    if history.timeSeries.isEmpty && history.upcoming.isEmpty {
                        SupplierEmptyView(title: "mobile_supplier.empty.no_forecast", message: "mobile_supplier.empty.forecast_message")
                    }
                }
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("supplier_portal.analytics.demand.text.demand_forecast")
        .task { await viewModel.load() }
        .refreshable { await viewModel.load(silent: true) }
    }
}
