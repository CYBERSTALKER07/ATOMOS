import SwiftUI

struct EmpathyEngineSection: View {
    @Binding var globalAutoOrder: Bool
    @Binding var showHistoryAlert: Bool
    var toggleAction: (Bool, Bool) async -> Void

    var body: some View {
        LabCard {
            VStack(alignment: .leading, spacing: 0) {
                SectionLabel(title: "Empathy Engine", icon: "arrow.triangle.2.circlepath")

                HStack(spacing: AppTheme.spacingMD) {
                    ZStack {
                        RoundedRectangle(cornerRadius: AppTheme.radiusSM)
                            .fill(AppTheme.surfaceElevated)
                            .frame(width: 36, height: 36)
                        Image(systemName: "arrow.triangle.2.circlepath")
                            .font(.system(size: 14, weight: .semibold))
                            .foregroundStyle(AppTheme.textPrimary)
                    }

                    VStack(alignment: .leading, spacing: 2) {
                        Text("mobile_retailer.ui.auto_order_everything")
                            .font(.system(.subheadline, design: .rounded, weight: .medium))
                            .foregroundStyle(AppTheme.textPrimary)
                        Text("mobile_retailer.ui.auto_order_all_previously_ordered_products")
                            .font(.system(.caption, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                    }

                    Spacer()

                    Toggle("", isOn: Binding(
                        get: { globalAutoOrder },
                        set: { newVal in
                            globalAutoOrder = newVal
                            if newVal {
                                showHistoryAlert = true
                            } else {
                                Task { await toggleAction(false, false) }
                            }
                        }
                    ))
                        .tint(AppTheme.accent)
                        .labelsHidden()
                }
                .padding(.horizontal, AppTheme.spacingLG)
                .padding(.vertical, AppTheme.spacingMD)

                if globalAutoOrder {
                    HStack(spacing: AppTheme.spacingSM) {
                        Image(systemName: "checkmark.circle.fill")
                            .font(.system(size: 12))
                            .foregroundStyle(AppTheme.success)
                        Text("mobile_retailer.ui.global_auto_order_is_active_this_overrides_individual_supplier_a")
                            .font(.system(.caption2, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                    }
                    .padding(.horizontal, AppTheme.spacingLG)
                    .padding(.bottom, AppTheme.spacingMD)
                    .transition(.move(edge: .top).combined(with: .opacity))
                }
            }
        }
        .animation(AnimationConstants.express, value: globalAutoOrder)
        .alert("Use Previous Analytics?", isPresented: $showHistoryAlert, actions: {
            Button("mobile_retailer.ui.use_history") {
                Task { await toggleAction(true, true) }
            }
            Button("mobile_retailer.ui.start_fresh") {
                Task { await toggleAction(true, false) }
            }
            Button("common.action.cancel", role: .cancel) {
                globalAutoOrder = false
            }
        }, message: {
            Text("mobile_retailer.ui.use_existing_order_history_for_predictions_or_start_fresh_starti")
        })
    }
}
