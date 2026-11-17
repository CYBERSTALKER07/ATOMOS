import SwiftUI

struct GeoReportView: View {
    @State private var vm = SupplyLanesViewModel()

    var body: some View {
        Group {
            if vm.loading && vm.lanes.isEmpty {
                SupplierLoadingView(title: "mobile_supplier.loading.geo_report")
            } else if let error = vm.error, vm.lanes.isEmpty {
                SupplierErrorView(message: error) { Task { await vm.load() } }
            } else if vm.lanes.isEmpty {
                SupplierEmptyView(title: "mobile_supplier.empty.no_coverage", message: "mobile_supplier.empty.coverage_lanes")
            } else {
                GeoReportLanesList(lanes: vm.lanes)
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("supplier_portal.geo_report.text.geo_report")
        .task { await vm.load() }
        .refreshable {
            PegasusHaptics.light()
            await vm.load(silent: true)
        }
    }
}
