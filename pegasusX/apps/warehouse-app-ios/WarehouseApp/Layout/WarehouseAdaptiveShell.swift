import SwiftUI
import Network

struct WarehouseAdaptiveShell: View {
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @Environment(\.scenePhase) private var scenePhase
    @Environment(WarehouseRealtimeHub.self) private var realtimeHub
    @State private var sidebarSelection: WarehouseSection? = .dashboard
    @State private var compactTab: WarehouseCompactTab = .dashboard
    @State private var refreshEpoch = 0
    @State private var pathMonitor: NWPathMonitor?
    @State private var wasOffline = false

    private var effectiveRefreshEpoch: Int { refreshEpoch + realtimeHub.refreshEpoch }

    var body: some View {
        Group {
            if horizontalSizeClass == .regular {
                regularShell
            } else {
                compactShell
            }
        }
        .onChange(of: scenePhase) { phase in
            if phase == .active {
                refreshEpoch += 1
            }
        }
        .onAppear { startNetworkMonitor() }
        .onDisappear {
            pathMonitor?.cancel()
            pathMonitor = nil
        }
    }

    private var regularShell: some View {
        NavigationSplitView {
            List(selection: $sidebarSelection) {
                Section("Primary") {
                    sidebarRows(WarehouseSection.primarySections)
                }
                Section("Fulfillment") {
                    sidebarRows(WarehouseSection.fulfillmentSections)
                }
                Section("Inventory") {
                    sidebarRows(WarehouseSection.inventorySections)
                }
                Section("Operations") {
                    sidebarRows(WarehouseSection.operationsSections)
                }
                Section("Portal only") {
                    sidebarRows(WarehouseSection.portalSections)
                }
            }
            .navigationTitle("Pegasus Warehouse")
            .listStyle(.sidebar)
        } detail: {
            if let section = sidebarSelection {
                sectionView(section)
                    .id("\(section.id)-\(effectiveRefreshEpoch)")
            } else {
                ContentUnavailableView("Select a section", systemImage: "sidebar.left")
            }
        }
        .navigationSplitViewStyle(.balanced)
    }

    private var compactShell: some View {
        TabView(selection: $compactTab) {
            sectionView(.dashboard)
                .id("dash-\(effectiveRefreshEpoch)")
                .tabItem { Label("Dashboard", systemImage: WarehouseSection.dashboard.icon) }
                .tag(WarehouseCompactTab.dashboard)

            sectionView(.orders)
                .id("orders-\(effectiveRefreshEpoch)")
                .tabItem { Label("Orders", systemImage: WarehouseSection.orders.icon) }
                .tag(WarehouseCompactTab.orders)

            sectionView(.dispatch)
                .id("dispatch-\(effectiveRefreshEpoch)")
                .tabItem { Label("Dispatch", systemImage: WarehouseSection.dispatch.icon) }
                .tag(WarehouseCompactTab.dispatch)

            NavigationStack {
                MoreHubView()
                    .id("more-hub-\(effectiveRefreshEpoch)")
            }
            .tabItem { Label("More", systemImage: "ellipsis.circle") }
            .tag(WarehouseCompactTab.more)
        }
    }

    @ViewBuilder
    private func sidebarRows(_ sections: [WarehouseSection]) -> some View {
        ForEach(sections) { section in
            Label(section.rawValue, systemImage: section.icon)
                .tag(section)
        }
    }

    @ViewBuilder
    private func sectionView(_ section: WarehouseSection) -> some View {
        switch section {
        case .dashboard:
            DashboardView()
        case .orders:
            OrdersView()
        case .drivers:
            DriversView()
        case .vehicles:
            VehiclesView()
        case .inventory:
            InventoryView()
        case .dispatch:
            DispatchView()
        case .analytics:
            AnalyticsView()
        case .treasury:
            TreasuryView()
        case .staff:
            StaffView()
        case .manifests:
            ManifestsView()
        case .dispatchSettings:
            NavigationStack { DispatchSettingsView() }
        case .fleetLiveMap:
            NavigationStack { FleetLiveMapView() }
        case .transferActions:
            NavigationStack { TransferActionsView() }
        case .products:
            ProductsView()
        case .supplyRequests:
            NavigationStack { SupplyRequestsHubView() }
        case .replenishment:
            NavigationStack { ReplenishmentView() }
        case .demandForecast:
            DemandForecastView()
        case .retailers:
            CRMView()
        case .returns:
            ReturnsView()
        case .paymentConfig:
            NavigationStack { PaymentConfigView() }
        case .opsSettings:
            NavigationStack { OpsSettingsView() }
        case .notifications:
            NavigationStack { NotificationInboxView() }
        case .portalSetup, .portalProfile, .portalSearch:
            if let feature = section.portalFeature {
                PortalHandoffView(feature: feature)
            }
        }
    }

    private func startNetworkMonitor() {
        guard pathMonitor == nil else { return }
        let monitor = NWPathMonitor()
        monitor.pathUpdateHandler = { path in
            DispatchQueue.main.async {
                if path.status == .satisfied {
                    if wasOffline { refreshEpoch += 1 }
                    wasOffline = false
                } else {
                    wasOffline = true
                }
            }
        }
        monitor.start(queue: DispatchQueue(label: "com.pegasusx.warehouse.network"))
        pathMonitor = monitor
    }
}
