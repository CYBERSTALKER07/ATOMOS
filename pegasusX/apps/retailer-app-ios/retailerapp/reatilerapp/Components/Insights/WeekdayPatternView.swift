import SwiftUI
import Charts

struct WeekdayPatternView: View {
    let data: [DayOfWeekPattern]

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("Ordering Pattern by Day")
                .font(.system(.subheadline, design: .rounded, weight: .semibold))
                .foregroundStyle(AppTheme.textPrimary)

            Chart(data) { item in
                BarMark(
                    x: .value("Day", String(item.weekday.prefix(3))),
                    y: .value("Orders", item.count)
                )
                .foregroundStyle(AppTheme.accent)
                .clipShape(.rect(cornerRadius: 4))
            }
            .frame(height: 160)

            // Avg spend per day
            HStack {
                ForEach(data) { d in
                    VStack(spacing: 2) {
                        Text(String(d.weekday.prefix(3)))
                            .font(.system(.caption2, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                        Text(abbreviate(d.avg))
                            .font(.system(.caption2, design: .rounded, weight: .bold))
                            .foregroundStyle(AppTheme.textPrimary)
                    }
                    .frame(maxWidth: .infinity)
                }
            }
        }
        .padding(AppTheme.spacingLG)
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
    }

    private func abbreviate(_ value: Int) -> String {
        if value >= 1_000_000 { return "\(value / 1_000_000)M" }
        if value >= 1_000 { return "\(value / 1_000)K" }
        return "\(value)"
    }
}
