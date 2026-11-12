import SwiftUI

struct PayloadOverrideView: View {
    @State private var vm = PayloadOverrideViewModel()

    var body: some View {
        @Bindable var vm = vm

        Group {
            if vm.loading && vm.manifests.isEmpty {
                FactoryLoadingView(
                    title: "Loading payload override",
                    message: "Fetching live loading manifests."
                )
            } else if let error = vm.error {
                FactoryErrorView(message: error) {
                    Task { await vm.load() }
                }
            } else if vm.manifests.isEmpty {
                FactoryStateView(
                    kind: .empty,
                    headline: "No loading manifests",
                    message: "Payload override requires manifests in LOADING state."
                )
            } else {
                ScrollView {
                    VStack(spacing: LabTheme.spacingLG) {
                        if let processingError = vm.processingError {
                            Text(processingError)
                                .font(.caption)
                                .foregroundStyle(.red)
                                .padding()
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .background(Color.red.opacity(0.1))
                                .clipShape(RoundedRectangle(cornerRadius: LabTheme.radiusMD, style: .continuous))
                                .accessibilityLabel("Override error: \(processingError)")
                        }

                        ForEach(vm.manifests) { manifest in
                            OverrideManifestCard(
                                manifest: manifest,
                                isProcessing: vm.isProcessing,
                                manifestsCount: vm.manifests.count,
                                onCancelManifest: {
                                    vm.promptCancelManifest(manifest)
                                },
                                onMoveTransfer: { transfer in
                                    vm.promptMove(manifestId: manifest.id, transfer: transfer)
                                },
                                onReleaseTransfer: { transfer in
                                    vm.promptRelease(manifestId: manifest.id, transfer: transfer)
                                }
                            )
                        }
                    }
                    .padding()
                }
                .scrollDismissesKeyboard(.interactively)
            }
        }
        .background(LabTheme.background)
        .navigationTitle("portal.nav.payload_override")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") {
                    PegasusHaptics.selection()
                    Task { await vm.load() }
                }
                .frame(minWidth: 44, minHeight: 44)
                .disabled(vm.isProcessing)
                .accessibilityLabel("Refresh payload override")
            }
        }
        .task {
            vm.startRealtime()
            await vm.load()
        }
        .onDisappear {
            vm.stopRealtime()
        }
        .sheet(item: $vm.moveCandidate) { candidate in
            MoveTransferSheet(
                candidate: candidate,
                manifests: vm.manifests,
                selectedTargetManifestId: $vm.selectedTargetManifestId,
                isProcessing: vm.isProcessing,
                onCancel: { vm.dismissMove() },
                onMove: { targetId in
                    Task { await vm.moveTransfer(candidate, targetManifestId: targetId) }
                }
            )
        }
        .alert("Remove Transfer", isPresented: $vm.showReleaseAlert) {
            Button("mobile_factory.ui.keep", role: .cancel) {
                vm.clearReleaseAlert()
            }
            Button("mobile_factory.ui.release", role: .destructive) {
                Task { await vm.confirmReleaseTransfer() }
            }
        } message: {
            Text("mobile_factory.ui.release_transfer_back_to_approved_so_it_can_be_reassigned")
        }
        .alert("Cancel Manifest", isPresented: $vm.showCancelManifestAlert) {
            Button("mobile_factory.ui.keep", role: .cancel) {
                vm.clearCancelManifestAlert()
            }
            Button("mobile_factory.ui.cancel_manifest_2", role: .destructive) {
                Task { await vm.confirmCancelManifest() }
            }
        } message: {
            Text("mobile_factory.ui.cancel_manifest_and_return_all_linked_transfers_to_approved")
        }
    }
}
