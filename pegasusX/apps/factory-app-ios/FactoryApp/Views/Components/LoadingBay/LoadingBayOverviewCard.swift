import SwiftUI

struct LoadingBayOverviewCard: View {
    let readyCount: Int
    let loadingCount: Int
    let dispatchedCount: Int

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingLG) {
            VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                Text("mobile_factory.ui.loading_bay_flow")
                    .font(.title2.bold())
                Text("mobile_factory.ui.track_approved_transfers_active_loading_work_and_dispatched_volu")
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
