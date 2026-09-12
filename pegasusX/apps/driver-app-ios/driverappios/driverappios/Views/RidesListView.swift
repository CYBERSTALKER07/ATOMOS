//
//  RidesListView.swift
//  driverappios
//

import CoreLocation
import SwiftUI

/// Tab 2: "Rides" — upcoming routes with full order details, premium card UI
struct RidesListView: View {
    @Bindable var vm: FleetViewModel
    @State private var driverSocketState = DriverSocketState.shared
    @State private var loadingMode = false
    @State private var showEarlyComplete = false
    @State private var isRequestingEarlyComplete = false
    @State private var earlyCompleteMessage: String?
    var onRequestEarlyComplete: ((String, String) -> Void)?

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                ManifestHeader(loadingMode: $loadingMode, pendingMissionsCount: vm.pendingMissions.count)

                // MARK: - Mission Cards

                // LEO: Ghost Stop Prevention banner
                if vm.awaitingSeal {
                    ExplainStatusBanner(
                        explain: vm.gateExplain,
                        fallbackTitle: "AWAITING SEAL",
                        fallbackDetail: "Manifest is \(vm.manifestState ?? "not sealed"). Payloader must complete loading and seal before you can depart."
                    )
                    .padding(.horizontal, LabTheme.s16)
                    .padding(.bottom, 8)
                }

                RemainingStopsStepper(stops: RemainingStops.remaining(vm.orders)) { id in
                    if let mission = vm.pendingMissions.first(where: { $0.id == id }) {
                        vm.selectMission(mission)
                    }
                }
                .padding(.horizontal, LabTheme.s16)
                .padding(.bottom, 8)

                if vm.isLoadingMissions {
                    ManifestLoadingView()
                } else if vm.pendingMissions.isEmpty {
                    ManifestEmptyView()
                } else {
                    let displayMissions = loadingMode ? Array(vm.pendingMissions.reversed()) : vm.pendingMissions
                    LazyVStack(spacing: 14) {
                        ForEach(Array(displayMissions.enumerated()), id: \.element.id) { index, mission in
                            HStack(spacing: 8) {
                                RideCard(
                                    mission: mission,
                                    index: index,
                                    loadSeqLabel: loadSeqLabel(for: index, total: displayMissions.count),
                                    location: vm.location,
                                    onSelect: { vm.selectMission(mission) }
                                )

                                if !loadingMode && displayMissions.count > 1 {
                                    VStack(spacing: 4) {
                                        Button {
                                            vm.moveOrder(from: index, to: index - 1)
                                        } label: {
                                            Image(systemName: "chevron.up")
                                                .font(.system(size: 14, weight: .bold))
                                                .foregroundStyle(index == 0 ? LabTheme.fgTertiary : LabTheme.fg)
                                                .frame(width: 32, height: 32)
                                                .background(LabTheme.fg.opacity(0.06), in: Circle())
                                        }
                                        .disabled(index == 0)

                                        Button {
                                            vm.moveOrder(from: index, to: index + 1)
                                        } label: {
                                            Image(systemName: "chevron.down")
                                                .font(.system(size: 14, weight: .bold))
                                                .foregroundStyle(index == displayMissions.count - 1 ? LabTheme.fgTertiary : LabTheme.fg)
                                                .frame(width: 32, height: 32)
                                                .background(LabTheme.fg.opacity(0.06), in: Circle())
                                        }
                                        .disabled(index == displayMissions.count - 1)
                                    }
                                }
                            }
                        }
                    }
                    .padding(.horizontal, LabTheme.s16)
                }
            }
            .padding(.bottom, 20)
        }
        .scrollIndicators(.hidden)
        .background(LabTheme.bg)
        .refreshable {
            await vm.loadMissions()
        }
        .overlay(alignment: .bottomTrailing) {
            // Edge 27: Early Complete FAB
            if !vm.pendingMissions.isEmpty {
                Button {
                    showEarlyComplete = true
                } label: {
                    Image(systemName: "exclamationmark.triangle.fill")
                        .font(.system(size: 20, weight: .bold))
                        .foregroundStyle(.white)
                        .frame(width: 56, height: 56)
                        .background(LabTheme.destructive, in: Circle())
                        .shadow(color: LabTheme.destructive.opacity(0.3), radius: 8, y: 4)
                }
                .padding(24)
            }
        }
        .sheet(isPresented: $showEarlyComplete) {
            EarlyCompleteSheet(onConfirm: { reason, note in
                showEarlyComplete = false
                if let onRequestEarlyComplete {
                    onRequestEarlyComplete(reason, note)
                } else {
                    Task { await submitEarlyComplete(reason: reason, note: note) }
                }
            })
            .presentationDetents([.medium])
        }
        .onChange(of: driverSocketState.reconnectEpoch) { _, _ in
            let hadInFlight = isRequestingEarlyComplete
            if isRequestingEarlyComplete {
                isRequestingEarlyComplete = false
                earlyCompleteMessage = DriverReconnectRecovery.hint
            }
            Task {
                await DriverReconnectRecovery.recoverInFlight(wasInFlight: hadInFlight)
                await vm.loadMissions(silent: true)
            }
        }
        .onChange(of: driverSocketState.eventSequence) { _, _ in
            vm.handleSocketEvent(driverSocketState.lastEvent)
        }
        .safeAreaInset(edge: .bottom) {
            if let earlyCompleteMessage {
                Text(earlyCompleteMessage)
                    .font(.footnote)
                    .foregroundStyle(.secondary)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 8)
                    .background(.bar)
            }
        }
    }

    private func submitEarlyComplete(reason: String, note: String) async {
        isRequestingEarlyComplete = true
        earlyCompleteMessage = nil
        defer { isRequestingEarlyComplete = false }
        do {
            _ = try await APIClient.shared.requestEarlyComplete(reason: reason, note: note)
            earlyCompleteMessage = "Early complete request submitted."
            await vm.loadMissions()
        } catch {
            earlyCompleteMessage = error.localizedDescription
        }
    }

    // MARK: - Load Sequence Label

    private func loadSeqLabel(for index: Int, total: Int) -> String? {
        guard loadingMode else { return nil }
        if index == 0 { return "Load #\(index + 1) · Back of Truck" }
        if index == total - 1 { return "Load #\(index + 1) · By the Doors" }
        return "Load #\(index + 1)"
    }
}

#Preview {
    RidesListView(vm: FleetViewModel())
}
