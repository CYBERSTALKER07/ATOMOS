import SwiftUI
import Charts

struct CategoryBreakdownView: View {
    let data: [CategorySpend]

    private var maxTotal: Int { data.map(\.total).max() ?? 1 }

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("mobile_retailer.ui.spending_by_category")
                .font(.system(.subheadline, design: .rounded, weight: .semibold))
                .foregroundStyle(AppTheme.textPrimary)

            Chart(data) { item in
                BarMark(
                    x: .value("Amount", item.total),
                    y: .value("Category", item.category)
                )
                .foregroundStyle(AppTheme.accent.opacity(0.8))
                .clipShape(.rect(cornerRadius: 4))
            }
            .chartXAxis {
                AxisMarks { value in
                    AxisValueLabel {
                        if let v = value.as(Int.self) {
                            Text(abbreviate(v))
                                .font(.system(size: 10, design: .rounded))
                        }
                    }
                }
            }
            .frame(height: CGFloat(data.count * 36))
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
