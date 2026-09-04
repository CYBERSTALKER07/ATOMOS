//
//  HomeView.swift
//  driverappios
//

import SwiftUI
import SwiftData
import CoreLocation
import MapKit

/// Default landing page — driver summary, today's stats, quick actions
struct HomeView: View {
    @Bindable var vm: FleetViewModel
    let onOpenMap: () -> Void
    var onHandoffNavigate: ((HandoffDestination) -> Void)? = nil
    @Environment(\.modelContext) private var modelContext
    @Environment(\.horizontalSizeClass) private var sizeClass

    @State private var appeared = false
    @State private var showNotificationInbox = false
    @State private var showSupplyTransfers = false
    @State private var handoffAlertMessage: String?
    @State private var pulseEvents: [PulseEvent] = []
    @State private var pulseLoading = true
    @State private var pulseError: String?
    @State private var driverSocketState = DriverSocketState.shared
    @State private var showRescueSheet = false
    @State private var showSyncQueue = false
    @State private var offlinePending = 0

    var body: some View {
        HStack(alignment: .top, spacing: 0) {
            if sizeClass == .regular {
                RemainingStopsStepper(stops: RemainingStops.remaining(vm.orders)) { id in
                    _ = id
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
                        Text(vm.hasActiveRoute ? "MISSION ACTIVE" : greetingText)
                            .font(.system(size: 10, weight: .black, design: .monospaced)) // Tactical weight
                            .foregroundStyle(vm.hasActiveRoute ? LabTheme.live : LabTheme.fgTertiary)
                            .tracking(1.4) // Increased tracking

                        Text(vm.driverName)
                            .font(.system(size: 32, weight: .bold)) // Slightly larger
                            .foregroundStyle(LabTheme.fg)
                    }

                    Spacer()

                    Button { showNotificationInbox = true } label: {
                        Image(systemName: "bell")
                            .font(.system(size: 18, weight: .medium))
                            .foregroundStyle(LabTheme.fgSecondary)
                    }
                    .padding(.top, 8)
                }
                .padding(.horizontal, LabTheme.s4)
                .padding(.top, 60)

                PulseStrip(events: pulseEvents, loading: pulseLoading, error: pulseError)
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
                    FactorySupplyCard(showSupplyTransfers: $showSupplyTransfers)
                        .staggeredAppear(index: 2)
                }

                // MARK: - Transit Control Card
                TransitControlCard(vm: vm)
                    .staggeredAppear(index: TokenStore.shared.isFactoryScopedDriver ? 3 : 2)

                if offlinePending > 0 {
                    Button { showSyncQueue = true } label: {
                        Text("Offline sync queue · \(offlinePending)")
                            .font(.system(size: 12, weight: .semibold))
                            .foregroundStyle(LabTheme.warning)
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .padding(12)
                            .labCard()
                    }
                    .buttonStyle(.plain)
                }

                if sizeClass != .regular {
                    RemainingStopsStepper(stops: RemainingStops.remaining(vm.orders)) { id in
                        _ = id
                        onHandoffNavigate?(.manifestList)
                    }
                }

                ManifestBulletMeter(state: vm.manifestState, usedVU: nil, maxVU: vm.maxVolumeVU)
                FieldMoneyStrip(counts: MoneyHealthCounts.from(orders: vm.orders))
                fieldEarningsRow
                fieldHistoryPeek

                // MARK: - Today Summary Card
                TodaySummaryCard(vm: vm)
                    .staggeredAppear(index: TokenStore.shared.isFactoryScopedDriver ? 4 : 3)

                // MARK: - Open Map CTA
                MapButton(pendingCount: vm.pendingMissions.count, onOpenMap: onOpenMap)
                    .staggeredAppear(index: TokenStore.shared.isFactoryScopedDriver ? 5 : 4)

                // MARK: - Quick Actions
                QuickActionsSection(showRescueSheet: $showRescueSheet)
                    .staggeredAppear(index: TokenStore.shared.isFactoryScopedDriver ? 6 : 5)

                // MARK: - Recent Activity
                RecentActivitySection(vm: vm)
                    .staggeredAppear(index: TokenStore.shared.isFactoryScopedDriver ? 7 : 6)
            }
            .padding(.horizontal, LabTheme.s16)
            .padding(.bottom, 20)
        }
        .scrollIndicators(.hidden)
        }
        .background(LabTheme.bg)
        .sheet(isPresented: $showNotificationInbox) {
            DriverNotificationInboxView { link in
                handleHandoffLink(link)
            }
                .presentationDetents([.large])
                .presentationDragIndicator(.visible)
        }
        .alert("Handoff", isPresented: Binding(
            get: { handoffAlertMessage != nil },
            set: { if !$0 { handoffAlertMessage = nil } }
        )) {
            Button("OK", role: .cancel) { handoffAlertMessage = nil }
        } message: {
            Text(handoffAlertMessage ?? "")
        }
        .sheet(isPresented: $showSupplyTransfers) {
            SupplyTransfersView()
                .presentationDetents([.large])
                .presentationDragIndicator(.visible)
        }
        .sheet(isPresented: $showRescueSheet) {
            RequestRescueSheet()
                .presentationDetents([.medium])
                .presentationDragIndicator(.visible)
        }
        .sheet(isPresented: $showSyncQueue) {
            SyncQueueView()
                .presentationDetents([.large])
        }
        .task {
            await vm.loadMissions()
            await loadPulse()
            DriverOfflineQueue.shared.attach(container: modelContext.container)
            offlinePending = DriverOfflineQueue.shared.pendingCount()
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
                .font(.system(size: 11, weight: .heavy, design: .monospaced))
                .foregroundStyle(LabTheme.fgTertiary)
            if vm.earnings == nil {
                Text("unavailable")
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(LabTheme.fgTertiary)
            } else if code.isEmpty {
                Text(today == 0 ? "empty" : "\(today ?? 0)")
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundStyle(LabTheme.fg)
            } else {
                Text(today == 0 ? "empty" : "\(today ?? 0) \(code)")
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundStyle(LabTheme.fg)
            }
        }
        .padding(LabTheme.s16)
        .labCard()
    }

    private var fieldHistoryPeek: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("HISTORY · 30D")
                .font(.system(size: 11, weight: .heavy, design: .monospaced))
                .foregroundStyle(LabTheme.fgTertiary)
            if vm.historyRows.isEmpty {
                Text("empty")
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(LabTheme.fgTertiary)
            } else {
                ForEach(vm.historyRows.prefix(3)) { row in
                    Text("\(row.orderId) · \(row.status)")
                        .font(.system(size: 12, weight: .medium))
                        .foregroundStyle(LabTheme.fgSecondary)
                }
            }
        }
        .padding(LabTheme.s16)
        .labCard()
    }

    private func handleHandoffLink(_ link: String) {
        switch HandoffPathResolver.resolve(link) {
        case .home:
            onHandoffNavigate?(.home)
        case .fleetMap:
            onHandoffNavigate?(.fleetMap)
        case .manifestList, .manifestDetail:
            onHandoffNavigate?(.manifestList)
        case .orderDetail:
            onHandoffNavigate?(.manifestList)
        case .unresolved:
            handoffAlertMessage = "Open in portal — no native route for this link"
        }
    }

    private func loadPulse() async {
        pulseLoading = true
        pulseError = nil
        defer { pulseLoading = false }
        do {
            let response = try await APIClient.shared.getPulse()
            let result = PulseHonesty.apply(ok: true, incoming: response.events, previous: pulseEvents)
            pulseEvents = result.events
            pulseError = result.error
        } catch {
            let result = PulseHonesty.apply(ok: false, incoming: nil, previous: pulseEvents)
            pulseEvents = result.events
            pulseError = result.error
        }
    }

    // MARK: - Greeting

    private var greetingText: String {
        let hour = Calendar.current.component(.hour, from: Date())
        switch hour {
        case 5..<12:  return "GOOD MORNING"
        case 12..<17: return "GOOD AFTERNOON"
        case 17..<21: return "GOOD EVENING"
        default:      return "GOOD NIGHT"
        }
    }

    // MARK: - Status Chip

    private func statusChip(icon: String, label: String, active: Bool, tint: Color? = nil) -> some View {
        let chipColor = tint ?? (active ? LabTheme.fg : LabTheme.fgTertiary)
        return HStack(spacing: 6) {
            Image(systemName: icon)
                .font(.system(size: 11, weight: .black)) // Tactical bold
            Text(label.uppercased()) // Constant uppercase
                .font(.system(size: 11, weight: .bold, design: .monospaced))
        }
        .foregroundStyle(chipColor)
        .padding(.horizontal, 12)
        .padding(.vertical, 8)
        .background {
            RoundedRectangle(cornerRadius: LabTheme.buttonRadius, style: .continuous) // Using token
                .fill(LabTheme.card)
                .stroke(LabTheme.separator.opacity(0.12), lineWidth: 1) // Modern stroke
        }
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
