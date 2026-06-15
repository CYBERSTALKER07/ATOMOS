import SwiftUI

struct ManifestsView: View {
    @Environment(\.dismiss) private var dismiss
    @State private var manifests: [Manifest] = []
    @State private var loading = true
    @State private var error: String?
    @State private var selectedManifestID: String?

    var body: some View {
        NavigationSplitView {
            Group {
                if loading {
                    FactoryLoadingView(
                        title: "Loading manifests",
                        message: "Fetching outbound manifest lifecycle states."
                    )
                } else if let error {
                    FactoryErrorView(message: error) {
                        Task { await load() }
                    }
                } else if manifests.isEmpty {
                    FactoryStateView(
                        kind: .empty,
                        headline: "No manifests",
                        message: "Dispatch transfers to create a manifest draft."
                    )
                } else {
                    List(selection: $selectedManifestID) {
                        Section {
                            FactorySectionHeader(
                                title: "Outbound manifests",
                                subtitle: "\(manifests.count) manifests in the factory queue"
                            )
                            .listRowInsets(EdgeInsets(top: 8, leading: 0, bottom: 8, trailing: 0))
                            .listRowBackground(Color.clear)
                        }

                        Section {
                            ForEach(manifests) { manifest in
                                ManifestRow(manifest: manifest)
                                    .tag(manifest.id)
                            }
                        }
                    }
                    .listStyle(.plain)
                }
            }
            .navigationTitle("Manifests")
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button("Close", systemImage: "xmark") { dismiss() }
                        .labelStyle(.iconOnly)
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Refresh", systemImage: "arrow.clockwise") {
                        Task { await load() }
                    }
                    .labelStyle(.iconOnly)
                }
            }
        } detail: {
            if let selectedManifestID {
                ManifestDetailView(manifestId: selectedManifestID)
            } else {
                ContentUnavailableView("Select a Manifest", systemImage: "list.clipboard", description: Text("Choose a manifest to run the LEO lifecycle."))
            }
        }
        .task { await load() }
    }

    @MainActor
    private func load() async {
        loading = true
        error = nil
        do {
            let response = try await FactoryService.manifests()
            manifests = response.manifests
            if selectedManifestID == nil {
                selectedManifestID = manifests.first?.id
            }
        } catch {
            self.error = error.localizedDescription
        }
        loading = false
    }
}

private struct ManifestRow: View {
    let manifest: Manifest

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
            Text(manifest.truckPlate.isEmpty ? String(manifest.truckId.prefix(8)) : manifest.truckPlate)
                .font(.subheadline.bold())
            Text("Manifest \(manifest.id.prefix(8))")
                .font(.footnote.monospaced())
                .foregroundStyle(.secondary)
            HStack {
                FactoryStatusBadge(text: manifest.state)
                Spacer()
                Text("\(Int(manifest.totalVolumeVU)) VU")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            if let next = ManifestLifecycleAction.next(for: manifest.state) {
                Text(next.label)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
        .padding(.vertical, LabTheme.spacingXS)
    }
}
