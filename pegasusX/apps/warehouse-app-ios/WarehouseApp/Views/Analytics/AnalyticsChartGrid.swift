import SwiftUI

struct AnalyticsChartGrid: View {
    let daily: [DailyMetric]

<<<<<<< HEAD
    var body: some View {
        VStack(spacing: LabTheme.spacingMD) {
            if !daily.isEmpty {
                DailyRevenueChart(daily: daily)
            }
        }
    }
}

private struct DailyRevenueChart: View {
    let daily: [DailyMetric]

=======
>>>>>>> 5fbd72145092e2ede05adb999b291e8ffbaa19a8
    private var maxRevenue: Int {
        max(daily.map(\.revenue).max() ?? 1, 1)
    }

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
            Text("Daily Revenue")
                .font(.title3.bold())
            HStack(alignment: .bottom, spacing: 4) {
                ForEach(daily) { day in
                    VStack(spacing: 4) {
                        RoundedRectangle(cornerRadius: LabTheme.radiusSM)
                            .fill(Color.accentColor)
                            .frame(
                                maxWidth: .infinity,
                                minHeight: 4,
                                idealHeight: CGFloat(day.revenue) / CGFloat(maxRevenue) * 96,
                                maxHeight: 96
                            )
                        Text(String(day.date.suffix(5)))
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                            .lineLimit(1)
                    }
                }
            }
            .frame(maxWidth: .infinity, minHeight: 120, alignment: .bottom)
            if let peak = daily.map(\.revenue).max() {
                Text("Peak day: \(peak.formatted()) UZS")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
        .labCard()
    }
}
