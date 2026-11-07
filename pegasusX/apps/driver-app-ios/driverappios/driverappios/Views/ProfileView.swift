//
//  ProfileView.swift
//  driverappios
//

import SwiftUI
import SwiftData

/// Tab 3: "Profile" — driver config, ride history, offline verifier access
struct ProfileView: View {
    @Environment(\.modelContext) private var modelContext
    @Bindable var vm: FleetViewModel
    @State private var showOfflineVerifier = false
    @State private var showSyncQueue = false
    @State private var showEndSession = false
    @State private var showFleetDVIR = false
    @State private var showMidRouteRescue = false
    @State private var showYardGate = false
    @State private var pendingCount = 0

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: LabTheme.s24) {
                // MARK: - Header
                ProfileHeader()

                // MARK: - Driver Card
                DriverCard(vm: vm)

                // MARK: - Quick Actions
                QuickActions(
                    onOfflineVerifier: { showOfflineVerifier = true },
                    onSyncQueue: { showSyncQueue = true },
                    onEndSession: { showEndSession = true },
                    onFleetDVIR: { showFleetDVIR = true },
                    onMidRouteRescue: { showMidRouteRescue = true },
                    onYardGate: { showYardGate = true },
                    pendingCount: pendingCount
                )

                // MARK: - Ride History
                HistorySection(vm: vm)

                // MARK: - Stats
                StatsSection(vm: vm)
            }
            .padding(.horizontal, LabTheme.s16)
            .padding(.bottom, 20)
            .labReadableWidth()
        }
        .background(LabTheme.bg)
        .sheet(isPresented: $showFleetDVIR) {
            FleetDVIRView(onBack: { showFleetDVIR = false })
                .presentationDetents([.large])
                .presentationDragIndicator(.visible)
        }
        .sheet(isPresented: $showMidRouteRescue) {
            MidRouteRescueView(onBack: { showMidRouteRescue = false })
                .presentationDetents([.large])
                .presentationDragIndicator(.visible)
        }
        .sheet(isPresented: $showYardGate) {
            YardGateView(onBack: { showYardGate = false })
                .presentationDetents([.large])
                .presentationDragIndicator(.visible)
        }
        .sheet(isPresented: $showOfflineVerifier) {
            OfflineVerifierView(modelContext: modelContext)
                .presentationDetents([.large])
                .presentationDragIndicator(.visible)
        }
        .sheet(isPresented: $showSyncQueue) {
            SyncQueueView()
                .presentationDetents([.large])
                .presentationDragIndicator(.visible)
        }
        .sheet(isPresented: $showEndSession) {
            EndSessionView(vm: vm)
                .presentationDetents([.medium, .large])
                .presentationDragIndicator(.hidden)
        }
        .task {
            DriverOfflineQueue.shared.attach(container: modelContext.container)
            pendingCount = DriverOfflineQueue.shared.pendingCount()
            await vm.loadEarningsAndHistory()
        }
        .onChange(of: showSyncQueue) { _, open in
            if !open { pendingCount = DriverOfflineQueue.shared.pendingCount() }
        }
    }
}

#Preview {
    ProfileView(vm: FleetViewModel())
        .modelContainer(for: [OfflineDelivery.self, QueuedDriverAction.self], inMemory: true)
}
