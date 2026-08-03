import SwiftUI
import Charts

let stateColors: [String: Color] = [
    "COMPLETED": .green,
    "ARRIVED": .blue,
    "IN_TRANSIT": .orange,
    "PENDING": .yellow,
    "LOADED": .purple,
    "CANCELLED": .pink,
    "CANCELLED_BY_ADMIN": .red,
]

struct OrdersByStateView: View {
    let data: [OrderStateCount]

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("Orders by Status")
                .font(.system(.subheadline, design: .rounded, weight: .semibold))
                .foregroundStyle(AppTheme.textPrimary)

            Chart(data) { item in
                SectorMark(
                    angle: .value("Count", item.count),
                    innerRadius: .ratio(0.6),
                    angularInset: 1.5
                )
                .foregroundStyle(stateColors[item.state] ?? .gray)
                .cornerRadius(3)
            }
            .frame(height: 200)

            // Legend
            FlowLayout(spacing: 8) {
                ForEach(data) { item in
                    HStack(spacing: 4) {
                        Circle()
                            .fill(stateColors[item.state] ?? .gray)
                            .frame(width: 8, height: 8)
                        Text("\(item.state.replacingOccurrences(of: "_", with: " ")) (\(item.count))")
                            .font(.system(.caption2, design: .rounded))
                            .foregroundStyle(AppTheme.textSecondary)
                    }
                }
            }
        }
        .padding(AppTheme.spacingLG)
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
    }
}
