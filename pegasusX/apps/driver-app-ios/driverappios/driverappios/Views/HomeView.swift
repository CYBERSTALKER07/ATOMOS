//
//  HomeView.swift
//  driverappios
//

import CoreLocation
import MapKit
import SwiftData
import SwiftUI

/// Default landing page — driver summary, today's stats, quick actions
struct HomeView: View {
    @Bindable var vm: FleetViewModel
    let onOpenMap: () -> Void
    var onHandoffNavigate: ((HandoffDestination) -> Void)? = nil
    @Environment(\.modelContext) private var modelContext
    @Environment(\.horizontalSizeClass) private var sizeClass

    @State private var homeVM = DriverHomeViewModel()
    @State private var driverSocketState = DriverSocketState.shared

    var body: some View {
        @Bindable var homeVM = homeVM
        HStack(alignment: .top, spacing: 0) {
            if sizeClass == .regular {
                RemainingStopsStepper(stops: RemainingStops.remaining(vm.orders)) { _ in
                    onHandoffNavigate?(.manifestList)
                }
                .frame(width: 320)
                .padding(.top, 24)
            }
            ScrollView {
                VStack(alignment: .leading, spacing: 20) {
                    // MARK: - Greeting
                    HStack(alignment: .top) {
                        VStack(alignment: .leading, spacing: 6) {
                            Text(vm.hasActiveRoute ? "MISSION ACTIVE" : homeVM.greetingText())
                                .font(.system(.caption2, design: .monospaced, weight: .black))
                                .foregroundStyle(vm.hasActiveRoute ? LabTheme.live : LabTheme.fgTertiary)
                                .tracking(1.4)

                            Text(vm.driverName)
                                .font(.system(.largeTitle, weight: .bold))
                                .foregroundStyle(LabTheme.fg)
                        }

                        Spacer()

                        Button {
                            Haptics.light()
                            homeVM.showNotificationInbox = true
                        } label: {
                            Image(systemName: "bell")
                                .font(.system(.body, weight: .medium))
                                .foregroundStyle(LabTheme.fgSecondary)
                                .frame(width: 44, height: 44)
                                .contentShape(Rectangle())
                        }
                        .accessibilityLabel("Notifications")
                        .padding(.top, 4)
                    }
                    .padding(.horizontal, LabTheme.s4)
                    .padding(.top, sizeClass == .regular ? 20 : 54)

                    PulseStrip(
                        events: homeVM.pulseEvents,
                        loading: homeVM.pulseLoading,
                        error: homeVM.pulseError
                    )
                    .padding(.horizontal, LabTheme.s4)
                    .staggeredAppear(index: 0)

                    // MARK: - Status Chip
                    HStack(spacing: 10) {
                        statusChip(
                            icon: "truck.box.fill",
                            label: vm.licensePlate,
                            active: true
                        )
                        if vm.isReturning {
                            statusChip(
                                icon: "house.fill",
                                label: "Returning",
                                active: true,
                                tint: LabTheme.warning
                            )
                        } else {
                            statusChip(
                                icon: vm.hasActiveRoute ? "antenna.radiowaves.left.and.right" : "moon.zzz.fill",
                                label: vm.hasActiveRoute ? "On Route" : "Idle",
                                active: vm.hasActiveRoute
                            )
                        }
                    }
                    .staggeredAppear(index: 0)

                    // MARK: - Vehicle Info Card
                    if !vm.truckId.isEmpty && vm.truckId != "—" {
                        VehicleInfoCard(vm: vm)
                            .staggeredAppear(index: 1)
                    }

                    // MARK: - Factory supply
                    if TokenStore.shared.isFactoryScopedDriver {
                        FactorySupplyCard(showSupplyTransfers: $homeVM.showSupplyTransfers)
                            .staggeredAppear(index: 2)
                    }

                    // MARK: - Transit Control Card
                    TransitControlCard(vm: vm)
                        .staggeredAppear(index: TokenStore.shared.isFactoryScopedDriver ? 3 : 2)

                    if homeVM.offlinePending > 0 {
                        Button {
                            Haptics.light()
                            homeVM.showSyncQueue = true
                        } label: {
                            Text("Offline sync queue · \(homeVM.offlinePending)")
                                .font(.system(.caption, weight: .semibold))
                                .foregroundStyle(LabTheme.warning)
                                .frame(maxWidth: .infinity, minHeight: 44, alignment: .leading)
                                .padding(.horizontal, 12)
                                .labCard()
                        }
                        .buttonStyle(.plain)
                        .accessibilityLabel("Offline sync queue, \(homeVM.offlinePending) pending actions")
                    }

                    if sizeClass != .regular {
                        RemainingStopsStepper(stops: RemainingStops.remaining(vm.orders)) { _ in
                            onHandoffNavigate?(.manifestList)
                        }
                    }

                    ManifestBulletMeter(state: vm.manifestState, usedVU: nil, maxVU: vm.maxVolumeVU)
                    FieldMoneyStrip(counts: MoneyHealthCounts.from(orders: vm.orders))
                    fieldEarningsRow
                    fieldHistoryPeek

                    // MARK: - Today Summary Card
                    TodaySummaryCard(
                        vm: vm,
                        onSelectPending: { onHandoffNavigate?(.manifestList) },
                        onSelectDone: { onHandoffNavigate?(.manifestList) },
                        onSelectRevenue: { onHandoffNavigate?(.home) }
                    )
                    .staggeredAppear(index: TokenStore.shared.isFactoryScopedDriver ? 4 : 3)

                    // MARK: - Open Map CTA
                    MapButton(pendingCount: vm.pendingMissions.count, onOpenMap: onOpenMap)
                        .staggeredAppear(index: TokenStore.shared.isFactoryScopedDriver ? 5 : 4)

                    // MARK: - Quick Actions
                    QuickActionsSection(
                        showRescueSheet: $homeVM.showRescueSheet,
                        onScanQR: { homeVM.showQRScanner = true },
                        onOfflineVerify: { homeVM.showOfflineVerifier = true }
                    )
                    .staggeredAppear(index: TokenStore.shared.isFactoryScopedDriver ? 6 : 5)

                    // MARK: - Recent Activity
                    RecentActivitySection(vm: vm, onSelectMission: { homeVM.selectedRecentMission = $0 })
                        .staggeredAppear(index: TokenStore.shared.isFactoryScopedDriver ? 7 : 6)
                }
                .padding(.horizontal, LabTheme.s16)
                .padding(.bottom, 20)
                .labReadableWidth()
            }
            .scrollIndicators(.hidden)
            .scrollDismissesKeyboard(.interactively)
        }
        .background(LabTheme.bg)
        .sheet(isPresented: $homeVM.showNotificationInbox) {
            DriverNotificationInboxView { link in
                homeVM.handleHandoffLink(link, onNavigate: onHandoffNavigate)
            }
            .presentationDetents([.large])
            .presentationDragIndicator(.visible)
        }
        .alert("Handoff", isPresented: $homeVM.showHandoffAlert) {
            Button("OK", role: .cancel) { homeVM.clearHandoffAlert() }
        } message: {
            Text(homeVM.handoffAlertMessage ?? "")
        }
        .sheet(isPresented: $homeVM.showSupplyTransfers) {
            SupplyTransfersView()
                .presentationDetents([.large])
                .presentationDragIndicator(.visible)
        }
        .sheet(isPresented: $homeVM.showRescueSheet) {
            RequestRescueSheet()
                .presentationDetents([.medium])
                .presentationDragIndicator(.visible)
        }
        .sheet(isPresented: $homeVM.showSyncQueue) {
            SyncQueueView()
                .presentationDetents([.large])
        }
        .sheet(isPresented: $homeVM.showQRScanner) {
            QRScannerView(
                onValidated: { _, _ in homeVM.showQRScanner = false },
                onCancel: { homeVM.showQRScanner = false }
            )
            .presentationDetents([.large])
            .presentationDragIndicator(.visible)
        }
        .sheet(isPresented: $homeVM.showOfflineVerifier) {
            OfflineVerifierView(modelContext: modelContext)
                .presentationDetents([.large])
                .presentationDragIndicator(.visible)
        }
        .sheet(item: $homeVM.selectedRecentMission) { mission in
            MissionDetailSheet(
                mission: mission,
                driverLocation: vm.displayLocation ?? vm.location,
                isInRange: true,
                distance: 0,
                onScan: {},
                onCorrection: {}
            )
            .presentationDetents([.medium, .large])
            .presentationDragIndicator(.visible)
        }
        .task {
            await vm.loadMissions()
            await homeVM.loadPulse()
            homeVM.refreshOfflinePending(container: modelContext.container)
        }
        .onChange(of: driverSocketState.eventSequence) { _, _ in
            vm.handleSocketEvent(driverSocketState.lastEvent)
        }
    }

    private var fieldEarningsRow: some View {
        let code = packCurrency(MarketPackStore.pack)
        let today = vm.earnings?.todayMinor
        return VStack(alignment: .leading, spacing: 6) {
            Text("EARNINGS")
                .font(.system(.caption2, design: .monospaced, weight: .heavy))
                .foregroundStyle(LabTheme.fgTertiary)
            if vm.earnings == nil {
                Text("unavailable")
                    .font(.system(.footnote, weight: .medium))
                    .foregroundStyle(LabTheme.fgTertiary)
            } else if code.isEmpty {
                Text(today == 0 ? "empty" : "\(today ?? 0)")
                    .font(.system(.footnote, weight: .semibold))
                    .foregroundStyle(LabTheme.fg)
            } else {
                Text(today == 0 ? "empty" : "\(today ?? 0) \(code)")
                    .font(.system(.footnote, weight: .semibold))
                    .foregroundStyle(LabTheme.fg)
            }
        }
        .padding(LabTheme.s16)
        .labCard()
        .accessibilityElement(children: .combine)
    }

    private var fieldHistoryPeek: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("HISTORY · 30D")
                .font(.system(.caption2, design: .monospaced, weight: .heavy))
                .foregroundStyle(LabTheme.fgTertiary)
            if vm.historyRows.isEmpty {
                Text("empty")
                    .font(.system(.footnote, weight: .medium))
                    .foregroundStyle(LabTheme.fgTertiary)
            } else {
                ForEach(vm.historyRows.prefix(3)) { row in
                    Text("\(row.orderId) · \(row.status)")
                        .font(.system(.caption, weight: .medium))
                        .foregroundStyle(LabTheme.fgSecondary)
                }
            }
        }
        .padding(LabTheme.s16)
        .labCard()
        .accessibilityElement(children: .combine)
    }

    private func statusChip(icon: String, label: String, active: Bool, tint: Color? = nil) -> some View {
        let chipColor = tint ?? (active ? LabTheme.fg : LabTheme.fgTertiary)
        return HStack(spacing: 6) {
            Image(systemName: icon)
                .font(.system(.caption2, weight: .black))
            Text(label.uppercased())
                .font(.system(.caption2, design: .monospaced, weight: .bold))
        }
        .foregroundStyle(chipColor)
        .padding(.horizontal, 12)
        .padding(.vertical, 8)
        .background {
            RoundedRectangle(cornerRadius: LabTheme.buttonRadius, style: .continuous)
                .fill(LabTheme.card)
                .stroke(LabTheme.separator.opacity(0.12), lineWidth: 1)
        }
        .accessibilityElement(children: .combine)
        .accessibilityLabel("Status: \(label)")
    }
}

// MARK: - Pulse Animation Modifier

struct PulseModifier: ViewModifier {
    @State private var isPulsing = false

    func body(content: Content) -> some View {
        content
            .opacity(isPulsing ? 0.3 : 1.0)
            .animation(.easeInOut(duration: 1.0).repeatForever(autoreverses: true), value: isPulsing)
            .onAppear { isPulsing = true }
    }
}

#Preview {
    HomeView(vm: FleetViewModel(), onOpenMap: {})
}
