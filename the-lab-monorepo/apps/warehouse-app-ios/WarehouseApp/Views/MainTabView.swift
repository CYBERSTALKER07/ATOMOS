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
            Tab(AppTab.dashboard.rawValue, systemImage: AppTab.dashboard.icon, value: .dashboard) {
                DashboardView()
            }
            Tab(AppTab.orders.rawValue, systemImage: AppTab.orders.icon, value: .orders) {
                OrdersView()
            }
            Tab(AppTab.drivers.rawValue, systemImage: AppTab.drivers.icon, value: .drivers) {
                DriversView()
            }
            Tab(AppTab.vehicles.rawValue, systemImage: AppTab.vehicles.icon, value: .vehicles) {
                VehiclesView()
            }
            Tab(AppTab.inventory.rawValue, systemImage: AppTab.inventory.icon, value: .inventory) {
                InventoryView()
            }
            Tab(AppTab.dispatch.rawValue, systemImage: AppTab.dispatch.icon, value: .dispatch) {
                DispatchView()
            }
            Tab(AppTab.analytics.rawValue, systemImage: AppTab.analytics.icon, value: .analytics) {
                AnalyticsView()
            }
            Tab(AppTab.treasury.rawValue, systemImage: AppTab.treasury.icon, value: .treasury) {
                TreasuryView()
            }
            Tab(AppTab.staff.rawValue, systemImage: AppTab.staff.icon, value: .staff) {
                StaffView()
            }
        }
    }
}
