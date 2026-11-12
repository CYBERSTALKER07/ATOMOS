import SwiftUI

struct ManifestDetailView: View {
    let manifestId: String
    @State private var vm: ManifestDetailViewModel

    init(manifestId: String) {
        self.manifestId = manifestId
        _vm = State(initialValue: ManifestDetailViewModel(manifestId: manifestId))
    }

    var body: some View {
        Group {
            if vm.loading && vm.detail == nil {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let error = vm.error, vm.detail == nil {
                ContentUnavailableView {
                    Label("mobile_factory.ui.error", systemImage: "exclamationmark.triangle")
                } description: {
                    Text(error)
                } actions: {
                    Button("common.action.retry") {
                        PegasusHaptics.selection()
                        Task { await vm.load() }
                    }
                    .frame(minWidth: 44, minHeight: 44)
                }
            } else if let detail = vm.detail {
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
                            Button(vm.acting ? "Applying…" : next.label) {
                                Task { await vm.runLifecycle(next) }
                            }
                            .frame(minWidth: 44, minHeight: 44)
                            .disabled(vm.acting)
                            .accessibilityLabel("\(next.label) for manifest")
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
                                    .frame(minHeight: 44, alignment: .leading)
                                    .accessibilityLabel("Transfer \(transfer.id), status \(transfer.state)")
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
                                    Text(transition.action)
                                        .font(.subheadline.bold())
                                    Text(L10n.format("mobile_factory.ui.fromstate_tostate", "\(transition.fromState)", "\(transition.toState)"))
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                }
                                .frame(minHeight: 44, alignment: .leading)
                                .accessibilityElement(children: .combine)
                                .accessibilityLabel("\(transition.action): from \(transition.fromState) to \(transition.toState)")
                            }
                        }
                    }
                }
            }
        }
        .navigationTitle("warehouse_portal.manifests.text.manifest")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") {
                    PegasusHaptics.selection()
                    Task { await vm.load() }
                }
                .labelStyle(.iconOnly)
                .frame(minWidth: 44, minHeight: 44)
                .accessibilityLabel("Refresh manifest detail")
            }
        }
        .task(id: manifestId) {
            vm.startRealtime()
            await vm.load()
        }
        .onDisappear {
            vm.stopRealtime()
        }
    }
}
