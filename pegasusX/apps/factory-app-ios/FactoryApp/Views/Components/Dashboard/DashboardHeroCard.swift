import SwiftUI

struct DashboardHeroCard: View {
    let stats: DashboardStats

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingLG) {
            VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                Text("Outbound floor status")
                    .font(.title2.bold())
                Text("\(stats.pendingTransfers + stats.loadingTransfers) transfers are active across release and bay lanes.")
                    .font(.body)
                    .foregroundStyle(.secondary)
            }

            HStack(spacing: LabTheme.spacingSM) {
                OverviewMetric(label: "Queued", value: "\(stats.pendingTransfers)")
                OverviewMetric(label: "Loading", value: "\(stats.loadingTransfers)")
                OverviewMetric(label: "Critical", value: "\(stats.criticalInsights)")
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .labCard()
        .padding(.horizontal)
    }
}

struct OverviewMetric: View {
    let label: String
    let value: String

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
            Text(value)
                .font(.title3.bold())
            Text(label)
                .font(.footnote)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(LabTheme.spacingMD)
        .background(LabTheme.tertiaryBackground, in: RoundedRectangle(cornerRadius: LabTheme.radiusMD))
    }
}
