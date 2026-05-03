import SwiftUI

private struct MoveTransferCandidate: Identifiable {
    let sourceManifestID: String
    let transfer: ManifestTransfer

    var id: String { transfer.id }
}

private struct CancelTransferCandidate: Identifiable {
    let manifestID: String
    let transfer: ManifestTransfer

    var id: String { transfer.id }
}

struct PayloadOverrideView: View {
    @State private var manifests: [Manifest] = []
    @State private var loading = true
    @State private var error: String?
    @State private var actingKey: String?
    @State private var moveCandidate: MoveTransferCandidate?
    @State private var selectedTargetManifestID = ""
    @State private var cancelTransferCandidate: CancelTransferCandidate?
    @State private var cancelManifestCandidate: Manifest?

    var body: some View {
        NavigationStack {
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
                        Button("Retry") { load() }
                    }
                } else if manifests.isEmpty {
                    ContentUnavailableView(
                        "No Loading Manifests",
                        systemImage: "shippingbox",
                        description: Text("Payload override becomes available when at least one manifest reaches loading.")
                    )
                } else {
                    ScrollView {
                        VStack(alignment: .leading, spacing: LabTheme.spacingLG) {
                            OverrideSummaryCard(manifests: manifests)

                            LazyVStack(spacing: LabTheme.spacingSM) {
                                ForEach(Array(manifests.enumerated()), id: \.element.id) { index, manifest in
                                    OverrideManifestCard(
                                        manifest: manifest,
                                        canMoveTransfers: manifests.contains(where: { $0.id != manifest.id }),
                                        actingKey: actingKey,
                                        onMove: { transfer in moveCandidate = MoveTransferCandidate(sourceManifestID: manifest.id, transfer: transfer) },
                                        onRelease: { transfer in cancelTransferCandidate = CancelTransferCandidate(manifestID: manifest.id, transfer: transfer) },
                                        onCancelManifest: { cancelManifestCandidate = manifest }
                                    )
                                    .staggeredAppear(index: index)
                                }
                            }
                        }
                        .padding()
                    }
                }
            }
            .background(LabTheme.background)
            .navigationTitle("Payload Override")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Refresh", systemImage: "arrow.clockwise") {
                        load()
                    }
                    .labelStyle(.iconOnly)
                }
            }
            .task { load() }
            .sheet(item: $moveCandidate, onDismiss: { selectedTargetManifestID = "" }) { candidate in
                MoveTransferSheet(
                    candidate: candidate,
                    manifests: manifests.filter { $0.id != candidate.sourceManifestID },
                    selectedTargetManifestID: $selectedTargetManifestID,
                    isWorking: actingKey == candidate.id,
                    onMove: {
                        Task { await rebalance(candidate: candidate, targetManifestID: selectedTargetManifestID) }
                    }
                )
            }
            .alert(
                "Release transfer?",
                isPresented: Binding(
                    get: { cancelTransferCandidate != nil },
                    set: { if !$0 { cancelTransferCandidate = nil } }
                ),
                presenting: cancelTransferCandidate
            ) { candidate in
                Button("Release", role: .destructive) {
                    Task { await cancelTransfer(candidate: candidate) }
                }
                Button("Keep", role: .cancel) { }
            } message: { candidate in
                Text("Release transfer \(candidate.transfer.id.prefix(8)) back to APPROVED so it can be reassigned.")
            }
            .alert(
                "Cancel manifest?",
                isPresented: Binding(
                    get: { cancelManifestCandidate != nil },
                    set: { if !$0 { cancelManifestCandidate = nil } }
                ),
                presenting: cancelManifestCandidate
            ) { manifest in
                Button("Cancel manifest", role: .destructive) {
                    Task { await cancelManifest(manifest) }
                }
                Button("Keep", role: .cancel) { }
            } message: { manifest in
                Text("Cancel manifest \(manifest.id.prefix(8)) and return all linked transfers to APPROVED.")
            }
        }
    }

    private func load() {
        loading = true
        error = nil

        Task {
            do {
                manifests = try await FactoryService.loadingManifests().manifests.filter { $0.state == "LOADING" }
            } catch {
                self.error = error.localizedDescription
            }

            loading = false
        }
    }

    @MainActor
    private func rebalance(candidate: MoveTransferCandidate, targetManifestID: String) async {
        guard !targetManifestID.isEmpty else { return }
        actingKey = candidate.id

        do {
            _ = try await FactoryService.rebalanceManifestTransfer(
                sourceManifestId: candidate.sourceManifestID,
                targetManifestId: targetManifestID,
                transferId: candidate.transfer.id
            )
            moveCandidate = nil
            selectedTargetManifestID = ""
            manifests = try await FactoryService.loadingManifests().manifests.filter { $0.state == "LOADING" }
        } catch {
            self.error = error.localizedDescription
        }

        actingKey = nil
    }

    @MainActor
    private func cancelTransfer(candidate: CancelTransferCandidate) async {
        actingKey = candidate.id

        do {
            _ = try await FactoryService.cancelManifestTransfer(
                manifestId: candidate.manifestID,
                transferId: candidate.transfer.id
            )
            cancelTransferCandidate = nil
            manifests = try await FactoryService.loadingManifests().manifests.filter { $0.state == "LOADING" }
        } catch {
            self.error = error.localizedDescription
        }

        actingKey = nil
    }

    @MainActor
    private func cancelManifest(_ manifest: Manifest) async {
        actingKey = manifest.id

        do {
            _ = try await FactoryService.cancelManifest(manifestId: manifest.id)
            cancelManifestCandidate = nil
            manifests = try await FactoryService.loadingManifests().manifests.filter { $0.state == "LOADING" }
        } catch {
            self.error = error.localizedDescription
        }

        actingKey = nil
    }
}

private struct OverrideSummaryCard: View {
    let manifests: [Manifest]

    var body: some View {
        let transferCount = manifests.reduce(into: 0) { $0 += $1.transfers.count }

        VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
            Text("Live manifest override")
                .font(.title2.bold())
            Text("\(manifests.count) loading manifests, \(transferCount) transfers available for rebalance or release.")
                .font(.body)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .labCard()
    }
}

private struct OverrideManifestCard: View {
    let manifest: Manifest
    let canMoveTransfers: Bool
    let actingKey: String?
    let onMove: (ManifestTransfer) -> Void
    let onRelease: (ManifestTransfer) -> Void
    let onCancelManifest: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingMD) {
            HStack(alignment: .top, spacing: LabTheme.spacingMD) {
                VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                    Text(manifest.truckPlate.isEmpty ? String(manifest.truckId.prefix(8)) : manifest.truckPlate)
                        .font(.subheadline.bold())
                    Text("Manifest \(manifest.id.prefix(8))")
                        .font(.footnote)
                        .foregroundStyle(.secondary)
                }
                Spacer()
                Button("Cancel Manifest", action: onCancelManifest)
                    .buttonStyle(.bordered)
                    .disabled(actingKey != nil)
            }

            VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
                ProgressView(
                    value: min(1, manifest.totalVolumeVU / max(manifest.maxCapacityVU, 1))
                )
                HStack(spacing: LabTheme.spacingSM) {
                    OverrideMetric(label: "Volume", value: volumeLabel(manifest.totalVolumeVU))
                    OverrideMetric(label: "Capacity", value: volumeLabel(manifest.maxCapacityVU))
                    OverrideMetric(label: "Transfers", value: "\(manifest.transfers.count)")
                }
            }

            if manifest.transfers.isEmpty {
                Text("No transfers are assigned to this manifest.")
                    .font(.footnote)
                    .foregroundStyle(.secondary)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(LabTheme.spacingMD)
                    .background(LabTheme.secondaryBackground, in: RoundedRectangle(cornerRadius: LabTheme.radiusMD))
            } else {
                LazyVStack(spacing: LabTheme.spacingSM) {
                    ForEach(manifest.transfers) { transfer in
                        OverrideTransferCard(
                            transfer: transfer,
                            canMove: canMoveTransfers,
                            busy: actingKey == transfer.id,
                            onMove: { onMove(transfer) },
                            onRelease: { onRelease(transfer) }
                        )
                    }
                }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .labCard()
    }
}

private struct OverrideTransferCard: View {
    let transfer: ManifestTransfer
    let canMove: Bool
    let busy: Bool
    let onMove: () -> Void
    let onRelease: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingMD) {
            HStack(alignment: .top, spacing: LabTheme.spacingMD) {
                VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                    Text(transfer.productName.isEmpty ? "Transfer \(transfer.id.prefix(8))" : transfer.productName)
                        .font(.subheadline.bold())
                    Text(transfer.id.prefix(8))
                        .font(.footnote)
                        .foregroundStyle(.secondary)
                }
                Spacer()
                Text(transfer.state)
                    .font(.footnote.bold())
                    .padding(.horizontal, LabTheme.spacingSM)
                    .padding(.vertical, LabTheme.spacingXS)
                    .background(LabTheme.fill, in: Capsule())
            }

            HStack(spacing: LabTheme.spacingSM) {
                OverrideMetric(label: "Qty", value: "\(transfer.quantity)")
                OverrideMetric(label: "Volume", value: volumeLabel(transfer.volumeVU))
            }

            HStack(spacing: LabTheme.spacingSM) {
                Button("Move", action: onMove)
                    .buttonStyle(.borderedProminent)
                    .frame(maxWidth: .infinity)
                    .disabled(!canMove || busy)

                Button("Release", action: onRelease)
                    .buttonStyle(.bordered)
                    .frame(maxWidth: .infinity)
                    .disabled(busy)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(LabTheme.spacingMD)
        .background(LabTheme.secondaryBackground, in: RoundedRectangle(cornerRadius: LabTheme.radiusMD))
    }
}

private struct OverrideMetric: View {
    let label: String
    let value: String

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
            Text(value)
                .font(.subheadline.bold())
            Text(label)
                .font(.footnote)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(LabTheme.spacingMD)
        .background(LabTheme.tertiaryBackground, in: RoundedRectangle(cornerRadius: LabTheme.radiusMD))
    }
}

private struct MoveTransferSheet: View {
    let candidate: MoveTransferCandidate
    let manifests: [Manifest]
    @Binding var selectedTargetManifestID: String
    let isWorking: Bool
    let onMove: () -> Void
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            List {
                Section("Select target manifest") {
                    if manifests.isEmpty {
                        Text("No alternate loading manifest is available right now.")
                            .foregroundStyle(.secondary)
                    } else {
                        ForEach(manifests) { manifest in
                            Button {
                                selectedTargetManifestID = manifest.id
                            } label: {
                                HStack {
                                    VStack(alignment: .leading, spacing: 2) {
                                        Text(manifest.truckPlate.isEmpty ? String(manifest.truckId.prefix(8)) : manifest.truckPlate)
                                        Text("\(volumeLabel(manifest.totalVolumeVU)) / \(volumeLabel(manifest.maxCapacityVU))")
                                            .font(.caption)
                                            .foregroundStyle(.secondary)
                                    }
                                    Spacer()
                                    if selectedTargetManifestID == manifest.id {
                                        Image(systemName: "checkmark.circle.fill")
                                            .foregroundStyle(.tint)
                                    }
                                }
                            }
                            .buttonStyle(.plain)
                        }
                    }
                }
            }
            .navigationTitle("Move Transfer")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Close") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button(isWorking ? "Moving…" : "Move") {
                        onMove()
                    }
                    .disabled(selectedTargetManifestID.isEmpty || isWorking)
                }
            }
        }
    }
}
