import SwiftUI

private struct MoveCandidate {
    let sourceManifestId: String
    let transfer: ManifestTransfer
}

private struct CancelTransferCandidate {
    let manifestId: String
    let transfer: ManifestTransfer
}

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
                                overrideManifestCard(manifest)
                            }
                        }
                        .padding()
                    }
                }
            }
            .navigationTitle("Payload Override")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Refresh", systemImage: "arrow.clockwise") {
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
            // Move Transfer Sheet
            .sheet(item: Binding<MoveCandidate?>(
                get: { moveCandidate },
                set: { moveCandidate = $0 }
            )) { candidate in
                moveTransferSheet(candidate)
            }
            // Release Transfer Confirmation
            .alert("Remove Transfer", isPresented: Binding(
                get: { cancelTransferCandidate != nil },
                set: { if !$0 { cancelTransferCandidate = nil } }
            )) {
                Button("Keep", role: .cancel) {}
                Button("Release", role: .destructive) {
                    if let candidate = cancelTransferCandidate {
                        Task { await releaseTransfer(candidate) }
                    }
                }
            } message: {
                Text("Release transfer back to APPROVED so it can be reassigned.")
            }
            // Cancel Manifest Confirmation
            .alert("Cancel Manifest", isPresented: Binding(
                get: { cancelManifestCandidate != nil },
                set: { if !$0 { cancelManifestCandidate = nil } }
            )) {
                Button("Keep", role: .cancel) {}
                Button("Cancel Manifest", role: .destructive) {
                    if let manifest = cancelManifestCandidate {
                        Task { await cancelManifest(manifest) }
                    }
                }
            } message: {
                Text("Cancel manifest and return all linked transfers to APPROVED.")
            }
        }
    }

    @ViewBuilder
    private func overrideManifestCard(_ manifest: Manifest) -> some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingMD) {
            HStack {
                VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                    Text(manifest.truckPlate.isEmpty ? String(manifest.truckId.prefix(8)) : manifest.truckPlate)
                        .font(.headline)
                    Text("Manifest \(manifest.id.prefix(8))")
                        .font(.caption.monospaced())
                        .foregroundStyle(.secondary)
                }
                Spacer()
                Button("Cancel Manifest", role: .destructive) {
                    cancelManifestCandidate = manifest
                }
                .buttonStyle(.bordered)
                .disabled(isProcessing)
            }

            let capacity = manifest.maxCapacityVU > 0 ? manifest.maxCapacityVU : 1.0
            let progress = min(max(manifest.totalVolumeVU / capacity, 0.0), 1.0)
            
            ProgressView(value: progress)
                .tint(progress > 0.95 ? .red : .blue)
            
            HStack {
                metricBox("Volume", "\(Int(manifest.totalVolumeVU)) VU")
                metricBox("Capacity", "\(Int(manifest.maxCapacityVU)) VU")
                metricBox("Transfers", "\(manifest.transfers.count)")
            }
            
            if manifest.transfers.isEmpty {
                Text("No transfers are assigned to this manifest.")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
                    .padding()
                    .frame(maxWidth: .infinity)
                    .background(Color(uiColor: .secondarySystemBackground))
                    .cornerRadius(LabTheme.radiusSM)
            } else {
                ForEach(manifest.transfers) { transfer in
                    transferRow(transfer: transfer, manifestId: manifest.id)
                }
            }
        }
        .padding()
        .background(Color(uiColor: .secondarySystemBackground))
        .cornerRadius(LabTheme.radiusLG)
    }

    @ViewBuilder
    private func transferRow(transfer: ManifestTransfer, manifestId: String) -> some View {
        VStack(spacing: LabTheme.spacingSM) {
            HStack {
                VStack(alignment: .leading) {
                    Text(transfer.productName.isEmpty ? "Transfer \(transfer.id.prefix(8))" : transfer.productName)
                        .font(.subheadline.bold())
                    Text(transfer.id.prefix(8))
                        .font(.caption.monospaced())
                        .foregroundStyle(.secondary)
                }
                Spacer()
                FactoryStatusBadge(text: transfer.state)
            }
            
            HStack {
                metricBox("Qty", "\(transfer.quantity)")
                metricBox("Volume", "\(Int(transfer.volumeVU)) VU")
            }
            
            HStack {
                Button("Move") {
                    moveCandidate = MoveCandidate(sourceManifestId: manifestId, transfer: transfer)
                }
                .buttonStyle(.borderedProminent)
                .disabled(isProcessing || manifests.count <= 1)
                
                Button("Release") {
                    cancelTransferCandidate = CancelTransferCandidate(manifestId: manifestId, transfer: transfer)
                }
                .buttonStyle(.bordered)
                .disabled(isProcessing)
            }
        }
        .padding()
        .background(Color(uiColor: .systemBackground))
        .cornerRadius(LabTheme.radiusMD)
    }

    @ViewBuilder
    private func metricBox(_ label: String, _ value: String) -> some View {
        VStack(spacing: LabTheme.spacingXXS) {
            Text(value)
                .font(.headline)
            Text(label)
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, LabTheme.spacingSM)
        .background(Color(uiColor: .systemBackground))
        .cornerRadius(LabTheme.radiusSM)
    }

    @ViewBuilder
    private func moveTransferSheet(_ candidate: MoveCandidate) -> some View {
        NavigationStack {
            let targets = manifests.filter { $0.id != candidate.sourceManifestId }
            ResponsiveGridContentWrapper {
                ForEach(targets, selection: $selectedTargetManifestId) { target in
                VStack(alignment: .leading) {
                    Text(target.truckPlate.isEmpty ? String(target.truckId.prefix(8)) : target.truckPlate)
                        .font(.headline)
                    Text("\(Int(target.totalVolumeVU)) / \(Int(target.maxCapacityVU)) VU")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                }
                .tag(target.id)
            }
            .navigationTitle("Move Transfer")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { moveCandidate = nil }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Move") {
                        if let targetId = selectedTargetManifestId {
                            Task { await moveTransfer(candidate, targetManifestId: targetId) }
                        }
                    }
                    .disabled(selectedTargetManifestId == nil || isProcessing)
                }
            }
        }
        .presentationDetents([.medium])
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
