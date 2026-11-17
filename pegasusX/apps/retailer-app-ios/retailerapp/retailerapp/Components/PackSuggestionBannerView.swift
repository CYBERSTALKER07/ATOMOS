import SwiftUI

struct PackSuggestionBannerView: View {
    let suggestions: [PackMultipleSuggestion]
    let onApply: (String, Int) -> Void

    var body: some View {
        LabCard {
            VStack(alignment: .leading, spacing: AppTheme.spacingSM) {
                Label("Pack Size Adjustment Required", systemImage: "shippingbox.fill")
                    .font(.system(.subheadline, design: .rounded, weight: .semibold))
                    .foregroundStyle(AppTheme.warning)

                ForEach(suggestions) { suggestion in
                    VStack(alignment: .leading, spacing: AppTheme.spacingXS) {
                        Text("\(suggestion.skuName ?? suggestion.skuId) · Requested \(suggestion.requested) (pack ×\(suggestion.packSize))")
                            .font(.system(.caption, design: .rounded, weight: .medium))
                            .foregroundStyle(AppTheme.textPrimary)

                        HStack(spacing: AppTheme.spacingSM) {
                            packChip("Floor \(suggestion.suggestedFloor)", value: suggestion.suggestedFloor, skuId: suggestion.skuId)
                            packChip("Nearest \(suggestion.suggestedNearest)", value: suggestion.suggestedNearest, skuId: suggestion.skuId, highlighted: true)
                            packChip("Ceil \(suggestion.suggestedCeil)", value: suggestion.suggestedCeil, skuId: suggestion.skuId)
                        }
                    }
                    .padding(.top, 2)
                }
            }
            .padding(AppTheme.spacingLG)
        }
    }

    private func packChip(_ title: String, value: Int, skuId: String, highlighted: Bool = false) -> some View {
        Button {
            Haptics.selection()
            onApply(skuId, value)
        } label: {
            Text(title)
                .font(.system(.caption2, design: .rounded, weight: .semibold))
                .foregroundStyle(highlighted ? .white : AppTheme.accent)
                .padding(.horizontal, 10)
                .padding(.vertical, 6)
                .background(highlighted ? AppTheme.accent : AppTheme.accentSoft.opacity(0.35))
                .clipShape(.capsule)
                .pegasusMinimumHitTarget()
        }
        .buttonStyle(.plain)
        .accessibilityLabel("Set \(skuId) quantity to \(value)")
    }
}

struct OrderLineFlagPills: View {
    let mustHave: Bool
    let allowSubstitutes: Bool
    let packSize: Int
    let currentQuantity: Int
    let onToggleMustHave: (() -> Void)?
    let onToggleAllowSubstitutes: (() -> Void)?
    let onSnapPack: (() -> Void)?

    var body: some View {
        HStack(spacing: 6) {
            if let onToggleMustHave {
                pillButton(
                    title: "Must-have",
                    active: mustHave,
                    activeColor: AppTheme.warning,
                    action: onToggleMustHave
                )
            }
            if let onToggleAllowSubstitutes {
                pillButton(
                    title: "Allow subs",
                    active: allowSubstitutes,
                    activeColor: AppTheme.accent,
                    action: onToggleAllowSubstitutes
                )
            }
            if let onSnapPack, packSize > 1, currentQuantity % packSize != 0 {
                let snapped = RetailerPackHelper.snapToPackMultiple(currentQuantity, packSize: packSize, mode: .nearest)
                pillButton(
                    title: "Snap → \(snapped)",
                    active: true,
                    activeColor: AppTheme.info,
                    action: onSnapPack
                )
            }
        }
    }

    private func pillButton(title: String, active: Bool, activeColor: Color, action: @escaping () -> Void) -> some View {
        Button {
            Haptics.selection()
            action()
        } label: {
            Text(title)
                .font(.system(.caption2, design: .rounded, weight: .semibold))
                .foregroundStyle(active ? .white : AppTheme.textSecondary)
                .padding(.horizontal, 8)
                .padding(.vertical, 4)
                .background(active ? activeColor : AppTheme.surfaceElevated)
                .clipShape(.capsule)
                .pegasusMinimumHitTarget()
        }
        .buttonStyle(.plain)
    }
}
