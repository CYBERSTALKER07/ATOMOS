import SwiftUI

struct StatsSection: View {
    var vm: FleetViewModel

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("portal.nav.earnings")
                .font(.system(size: 17, weight: .bold))
                .foregroundStyle(LabTheme.fg)
                .padding(.horizontal, LabTheme.s8)

            HStack(spacing: 12) {
                StatCard(title: "Total Volume", value: totalValue, icon: "banknote.fill")
                StatCard(title: "Deliveries", value: deliveries, icon: "shippingbox.fill")
            }
        }
    }

    private var totalValue: String {
        let total = vm.earnings?.totalVolume ?? 0
        return total > 0 ? Int(total).formattedAmount : "—"
    }

    private var deliveries: String {
        let count = vm.earnings?.totalDeliveries ?? 0
        return count > 0 ? "\(count)" : "—"
    }
}

struct StatCard: View {
    let title: String
    let value: String
    let icon: String

    var body: some View {
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
