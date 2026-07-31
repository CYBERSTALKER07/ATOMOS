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
    @State private var showEndSession = false

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
                    onEndSession: { showEndSession = true }
                )

                // MARK: - Ride History
                HistorySection(vm: vm)

                // MARK: - Stats
                StatsSection(vm: vm)
            }
            .padding(.horizontal, LabTheme.s16)
            .padding(.bottom, 20)
        }
        .background(LabTheme.bg)
        .sheet(isPresented: $showOfflineVerifier) {
            OfflineVerifierView(modelContext: modelContext)
                .presentationDetents([.large])
                .presentationDragIndicator(.visible)
        }
        .sheet(isPresented: $showEndSession) {
            EndSessionView(vm: vm)
                .presentationDetents([.medium, .large])
                .presentationDragIndicator(.hidden)
        }
        .task {
            await vm.loadEarningsAndHistory()
        }
    }
}

#Preview {
    ProfileView(vm: FleetViewModel())
        .modelContainer(for: OfflineDelivery.self, inMemory: true)
}
