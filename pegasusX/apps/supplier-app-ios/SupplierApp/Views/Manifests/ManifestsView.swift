import SwiftUI

struct ManifestsView: View {
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    @State private var viewModel = ManifestsViewModel()

    var body: some View {
        Group {
            if viewModel.loading && viewModel.rows.isEmpty {
                SupplierLoadingView(title: "mobile_supplier.loading.manifests")
            } else if let error = viewModel.error, viewModel.rows.isEmpty {
                SupplierErrorView(message: error) { Task { await viewModel.load() } }
            } else if viewModel.rows.isEmpty {
                SupplierEmptyView(title: "mobile_supplier.empty.no_manifests", message: "mobile_supplier.empty.manifests_message")
            } else {
                ManifestsList(items: viewModel.rows)
            }
        }
        .navigationTitle("portal.nav.manifests")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                NavigationLink { ManifestExceptionsView() } label: {
                    Text("mobile_supplier.ui.gate")
                }
            }
        }
        .task { await viewModel.load() }
        .refreshable { await viewModel.load(silent: true) }
        .silentRealtimeRefresh(
            refreshEpoch: realtimeHub.refreshEpoch,
            reconnectEpoch: realtimeHub.reconnectEpoch
        ) { silent in
            Task { await viewModel.load(silent: silent) }
        }
    }
}
