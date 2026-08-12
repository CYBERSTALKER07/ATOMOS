import SwiftUI
import Network

struct WarehouseAdaptiveShell: View {
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @Environment(WarehouseRealtimeHub.self) private var realtimeHub
    @State private var sidebarSelection: WarehouseSection? = .dashboard
    @State private var isSidebarExpanded = true
    @State private var compactTab: WarehouseCompactTab = .dashboard
    @State private var pathMonitor: NWPathMonitor?
    @State private var wasOffline = false

    var body: some View {
        Group {
            if horizontalSizeClass == .regular {
                regularShell
            } else {
                compactShell
            }
        }
        .onAppear { startNetworkMonitor() }
        .onDisappear {
            pathMonitor?.cancel()
            pathMonitor = nil
        }
    }

    private var regularShell: some View {
        CollapsibleSidebar(
            title: "Pegasus Warehouse",
            isExpanded: $isSidebarExpanded,
            selection: $sidebarSelection,
            groups: sidebarGroups,
        ) {
            if let section = sidebarSelection {
                sectionView(section)
            } else {
                ContentUnavailableView("Select a section", systemImage: "sidebar.left")
            }
        }
    }

    private var sidebarGroups: [(title: String, items: [CollapsibleSidebarItem<WarehouseSection>])] {
        [
            ("Primary", items(for: WarehouseSection.primarySections)),
            ("Fulfillment", items(for: WarehouseSection.fulfillmentSections)),
            ("Inventory", items(for: WarehouseSection.inventorySections)),
            ("Operations", items(for: WarehouseSection.operationsSections)),
            ("Portal only", items(for: WarehouseSection.portalSections)),
        ]
    }

    private func items(for sections: [WarehouseSection]) -> [CollapsibleSidebarItem<WarehouseSection>] {
        sections.map { CollapsibleSidebarItem(tag: $0, label: $0.rawValue, icon: $0.icon) }
    }

    private var compactShell: some View {
        TabView(selection: $compactTab) {
            sectionView(.dashboard)
                .tabItem { Label("portal.nav.dashboard", systemImage: WarehouseSection.dashboard.icon) }
                .tag(WarehouseCompactTab.dashboard)

            sectionView(.orders)
                .tabItem { Label("portal.nav.orders", systemImage: WarehouseSection.orders.icon) }
                .tag(WarehouseCompactTab.orders)

            sectionView(.dispatch)
                .tabItem { Label("portal.nav.dispatch", systemImage: WarehouseSection.dispatch.icon) }
                .tag(WarehouseCompactTab.dispatch)

            NavigationStack {
                MoreHubView()
            }
            .tabItem { Label("mobile_warehouse.ui.more", systemImage: "ellipsis.circle") }
            .tag(WarehouseCompactTab.more)
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
        case .preorders:
            NavigationStack { PreordersView() }
        case .stockCommitments:
            NavigationStack { StockCommitmentsView() }
        case .tomorrowBoard:
            NavigationStack { TomorrowBoardView() }
        case .replenishment:
            NavigationStack { ReplenishmentView() }
        case .demandForecast:
            DemandForecastView()
        case .retailers:
            CRMView()
        case .returns:
            ReturnsView()
        case .coldChain:
            ColdChainView()
        case .laborCapacity:
            LaborCapacityView()
        case .exceptions:
            ExceptionsView()
        case .claims:
            ClaimsView()
        case .rescues:
            RescuesView()
        case .paymentConfig:
            NavigationStack { PaymentConfigView() }
        case .opsSettings:
            NavigationStack { OpsSettingsView() }
        case .returnPolicy:
            NavigationStack { ReturnPolicySettingsView() }
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
