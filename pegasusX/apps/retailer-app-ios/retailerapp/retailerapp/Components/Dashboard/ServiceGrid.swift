import SwiftUI

struct ServiceGrid: View {
    let activeOrdersCount: Int
    let predictionsCount: Int

    var body: some View {
        VStack(spacing: AppTheme.spacingMD) {
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

            HStack(spacing: AppTheme.spacingMD) {
                NavigationLink {
                    OrdersView()
                } label: {
                    ServiceTileView(title: "Orders", icon: "shippingbox.fill", subtitle: "\(activeOrdersCount) active", height: 120)
                }
                .buttonStyle(.pressable)

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
