import SwiftUI

struct FleetOrdersView: View {
    @State private var rows: [SupplierFleetOrderRow] = []
    @State private var loading = true
    @State private var error: String?

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading fleet orders…")
            } else if let error {
                SupplierErrorView(message: error) { Task { await load() } }
            } else if rows.isEmpty {
                SupplierEmptyView(title: "No fleet orders", message: "Active route assignments appear here.")
            } else {
                ResponsiveGridContentWrapper {
                    ForEach(rows) { row in
                        VStack(alignment: .leading, spacing: 4) {
                            Text(row.orderId).font(.headline)
                            Text(L10n.format("mobile_supplier.ui.status_driver_driverid", "\(row.status)", "\(row.driverId ?? "—")")).font(.subheadline)
                            if let routeId = row.routeId { Text(L10n.format("mobile_supplier.ui.route_routeid_2", "\(routeId)")).font(.caption) }
                        }
                    }
                }
            }
        }
        .navigationTitle("supplier_portal.fleet.orders.text.fleet_orders")
        .task { await load() }
    }

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            rows = try await SupplierOperationsService.fleetOrders()
        } catch {
            self.error = error.localizedDescription
        }
    }
}
