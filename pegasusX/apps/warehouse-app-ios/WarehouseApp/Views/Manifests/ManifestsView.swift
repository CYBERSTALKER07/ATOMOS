import SwiftUI

struct ManifestsView: View {
    @State private var manifests: [Manifest] = []
    @State private var loading = true
    @State private var error: String?

    var body: some View {
        NavigationStack {
            Group {
                if loading {
                    ProgressView()
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else if let error {
                    ContentUnavailableView {
                        Label("mobile_warehouse.ui.error", systemImage: "exclamationmark.triangle")
                    } description: {
                        Text(error)
                    } actions: {
                        Button("common.action.retry") { load() }
                    }
                } else if manifests.isEmpty {
                    ContentUnavailableView("No Manifests", systemImage: "doc.on.doc", description: Text("mobile_warehouse.ui.no_manifests_found"))
                } else {
                    ResponsiveGridContentWrapper {
                        ForEach(manifests) { manifest in
                        HStack {
                            VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                Text(String(manifest.manifestId.prefix(8)))
                                    .font(.headline.monospaced())
                                Text(L10n.format("mobile_warehouse.ui.stopcount_stops_drivername", "\(manifest.stopCount)", "\(manifest.driverName)"))
                                    .font(.subheadline)
                                    .foregroundStyle(.secondary)
                            }
                            Spacer()
                        }
                    }
                }
                }
            }
            .background(LabTheme.background)
            .navigationTitle("portal.nav.manifests")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") { load() }
                }
            }
            .task { load() }
            .refreshable { load() }
        }
    }

    private func load() {
        loading = true
        error = nil
        Task {
            do {
                let resp = try await WarehouseService.manifests()
                manifests = resp.manifests
            } catch {
                self.error = error.localizedDescription
            }
            loading = false
        }
    }
}
