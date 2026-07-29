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
                header

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
    }

    // MARK: - Header

    private var header: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("DRIVER")
                .font(.system(size: 10, weight: .heavy, design: .monospaced))
                .foregroundStyle(LabTheme.fgTertiary)
                .tracking(1.2)

            Text("Profile")
                .font(.system(size: 28, weight: .bold))
                .foregroundStyle(LabTheme.fg)
        }
        .padding(.top, 60)
        .padding(.horizontal, LabTheme.s4)
    }


}

#Preview {
    ProfileView(vm: FleetViewModel())
        .modelContainer(for: OfflineDelivery.self, inMemory: true)
}
