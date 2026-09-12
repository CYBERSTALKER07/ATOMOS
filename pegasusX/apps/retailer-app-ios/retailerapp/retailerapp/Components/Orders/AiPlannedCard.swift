import SwiftUI

struct AiPlannedCard: View {
    let item: RetailerAIPrediction
    let onConfirm: () -> Void
    let onReject: () -> Void

    var body: some View {
        HStack(spacing: AppTheme.spacingMD) {
            ZStack {
                Circle().stroke(AppTheme.separator.opacity(0.3), lineWidth: 3).frame(width: 40, height: 40)
                Text(String(item.statusLabel.prefix(7)))
                    .font(.system(size: 8, weight: .bold, design: .rounded))
                    .foregroundStyle(AppTheme.warning)
            }

            VStack(alignment: .leading, spacing: 2) {
                Text(item.title)
                    .font(.system(.subheadline, design: .rounded, weight: .semibold))
                    .foregroundStyle(AppTheme.textPrimary)
                    .lineLimit(1)
                Text("\(item.deliveryLabel) · \(item.statusLabel)")
                    .font(.system(.caption2, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
            }

            Spacer()

            VStack(alignment: .trailing, spacing: 4) {
                Text(item.formattedTotal)
                    .font(.system(.caption, design: .rounded, weight: .bold))
                    .foregroundStyle(AppTheme.textPrimary)

                HStack(spacing: 6) {
                    Button {
                        Haptics.medium()
                        onConfirm()
                    } label: {
                        Text("Confirm")
                            .font(.system(size: 11, weight: .bold, design: .rounded))
                            .foregroundStyle(.white)
                            .padding(.horizontal, 10).padding(.vertical, 5)
                            .background(AppTheme.accent)
                            .clipShape(.capsule)
                    }
                    Button {
                        Haptics.light()
                        onReject()
                    } label: {
                        Text("Reject")
                            .font(.system(size: 11, weight: .bold, design: .rounded))
                            .foregroundStyle(AppTheme.destructive)
                            .padding(.horizontal, 10).padding(.vertical, 5)
                            .background(AppTheme.destructive.opacity(0.1))
                            .clipShape(.capsule)
                    }
                }
            }
        }
        .padding(AppTheme.spacingMD)
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
        .shadow(color: AppTheme.shadowColor, radius: 3, y: 1)
    }
}
