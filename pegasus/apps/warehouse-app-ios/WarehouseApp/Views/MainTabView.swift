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
    @Environment(\.scenePhase) private var scenePhase
    @State private var selectedTab: AppTab = .dashboard
    @State private var refreshEpoch: Int = 0

    var body: some View {
        TabView(selection: $selectedTab) {
            DashboardView()
                .id("\(AppTab.dashboard.rawValue)-\(refreshEpoch)")
                .tabItem { Label(AppTab.dashboard.rawValue, systemImage: AppTab.dashboard.icon) }
                .tag(AppTab.dashboard)
            OrdersView()
                .id("\(AppTab.orders.rawValue)-\(refreshEpoch)")
                .tabItem { Label(AppTab.orders.rawValue, systemImage: AppTab.orders.icon) }
                .tag(AppTab.orders)
            DriversView()
                .id("\(AppTab.drivers.rawValue)-\(refreshEpoch)")
                .tabItem { Label(AppTab.drivers.rawValue, systemImage: AppTab.drivers.icon) }
                .tag(AppTab.drivers)
            VehiclesView()
                .id("\(AppTab.vehicles.rawValue)-\(refreshEpoch)")
                .tabItem { Label(AppTab.vehicles.rawValue, systemImage: AppTab.vehicles.icon) }
                .tag(AppTab.vehicles)
            InventoryView()
                .id("\(AppTab.inventory.rawValue)-\(refreshEpoch)")
                .tabItem { Label(AppTab.inventory.rawValue, systemImage: AppTab.inventory.icon) }
                .tag(AppTab.inventory)
            DispatchView()
                .id("\(AppTab.dispatch.rawValue)-\(refreshEpoch)")
                .tabItem { Label(AppTab.dispatch.rawValue, systemImage: AppTab.dispatch.icon) }
                .tag(AppTab.dispatch)
            AnalyticsView()
                .id("\(AppTab.analytics.rawValue)-\(refreshEpoch)")
                .tabItem { Label(AppTab.analytics.rawValue, systemImage: AppTab.analytics.icon) }
                .tag(AppTab.analytics)
            TreasuryView()
                .id("\(AppTab.treasury.rawValue)-\(refreshEpoch)")
                .tabItem { Label(AppTab.treasury.rawValue, systemImage: AppTab.treasury.icon) }
                .tag(AppTab.treasury)
            StaffView()
                .id("\(AppTab.staff.rawValue)-\(refreshEpoch)")
                .tabItem { Label(AppTab.staff.rawValue, systemImage: AppTab.staff.icon) }
                .tag(AppTab.staff)
        }
        .buttonStyle(PressableButtonStyle())
        .onChange(of: scenePhase) { phase in
            if phase == .active {
                refreshEpoch += 1
            }
        }
    }
}
