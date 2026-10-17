import SwiftUI

struct DeliveryZonesView: View {
    @State private var loading = true
    @State private var error: String?
    @State private var warehouses: [SupplierTopologyWarehouse] = []

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading delivery zones…")
            } else if let error {
                SupplierErrorView(message: error) { Task { await load() } }
            } else if warehouses.isEmpty {
                SupplierEmptyView(title: "No coverage", message: "No warehouse coverage configured.")
            } else {
                DeliveryZonesList(warehouses: warehouses)
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("supplier_portal.delivery_zones.text.delivery_zones")
        .task { await load() }
    }

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let resp = try await SupplierOperationsService.topology()
            warehouses = resp.warehouses
        } catch {
            self.error = error.localizedDescription
        }
    }
}
