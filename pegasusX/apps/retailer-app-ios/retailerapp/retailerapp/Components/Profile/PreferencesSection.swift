import SwiftUI

struct PreferencesSection: View {
    @Binding var aiAutoOrder: Bool
    @Binding var notificationsEnabled: Bool
    var onAutoOrderToggle: (Bool) -> Void = { _ in }

    var body: some View {
        LabCard {
            VStack(alignment: .leading, spacing: 0) {
                SectionLabel(title: "Preferences", icon: "slider.horizontal.3")

                SettingsToggle(
                    icon: "sparkles",
                    title: "AI Auto-Order",
                    subtitle: "Server-backed global auto-order (same as Empathy Engine)",
                    color: AppTheme.accent,
                    isOn: Binding(
                        get: { aiAutoOrder },
                        set: { newVal in
                            aiAutoOrder = newVal
                            onAutoOrderToggle(newVal)
                        }
                    )
                )

                Rectangle().fill(AppTheme.separator.opacity(0.3)).frame(height: AppTheme.separatorHeight).padding(.leading, 60)

                SettingsToggle(icon: "bell.fill", title: "Notifications", subtitle: "Push notification alerts", color: AppTheme.info, isOn: $notificationsEnabled)
            }
        }
    }
}
