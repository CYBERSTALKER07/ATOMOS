import SwiftUI

struct AiPredictionSection: View {
    let predictions: [RetailerAIPrediction]
    @Binding var actingId: String?
    let onConfirm: (RetailerAIPrediction) async -> Void
    let onReject: (RetailerAIPrediction) async -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
            RetailerSectionHeader(title: "AI Predictions", icon: "sparkles", count: predictions.count)

            ForEach(Array(predictions.enumerated()), id: \.element.id) { index, item in
                PredictionCard(item: item, actingId: $actingId, onConfirm: onConfirm, onReject: onReject)
                    .staggeredSlideIn(index: index)
            }
        }
    }
}

struct PredictionCard: View {
    let item: RetailerAIPrediction
    @Binding var actingId: String?
    let onConfirm: (RetailerAIPrediction) async -> Void
    let onReject: (RetailerAIPrediction) async -> Void

    var body: some View {
        LabCard {
            HStack(spacing: AppTheme.spacingMD) {
                ZStack {
                    Circle()
                        .stroke(AppTheme.separator.opacity(0.3), lineWidth: 3)
                        .frame(width: 44, height: 44)
                    Text(String(item.statusLabel.prefix(7)))
                        .font(.system(size: 8, weight: .bold, design: .rounded))
                        .foregroundStyle(AppTheme.warning)
                }

                VStack(alignment: .leading, spacing: 3) {
                    Text(item.title)
                        .font(.system(.subheadline, design: .rounded, weight: .semibold))
                        .foregroundStyle(AppTheme.textPrimary)
                    Text("\(item.deliveryLabel) · \(item.statusLabel)")
                        .font(.caption)
                        .foregroundStyle(AppTheme.textTertiary)
                        .lineLimit(2)
                }

                Spacer(minLength: 0)

                VStack(spacing: 6) {
                    Text(item.formattedTotal)
                        .font(.system(.title3, design: .rounded, weight: .bold))
                        .foregroundStyle(AppTheme.textPrimary)
                    Text("\(item.quantity) units")
                        .font(.system(size: 9, weight: .medium, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)

                    HStack(spacing: 8) {
                        Button {
                            guard actingId == nil else { return }
                            Task { await onConfirm(item) }
                        } label: {
                            Text("Confirm")
                                .font(.system(size: 11, weight: .bold, design: .rounded))
                        }
                        .disabled(actingId != nil)
                        Button {
                            guard actingId == nil else { return }
                            Task { await onReject(item) }
                        } label: {
                            Text("Reject")
                                .font(.system(size: 11, weight: .bold, design: .rounded))
                                .foregroundStyle(AppTheme.destructive)
                        }
                        .disabled(actingId != nil)
                    }
                }
            }
            .padding(AppTheme.spacingLG)
        }
    }
}
