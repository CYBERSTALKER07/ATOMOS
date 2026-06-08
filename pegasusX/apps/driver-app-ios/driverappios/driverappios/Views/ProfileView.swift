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
                driverCard

                // MARK: - Quick Actions
                quickActions

                // MARK: - Ride History
                historySection

                // MARK: - Stats
                statsSection
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

    // MARK: - Driver Card

    private var driverCard: some View {
        VStack(spacing: 16) {
            HStack(spacing: 14) {
                // Avatar
                ZStack {
                    Circle()
                        .fill(LabTheme.fg)
                        .frame(width: 52, height: 52)

                    Text(String(vm.driverName.prefix(1)))
                        .font(.system(size: 22, weight: .bold))
                        .foregroundStyle(LabTheme.buttonFg)
                }

                VStack(alignment: .leading, spacing: 4) {
                    Text(vm.driverName)
                        .font(.system(size: 17, weight: .bold))
                        .foregroundStyle(LabTheme.fg)

                    Text(vm.driverId)
                        .font(.system(size: 12, weight: .semibold, design: .monospaced))
                        .foregroundStyle(LabTheme.fgSecondary)
                }

                Spacer()

                StatusPill(
                    label: vm.hasActiveRoute ? "ON DUTY" : "IDLE",
                    color: vm.hasActiveRoute ? LabTheme.success : LabTheme.fgSecondary
                )
            }

            // Info grid
            HStack(spacing: 12) {
                infoTile("Truck", vm.truckId, icon: "truck.box.fill")
                infoTile("Plate", vm.licensePlate, icon: "car.fill")
                infoTile("Capacity", "\(Int(vm.maxVolumeVU)) VU", icon: "shippingbox.fill")
                infoTile("Done", "\(vm.completedIds.count)", icon: "checkmark.circle.fill")
            }
        }
        .padding(LabTheme.s20)
        .labCard()
    }

    // MARK: - Info Tile

    private func infoTile(_ label: String, _ value: String, icon: String) -> some View {
        VStack(spacing: 6) {
            Image(systemName: icon)
                .font(.system(size: 14))
                .foregroundStyle(LabTheme.fgSecondary)

            Text(value)
                .font(.system(size: 14, weight: .bold, design: .monospaced))
                .foregroundStyle(LabTheme.fg)

            Text(label)
                .font(.system(size: 10, weight: .medium))
                .foregroundStyle(LabTheme.fgTertiary)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 12)
        .background(LabTheme.fg.opacity(0.03), in: .rect(cornerRadius: 12))
    }

    // MARK: - Quick Actions

    private var quickActions: some View {
        VStack(spacing: 10) {
            actionRow(icon: "shield.checkered", title: "Offline Verifier", subtitle: "Hash manifest protocol") {
                showOfflineVerifier = true
            }
            actionRow(icon: "arrow.triangle.2.circlepath", title: "Sync Queue", subtitle: "Upload pending deliveries") {
                Haptics.light()
            }
            actionRow(icon: "gearshape.fill", title: "Settings", subtitle: "App configuration") {
                Haptics.light()
            }
            actionRow(icon: "rectangle.portrait.and.arrow.right", title: "End Session", subtitle: "Go offline and sign out", destructive: true) {
                Haptics.medium()
                showEndSession = true
            }
        }
    }

    private func actionRow(icon: String, title: String, subtitle: String, destructive: Bool = false, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            HStack(spacing: 14) {
                Image(systemName: icon)
                    .font(.system(size: 15, weight: .semibold))
                    .foregroundStyle(destructive ? LabTheme.destructive : LabTheme.fg)
                    .frame(width: 36, height: 36)
                    .background((destructive ? LabTheme.destructive : LabTheme.fg).opacity(0.06), in: .rect(cornerRadius: 10))

                VStack(alignment: .leading, spacing: 2) {
                    Text(title)
                        .font(.system(size: 15, weight: .semibold))
                        .foregroundStyle(destructive ? LabTheme.destructive : LabTheme.fg)
                    Text(subtitle)
                        .font(.system(size: 12, weight: .medium))
                        .foregroundStyle(LabTheme.fgSecondary)
                }

                Spacer()

                Image(systemName: "chevron.right")
                    .font(.system(size: 11, weight: .bold))
                    .foregroundStyle(LabTheme.fgTertiary)
            }
            .padding(LabTheme.s16)
            .labCard()
        }
        .buttonStyle(.pressable)
    }

    // MARK: - History Section

    private var historySection: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                Text("Ride History")
                    .font(.system(size: 17, weight: .bold))
                    .foregroundStyle(LabTheme.fg)

                Spacer()

                Text("\(vm.completedMissions.count) rides")
                    .font(.system(size: 12, weight: .medium, design: .monospaced))
                    .foregroundStyle(LabTheme.fgTertiary)
            }
            .padding(.horizontal, LabTheme.s8)

            if vm.completedMissions.isEmpty {
                VStack(spacing: 10) {
                    Image(systemName: "clock.arrow.circlepath")
                        .font(.system(size: 24))
                        .foregroundStyle(LabTheme.fgTertiary)

                    Text("No completed rides yet")
                        .font(.subheadline.weight(.medium))
                        .foregroundStyle(LabTheme.fgSecondary)
                }
                .frame(maxWidth: .infinity)
                .padding(.vertical, 30)
                .labCard()
            } else {
                ForEach(Array(vm.completedMissions.enumerated()), id: \.element.id) { index, mission in
                    historyRow(mission, index: index)
                }
            }
        }
    }

    private func historyRow(_ mission: Mission, index: Int) -> some View {
        HStack(spacing: 14) {
            ZStack {
                Circle()
                    .fill(LabTheme.success.opacity(0.15))
                    .frame(width: 36, height: 36)

                Image(systemName: "checkmark")
                    .font(.system(size: 13, weight: .bold))
                    .foregroundStyle(LabTheme.success)
            }

            VStack(alignment: .leading, spacing: 3) {
                Text(mission.order_id)
                    .font(.system(size: 13, weight: .bold, design: .monospaced))
                    .foregroundStyle(LabTheme.fg)

                Text("\(mission.gateway) · \(mission.amount.formattedAmount)")
                    .font(.system(size: 11, weight: .medium))
                    .foregroundStyle(LabTheme.fgSecondary)
            }

            Spacer()

            StatusPill(label: "DELIVERED", color: LabTheme.success)
        }
        .padding(LabTheme.s16)
        .labCard()
        .staggeredAppear(index: index)
    }

    // MARK: - Stats Section

    private var statsSection: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("Session Stats")
                .font(.system(size: 17, weight: .bold))
                .foregroundStyle(LabTheme.fg)
                .padding(.horizontal, LabTheme.s8)

            HStack(spacing: 12) {
                statCard("Total Value", totalValue, icon: "banknote.fill")
                statCard("Avg Distance", "—", icon: "location.fill")
            }
        }
    }

    private var totalValue: String {
        let total = vm.completedMissions.reduce(0) { $0 + $1.amount }
        return total > 0 ? total.formattedAmount : "—"
    }

    private func statCard(_ title: String, _ value: String, icon: String) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Image(systemName: icon)
                .font(.system(size: 14))
                .foregroundStyle(LabTheme.fgSecondary)

            Text(value)
                .font(.system(size: 15, weight: .bold, design: .monospaced))
                .foregroundStyle(LabTheme.fg)
                .lineLimit(1)
                .minimumScaleFactor(0.7)

            Text(title)
                .font(.system(size: 11, weight: .medium))
                .foregroundStyle(LabTheme.fgTertiary)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(LabTheme.s16)
        .labCard()
    }
}

#Preview {
    ProfileView(vm: FleetViewModel())
        .modelContainer(for: OfflineDelivery.self, inMemory: true)
}
