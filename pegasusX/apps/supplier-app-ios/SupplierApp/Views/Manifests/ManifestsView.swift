import SwiftUI

struct ManifestsView: View {
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    @State private var rows: [SupplierManifestRow] = []
    @State private var loading = true
    @State private var error: String?

    var body: some View {
        Group {
            if loading && rows.isEmpty {
                SupplierLoadingView(title: "Loading manifests…")
            } else if let error {
                SupplierErrorView(message: error) { Task { await load() } }
            } else if rows.isEmpty {
                SupplierEmptyView(title: "No manifests", message: "Loading manifests will appear here.")
            } else {
                ManifestsList(items: rows)
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
        .task { await load() }
        .refreshable { await load(silent: true) }
        .silentRealtimeRefresh(
            refreshEpoch: realtimeHub.refreshEpoch,
            reconnectEpoch: realtimeHub.reconnectEpoch
        ) { silent in
            Task { await load(silent: silent) }
        }
    }

    private func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        defer { if !silent { loading = false } }
        do {
            rows = try await SupplierOperationsService.manifests()
        } catch {
            if !silent { self.error = error.localizedDescription }
        }
    }
}
