import SwiftUI

struct HistorySection: View {
    var vm: FleetViewModel

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                Text("mobile_driver.ui.ride_history")
                    .font(.system(.body, weight: .bold))
                    .foregroundStyle(LabTheme.fg)

                Spacer()

                Text(L10n.format("mobile_driver.ui.count_rides", "\(vm.historyRows.count)"))
                    .font(.system(.caption, design: .monospaced, weight: .medium))
                    .foregroundStyle(LabTheme.fgTertiary)
            }
            .padding(.horizontal, LabTheme.s8)

            if vm.historyRows.isEmpty {
                VStack(spacing: 10) {
                    Image(systemName: "clock.arrow.circlepath")
                        .font(.system(.title2))
                        .foregroundStyle(LabTheme.fgTertiary)

                    Text("mobile_driver.ui.no_completed_rides_yet")
                        .font(.subheadline.weight(.medium))
                        .foregroundStyle(LabTheme.fgSecondary)
                }
                .frame(maxWidth: .infinity)
                .padding(.vertical, 30)
                .labCard()
            } else {
                ForEach(Array(vm.historyRows.enumerated()), id: \.element.id) { index, row in
                    HistoryRow(row: row, index: index)
                }
            }
        }
    }
}

struct HistoryRow: View {
    let row: DriverHistoryRow
    let index: Int

    var body: some View {
        HStack(spacing: 14) {
            ZStack {
                Circle()
                    .fill(LabTheme.success.opacity(0.15))
                    .frame(width: 36, height: 36)

                Image(systemName: "checkmark")
                    .font(.system(.footnote, weight: .bold))
                    .foregroundStyle(LabTheme.success)
            }

            VStack(alignment: .leading, spacing: 3) {
                Text(row.orderId)
                    .font(.system(.footnote, design: .monospaced, weight: .bold))
                    .foregroundStyle(LabTheme.fg)

                Text("\(row.status.isEmpty ? "COMPLETED" : row.status) · \(Int(row.totalMinor).formattedAmount)")
                    .font(.system(.caption2, weight: .medium))
                    .foregroundStyle(LabTheme.fgSecondary)
            }

            Spacer()

            StatusPill(label: row.status.isEmpty ? "DELIVERED" : row.status, color: LabTheme.success)
        }
        .padding(LabTheme.s16)
        .labCard()
        .staggeredAppear(index: index)
    }
}
