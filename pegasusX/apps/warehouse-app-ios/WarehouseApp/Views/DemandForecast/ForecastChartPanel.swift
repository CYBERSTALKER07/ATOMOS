import SwiftUI

struct ForecastChartPanel: View {
    let criticalCount: Int
    let urgentCount: Int
    let normalCount: Int
    
    private let summaryColumns = [
        GridItem(.flexible(), spacing: LabTheme.spacingSM),
        GridItem(.flexible(), spacing: LabTheme.spacingSM),
        GridItem(.flexible(), spacing: LabTheme.spacingSM),
    ]
    
    var body: some View {
        LazyVGrid(columns: summaryColumns, spacing: LabTheme.spacingSM) {
            ForecastSummaryCard(
                title: "Critical",
                count: criticalCount,
                subtitle: "< 2 days",
                tint: LabTheme.destructive,
                index: 0
            )
            ForecastSummaryCard(
                title: "Urgent",
                count: urgentCount,
                subtitle: "< 5 days",
                tint: LabTheme.warning,
                index: 1
            )
            ForecastSummaryCard(
                title: "Healthy",
                count: normalCount,
                subtitle: "5+ days",
                tint: LabTheme.success,
                index: 2
            )
        }
    }
}

private struct ForecastSummaryCard: View {
    let title: String
    let count: Int
    let subtitle: String
    let tint: Color
    let index: Int

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
            Text(title)
                .font(.caption2)
                .foregroundStyle(.secondary)
            Text("\(count)")
                .font(.title2.bold())
                .foregroundStyle(tint)
            Text(subtitle)
                .font(.caption2)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .labCard()
        .staggeredAppear(index: index)
    }
}
