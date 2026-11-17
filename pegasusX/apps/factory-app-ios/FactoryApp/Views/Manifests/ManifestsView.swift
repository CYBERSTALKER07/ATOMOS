import SwiftUI

struct ManifestsView: View {
    var showsCloseButton: Bool = false
    @Environment(\.dismiss) private var dismiss
    @State private var vm = ManifestsViewModel()

    var body: some View {
        Group {
            if vm.loading && vm.manifests.isEmpty {
                FactoryLoadingView(
                    title: "Loading manifests",
                    message: "Fetching outbound manifest lifecycle states."
                )
            } else if let error = vm.error {
                FactoryErrorView(message: error) {
                    Task { await vm.load() }
                }
            } else if vm.manifests.isEmpty {
                FactoryStateView(
                    kind: .empty,
                    headline: "No manifests",
                    message: "Dispatch transfers to create a manifest draft."
                )
            } else {
                List {
                    Section {
                        FactorySectionHeader(
                            title: "Outbound manifests",
                            subtitle: "\(vm.manifests.count) manifests in the factory queue"
                        )
                        .listRowInsets(EdgeInsets(top: 8, leading: 0, bottom: 8, trailing: 0))
                        .listRowBackground(Color.clear)
                    }

                    Section {
                        ForEach(vm.manifests) { manifest in
                            NavigationLink {
                                ManifestDetailView(manifestId: manifest.id)
                            } label: {
                                ManifestRow(manifest: manifest)
                            }
                        }
                    }
                }
                .scrollDismissesKeyboard(.interactively)
            }
        }
        .navigationTitle("portal.nav.manifests")
        .toolbar {
            if showsCloseButton {
                ToolbarItem(placement: .topBarLeading) {
                    Button("common.action.close", systemImage: "xmark") {
                        PegasusHaptics.selection()
                        dismiss()
                    }
                    .labelStyle(.iconOnly)
                    .frame(minWidth: 44, minHeight: 44)
                    .accessibilityLabel("Close manifests")
                }
            }
            ToolbarItem(placement: .topBarTrailing) {
                Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") {
                    PegasusHaptics.selection()
                    Task { await vm.load() }
                }
                .labelStyle(.iconOnly)
                .frame(minWidth: 44, minHeight: 44)
                .accessibilityLabel("Refresh manifests")
            }
        }
        .task {
            vm.startRealtime()
            await vm.load()
        }
        .onDisappear {
            vm.stopRealtime()
        }
    }
}

struct ManifestRow: View {
    let manifest: Manifest

    var body: some View {
        let plate = manifest.truckPlate.isEmpty ? String(manifest.truckId.prefix(8)) : manifest.truckPlate

        VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
            Text(plate)
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
        .frame(minHeight: 44)
        .accessibilityElement(children: .combine)
        .accessibilityLabel("Manifest \(manifest.id.prefix(8)), truck \(plate), status \(manifest.state), \(Int(manifest.totalVolumeVU)) volume units")
    }
}
