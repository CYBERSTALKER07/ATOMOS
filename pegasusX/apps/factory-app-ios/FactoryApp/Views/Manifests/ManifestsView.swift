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
                    ProgressView()
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else if let error {
                    ContentUnavailableView {
                        Label("Error", systemImage: "exclamationmark.triangle")
                    } description: {
                        Text(error)
                    } actions: {
                        Button("Retry") { Task { await load() } }
                    }
                } else if manifests.isEmpty {
                    ContentUnavailableView("No Manifests", systemImage: "list.clipboard", description: Text("Dispatch transfers to create a manifest draft."))
                } else {
                    List(selection: $selectedManifestID) {
                        ForEach(manifests) { manifest in
                            VStack(alignment: .leading, spacing: 4) {
                                Text(manifest.id)
                                    .font(.footnote.monospaced())
                                HStack {
                                    Text(manifest.state).font(.caption.bold())
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
                            .tag(manifest.id)
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
