import SwiftUI

struct RecentActivitySection: View {
    var vm: FleetViewModel

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            DriverSectionHeader(title: "Recent")
                .padding(.horizontal, LabTheme.s4)

            if vm.completedMissions.isEmpty {
                DriverEmptyView(
                    icon: "clock.arrow.circlepath",
                    title: "No deliveries yet",
                    message: "Completed drops will appear here."
                )
            } else {
                ForEach(vm.completedMissions.prefix(3)) { mission in
                    HStack(spacing: 12) {
                        ZStack {
                            Circle()
                                .fill(LabTheme.success.opacity(0.12))
                                .frame(width: 32, height: 32)
                            Image(systemName: "checkmark")
                                .font(.system(size: 11, weight: .bold))
                                .foregroundStyle(LabTheme.success)
                        }

                        VStack(alignment: .leading, spacing: 2) {
                            Text(mission.order_id)
                                .font(.system(size: 12, weight: .bold, design: .monospaced))
                                .foregroundStyle(LabTheme.fg)
                            Text(mission.amount.formattedAmount)
                                .font(.system(size: 11, weight: .medium))
                                .foregroundStyle(LabTheme.fgSecondary)
                        }

                        Spacer()

                        Text(mission.gateway)
                            .font(.system(size: 10, weight: .bold))
                            .foregroundStyle(LabTheme.fgTertiary)
                    }
                    .padding(LabTheme.s12)
                    .labCard()
                }
            }
        }
    }
}
