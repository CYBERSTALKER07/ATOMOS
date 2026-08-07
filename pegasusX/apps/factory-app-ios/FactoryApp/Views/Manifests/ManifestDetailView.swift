import SwiftUI

struct ManifestDetailView: View {
    let manifestId: String

    @State private var realtimeClient = FactoryRealtimeClient()
    @State private var detail: ManifestDetailSnapshot?
    @State private var loading = true
    @State private var acting = false
    @State private var error: String?

    var body: some View {
        Group {
            if loading && detail == nil {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let error {
                ContentUnavailableView {
                    Label("mobile_factory.ui.error", systemImage: "exclamationmark.triangle")
                } description: {
                    Text(error)
                } actions: {
                    Button("common.action.retry") { Task { await load() } }
                }
            } else if let detail {
                ResponsiveGridContentWrapper {
                    Section {
                        Text(detail.manifest.manifestId)
                            .font(.footnote.monospaced())
                        LabeledContent("State", value: detail.manifest.state)
                        LabeledContent("Route", value: detail.routeId.isEmpty ? "—" : detail.routeId)
                        LabeledContent("Orders", value: "\(detail.orderCount)")
                    }

                    if let next = ManifestLifecycleAction.next(for: detail.manifest.state) {
                        Section {
                            Button(acting ? "Applying…" : next.label) {
                                Task { await runLifecycle(next) }
                            }
                            .disabled(acting)
                        }
                    }

                    Section("Transfers") {
                        if detail.transfers.isEmpty {
                            Text("factory_portal.manifests._id_.text.no_transfers_on_this_manifest")
                                .foregroundStyle(.secondary)
                        } else {
                            ForEach(detail.transfers) { transfer in
                                Text(L10n.format("mobile_factory.ui.id_state", "\(transfer.id)", "\(transfer.state)"))
                                    .font(.footnote.monospaced())
                            }
                        }
                    }

                    Section("Transitions") {
                        if detail.transitions.isEmpty {
                            Text("mobile_factory.ui.no_transitions_recorded")
                                .foregroundStyle(.secondary)
                        } else {
                            ForEach(detail.transitions) { transition in
                                VStack(alignment: .leading, spacing: 2) {
                                    Text(transition.action).font(.subheadline.bold())
                                    Text(L10n.format("mobile_factory.ui.fromstate_tostate", "\(transition.fromState)", "\(transition.toState)"))
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                }
                            }
                        }
                    }
                }
            }
        }
        .navigationTitle("warehouse_portal.manifests.text.manifest")
        .task(id: manifestId) { await load() }
        .onAppear {
            realtimeClient.connect(
                onStateChange: { _ in },
                onEvent: { event in
                    guard event.eventType == .manifestUpdate else { return }
                    Task { await load(silent: true) }
                },
                onReconnect: {
                    acting = false
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
            detail = try await FactoryService.manifestDetail(id: manifestId)
        } catch {
            self.error = error.localizedDescription
        }
        if !silent {
            loading = false
        }
    }

    @MainActor
    private func runLifecycle(_ action: ManifestLifecycleAction) async {
        acting = true
        do {
            _ = try await FactoryService.transitionManifest(id: manifestId, action: action.rawValue)
            await load()
        } catch {
            self.error = error.localizedDescription
        }
        acting = false
    }
}
