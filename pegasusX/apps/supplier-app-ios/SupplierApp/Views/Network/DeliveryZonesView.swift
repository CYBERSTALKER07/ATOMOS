import SwiftUI

struct DeliveryZonesView: View {
    @State private var vm = TopologyNodesListViewModel()

    var body: some View {
        Group {
            if vm.loading && vm.warehouses.isEmpty {
                SupplierLoadingView(title: "mobile_supplier.loading.delivery_zones")
            } else if let error = vm.error, vm.warehouses.isEmpty {
                SupplierErrorView(message: error) { Task { await vm.load() } }
            } else if vm.warehouses.isEmpty {
                SupplierEmptyView(title: "mobile_supplier.empty.no_coverage", message: "mobile_supplier.empty.coverage_warehouses")
            } else {
                DeliveryZonesList(warehouses: vm.warehouses)
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("supplier_portal.delivery_zones.text.delivery_zones")
        .task { await vm.load() }
        .refreshable {
            PegasusHaptics.light()
            await vm.load(silent: true)
        }
    }
}
