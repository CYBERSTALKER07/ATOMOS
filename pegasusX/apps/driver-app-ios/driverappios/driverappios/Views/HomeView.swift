//
//  HomeView.swift
//  driverappios
//

import SwiftUI
import CoreLocation
import MapKit

/// Default landing page — driver summary, today's stats, quick actions
struct HomeView: View {
    @Bindable var vm: FleetViewModel
    let onOpenMap: () -> Void
    var onHandoffNavigate: ((HandoffDestination) -> Void)? = nil

    @State private var appeared = false
    @State private var showNotificationInbox = false
    @State private var showSupplyTransfers = false
    @State private var handoffAlertMessage: String?
    @State private var pulseEvents: [PulseEvent] = []
    @State private var pulseLoading = true
    @State private var driverSocketState = DriverSocketState.shared
    @State private var showRescueSheet = false

    var body: some View {
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

                PulseStrip(events: pulseEvents, loading: pulseLoading)
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
        .task {
            await vm.loadMissions()
            await loadPulse()
        }
        .onChange(of: driverSocketState.eventSequence) { _, _ in
            vm.handleSocketEvent(driverSocketState.lastEvent)
        }
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
        defer { pulseLoading = false }
        do {
            let response = try await APIClient.shared.getPulse()
            pulseEvents = response.events
        } catch {
            pulseEvents = []
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
