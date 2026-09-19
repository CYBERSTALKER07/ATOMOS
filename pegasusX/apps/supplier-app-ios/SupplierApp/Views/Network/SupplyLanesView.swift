import SwiftUI

struct SupplyLanesView: View {
    @State private var loading = true
    @State private var error: String?
    @State private var lanes: [SupplierSupplyLaneRow] = []

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading supply lanes…")
            } else if let error {
                SupplierErrorView(message: error) { Task { await load() } }
            } else if lanes.isEmpty {
                SupplierEmptyView(title: "No lanes", message: "No active warehouse lanes. Configure nodes on topology.")
            } else {
                SupplyLanesList(lanes: lanes)
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("supplier_portal.supply_lanes.text.supply_lanes")
        .task { await load() }
    }

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            lanes = try await SupplierOperationsService.supplyLanes()
        } catch {
            self.error = error.localizedDescription
        }
    }
}
