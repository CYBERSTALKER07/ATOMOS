import SwiftUI

struct MoveTransferCandidate: Identifiable {
    let sourceManifestID: String
    let transfer: ManifestTransfer

    var id: String { transfer.id }
}

struct CancelTransferCandidate: Identifiable {
    let manifestID: String
    let transfer: ManifestTransfer

    var id: String { transfer.id }
}

struct PayloadOverrideView: View {
    @Environment(\.scenePhase) private var scenePhase
    @State private var realtimeClient = FactoryRealtimeClient()
    @State private var manifests: [Manifest] = []
    @State private var loading = true
    @State private var error: String?
    @State private var actingKey: String?
    @State private var refreshing = false
    @State private var staleMessage: String?
    @State private var lastSyncedAt: Date?
    @State private var moveCandidate: MoveTransferCandidate?
    @State private var selectedTargetManifestID = ""
    @State private var cancelTransferCandidate: CancelTransferCandidate?
    @State private var cancelManifestCandidate: Manifest?

    private var runtimeStatus: String {
        if refreshing {
            return "Refreshing live manifests — last sync \(overrideSyncText(lastSyncedAt))"
        }

        if let staleMessage {
            return staleMessage
        }

        if lastSyncedAt != nil {
            return "Live sync active — last sync \(overrideSyncText(lastSyncedAt))"
        }

        return "Waiting for first sync"
    }

    var body: some View {
        NavigationStack {
            Group {
                if loading {
                    FactoryLoadingState(
                        title: "Loading override",
                        message: "Fetching active loading manifests for payload override."
                    )
                } else if let error {
                    FactoryErrorView(message: error) {
                        Task { await load() }
                    }
                } else if manifests.isEmpty {
                    FactoryStateView(
                        kind: .empty,
                        headline: "No Loading Manifests",
                        message: "Payload override becomes available when at least one manifest reaches loading."
                    )
                } else {
                    ScrollView {
                        VStack(alignment: .leading, spacing: LabTheme.spacingLG) {
                            OverrideSummaryCard(
                                manifests: manifests,
                                runtimeStatus: runtimeStatus,
                                stale: staleMessage != nil
                            )

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
            .navigationTitle("portal.nav.payload_override")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") {
                        Task { await load(background: !manifests.isEmpty) }
                    }
                    .labelStyle(.iconOnly)
                }
            }
            .task { await load() }
            .task {
                while !Task.isCancelled {
                    try? await Task.sleep(for: .seconds(30))
                    if actingKey == nil {
                        await load(background: true)
                    }
                }
            }
            .onChange(of: scenePhase) { _, newPhase in
                if newPhase == .active {
                    Task { await load(background: !manifests.isEmpty) }
                }
            }
            .onAppear {
                realtimeClient.connect(
                    onStateChange: { _ in },
                    onEvent: { event in
                        guard let eventType = event.eventType else { return }
                        guard eventType == .transferUpdate || eventType == .manifestUpdate else { return }
                        if actingKey == nil {
                            Task { await load(background: !manifests.isEmpty) }
                        }
                    }
                )
            }
            .onDisappear {
                realtimeClient.disconnect()
            }
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
                Button("mobile_factory.ui.release", role: .destructive) {
                    Task { await cancelTransfer(candidate: candidate) }
                }
                Button("mobile_factory.ui.keep", role: .cancel) { }
            } message: { candidate in
                Text(L10n.format("mobile_factory.ui.release_transfer_prefix_back_to_approved_so_it_can_be_reassigned", "\(candidate.transfer.id.prefix(8))"))
            }
            .alert(
                "Cancel manifest?",
                isPresented: Binding(
                    get: { cancelManifestCandidate != nil },
                    set: { if !$0 { cancelManifestCandidate = nil } }
                ),
                presenting: cancelManifestCandidate
            ) { manifest in
                Button("mobile_factory.ui.cancel_manifest", role: .destructive) {
                    Task { await cancelManifest(manifest) }
                }
                Button("mobile_factory.ui.keep", role: .cancel) { }
            } message: { manifest in
                Text(L10n.format("mobile_factory.ui.cancel_manifest_prefix_and_return_all_linked_transfers_to_approved", "\(manifest.id.prefix(8))"))
            }
        }
    }

    @MainActor
    private func load(background: Bool = false) async {
        if background {
            refreshing = true
        } else if manifests.isEmpty {
            loading = true
            error = nil
        }

        do {
            manifests = try await FactoryService.loadingManifests().manifests.filter { $0.state == "LOADING" }
            staleMessage = nil
            error = nil
            lastSyncedAt = Date()
        } catch {
            let message = error.localizedDescription
            if manifests.isEmpty {
                self.error = message
            } else {
                staleMessage = "Showing last synced manifests. \(message)"
            }
        }

        loading = false
        refreshing = false
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
            await load(background: true)
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
            await load(background: true)
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
            await load(background: true)
        } catch {
            self.error = error.localizedDescription
        }

        actingKey = nil
    }
}

struct OverrideSummaryCard: View {
    let manifests: [Manifest]
    let runtimeStatus: String
    let stale: Bool

    var body: some View {
        let transferCount = manifests.reduce(into: 0) { $0 += $1.transfers.count }

        VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
            Text("mobile_factory.ui.live_manifest_override")
                .font(.title2.bold())
            Text(L10n.format("mobile_factory.ui.count_loading_manifests_transfercount_transfers_available_for_rebalance_", "\(manifests.count)", "\(transferCount)"))
                .font(.body)
                .foregroundStyle(.secondary)
            Text(runtimeStatus)
                .font(.footnote.weight(.medium))
                .foregroundStyle(stale ? Color.red : .secondary)
                .frame(maxWidth: .infinity, alignment: .leading)
                .padding(.horizontal, LabTheme.spacingMD)
                .padding(.vertical, LabTheme.spacingSM)
                .background(stale ? Color.red.opacity(0.1) : LabTheme.tertiaryBackground, in: RoundedRectangle(cornerRadius: LabTheme.radiusMD))
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .labCard()
    }
}

struct OverrideManifestCard: View {
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
                    Text(L10n.format("mobile_factory.ui.manifest_prefix", "\(manifest.id.prefix(8))"))
                        .font(.footnote)
                        .foregroundStyle(.secondary)
                }
                Spacer()
                Button("mobile_factory.ui.cancel_manifest_2", action: onCancelManifest)
                    .buttonStyle(.bordered)
                    .disabled(actingKey != nil)
            }

            VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
                ProgressView(
                    value: min(1, manifest.totalVolumeVU / max(manifest.maxCapacityVU, 1))
                )
                HStack(spacing: LabTheme.spacingSM) {
                    OverrideMetric(label: "Volume", value: overrideVolumeLabel(manifest.totalVolumeVU))
                    OverrideMetric(label: "Capacity", value: overrideVolumeLabel(manifest.maxCapacityVU))
                    OverrideMetric(label: "Transfers", value: "\(manifest.transfers.count)")
                }
            }

            if manifest.transfers.isEmpty {
                Text("mobile_factory.ui.no_transfers_are_assigned_to_this_manifest")
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

struct OverrideTransferCard: View {
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
                FactoryStatusBadge(text: transfer.state)
            }

            HStack(spacing: LabTheme.spacingSM) {
                OverrideMetric(label: "Qty", value: "\(transfer.quantity)")
                OverrideMetric(label: "Volume", value: overrideVolumeLabel(transfer.volumeVU))
            }

            HStack(spacing: LabTheme.spacingSM) {
                Button("mobile_factory.ui.move", action: onMove)
                    .buttonStyle(.borderedProminent)
                    .frame(maxWidth: .infinity)
                    .disabled(!canMove || busy)

                Button("mobile_factory.ui.release", action: onRelease)
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

struct OverrideMetric: View {
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

struct MoveTransferSheet: View {
    let candidate: MoveTransferCandidate
    let manifests: [Manifest]
    @Binding var selectedTargetManifestID: String
    let isWorking: Bool
    let onMove: () -> Void
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            ResponsiveGridContentWrapper {
                Section("Select target manifest") {
                    if manifests.isEmpty {
                        Text("mobile_factory.ui.no_alternate_loading_manifest_is_available_right_now")
                            .foregroundStyle(.secondary)
                    } else {
                        ForEach(manifests) { manifest in
                            Button {
                                selectedTargetManifestID = manifest.id
                            } label: {
                                HStack {
                                    VStack(alignment: .leading, spacing: 2) {
                                        Text(manifest.truckPlate.isEmpty ? String(manifest.truckId.prefix(8)) : manifest.truckPlate)
                                        Text(L10n.format("mobile_factory.ui.overridevolumelabel_overridevolumelabel_2", "\(overrideVolumeLabel(manifest.totalVolumeVU))", "\(overrideVolumeLabel(manifest.maxCapacityVU))"))
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
            .navigationTitle("mobile_factory.ui.move_transfer")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("common.action.close") { dismiss() }
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

func overrideVolumeLabel(_ value: Double) -> String {
    value.rounded(.towardZero) == value ? "\(Int(value)) VU" : String(format: "%.1f VU", value)
}

func overrideSyncText(_ value: Date?) -> String {
    guard let value else { return "waiting" }
    return value.formatted(date: .omitted, time: .shortened)
}
