import SwiftUI

struct LoadingBayOverviewCard: View {
    let readyCount: Int
    let loadingCount: Int
    let dispatchedCount: Int

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingLG) {
            VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                Text("Loading bay flow")
                    .font(.title2.bold())
                Text("Track approved transfers, active loading work, and dispatched volume from one queue.")
                    .font(.body)
                    .foregroundStyle(.secondary)
            }

            HStack(spacing: LabTheme.spacingSM) {
                BayOverviewMetric(label: "Ready", value: "\(readyCount)")
                BayOverviewMetric(label: "Loading", value: "\(loadingCount)")
                BayOverviewMetric(label: "Out", value: "\(dispatchedCount)")
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .labCard()
    }
}

struct BayOverviewMetric: View {
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
