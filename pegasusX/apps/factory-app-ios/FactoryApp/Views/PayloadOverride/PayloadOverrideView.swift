import SwiftUI

struct PayloadOverrideView: View {
    @State private var manifests: [Manifest] = []
    @State private var loading = true
    @State private var error: String?
    @State private var realtimeClient = FactoryRealtimeClient()
    
    @State private var moveCandidate: MoveCandidate?
    @State private var selectedTargetManifestId: String?
    
    @State private var cancelTransferCandidate: CancelTransferCandidate?
    @State private var cancelManifestCandidate: Manifest?
    
    @State private var isProcessing = false
    @State private var processingError: String?

    var body: some View {
        NavigationStack {
            Group {
                if loading && manifests.isEmpty {
                    FactoryLoadingView(
                        title: "Loading payload override",
                        message: "Fetching live loading manifests."
                    )
                } else if let error {
                    FactoryErrorView(message: error) {
                        Task { await load() }
                    }
                } else if manifests.isEmpty {
                    FactoryStateView(
                        kind: .empty,
                        headline: "No loading manifests",
                        message: "Payload override requires manifests in LOADING state."
                    )
                } else {
                    ScrollView {
                        VStack(spacing: LabTheme.spacingLG) {
                            if let processingError {
                                Text(processingError)
                                    .font(.caption)
                                    .foregroundStyle(.red)
                                    .padding()
                                    .frame(maxWidth: .infinity, alignment: .leading)
                                    .background(Color.red.opacity(0.1))
                                    .cornerRadius(LabTheme.radiusMD)
                            }
                            
                            ForEach(manifests) { manifest in
                                OverrideManifestCard(
                                    manifest: manifest,
                                    isProcessing: isProcessing,
                                    manifestsCount: manifests.count,
                                    onCancelManifest: {
                                        cancelManifestCandidate = manifest
                                    },
                                    onMoveTransfer: { transfer in
                                        moveCandidate = MoveCandidate(sourceManifestId: manifest.id, transfer: transfer)
                                    },
                                    onReleaseTransfer: { transfer in
                                        cancelTransferCandidate = CancelTransferCandidate(manifestId: manifest.id, transfer: transfer)
                                    }
                                )
                            }
                        }
                        .padding()
                    }
                }
            }
            .navigationTitle("portal.nav.payload_override")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") {
                        Task { await load() }
                    }
                    .disabled(isProcessing)
                }
            }
            .task { await load() }
            .onAppear {
                realtimeClient.connect(
                    onStateChange: { _ in },
                    onEvent: { event in
                        guard event.type.hasPrefix("MANIFEST_") || event.type.hasPrefix("TRANSFER_") || event.type.hasPrefix("WAREHOUSE_TRANSFER_") else { return }
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
            // Move Transfer Sheet
            .sheet(item: Binding<MoveCandidate?>(
                get: { moveCandidate },
                set: { moveCandidate = $0 }
            )) { candidate in
                MoveTransferSheet(
                    candidate: candidate,
                    manifests: manifests,
                    selectedTargetManifestId: $selectedTargetManifestId,
                    isProcessing: isProcessing,
                    onCancel: { moveCandidate = nil },
                    onMove: { targetId in
                        Task { await moveTransfer(candidate, targetManifestId: targetId) }
                    }
                )
            }
            // Release Transfer Confirmation
            .alert("Remove Transfer", isPresented: Binding(
                get: { cancelTransferCandidate != nil },
                set: { if !$0 { cancelTransferCandidate = nil } }
            )) {
                Button("mobile_factory.ui.keep", role: .cancel) {}
                Button("mobile_factory.ui.release", role: .destructive) {
                    if let candidate = cancelTransferCandidate {
                        Task { await releaseTransfer(candidate) }
                    }
                }
            } message: {
                Text("mobile_factory.ui.release_transfer_back_to_approved_so_it_can_be_reassigned")
            }
            // Cancel Manifest Confirmation
            .alert("Cancel Manifest", isPresented: Binding(
                get: { cancelManifestCandidate != nil },
                set: { if !$0 { cancelManifestCandidate = nil } }
            )) {
                Button("mobile_factory.ui.keep", role: .cancel) {}
                Button("mobile_factory.ui.cancel_manifest_2", role: .destructive) {
                    if let manifest = cancelManifestCandidate {
                        Task { await cancelManifest(manifest) }
                    }
                }
            } message: {
                Text("mobile_factory.ui.cancel_manifest_and_return_all_linked_transfers_to_approved")
            }
        }
    }



    @MainActor
    private func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        do {
            let response = try await FactoryService.loadingManifests()
            manifests = response.manifests
        } catch {
            self.error = error.localizedDescription
        }
        if !silent { loading = false }
    }

    @MainActor
    private func moveTransfer(_ candidate: MoveCandidate, targetManifestId: String) async {
        isProcessing = true
        processingError = nil
        do {
            _ = try await FactoryService.rebalanceManifestTransfer(
                sourceManifestId: candidate.sourceManifestId,
                targetManifestId: targetManifestId,
                transferId: candidate.transfer.id
            )
            moveCandidate = nil
            selectedTargetManifestId = nil
            await load(silent: true)
        } catch {
            processingError = error.localizedDescription
        }
        isProcessing = false
    }

    @MainActor
    private func releaseTransfer(_ candidate: CancelTransferCandidate) async {
        isProcessing = true
        processingError = nil
        do {
            _ = try await FactoryService.cancelManifestTransfer(
                manifestId: candidate.manifestId,
                transferId: candidate.transfer.id
            )
            cancelTransferCandidate = nil
            await load(silent: true)
        } catch {
            processingError = error.localizedDescription
        }
        isProcessing = false
    }

    @MainActor
    private func cancelManifest(_ manifest: Manifest) async {
        isProcessing = true
        processingError = nil
        do {
            _ = try await FactoryService.cancelManifest(manifestId: manifest.id)
            cancelManifestCandidate = nil
            await load(silent: true)
        } catch {
            processingError = error.localizedDescription
        }
        isProcessing = false
    }
}

extension MoveCandidate: Identifiable {
    var id: String { transfer.id }
}
