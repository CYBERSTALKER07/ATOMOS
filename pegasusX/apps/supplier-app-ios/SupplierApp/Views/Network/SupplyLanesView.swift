import SwiftUI

struct SupplyLanesView: View {
    @State private var vm = SupplyLanesViewModel()

    var body: some View {
        Group {
            if vm.loading && vm.lanes.isEmpty {
                SupplierLoadingView(title: "mobile_supplier.loading.supply_lanes")
            } else if let error = vm.error, vm.lanes.isEmpty {
                SupplierErrorView(message: error) { Task { await vm.load() } }
            } else if vm.lanes.isEmpty {
                SupplierEmptyView(title: "mobile_supplier.empty.no_lanes", message: "mobile_supplier.empty.lanes_message")
            } else {
                SupplyLanesList(lanes: vm.lanes)
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("supplier_portal.supply_lanes.text.supply_lanes")
        .task { await vm.load() }
        .refreshable {
            PegasusHaptics.light()
            await vm.load(silent: true)
        }
    }
}
