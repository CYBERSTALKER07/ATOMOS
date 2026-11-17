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
                        Rectangle()
                            .fill(AppTheme.separator.opacity(0.3))
                            .frame(height: AppTheme.separatorHeight)
                            .padding(.leading, 60)
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
                .font(.system(.caption, weight: .semibold))
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
