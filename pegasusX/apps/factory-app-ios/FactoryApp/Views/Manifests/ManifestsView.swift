import SwiftUI

struct ManifestsView: View {
    @Environment(\.dismiss) private var dismiss
    @State private var realtimeClient = FactoryRealtimeClient()
    @State private var manifests: [Manifest] = []
    @State private var loading = true
    @State private var error: String?
    @State private var selectedManifestID: String?

    var body: some View {
        NavigationSplitView {
            Group {
                if loading && manifests.isEmpty {
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
                }
            }
            .navigationTitle("portal.nav.manifests")
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button("common.action.close", systemImage: "xmark") { dismiss() }
                        .labelStyle(.iconOnly)
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") {
                        Task { await load() }
                    }
                    .labelStyle(.iconOnly)
                }
            }
        } detail: {
            if let selectedManifestID {
                ManifestDetailView(manifestId: selectedManifestID)
            } else {
                ContentUnavailableView("Select a Manifest", systemImage: "list.clipboard", description: Text("mobile_factory.ui.choose_a_manifest_to_run_the_leo_lifecycle"))
            }
        }
        .task { await load() }
        .onAppear {
            realtimeClient.connect(
                onStateChange: { _ in },
                onEvent: { event in
                    guard event.eventType == .manifestUpdate || event.eventType == .transferUpdate else { return }
                    Task { await load(silent: true) }
                },
                onReconnect: {
                    Task { await load(silent: true) }
                }
            )
        }
        .onDisappear {
            realtimeClient.disconnect()
        }
    }

    @MainActor
    private func load(silent: Bool = false) async {
        if !silent {
            loading = true
        }
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
        if !silent {
            loading = false
        }
    }
}

private struct ManifestRow: View {
    let manifest: Manifest

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
            Text(manifest.truckPlate.isEmpty ? String(manifest.truckId.prefix(8)) : manifest.truckPlate)
                .font(.subheadline.bold())
            Text(L10n.format("mobile_factory.ui.manifest_prefix", "\(manifest.id.prefix(8))"))
                .font(.footnote.monospaced())
                .foregroundStyle(.secondary)
            HStack {
                FactoryStatusBadge(text: manifest.state)
                Spacer()
                Text(L10n.format("mobile_factory.ui.totalvolumevu_vu", "\(Int(manifest.totalVolumeVU))"))
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

struct PayloadLoadView: View {
    @State private var realtimeClient = FactoryRealtimeClient()
    @State private var manifests: [Manifest] = []
    @State private var loading = true
    @State private var error: String?
    @State private var actingId: String?

    var body: some View {
        NavigationStack {
            Group {
                if loading && manifests.isEmpty {
                    FactoryLoadingState(title: "Loading payloads", message: "Factory drafts and loading manifests.")
                } else if let error, manifests.isEmpty {
                    FactoryErrorView(message: error) { Task { await load() } }
                } else if manifests.isEmpty {
                    FactoryStateView(
                        kind: .empty,
                        headline: "No factory payloads",
                        message: "Dispatch creates FactoryTruckManifests drafts. Start loading and seal them here."
                    )
                } else {
                    List(manifests) { manifest in
                        VStack(alignment: .leading, spacing: 8) {
                            Text(manifest.id.prefix(16)).font(.footnote.monospaced())
                            Text(manifest.state).font(.caption).foregroundStyle(.secondary)
                            if let next = ManifestLifecycleAction.next(for: manifest.state),
                               next == .startLoading || next == .seal {
                                Button(actingId == manifest.id ? "Applying…" : next.label) {
                                    Task { await run(manifest, next) }
                                }
                                .disabled(actingId != nil)
                            }
                        }
                    }
                }
            }
            .navigationTitle("Payload / Load")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") {
                        Task { await load() }
                    }
                    .labelStyle(.iconOnly)
                }
            }
            .task { await load() }
            .onAppear {
                realtimeClient.connect(
                    onStateChange: { _ in },
                    onEvent: { event in
                        guard event.eventType == .manifestUpdate || event.eventType == .transferUpdate else { return }
                        Task { await load(silent: true) }
                    }
                )
            }
            .onDisappear { realtimeClient.disconnect() }
        }
    }

    @MainActor
    private func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        do {
            let response = try await FactoryService.manifests()
            manifests = response.manifests.filter { $0.state == "DRAFT" || $0.state == "LOADING" }
        } catch {
            if !silent { self.error = error.localizedDescription }
        }
        if !silent { loading = false }
    }

    @MainActor
    private func run(_ manifest: Manifest, _ action: ManifestLifecycleAction) async {
        actingId = manifest.id
        do {
            _ = try await FactoryService.transitionManifest(id: manifest.id, action: action.rawValue)
            await load(silent: true)
        } catch {
            self.error = error.localizedDescription
        }
        actingId = nil
    }
}
