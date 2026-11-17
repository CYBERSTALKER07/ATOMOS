import SwiftUI

struct FleetOrdersView: View {
    @State private var viewModel = FleetOrdersViewModel()

    var body: some View {
        Group {
            if viewModel.loading && viewModel.rows.isEmpty {
                SupplierLoadingView(title: "mobile_supplier.loading.fleet_orders")
            } else if let error = viewModel.error, viewModel.rows.isEmpty {
                SupplierErrorView(message: error) { Task { await viewModel.load() } }
            } else if viewModel.rows.isEmpty {
                SupplierEmptyView(title: "mobile_supplier.empty.no_fleet_orders", message: "mobile_supplier.empty.fleet_orders_message")
            } else {
                ResponsiveGridContentWrapper {
                    ForEach(viewModel.rows) { row in
                        VStack(alignment: .leading, spacing: 4) {
                            Text(row.orderId).font(.headline)
                            Text(L10n.format("mobile_supplier.ui.status_driver_driverid", "\(row.status)", "\(row.driverId ?? "—")")).font(.subheadline)
                            if let formattedTotal = Self.formatOrderTotal(totalMinor: row.totalMinor, currency: row.currency) {
                                Text(formattedTotal)
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                            if let routeId = row.routeId { Text(L10n.format("mobile_supplier.ui.route_routeid_2", "\(routeId)")).font(.caption) }
                        }
                        .accessibilityElement(children: .combine)
                    }
                }
            }
        }
        .navigationTitle("supplier_portal.fleet.orders.text.fleet_orders")
        .task { await viewModel.load() }
        .refreshable { await viewModel.load(silent: true) }
    }

    static func formatOrderTotal(totalMinor: Int64?, currency: String?) -> String? {
        guard let totalMinor else { return nil }
        let code = (currency?.isEmpty == false ? currency : nil) ?? "UZS"
        return MoneyFormat.minor(totalMinor, currency: code)
    }
}
