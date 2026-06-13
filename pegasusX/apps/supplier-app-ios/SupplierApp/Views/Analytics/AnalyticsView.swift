import SwiftUI

struct AnalyticsView: View {
    @State private var loading = true
    @State private var error: String?
    @State private var pendingOrders = 0
    @State private var inventorySKUs = 0

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading analytics…")
            } else if let error {
                SupplierErrorView(message: error) { Task { await load() } }
            } else {
                List {
                    Section("Operational KPIs") {
                        AnalyticsKpiRow(label: "Pending orders", value: "\(pendingOrders)")
                        AnalyticsKpiRow(label: "Inventory SKUs", value: "\(inventorySKUs)")
                    }
                    Section {
                        Text("KPIs sourced from supplier dashboard authority.")
                            .font(.footnote)
                            .foregroundStyle(.secondary)
                    }
                }
                .listStyle(.insetGrouped)
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("Analytics")
        .task { await load() }
    }

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let dash = try await SupplierService.dashboard()
            pendingOrders = dash.pendingOrders
            inventorySKUs = dash.inventorySKUs
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
