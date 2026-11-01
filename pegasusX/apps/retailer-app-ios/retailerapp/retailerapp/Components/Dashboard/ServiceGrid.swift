import SwiftUI

struct ServiceGrid: View {
    let activeOrdersCount: Int
    let predictionsCount: Int
    
    var body: some View {
        VStack(spacing: AppTheme.spacingMD) {
            // Row 1: two big tiles
            HStack(spacing: AppTheme.spacingMD) {
                NavigationLink {
                    CatalogView()
                } label: {
                    ServiceTileView(title: "Catalog", icon: "bag.fill", subtitle: "Browse products", height: 130)
                }
                .buttonStyle(.pressable)

                NavigationLink {
                    FutureDemandView()
                } label: {
                    ServiceTileView(title: "Reorder suggestions", icon: "sparkles", subtitle: "\(predictionsCount) items", height: 130)
                }
                .buttonStyle(.pressable)
            }

            // Row 2: one wide + two small
            HStack(spacing: AppTheme.spacingMD) {
                // Left: tall tile
                NavigationLink {
                    OrdersView()
                } label: {
                    ServiceTileView(title: "Orders", icon: "shippingbox.fill", subtitle: "\(activeOrdersCount) active", height: 120)
                }
                .buttonStyle(.pressable)

                // Right: two small stacked
                VStack(spacing: AppTheme.spacingMD) {
                    NavigationLink {
                        NotificationInboxView()
                    } label: {
                        ServiceTileView(title: "Inbox", icon: "tray.fill", subtitle: nil, height: 54)
                    }
                    .buttonStyle(.pressable)

                    NavigationLink {
                        HistoryView()
                    } label: {
                        ServiceTileView(title: "History", icon: "clock.fill", subtitle: nil, height: 54)
                    }
                    .buttonStyle(.pressable)
                }
            }

            // Row 3: three equal small tiles
            HStack(spacing: AppTheme.spacingMD) {
                NavigationLink {
                    ProcurementView()
                } label: {
                    ServiceTileSmall(title: "Procurement", icon: "chart.bar.fill")
                }
                .buttonStyle(.pressable)

                NavigationLink {
                    SearchView()
                } label: {
                    ServiceTileSmall(title: "Search", icon: "magnifyingglass")
                }
                .buttonStyle(.pressable)

                NavigationLink {
                    ProfileView()
                } label: {
                    ServiceTileSmall(title: "Profile", icon: "person.fill")
                }
                .buttonStyle(.pressable)
            }
        }
    }
}

struct ServiceTileView: View {
    let title: String
    let icon: String
    let subtitle: String?
    let height: Double
    
    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            Spacer()

            Image(systemName: icon)
                .font(.system(size: 28, weight: .semibold)) // Bold icons
                .foregroundStyle(AppTheme.accent)
                .padding(.bottom, AppTheme.spacingSM)

            Text(title)
                .font(.system(.subheadline, design: .rounded, weight: .bold)) // Bold titles
                .foregroundStyle(AppTheme.textPrimary)

            if let subtitle {
                Text(subtitle)
                    .font(.system(.caption2, design: .rounded, weight: .medium)) // Medium weight
                    .foregroundStyle(AppTheme.textTertiary)
                    .padding(.top, 2)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .frame(height: height)
        .padding(AppTheme.spacingMD)
        .background {
            RoundedRectangle(cornerRadius: AppTheme.radiusCard, style: .continuous)
                .fill(AppTheme.cardBackground)
                .overlay {
                    RoundedRectangle(cornerRadius: AppTheme.radiusCard, style: .continuous)
                        .stroke(AppTheme.separator.opacity(0.12), lineWidth: 1)
                }
        }
    }
}

struct ServiceTileSmall: View {
    let title: String
    let icon: String
    
    var body: some View {
        VStack(spacing: AppTheme.spacingSM) {
            Image(systemName: icon)
                .font(.system(size: 20, weight: .semibold)) // Bold icons
                .foregroundStyle(AppTheme.accent)

            Text(title)
                .font(.system(.caption2, design: .rounded, weight: .bold)) // Bold titles
                .foregroundStyle(AppTheme.textSecondary)
        }
        .frame(maxWidth: .infinity)
        .frame(height: 80)
        .background {
            RoundedRectangle(cornerRadius: AppTheme.radiusCard, style: .continuous)
                .fill(AppTheme.cardBackground)
                .overlay {
                    RoundedRectangle(cornerRadius: AppTheme.radiusCard, style: .continuous)
                        .stroke(AppTheme.separator.opacity(0.12), lineWidth: 1)
                }
        }
    }
}
