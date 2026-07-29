import SwiftUI

struct HistorySection: View {
    var vm: FleetViewModel
    
    var body: some View {
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
                    HistoryRow(mission: mission, index: index)
                }
            }
        }
    }
}

struct HistoryRow: View {
    let mission: Mission
    let index: Int
    
    var body: some View {
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
}
