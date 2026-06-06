import SwiftUI
import Network

enum CompactTab: Hashable {
    case dashboard
    case orders
    case fleet
    case more
}

struct SupplierAdaptiveShell: View {
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @Environment(TokenStore.self) private var tokenStore
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    @State private var sidebarSelection: SupplierSection? = .dashboard
    @State private var compactTab: CompactTab = .dashboard
    @State private var refreshEpoch = 0

    private var effectiveRefreshEpoch: Int { refreshEpoch + realtimeHub.refreshEpoch }
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
        NavigationSplitView {
            List(SupplierSection.sidebarSections, selection: $sidebarSelection) { section in
                Label(section.rawValue, systemImage: section.icon)
                    .tag(section)
            }
            .navigationTitle("Pegasus Supplier")
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
                .tabItem { Label("Dashboard", systemImage: SupplierSection.dashboard.icon) }
                .tag(CompactTab.dashboard)

            sectionView(.orders)
                .id("orders-\(effectiveRefreshEpoch)")
                .tabItem { Label("Orders", systemImage: SupplierSection.orders.icon) }
                .tag(CompactTab.orders)

            sectionView(.fleet)
                .id("fleet-\(effectiveRefreshEpoch)")
                .tabItem { Label("Fleet", systemImage: SupplierSection.fleet.icon) }
                .tag(CompactTab.fleet)

            NavigationStack {
                MoreHubView()
                    .id("more-hub-\(effectiveRefreshEpoch)")
            }
            .tabItem { Label("More", systemImage: "ellipsis.circle") }
            .tag(CompactTab.more)
        }
    }

    @ViewBuilder
    private func sectionView(_ section: SupplierSection) -> some View {
        switch section {
        case .dashboard:
            DashboardView()
        case .orders:
            OrdersView()
        case .fleet:
            FleetView()
        case .exceptions:
            ExceptionsView()
        case .shopClosed:
            ShopClosedView()
        case .negotiations:
            NegotiationsView()
        case .manifests:
            ManifestsView()
        case .dispatchPreview:
            DispatchPreviewView()
        case .activity:
            ActivityView()
        case .fleetOrders:
            FleetOrdersView()
        case .ledger:
            LedgerView()
        case .operations:
            OperationsView()
        case .inventory:
            InventoryView()
        case .earnings:
            EarningsView()
        case .profile:
            ProfileView()
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
        monitor.start(queue: DispatchQueue(label: "com.pegasusx.supplier.network"))
        pathMonitor = monitor
    }
}
