import SwiftUI

struct PreferencesSection: View {
    @Binding var aiAutoOrder: Bool
    @Binding var notificationsEnabled: Bool
    var preferencesStore: RetailerPreferencesStore = .shared
    var effectiveWorkspace: RetailerWorkspaceMode? = nil
    var onAutoOrderToggle: (Bool) -> Void = { _ in }
    var onLanguageChanged: (RetailerLanguage) -> Void = { _ in }
    var onWorkspaceModeChanged: (RetailerWorkspaceMode) -> Void = { _ in }

    @State private var autoOrderToggleValue = false

    var body: some View {
        let activeLanguage = preferencesStore.currentLanguage
        let activeWorkspace = preferencesStore.workspaceModeOverride ?? effectiveWorkspace ?? .fast

        LabCard {
            VStack(alignment: .leading, spacing: 0) {
                SectionLabel(title: L10n.string("mobile_retailer.preferences.title"), icon: "slider.horizontal.3")

                VStack(alignment: .leading, spacing: AppTheme.spacingSM) {
                    HStack(spacing: AppTheme.spacingMD) {
                        ZStack {
                            RoundedRectangle(cornerRadius: AppTheme.radiusSM, style: .continuous)
                                .fill(AppTheme.accent.opacity(0.12))
                                .frame(width: 34, height: 34)
                            Image(systemName: "globe")
                                .font(.system(.subheadline, weight: .semibold))
                                .foregroundStyle(AppTheme.accent)
                        }

                        VStack(alignment: .leading, spacing: 2) {
                            Text(L10n.string("mobile_retailer.preferences.language_title"))
                                .font(.system(.subheadline, design: .rounded, weight: .semibold))
                                .foregroundStyle(AppTheme.textPrimary)
                            Text(L10n.string("mobile_retailer.preferences.language_subtitle"))
                                .font(.system(.caption2, design: .rounded))
                                .foregroundStyle(AppTheme.textSecondary)
                        }

                        Spacer()
                    }

                    HStack(spacing: AppTheme.spacingSM) {
                        ForEach(RetailerLanguage.allCases) { language in
                            let isSelected = activeLanguage == language
                            Button {
                                guard activeLanguage != language else { return }
                                Haptics.selection()
                                preferencesStore.setLanguage(language)
                                onLanguageChanged(language)
                            } label: {
                                VStack(spacing: 2) {
                                    Text(language.codeLabel)
                                        .font(.system(.caption2, design: .rounded, weight: .bold))
                                    Text(language.nativeName)
                                        .font(.system(.caption, design: .rounded, weight: isSelected ? .bold : .medium))
                                        .lineLimit(1)
                                        .minimumScaleFactor(0.85)
                                }
                                .frame(maxWidth: .infinity, minHeight: AppTheme.minTouchTarget)
                                .padding(.horizontal, AppTheme.spacingXS)
                                .background(isSelected ? AppTheme.accent : AppTheme.surfaceElevated)
                                .foregroundStyle(isSelected ? .white : AppTheme.textPrimary)
                                .clipShape(.rect(cornerRadius: AppTheme.radiusButton))
                            }
                            .buttonStyle(.plain)
                            .accessibilityIdentifier("preferences-language-\(language.rawValue)")
                        }
                    }
                }
                .padding(.horizontal, AppTheme.spacingLG)
                .padding(.vertical, AppTheme.spacingMD)

                Rectangle()
                    .fill(AppTheme.separator.opacity(0.3))
                    .frame(height: AppTheme.separatorHeight)
                    .padding(.leading, 60)

                VStack(alignment: .leading, spacing: AppTheme.spacingSM) {
                    Text(L10n.string("mobile_retailer.preferences.workspace_title"))
                        .font(.system(.subheadline, design: .rounded, weight: .semibold))
                        .foregroundStyle(AppTheme.textPrimary)

                    HStack(spacing: AppTheme.spacingSM) {
                        workspaceButton(
                            mode: .fast,
                            label: L10n.string("mobile_retailer.workspace_mode.fast"),
                            isSelected: activeWorkspace == .fast
                        )
                        workspaceButton(
                            mode: .full,
                            label: L10n.string("mobile_retailer.workspace_mode.full"),
                            isSelected: activeWorkspace == .full
                        )
                    }
                }
                .padding(.horizontal, AppTheme.spacingLG)
                .padding(.vertical, AppTheme.spacingMD)

                Rectangle()
                    .fill(AppTheme.separator.opacity(0.3))
                    .frame(height: AppTheme.separatorHeight)
                    .padding(.leading, 60)

                SettingsToggle(
                    icon: "sparkles",
                    title: L10n.string("mobile_retailer.preferences.ai_auto_order_title"),
                    subtitle: L10n.string("mobile_retailer.preferences.ai_auto_order_subtitle"),
                    color: AppTheme.accent,
                    isOn: $autoOrderToggleValue
                )

                Rectangle()
                    .fill(AppTheme.separator.opacity(0.3))
                    .frame(height: AppTheme.separatorHeight)
                    .padding(.leading, 60)

                SettingsToggle(
                    icon: "bell.fill",
                    title: L10n.string("mobile_retailer.preferences.notifications_title"),
                    subtitle: L10n.string("mobile_retailer.preferences.notifications_subtitle"),
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

    private func workspaceButton(mode: RetailerWorkspaceMode, label: String, isSelected: Bool) -> some View {
        Button {
            Haptics.selection()
            preferencesStore.setWorkspaceMode(mode)
            onWorkspaceModeChanged(mode)
        } label: {
            Text(label)
                .font(.system(.caption, design: .rounded, weight: .semibold))
                .frame(maxWidth: .infinity, minHeight: AppTheme.minTouchTarget)
                .background(isSelected ? AppTheme.accent : AppTheme.surfaceElevated)
                .foregroundStyle(isSelected ? .white : AppTheme.textPrimary)
                .clipShape(.rect(cornerRadius: AppTheme.radiusButton))
        }
        .buttonStyle(.plain)
        .accessibilityIdentifier("preferences-workspace-\(mode.rawValue)")
    }
}
