import SwiftUI

struct AutoOrderPredictionsSection: View {
    let forecasts: [DemandForecast]

    var body: some View {
        LabCardWithHeader(title: "Active Predictions", icon: "sparkles") {
            VStack(spacing: AppTheme.spacingMD) {
                ForEach(forecasts) { forecast in
                    HStack(spacing: AppTheme.spacingMD) {
                        ZStack {
                            Circle()
                                .stroke(AppTheme.separator.opacity(0.3), lineWidth: 2)
                                .frame(width: 36, height: 36)
                            Circle()
                                .trim(from: 0, to: forecast.confidence)
                                .stroke(confidenceColor(forecast.confidence), style: StrokeStyle(lineWidth: 2, lineCap: .round))
                                .frame(width: 36, height: 36)
                                .rotationEffect(.degrees(-90))
                            Text(forecast.confidencePercent)
                                .font(.system(.caption2, design: .rounded, weight: .bold))
                                .foregroundStyle(confidenceColor(forecast.confidence))
                        }

                        VStack(alignment: .leading, spacing: 2) {
                            Text(forecast.productName)
                                .font(.system(.subheadline, design: .rounded, weight: .medium))
                                .foregroundStyle(AppTheme.textPrimary)
                                .lineLimit(1)
                            Text(L10n.format("mobile_retailer.ui.order_by_suggestedorderdate", "\(forecast.suggestedOrderDate)"))
                                .font(.system(.caption2, design: .rounded))
                                .foregroundStyle(AppTheme.textTertiary)
                        }

                        Spacer()

                        VStack(spacing: 1) {
                            Text("\(forecast.predictedQuantity)")
                                .font(.system(.headline, design: .rounded, weight: .bold))
                                .foregroundStyle(AppTheme.accent)
                            Text("retailer_desktop.auto_order.auto_order_list.text.units")
                                .font(.system(.caption2, design: .rounded, weight: .medium))
                                .foregroundStyle(AppTheme.textTertiary)
                        }
                    }
                    .accessibilityElement(children: .combine)
                    .accessibilityLabel("\(forecast.productName), \(forecast.predictedQuantity) units, \(forecast.confidencePercent) confidence")

                    if forecast.id != forecasts.last?.id {
                        Rectangle()
                            .fill(AppTheme.separator.opacity(0.15))
                            .frame(height: AppTheme.separatorHeight)
                    }
                }
            }
        }
    }

    private func confidenceColor(_ confidence: Double) -> Color {
        if confidence >= 0.8 { return AppTheme.success }
        if confidence >= 0.6 { return AppTheme.warning }
        return AppTheme.destructive
    }
}
