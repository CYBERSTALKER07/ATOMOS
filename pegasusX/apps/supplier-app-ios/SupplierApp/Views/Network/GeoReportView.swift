import SwiftUI

struct GeoReportView: View {
    @State private var loading = true
    @State private var error: String?
    @State private var lanes: [SupplierSupplyLaneRow] = []

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading geo report…")
            } else if let error {
                SupplierErrorView(message: error) { Task { await load() } }
            } else if lanes.isEmpty {
                SupplierEmptyView(title: "No coverage", message: "No active lanes to report on.")
            } else {
                GeoReportLanesList(lanes: lanes)
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("Geo report")
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
