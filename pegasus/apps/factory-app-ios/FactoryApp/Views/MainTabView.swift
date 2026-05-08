import SwiftUI
import Network

enum AppTab: String, CaseIterable {
    case dashboard = "Dashboard"
    case loadingBay = "Loading Bay"
    case transfers = "Transfers"
    case supplyRequests = "Supply"
    case payloadOverride = "Override"
    case fleet = "Fleet"
    case staff = "Staff"
    case insights = "Insights"

    var icon: String {
        switch self {
        case .dashboard: "square.grid.2x2"
        case .loadingBay: "shippingbox"
        case .transfers: "arrow.left.arrow.right"
        case .supplyRequests: "checklist"
        case .payloadOverride: "arrow.left.arrow.right.square"
        case .fleet: "truck.box"
        case .staff: "person.2"
        case .insights: "chart.bar.xaxis"
        }
    }
}

struct MainTabView: View {
    @Environment(\.scenePhase) private var scenePhase
    @State private var selectedTab: AppTab = .dashboard
    @State private var refreshEpoch: Int = 0
    @State private var pathMonitor: NWPathMonitor?
    @State private var wasOffline = false

    var body: some View {
        TabView(selection: $selectedTab) {
            DashboardView(
                onOpenSupplyRequests: { selectedTab = .supplyRequests },
                onOpenPayloadOverride: { selectedTab = .payloadOverride }
            )
                .id("\(AppTab.dashboard.rawValue)-\(refreshEpoch)")
                .tabItem { Label(AppTab.dashboard.rawValue, systemImage: AppTab.dashboard.icon) }
                .tag(AppTab.dashboard)
            LoadingBayView()
                .id("\(AppTab.loadingBay.rawValue)-\(refreshEpoch)")
                .tabItem { Label(AppTab.loadingBay.rawValue, systemImage: AppTab.loadingBay.icon) }
                .tag(AppTab.loadingBay)
            TransferListView()
                .id("\(AppTab.transfers.rawValue)-\(refreshEpoch)")
                .tabItem { Label(AppTab.transfers.rawValue, systemImage: AppTab.transfers.icon) }
                .tag(AppTab.transfers)
            SupplyRequestsView()
                .id("\(AppTab.supplyRequests.rawValue)-\(refreshEpoch)")
                .tabItem { Label(AppTab.supplyRequests.rawValue, systemImage: AppTab.supplyRequests.icon) }
                .tag(AppTab.supplyRequests)
            PayloadOverrideView()
                .id("\(AppTab.payloadOverride.rawValue)-\(refreshEpoch)")
                .tabItem { Label(AppTab.payloadOverride.rawValue, systemImage: AppTab.payloadOverride.icon) }
                .tag(AppTab.payloadOverride)
            FleetView()
                .id("\(AppTab.fleet.rawValue)-\(refreshEpoch)")
                .tabItem { Label(AppTab.fleet.rawValue, systemImage: AppTab.fleet.icon) }
                .tag(AppTab.fleet)
            StaffView()
                .id("\(AppTab.staff.rawValue)-\(refreshEpoch)")
                .tabItem { Label(AppTab.staff.rawValue, systemImage: AppTab.staff.icon) }
                .tag(AppTab.staff)
            InsightsView()
                .id("\(AppTab.insights.rawValue)-\(refreshEpoch)")
                .tabItem { Label(AppTab.insights.rawValue, systemImage: AppTab.insights.icon) }
                .tag(AppTab.insights)
        }
        .onChange(of: scenePhase) { phase in
            if phase == .active {
                refreshEpoch += 1
            }
        }
        .onAppear {
            guard pathMonitor == nil else { return }
            let monitor = NWPathMonitor()
            let queue = DispatchQueue(label: "com.pegasus.factory.main-tab.network")
            monitor.pathUpdateHandler = { path in
                DispatchQueue.main.async {
                    if path.status == .satisfied {
                        if wasOffline {
                            refreshEpoch += 1
                        }
                        wasOffline = false
                    } else {
                        wasOffline = true
                    }
                }
            }
            monitor.start(queue: queue)
            pathMonitor = monitor
        }
        .onDisappear {
            pathMonitor?.cancel()
            pathMonitor = nil
        }
    }
}
