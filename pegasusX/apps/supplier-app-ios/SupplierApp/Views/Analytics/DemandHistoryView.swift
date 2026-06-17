import SwiftUI

struct DemandHistoryView: View {
    @State private var history: SupplierDemandHistoryResponse?
    @State private var loading = true
    @State private var error: String?

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading demand forecast…")
            } else if let error {
                SupplierErrorView(message: error) { Task { await load() } }
            } else if let history {
                List {
                    if !history.timeSeries.isEmpty {
                        Section("Historical accuracy") {
                            ForEach(history.timeSeries) { point in
                                VStack(alignment: .leading, spacing: SupplierTheme.spacingXS) {
                                    Text(point.date).font(.headline)
                                    Text("Predicted \(Int(point.predictedQty)) · Actual \(Int(point.actualQty))")
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
                                    Text("\(row.retailerName) · \(row.date) · qty \(Int(row.predictedQty))")
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
                .listStyle(.insetGrouped)
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("Demand forecast")
        .task { await load() }
        .refreshable { await load(silent: true) }
    }

    @MainActor
    private func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        defer { loading = false }
        do {
            history = try await SupplierService.getDemandHistory()
        } catch {
            if !silent { self.error = error.localizedDescription }
        }
    }
}
