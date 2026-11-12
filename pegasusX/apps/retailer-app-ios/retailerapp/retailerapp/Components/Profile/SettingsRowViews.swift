import SwiftUI

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
            } else if item.view == "Capabilities" {
                NavigationLink(destination: CapabilitiesView()) {
                    SettingsRowContent(item: item)
                }
            } else if item.view == "Team" {
                NavigationLink(destination: TeamView()) {
                    SettingsRowContent(item: item)
                }
            } else if item.view == "Locations" {
                NavigationLink(destination: LocationsView()) {
                    SettingsRowContent(item: item)
                }
            } else if item.view == "StoreStock" {
                NavigationLink(destination: StoreStockView()) {
                    SettingsRowContent(item: item)
                }
            } else if item.view == "LocalSkus" {
                NavigationLink(destination: LocalSkusView()) {
                    SettingsRowContent(item: item)
                }
            } else if item.view == "POS" {
                NavigationLink(destination: PosView()) {
                    SettingsRowContent(item: item)
                }
            } else if item.view == "Shifts" {
                NavigationLink(destination: ShiftsView()) {
                    SettingsRowContent(item: item)
                }
            } else if item.view == "Sections" {
                NavigationLink(destination: SectionsView()) {
                    SettingsRowContent(item: item)
                }
            } else if item.view == "ReportsPro" {
                NavigationLink(destination: ReportsProView()) {
                    SettingsRowContent(item: item)
                }
            } else if item.view == "Assist" {
                NavigationLink(destination: AssistView()) {
                    SettingsRowContent(item: item)
                }
            } else if item.view == "Planograms" || item.view == "Planogram" {
                NavigationLink(destination: PlanogramView()) {
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
                RoundedRectangle(cornerRadius: AppTheme.radiusSM, style: .continuous)
                    .fill(AppTheme.accentSoft.opacity(0.4))
                    .frame(width: 34, height: 34)
                Image(systemName: item.icon)
                    .font(.system(.subheadline, weight: .semibold))
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
                .font(.system(.caption2, weight: .semibold))
                .foregroundStyle(AppTheme.textTertiary.opacity(0.5))
        }
        .frame(minHeight: 44)
        .padding(.horizontal, AppTheme.spacingLG)
        .padding(.vertical, AppTheme.spacingMD)
        .contentShape(Rectangle())
        .accessibilityElement(children: .combine)
        .accessibilityLabel(item.subtitle.map { "\(item.title), \($0)" } ?? item.title)
    }
}
