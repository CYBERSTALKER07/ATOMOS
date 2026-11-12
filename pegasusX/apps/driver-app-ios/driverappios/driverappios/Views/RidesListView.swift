//
//  RidesListView.swift
//  driverappios
//

import CoreLocation
import SwiftUI

/// Tab 2: "Rides" — upcoming routes with full order details, premium card UI
struct RidesListView: View {
    @Bindable var vm: FleetViewModel
    @State private var ridesVM = RidesListViewModel()
    @State private var driverSocketState = DriverSocketState.shared
    var onRequestEarlyComplete: ((String, String) -> Void)?

    var body: some View {
        @Bindable var ridesVM = ridesVM
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                ManifestHeader(loadingMode: $ridesVM.loadingMode, pendingMissionsCount: vm.pendingMissions.count)

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
                    let displayMissions = ridesVM.loadingMode ? Array(vm.pendingMissions.reversed()) : vm.pendingMissions
                    LazyVStack(spacing: 14) {
                        ForEach(Array(displayMissions.enumerated()), id: \.element.id) { index, mission in
                            HStack(spacing: 8) {
                                RideCard(
                                    mission: mission,
                                    index: index,
                                    loadSeqLabel: ridesVM.loadSeqLabel(for: index, total: displayMissions.count),
                                    location: vm.location,
                                    onSelect: { vm.selectMission(mission) }
                                )

                                if !ridesVM.loadingMode && displayMissions.count > 1 {
                                    VStack(spacing: 4) {
                                        Button {
                                            Haptics.light()
                                            vm.moveOrder(from: index, to: index - 1)
                                        } label: {
                                            Image(systemName: "chevron.up")
                                                .font(.system(.subheadline, weight: .bold))
                                                .foregroundStyle(index == 0 ? LabTheme.fgTertiary : LabTheme.fg)
                                                .frame(width: 44, height: 44)
                                                .background(LabTheme.fg.opacity(0.06), in: Circle())
                                        }
                                        .accessibilityLabel("Move order up")
                                        .disabled(index == 0)

                                        Button {
                                            Haptics.light()
                                            vm.moveOrder(from: index, to: index + 1)
                                        } label: {
                                            Image(systemName: "chevron.down")
                                                .font(.system(.subheadline, weight: .bold))
                                                .foregroundStyle(index == displayMissions.count - 1 ? LabTheme.fgTertiary : LabTheme.fg)
                                                .frame(width: 44, height: 44)
                                                .background(LabTheme.fg.opacity(0.06), in: Circle())
                                        }
                                        .accessibilityLabel("Move order down")
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
            .labReadableWidth()
        }
        .scrollIndicators(.hidden)
        .scrollDismissesKeyboard(.interactively)
        .background(LabTheme.bg)
        .refreshable {
            await vm.loadMissions()
        }
        .overlay(alignment: .bottomTrailing) {
            if !vm.pendingMissions.isEmpty {
                Button {
                    Haptics.medium()
                    ridesVM.showEarlyComplete = true
                } label: {
                    Image(systemName: "exclamationmark.triangle.fill")
                        .font(.system(.title3, weight: .bold))
                        .foregroundStyle(.white)
                        .frame(width: 56, height: 56)
                        .background(LabTheme.destructive, in: Circle())
                        .shadow(color: LabTheme.destructive.opacity(0.3), radius: 8, y: 4)
                }
                .accessibilityLabel("Report issue or early complete")
                .padding(24)
            }
        }
        .sheet(isPresented: $ridesVM.showEarlyComplete) {
            EarlyCompleteSheet(onConfirm: { reason, note in
                ridesVM.showEarlyComplete = false
                if let onRequestEarlyComplete {
                    onRequestEarlyComplete(reason, note)
                } else {
                    Task {
                        await ridesVM.submitEarlyComplete(reason: reason, note: note) {
                            await vm.loadMissions()
                        }
                    }
                }
            })
            .presentationDetents([.medium])
        }
        .onChange(of: driverSocketState.reconnectEpoch) { _, _ in
            Task {
                await ridesVM.handleReconnect {
                    await vm.loadMissions(silent: true)
                }
            }
        }
        .onChange(of: driverSocketState.eventSequence) { _, _ in
            vm.handleSocketEvent(driverSocketState.lastEvent)
        }
        .safeAreaInset(edge: .bottom) {
            if let earlyCompleteMessage = ridesVM.earlyCompleteMessage {
                Text(earlyCompleteMessage)
                    .font(.footnote)
                    .foregroundStyle(.secondary)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 8)
                    .background(.bar)
            }
        }
    }
}

#Preview {
    RidesListView(vm: FleetViewModel())
}
