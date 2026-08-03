import SwiftUI

struct AiPlannedCard: View {
    let forecast: DemandForecast
    let onPreorder: () -> Void

    var body: some View {
        HStack(spacing: AppTheme.spacingMD) {
            ZStack {
                Circle().stroke(AppTheme.separator.opacity(0.3), lineWidth: 3).frame(width: 40, height: 40)
                Circle()
                    .trim(from: 0, to: forecast.confidence)
                    .stroke(confidenceColor(forecast.confidence), style: StrokeStyle(lineWidth: 3, lineCap: .round))
                    .frame(width: 40, height: 40)
                    .rotationEffect(.degrees(-90))
                Text(forecast.confidencePercent)
                    .font(.system(size: 9, weight: .bold, design: .rounded))
                    .foregroundStyle(confidenceColor(forecast.confidence))
            }

            VStack(alignment: .leading, spacing: 2) {
                Text(forecast.productName)
                    .font(.system(.subheadline, design: .rounded, weight: .semibold))
                    .foregroundStyle(AppTheme.textPrimary)
                    .lineLimit(1)
                Text("Order by \(forecast.suggestedOrderDate)")
                    .font(.system(.caption2, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
            }

            Spacer()

            VStack(alignment: .trailing, spacing: 4) {
                Text("\(forecast.predictedQuantity) units")
                    .font(.system(.caption, design: .rounded, weight: .bold))
                    .foregroundStyle(AppTheme.textPrimary)

                Button {
                    Haptics.medium()
                    onPreorder()
                } label: {
                    Text("Pre-Order")
                        .font(.system(size: 11, weight: .bold, design: .rounded))
                        .foregroundStyle(.white)
                        .padding(.horizontal, 10).padding(.vertical, 5)
                        .background(AppTheme.accent)
                        .clipShape(.capsule)
                }
            }
        }
        .padding(AppTheme.spacingMD)
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
        .shadow(color: AppTheme.shadowColor, radius: 3, y: 1)
    }

    private func confidenceColor(_ confidence: Double) -> Color {
        if confidence >= 0.8 { return AppTheme.success }
        if confidence >= 0.6 { return AppTheme.warning }
        return AppTheme.destructive
    }
}
