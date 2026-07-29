import SwiftUI

struct SettingsItem: Identifiable {
    let id = UUID()
    let icon: String
    let title: String
    let subtitle: String?
    var view: String? = nil
}

struct SettingsSectionView: View {
    var title: String
    var icon: String
    var items: [SettingsItem]

    var body: some View {
        LabCard {
            VStack(alignment: .leading, spacing: 0) {
                SectionLabel(title: title, icon: icon)

                ForEach(Array(items.enumerated()), id: \.element.id) { index, item in
                    SettingsRow(item: item)

                    if index < items.count - 1 {
                        Rectangle().fill(AppTheme.separator.opacity(0.3)).frame(height: AppTheme.separatorHeight).padding(.leading, 60)
                    }
                }
            }
        }
    }
}

struct SectionLabel: View {
    var title: String
    var icon: String
    
    var body: some View {
        HStack(spacing: AppTheme.spacingSM) {
            Image(systemName: icon)
                .font(.system(size: 12, weight: .semibold))
                .foregroundStyle(AppTheme.accent)
            Text(title)
                .font(.system(.caption, design: .rounded, weight: .bold))
                .foregroundStyle(AppTheme.textTertiary)
                .textCase(.uppercase)
        }
        .padding(.horizontal, AppTheme.spacingLG)
        .padding(.top, AppTheme.spacingMD)
        .padding(.bottom, AppTheme.spacingSM)
    }
}

struct SettingsRow: View {
    var item: SettingsItem
    
    var body: some View {
        Group {
            if item.view == "FamilyMembers" {
                NavigationLink(destination: FamilyMembersView()) {
                    SettingsRowContent(item: item)
                }
            } else if item.view == "AccountProfile" {
                NavigationLink(destination: AccountProfileView()) {
                    SettingsRowContent(item: item)
                }
            } else if item.view == "SavedCards" {
                NavigationLink(destination: SavedCardsView()) {
                    SettingsRowContent(item: item)
                }
            } else {
                SettingsRowContent(item: item)
            }
        }
    }
}

struct SettingsRowContent: View {
    var item: SettingsItem
    
    var body: some View {
        HStack(spacing: AppTheme.spacingMD) {
            ZStack {
                RoundedRectangle(cornerRadius: AppTheme.radiusSM)
                    .fill(AppTheme.accentSoft.opacity(0.4))
                    .frame(width: 34, height: 34)
                Image(systemName: item.icon)
                    .font(.system(size: 14, weight: .semibold))
                    .foregroundStyle(AppTheme.accent)
            }

            VStack(alignment: .leading, spacing: 2) {
                Text(item.title)
                    .font(.system(.subheadline, design: .rounded, weight: .medium))
                    .foregroundStyle(AppTheme.textPrimary)
                if let subtitle = item.subtitle {
                    Text(subtitle)
                        .font(.system(.caption, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)
                }
            }

            Spacer()

            Image(systemName: "chevron.right")
                .font(.system(size: 11, weight: .semibold))
                .foregroundStyle(AppTheme.textTertiary.opacity(0.5))
        }
        .padding(.horizontal, AppTheme.spacingLG)
        .padding(.vertical, AppTheme.spacingMD)
    }
}

struct SettingsToggle: View {
    var icon: String
    var title: String
    var subtitle: String
    var color: Color
    @Binding var isOn: Bool
    
    var body: some View {
        HStack(spacing: AppTheme.spacingMD) {
            ZStack {
                RoundedRectangle(cornerRadius: AppTheme.radiusSM)
                    .fill(color.opacity(0.12))
                    .frame(width: 34, height: 34)
                Image(systemName: icon)
                    .font(.system(size: 14, weight: .semibold))
                    .foregroundStyle(color)
            }

            VStack(alignment: .leading, spacing: 2) {
                Text(title)
                    .font(.system(.subheadline, design: .rounded, weight: .medium))
                    .foregroundStyle(AppTheme.textPrimary)
                Text(subtitle)
                    .font(.system(.caption, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
            }

            Spacer()

            Toggle("", isOn: $isOn)
                .tint(AppTheme.accent)
                .labelsHidden()
        }
        .padding(.horizontal, AppTheme.spacingLG)
        .padding(.vertical, AppTheme.spacingMD)
    }
}
