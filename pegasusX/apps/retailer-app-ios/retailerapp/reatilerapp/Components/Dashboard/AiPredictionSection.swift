import SwiftUI

struct AiPredictionSection: View {
    let predictions: [DemandForecast]
    @Binding var preorderingId: String?
    let onPreorder: (DemandForecast) async -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
            RetailerSectionHeader(title: "AI Predictions", icon: "sparkles", count: predictions.count)

            ForEach(Array(predictions.enumerated()), id: \.element.id) { index, forecast in
                PredictionCard(forecast: forecast, preorderingId: $preorderingId, onPreorder: onPreorder)
                    .staggeredSlideIn(index: index)
            }
        }
    }
}

struct PredictionCard: View {
    let forecast: DemandForecast
    @Binding var preorderingId: String?
    let onPreorder: (DemandForecast) async -> Void
    
    var body: some View {
        LabCard {
            HStack(spacing: AppTheme.spacingMD) {
                // Confidence ring
                ZStack {
                    Circle()
                        .stroke(AppTheme.separator.opacity(0.3), lineWidth: 3)
                        .frame(width: 44, height: 44)
                    Circle()
                        .trim(from: 0, to: forecast.confidence)
                        .stroke(confidenceColor(forecast.confidence), style: StrokeStyle(lineWidth: 3, lineCap: .round))
                        .frame(width: 44, height: 44)
                        .rotationEffect(.degrees(-90))
                    Text(forecast.confidencePercent)
                        .font(.system(size: 10, weight: .bold, design: .rounded))
                        .foregroundStyle(confidenceColor(forecast.confidence))
                }

                VStack(alignment: .leading, spacing: 3) {
                    HStack(spacing: 6) {
                        Text(forecast.productName)
                            .font(.system(.subheadline, design: .rounded, weight: .semibold))
                            .foregroundStyle(AppTheme.textPrimary)
                        if forecast.isBlocked {
                            Text("Insufficient history")
                                .font(.system(size: 9, weight: .bold, design: .rounded))
                                .foregroundStyle(AppTheme.warning)
                                .padding(.horizontal, 6)
                                .padding(.vertical, 2)
                                .background(AppTheme.warning.opacity(0.12))
                                .clipShape(Capsule())
                        }
                    }

                    Text(forecast.reasoning)
                        .font(.caption)
                        .foregroundStyle(AppTheme.textTertiary)
                        .lineLimit(2)
                }

                Spacer(minLength: 0)

                VStack(spacing: 6) {
                    Text("\(forecast.predictedQuantity)")
                        .font(.system(.title3, design: .rounded, weight: .bold))
                        .foregroundStyle(AppTheme.textPrimary)
                    Text("units")
                        .font(.system(size: 9, weight: .medium, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)

                    Button {
                        guard preorderingId == nil else { return }
                        Task { await onPreorder(forecast) }
                    } label: {
                        Group {
                            if preorderingId == forecast.id {
                                ProgressView()
                                    .progressViewStyle(.circular)
                                    .tint(.white)
                            } else {
                                Image(systemName: "cart.badge.plus")
                                    .font(.system(size: 14, weight: .semibold))
                                    .foregroundStyle(.white)
                            }
                        }
                        .frame(width: 32, height: 32)
                        .background(AppTheme.accent)
                        .clipShape(.circle)
                    }
                    .disabled(preorderingId != nil)
                }
            }
            .padding(AppTheme.spacingLG)
        }
    }
    
    private func confidenceColor(_ confidence: Double) -> Color {
        if confidence >= 0.8 { return AppTheme.success }
        if confidence >= 0.6 { return AppTheme.warning }
        return AppTheme.destructive
    }
}
