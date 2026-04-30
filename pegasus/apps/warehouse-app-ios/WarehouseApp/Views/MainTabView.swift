import SwiftUI

enum AppTab: String, CaseIterable {
    case dashboard = "Dashboard"
    case orders = "Orders"
    case drivers = "Drivers"
    case vehicles = "Vehicles"
    case inventory = "Inventory"
    case dispatch = "Dispatch"
    case analytics = "Analytics"
    case treasury = "Treasury"
    case staff = "Staff"

    var icon: String {
        switch self {
        case .dashboard: "square.grid.2x2"
        case .orders: "cart"
        case .drivers: "person.badge.key"
        case .vehicles: "truck.box"
        case .inventory: "archivebox"
        case .dispatch: "paperplane"
        case .analytics: "chart.bar.xaxis"
        case .treasury: "banknote"
        case .staff: "person.2"
        }
    }
}

struct MainTabView: View {
    @State private var selectedTab: AppTab = .dashboard

    var body: some View {
        TabView(selection: $selectedTab) {
            DashboardView()
                .tabItem { Label(AppTab.dashboard.rawValue, systemImage: AppTab.dashboard.icon) }
                .tag(AppTab.dashboard)
            OrdersView()
                .tabItem { Label(AppTab.orders.rawValue, systemImage: AppTab.orders.icon) }
                .tag(AppTab.orders)
            DriversView()
                .tabItem { Label(AppTab.drivers.rawValue, systemImage: AppTab.drivers.icon) }
                .tag(AppTab.drivers)
            VehiclesView()
                .tabItem { Label(AppTab.vehicles.rawValue, systemImage: AppTab.vehicles.icon) }
                .tag(AppTab.vehicles)
            InventoryView()
                .tabItem { Label(AppTab.inventory.rawValue, systemImage: AppTab.inventory.icon) }
                .tag(AppTab.inventory)
            DispatchView()
                .tabItem { Label(AppTab.dispatch.rawValue, systemImage: AppTab.dispatch.icon) }
                .tag(AppTab.dispatch)
            AnalyticsView()
                .tabItem { Label(AppTab.analytics.rawValue, systemImage: AppTab.analytics.icon) }
                .tag(AppTab.analytics)
            TreasuryView()
                .tabItem { Label(AppTab.treasury.rawValue, systemImage: AppTab.treasury.icon) }
                .tag(AppTab.treasury)
            StaffView()
                .tabItem { Label(AppTab.staff.rawValue, systemImage: AppTab.staff.icon) }
                .tag(AppTab.staff)
        }
    }
}
