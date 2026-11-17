import SwiftUI

struct RoutePerformanceListView: View {
    @State private var viewModel = RoutePerformanceListViewModel()

    var body: some View {
        Group {
            if viewModel.loading && viewModel.rows.isEmpty {
                SupplierLoadingView(title: "mobile_supplier.loading.latest")
            } else if let error = viewModel.error, viewModel.rows.isEmpty {
                SupplierErrorView(message: error) { Task { await viewModel.load() } }
            } else if viewModel.rows.isEmpty {
                SupplierEmptyView(
                    title: "mobile_supplier.empty.no_active_routes",
                    message: "mobile_supplier.empty.routes_message"
                )
            } else {
                List(viewModel.rows) { row in
                    Text(L10n.format("mobile_supplier.ui.route_routeid_orderscompleted_orders", "\(row.routeId)", "\(row.ordersCompleted)"))
                        .accessibilityElement(children: .combine)
                        .accessibilityLabel("Route \(row.routeId), \(row.ordersCompleted) orders completed")
                }
            }
        }
        .navigationTitle("portal.nav.route_performance")
        .refreshable {
            PegasusHaptics.light()
            await viewModel.load(silent: true)
        }
        .task { await viewModel.load() }
    }
}
