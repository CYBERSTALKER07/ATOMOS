import SwiftUI

struct FocusedFactorySectionKey: FocusedValueKey {
    typealias Value = Binding<FactorySection?>
}

extension FocusedValues {
    var factorySection: Binding<FactorySection?>? {
        get { self[FocusedFactorySectionKey.self] }
        set { self[FocusedFactorySectionKey.self] = newValue }
    }
}

struct FactoryAdaptiveShell: View {
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @State private var sidebarSelection: FactorySection? = .dashboard
    @State private var isSidebarExpanded = true
    @State private var compactTab: FactoryCompactTab = .dashboard
    @State private var pendingTransferFilter: String?
    @State private var morePath: [FactorySection] = []

    private var network = NetworkLifecycleEngine.shared

    var body: some View {
        VStack(spacing: 0) {
            if !network.isOnline {
                FactoryRuntimeBanner(
                    tone: .offline,
                    message: network.state == .flapping
                        ? "Network unstable — stabilizing connection before syncing"
                        : "Offline — changes will sync once connectivity returns"
                )
                .padding(.horizontal)
                .padding(.vertical, 4)
            }

            Group {
                if horizontalSizeClass == .regular {
                    regularShell
                } else {
                    compactShell
                }
            }
        }
        .focusedSceneValue(\.factorySection, $sidebarSelection)
    }

    private var regularShell: some View {
        CollapsibleSidebar(
            title: "Pegasus Factory",
            isExpanded: $isSidebarExpanded,
            selection: $sidebarSelection,
            groups: sidebarGroups
        ) {
            if let section = sidebarSelection {
                NavigationStack {
                    sectionView(section)
                }
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
            NavigationStack {
                sectionView(.dashboard)
            }
            .tabItem { Label("portal.nav.dashboard", systemImage: FactorySection.dashboard.icon) }
            .tag(FactoryCompactTab.dashboard)

            NavigationStack {
                sectionView(.loadingBay)
            }
            .tabItem { Label("portal.nav.loading_bay", systemImage: FactorySection.loadingBay.icon) }
            .tag(FactoryCompactTab.loadingBay)

            NavigationStack {
                sectionView(.payloadLoad)
            }
            .tabItem { Label("portal.nav.payload", systemImage: FactorySection.payloadLoad.icon) }
            .tag(FactoryCompactTab.payload)

            NavigationStack {
                sectionView(.transfers)
            }
            .tabItem { Label("portal.nav.transfers", systemImage: FactorySection.transfers.icon) }
            .tag(FactoryCompactTab.transfers)

            NavigationStack(path: $morePath) {
                FactoryMoreHubView()
                    .navigationDestination(for: FactorySection.self) { section in
                        sectionView(section)
                    }
            }
            .tabItem { Label("mobile_factory.ui.more", systemImage: "ellipsis.circle") }
            .tag(FactoryCompactTab.more)
        }
    }

    private func openSecondarySection(_ section: FactorySection) {
        sidebarSelection = section
        if horizontalSizeClass != .regular {
            morePath = [section]
            compactTab = .more
        }
    }

    @ViewBuilder
    private func sectionView(_ section: FactorySection) -> some View {
        switch section {
        case .dashboard:
            DashboardView(
                onOpenSupplyRequests: { openSecondarySection(.supplyRequests) },
                onOpenPayloadOverride: { openSecondarySection(.payloadOverride) },
                onOpenManifestExceptions: { openSecondarySection(.manifestExceptions) },
                onOpenAnalytics: { openSecondarySection(.analytics) },
                onOpenInsights: { openSecondarySection(.insights) },
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
            LocationSettingsView()
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
}
