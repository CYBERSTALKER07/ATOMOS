import SwiftUI

struct ActivityView: View {
    @State private var viewModel = ActivityViewModel()

    var body: some View {
        Group {
            if viewModel.loading && viewModel.rows.isEmpty {
                SupplierLoadingView(title: "mobile_supplier.loading.activity")
            } else if let error = viewModel.error, viewModel.rows.isEmpty {
                SupplierErrorView(message: error) { Task { await viewModel.load() } }
            } else if viewModel.rows.isEmpty {
                SupplierEmptyView(title: "mobile_supplier.empty.no_recent_activity", message: "mobile_supplier.empty.activity_message")
            } else {
                ResponsiveGridContentWrapper {
                    ForEach(viewModel.rows) { row in
                        VStack(alignment: .leading, spacing: 4) {
                            Text(row.type).font(.headline)
                            Text(row.description).font(.subheadline)
                            Text(row.timestamp).font(.caption)
                        }
                        .accessibilityElement(children: .combine)
                    }
                }
            }
        }
        .navigationTitle("portal.nav.activity")
        .task { await viewModel.load() }
        .refreshable { await viewModel.load(silent: true) }
    }
}
