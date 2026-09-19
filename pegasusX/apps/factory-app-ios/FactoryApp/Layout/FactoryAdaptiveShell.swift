import SwiftUI
import Network

struct FactoryAdaptiveShell: View {
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @State private var sidebarSelection: FactorySection? = .dashboard
    @State private var isSidebarExpanded = true
    @State private var compactTab: FactoryCompactTab = .dashboard
    @State private var pendingTransferFilter: String?
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
            title: "Pegasus Factory",
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

    private var sidebarGroups: [(title: String, items: [CollapsibleSidebarItem<FactorySection>])] {
        [
            ("Primary", items(for: FactorySection.primarySections)),
            ("Operations", items(for: FactorySection.operationsSections)),
            ("Intelligence", items(for: FactorySection.intelligenceSections)),
        ]
    }

    private func items(for sections: [FactorySection]) -> [CollapsibleSidebarItem<FactorySection>] {
        sections.map { CollapsibleSidebarItem(tag: $0, label: $0.rawValue, icon: $0.icon) }
    }

    private var compactShell: some View {
        TabView(selection: $compactTab) {
            sectionView(.dashboard)
                .tabItem { Label("portal.nav.dashboard", systemImage: FactorySection.dashboard.icon) }
                .tag(FactoryCompactTab.dashboard)
            sectionView(.loadingBay)
                .tabItem { Label("portal.nav.loading_bay", systemImage: FactorySection.loadingBay.icon) }
                .tag(FactoryCompactTab.loadingBay)
            sectionView(.payloadLoad)
                .tabItem { Label("portal.nav.payload", systemImage: FactorySection.payloadLoad.icon) }
                .tag(FactoryCompactTab.payload)
            sectionView(.transfers)
                .tabItem { Label("portal.nav.transfers", systemImage: FactorySection.transfers.icon) }
                .tag(FactoryCompactTab.transfers)
            NavigationStack {
                FactoryMoreHubView { section in
                    sidebarSelection = section
                    compactTab = .dashboard
                }
            }
            .tabItem { Label("mobile_factory.ui.more", systemImage: "ellipsis.circle") }
            .tag(FactoryCompactTab.more)
        }
    }

    @ViewBuilder
    private func sectionView(_ section: FactorySection) -> some View {
        switch section {
        case .dashboard:
            DashboardView(
                onOpenSupplyRequests: { sidebarSelection = .supplyRequests },
                onOpenPayloadOverride: { sidebarSelection = .payloadOverride },
                onOpenManifestExceptions: { sidebarSelection = .manifestExceptions },
                onOpenAnalytics: { sidebarSelection = .analytics },
                onOpenInsights: { sidebarSelection = .insights },
                onOpenTransfers: { key in
                    pendingTransferFilter = key
                    sidebarSelection = .transfers
                    compactTab = .transfers
                },
                onOpenLoadingBay: {
                    sidebarSelection = .loadingBay
                    compactTab = .loadingBay
                }
            )
        case .loadingBay:
            LoadingBayView()
        case .transfers:
            TransferListView(initialFilter: pendingTransferFilter)
        case .fleet:
            FleetView()
        case .staff:
            StaffView()
        case .location:
            NavigationStack { LocationSettingsView() }
        case .supplyRequests:
            SupplyRequestsView()
        case .payloadOverride:
            PayloadOverrideView()
        case .payloadLoad:
            PayloadLoadView()
        case .manifests:
            ManifestsView()
        case .manifestExceptions:
            ManifestExceptionsView()
        case .insights:
            InsightsView()
        case .analytics:
            AnalyticsView()
        case .notifications:
            NotificationInboxView()
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
        monitor.start(queue: DispatchQueue(label: "com.pegasus.factory.network"))
        pathMonitor = monitor
    }
}
