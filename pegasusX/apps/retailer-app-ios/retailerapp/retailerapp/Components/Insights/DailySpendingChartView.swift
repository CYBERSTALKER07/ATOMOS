import SwiftUI
import Charts

struct DailySpendingChartView: View {
    let data: [RetailerDayExpense]

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("mobile_retailer.ui.daily_spending")
                .font(.system(.subheadline, design: .rounded, weight: .semibold))
                .foregroundStyle(AppTheme.textPrimary)

            Chart(data) { item in
                LineMark(
                    x: .value("Date", item.shortDate),
                    y: .value("Amount", item.total)
                )
                .foregroundStyle(AppTheme.accent)
                .interpolationMethod(.catmullRom)

                AreaMark(
                    x: .value("Date", item.shortDate),
                    y: .value("Amount", item.total)
                )
                .foregroundStyle(
                    .linearGradient(
                        colors: [AppTheme.accent.opacity(0.2), .clear],
                        startPoint: .top,
                        endPoint: .bottom
                    )
                )
                .interpolationMethod(.catmullRom)
            }
            .frame(height: 200)
            .chartXAxis {
                AxisMarks(values: .automatic(desiredCount: 6)) { _ in
                    AxisValueLabel()
                        .font(.system(size: 9, design: .rounded))
                }
            }
        }
        .padding(AppTheme.spacingLG)
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
    }
}
