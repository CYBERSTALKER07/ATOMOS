import SwiftUI

struct PayloadLoadView: View {
    @State private var vm = PayloadLoadViewModel()

    var body: some View {
        Group {
            if vm.loading && vm.manifests.isEmpty {
                FactoryLoadingState(
                    title: "Loading payloads",
                    message: "Factory drafts and loading manifests."
                )
            } else if let error = vm.error, vm.manifests.isEmpty {
                FactoryErrorView(message: error) {
                    Task { await vm.load() }
                }
            } else if vm.manifests.isEmpty {
                FactoryStateView(
                    kind: .empty,
                    headline: "No factory payloads",
                    message: "Dispatch creates FactoryTruckManifests drafts. Start loading and seal them here."
                )
            } else {
                List(vm.manifests) { manifest in
                    VStack(alignment: .leading, spacing: 8) {
                        HStack {
                            Text(manifest.id.prefix(16))
                                .font(.footnote.monospaced())
                            Spacer()
                            FactoryStatusBadge(text: manifest.state)
                        }
                        if let next = ManifestLifecycleAction.next(for: manifest.state),
                           next == .startLoading || next == .seal {
                            Button(vm.actingId == manifest.id ? "Applying…" : next.label) {
                                Task { await vm.run(manifest, next) }
                            }
                            .buttonStyle(.borderedProminent)
                            .frame(minWidth: 44, minHeight: 44, alignment: .leading)
                            .disabled(vm.actingId != nil)
                            .accessibilityLabel("\(next.label) for manifest \(manifest.id.prefix(8))")
                        }
                    }
                    .padding(.vertical, 4)
                    .accessibilityElement(children: .contain)
                }
                .scrollDismissesKeyboard(.interactively)
            }
        }
        .background(LabTheme.background)
        .navigationTitle("Payload / Load")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") {
                    PegasusHaptics.selection()
                    Task { await vm.load() }
                }
                .labelStyle(.iconOnly)
                .frame(minWidth: 44, minHeight: 44)
                .accessibilityLabel("Refresh payloads")
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
