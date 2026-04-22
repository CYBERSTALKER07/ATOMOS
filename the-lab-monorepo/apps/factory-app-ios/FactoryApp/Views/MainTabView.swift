import SwiftUI

enum AppTab: String, CaseIterable {
    case dashboard = "Dashboard"
    case loadingBay = "Loading Bay"
    case transfers = "Transfers"
    case fleet = "Fleet"
    case staff = "Staff"
    case insights = "Insights"

    var icon: String {
        switch self {
        case .dashboard: "square.grid.2x2"
            
        case .loadingBay: "shippingbox"
        case .transfers: "arrow.left.arrow.right"
        case .fleet: "truck.box"
        case .staff: "person.2"
        case .insights: "chart.bar.xaxis"
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
            LoadingBayView()
                .tabItem { Label(AppTab.loadingBay.rawValue, systemImage: AppTab.loadingBay.icon) }
                .tag(AppTab.loadingBay)
            TransferListView()
                .tabItem { Label(AppTab.transfers.rawValue, systemImage: AppTab.transfers.icon) }
                .tag(AppTab.transfers)
            FleetView()
                .tabItem { Label(AppTab.fleet.rawValue, systemImage: AppTab.fleet.icon) }
                .tag(AppTab.fleet)
            StaffView()
                .tabItem { Label(AppTab.staff.rawValue, systemImage: AppTab.staff.icon) }
                .tag(AppTab.staff)
            InsightsView()
                .tabItem { Label(AppTab.insights.rawValue, systemImage: AppTab.insights.icon) }
                .tag(AppTab.insights)
        }
    }
}
