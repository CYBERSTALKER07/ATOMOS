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
    @State private var profileVM = ProfileViewModel()

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: LabTheme.s24) {
                // MARK: - Header
                ProfileHeader()

                // MARK: - Driver Card
                DriverCard(vm: vm)

                // MARK: - Quick Actions
                QuickActions(
                    onOfflineVerifier: { profileVM.showOfflineVerifier = true },
                    onSyncQueue: { profileVM.showSyncQueue = true },
                    onEndSession: { profileVM.showEndSession = true },
                    onFleetDVIR: { profileVM.showFleetDVIR = true },
                    onMidRouteRescue: { profileVM.showMidRouteRescue = true },
                    onYardGate: { profileVM.showYardGate = true },
                    onSettings: { profileVM.showSettings = true },
                    pendingCount: profileVM.pendingCount
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
        .scrollDismissesKeyboard(.interactively)
        .background(LabTheme.bg)
        .sheet(isPresented: $profileVM.showFleetDVIR) {
            FleetDVIRView(onBack: { profileVM.showFleetDVIR = false })
                .presentationDetents([.large])
                .presentationDragIndicator(.visible)
        }
        .sheet(isPresented: $profileVM.showMidRouteRescue) {
            MidRouteRescueView(onBack: { profileVM.showMidRouteRescue = false })
                .presentationDetents([.large])
                .presentationDragIndicator(.visible)
        }
        .sheet(isPresented: $profileVM.showYardGate) {
            YardGateView(onBack: { profileVM.showYardGate = false })
                .presentationDetents([.large])
                .presentationDragIndicator(.visible)
        }
        .sheet(isPresented: $profileVM.showOfflineVerifier) {
            OfflineVerifierView(modelContext: modelContext)
                .presentationDetents([.large])
                .presentationDragIndicator(.visible)
        }
        .sheet(isPresented: $profileVM.showSyncQueue) {
            SyncQueueView()
                .presentationDetents([.large])
                .presentationDragIndicator(.visible)
        }
        .sheet(isPresented: $profileVM.showSettings) {
            DriverLanguageSettingsSheet(onDone: { profileVM.showSettings = false })
                .presentationDetents([.medium])
                .presentationDragIndicator(.visible)
        }
        .sheet(isPresented: $profileVM.showEndSession) {
            EndSessionView(vm: vm)
                .presentationDetents([.medium, .large])
                .presentationDragIndicator(.hidden)
        }
        .task {
            profileVM.attach(container: modelContext.container)
            await vm.loadEarningsAndHistory()
        }
        .onChange(of: profileVM.showSyncQueue) { _, open in
            if !open { profileVM.refreshPendingCount() }
        }
    }
}

private struct DriverLanguageSettingsSheet: View {
    let onDone: () -> Void
    @State private var localeManager = DriverLocaleManager.shared

    var body: some View {
        NavigationStack {
            List {
                Section {
                    ForEach(DriverLocaleManager.supportedLanguages) { option in
                        Button {
                            Haptics.light()
                            localeManager.setLanguage(option.code)
                        } label: {
                            HStack {
                                VStack(alignment: .leading, spacing: 2) {
                                    Text(option.displayName)
                                        .font(.headline)
                                        .foregroundStyle(LabTheme.fg)
                                    Text(option.code.uppercased())
                                        .font(.caption.monospaced())
                                        .foregroundStyle(LabTheme.fgTertiary)
                                }
                                Spacer()
                                if localeManager.currentCode == option.code {
                                    Image(systemName: "checkmark.circle.fill")
                                        .foregroundStyle(LabTheme.success)
                                }
                            }
                            .frame(minHeight: 44)
                        }
                    }
                } header: {
                    Text(L10n.string("mobile_driver.ui.language"))
                }
            }
            .navigationTitle(L10n.string("mobile_driver.ui.settings"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button(L10n.string("common.done")) {
                        onDone()
                    }
                }
            }
        }
    }
}

#Preview {
    ProfileView(vm: FleetViewModel())
        .modelContainer(for: [OfflineDelivery.self, QueuedDriverAction.self], inMemory: true)
}
