import SwiftUI

struct PreferencesSection: View {
    @Binding var aiAutoOrder: Bool
    @Binding var notificationsEnabled: Bool

    var body: some View {
        LabCard {
            VStack(alignment: .leading, spacing: 0) {
                SectionLabel(title: "Preferences", icon: "slider.horizontal.3")

                SettingsToggle(icon: "sparkles", title: "AI Auto-Order", subtitle: "Automatically place predicted orders", color: AppTheme.accent, isOn: $aiAutoOrder)

                Rectangle().fill(AppTheme.separator.opacity(0.3)).frame(height: AppTheme.separatorHeight).padding(.leading, 60)

                SettingsToggle(icon: "bell.fill", title: "Notifications", subtitle: "Push notification alerts", color: AppTheme.info, isOn: $notificationsEnabled)
            }
        }
    }
}
