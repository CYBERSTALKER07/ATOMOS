import SwiftUI

struct AutoOrderGlobalToggleCard: View {
    @Binding var globalAutoOrder: Bool
    let analyticsStartDate: String?
    let onToggle: (Bool) -> Void

    var body: some View {
        LabCard {
            VStack(alignment: .leading, spacing: 0) {
                HStack(spacing: AppTheme.spacingMD) {
                    ZStack {
                        RoundedRectangle(cornerRadius: AppTheme.radiusSM)
                            .fill(globalAutoOrder ? AppTheme.accent.opacity(0.15) : AppTheme.surfaceElevated)
                            .frame(width: 40, height: 40)
                        Image(systemName: "arrow.triangle.2.circlepath")
                            .font(.system(size: 16, weight: .semibold))
                            .foregroundStyle(globalAutoOrder ? AppTheme.accent : AppTheme.textSecondary)
                    }

                    VStack(alignment: .leading, spacing: 2) {
                        Text("retailer_desktop.auto_order.auto_order_rules.text.global_auto_order")
                            .font(.system(.subheadline, design: .rounded, weight: .semibold))
                            .foregroundStyle(AppTheme.textPrimary)
                        Text("retailer_desktop.auto_order.auto_order_rules.text.auto_order_everything_from_all_suppliers")
                            .font(.system(.caption, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                    }

                    Spacer()

                    Toggle("", isOn: Binding(
                        get: { globalAutoOrder },
                        set: { newVal in
                            globalAutoOrder = newVal
                            onToggle(newVal)
                        }
                    ))
                        .tint(AppTheme.accent)
                        .labelsHidden()
                }
                .padding(AppTheme.spacingLG)

                if globalAutoOrder {
                    HStack(spacing: AppTheme.spacingSM) {
                        Image(systemName: "checkmark.circle.fill")
                            .font(.system(size: 12))
                            .foregroundStyle(AppTheme.success)
                        Text("mobile_retailer.ui.global_auto_order_active_overrides_all_supplier_product_settings")
                            .font(.system(.caption2, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                    }
                    .padding(.horizontal, AppTheme.spacingLG)
                    .padding(.bottom, AppTheme.spacingMD)
                    .transition(.move(edge: .top).combined(with: .opacity))
                }

                if let dateStr = analyticsStartDate {
                    HStack(spacing: AppTheme.spacingSM) {
                        Image(systemName: "calendar.badge.clock")
                            .font(.system(size: 12))
                            .foregroundStyle(AppTheme.accent)
                        Text(L10n.format("mobile_retailer.ui.analytics_since_datestr_2", "\(dateStr)"))
                            .font(.system(.caption2, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                    }
                    .padding(.horizontal, AppTheme.spacingLG)
                    .padding(.bottom, AppTheme.spacingMD)
                }
            }
        }
        .animation(AnimationConstants.express, value: globalAutoOrder)
    }
}
