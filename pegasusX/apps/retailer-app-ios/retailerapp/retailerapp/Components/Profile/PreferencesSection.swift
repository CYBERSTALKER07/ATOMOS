import SwiftUI

struct PreferencesSection: View {
    @Binding var aiAutoOrder: Bool
    @Binding var notificationsEnabled: Bool
    var onAutoOrderToggle: (Bool) -> Void = { _ in }

    @State private var autoOrderToggleValue = false

    var body: some View {
        LabCard {
            VStack(alignment: .leading, spacing: 0) {
                SectionLabel(title: "Preferences", icon: "slider.horizontal.3")

                SettingsToggle(
                    icon: "sparkles",
                    title: "AI Auto-Order",
                    subtitle: "Server-backed global auto-order (same as Empathy Engine)",
                    color: AppTheme.accent,
                    isOn: $autoOrderToggleValue
                )

                Rectangle()
                    .fill(AppTheme.separator.opacity(0.3))
                    .frame(height: AppTheme.separatorHeight)
                    .padding(.leading, 60)

                SettingsToggle(
                    icon: "bell.fill",
                    title: "Notifications",
                    subtitle: "Push notification alerts",
                    color: AppTheme.info,
                    isOn: $notificationsEnabled
                )
            }
        }
        .onAppear {
            autoOrderToggleValue = aiAutoOrder
        }
        .onChange(of: aiAutoOrder) { _, newValue in
            if autoOrderToggleValue != newValue {
                autoOrderToggleValue = newValue
            }
        }
        .onChange(of: autoOrderToggleValue) { _, newValue in
            guard newValue != aiAutoOrder else { return }
            Haptics.selection()
            aiAutoOrder = newValue
            onAutoOrderToggle(newValue)
        }
    }
}
