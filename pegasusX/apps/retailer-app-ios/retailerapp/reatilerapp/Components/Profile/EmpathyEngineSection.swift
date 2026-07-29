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
                        Text("Auto-Order Everything")
                            .font(.system(.subheadline, design: .rounded, weight: .medium))
                            .foregroundStyle(AppTheme.textPrimary)
                        Text("Auto-order all previously ordered products")
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
                        Text("Global auto-order is active. This overrides individual supplier and product settings.")
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
            Button("Use History") {
                Task { await toggleAction(true, true) }
            }
            Button("Start Fresh") {
                Task { await toggleAction(true, false) }
            }
            Button("Cancel", role: .cancel) {
                globalAutoOrder = false
            }
        }, message: {
            Text("Use existing order history for predictions, or start fresh? Starting fresh requires at least 2 orders per product.")
        })
    }
}
