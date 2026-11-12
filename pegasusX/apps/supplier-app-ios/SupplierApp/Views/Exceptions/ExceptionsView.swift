import SwiftUI

struct ExceptionsView: View {
    @State private var viewModel = ExceptionsViewModel()

    var body: some View {
        Group {
            if viewModel.loading && viewModel.rows.isEmpty {
                SupplierLoadingView(title: "mobile_supplier.loading.exceptions")
            } else if let error = viewModel.error, viewModel.rows.isEmpty {
                SupplierErrorView(message: error) { Task { await viewModel.load() } }
            } else if viewModel.rows.isEmpty {
                SupplierEmptyView(
                    title: "mobile_supplier.empty.no_exceptions",
                    message: "mobile_supplier.empty.exceptions_message"
                )
            } else {
                ExceptionsList(rows: viewModel.rows, busyKey: viewModel.busyKey) { row in
                    Task { await viewModel.resolve(row) }
                }
            }
        }
        .navigationTitle("portal.nav.exceptions")
        .toolbar {
            ToolbarItem(placement: .primaryAction) {
                NavigationLink {
                    ClaimsView()
                } label: {
                    Text("portal.nav.claims")
                }
            }
        }
        .refreshable { await viewModel.load(silent: true) }
        .task { await viewModel.load() }
    }
}
