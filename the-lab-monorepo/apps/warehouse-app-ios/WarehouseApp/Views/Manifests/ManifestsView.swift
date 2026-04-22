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
                        Label("Error", systemImage: "exclamationmark.triangle")
                    } description: {
                        Text(error)
                    } actions: {
                        Button("Retry") { load() }
                    }
                } else if manifests.isEmpty {
                    ContentUnavailableView("No Manifests", systemImage: "doc.on.doc", description: Text("No manifests found"))
                } else {
                    List(manifests) { manifest in
                        HStack {
                            VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                Text(String(manifest.manifestId.prefix(8)))
                                    .font(.headline.monospaced())
                                Text("\(manifest.orderCount) orders · \(manifest.driverName)")
                                    .font(.subheadline)
                                    .foregroundStyle(.secondary)
                            }
                            Spacer()
                            Text(manifest.status)
                                .font(.caption.bold())
                                .padding(.horizontal, LabTheme.spacingSM)
                                .padding(.vertical, LabTheme.spacingXS)
                                .background(.quaternary, in: Capsule())
                        }
                    }
                    .listStyle(.insetGrouped)
                }
            }
            .background(LabTheme.background)
            .navigationTitle("Manifests")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Refresh", systemImage: "arrow.clockwise") { load() }
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
