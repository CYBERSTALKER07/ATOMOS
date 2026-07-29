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
    @State private var isSidebarExpanded = true
    @State private var compactTab: CompactTab = .dashboard
    @State private var clientPolicyMessage: String?
    @State private var clientPolicyForce = false
    @State private var pendingManifest: AutoUpdater.Manifest?

    @State private var pathMonitor: NWPathMonitor?
    @State private var wasOffline = false

    var body: some View {
        VStack(spacing: 0) {
            ClientPolicyBanner(
                message: clientPolicyMessage,
                force: clientPolicyForce,
                onUpdate: clientPolicyMessage == nil ? nil : {
                    AutoUpdater.shared.promptInstall(manifest: pendingManifest, force: clientPolicyForce)
                },
                onDismiss: clientPolicyForce ? nil : { clientPolicyMessage = nil },
            )
            Group {
                if horizontalSizeClass == .regular {
                    regularShell
                } else {
                    compactShell
                }
            }
        }
        .onAppear { startNetworkMonitor() }
        .onDisappear {
            pathMonitor?.cancel()
            pathMonitor = nil
        }
        .task(id: realtimeHub.reconnectEpoch) {
            await loadClientPolicy()
        }
    }

    private var regularShell: some View {
        CollapsibleSidebar(
            title: "Pegasus Supplier",
            isExpanded: $isSidebarExpanded,
            selection: $sidebarSelection,
            groups: sidebarGroups,
        ) {
            if let section = sidebarSelection {
                sectionView(section)
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                    .background(PegasusMonochromeTheme.background)
            } else {
                ContentUnavailableView("Select a section", systemImage: "sidebar.left")
            }
        }
    }

    private var sidebarGroups: [(title: String, items: [CollapsibleSidebarItem<SupplierSection>])] {
        [
            ("Primary", items(for: SupplierSection.compactTabs)),
            ("Operations", items(for: SupplierSection.opsSections)),
            ("Intelligence", items(for: SupplierSection.intelligenceSections)),
            ("Network", items(for: SupplierSection.networkSections)),
            ("Account", items(for: SupplierSection.accountSections)),
        ]
    }

    private func items(for sections: [SupplierSection]) -> [CollapsibleSidebarItem<SupplierSection>] {
        sections.map { CollapsibleSidebarItem(tag: $0, label: $0.rawValue, icon: $0.icon) }
    }

    private var compactShell: some View {
        TabView(selection: $compactTab) {
            sectionView(.dashboard)
                .tabItem { Label("Dashboard", systemImage: SupplierSection.dashboard.icon) }
                .tag(CompactTab.dashboard)

            sectionView(.orders)
                .tabItem { Label("Orders", systemImage: SupplierSection.orders.icon) }
                .tag(CompactTab.orders)

            sectionView(.fleet)
                .tabItem { Label("Fleet", systemImage: SupplierSection.fleet.icon) }
                .tag(CompactTab.fleet)

            NavigationStack {
                MoreHubView()
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
            OrdersHubView()
        case .fleet:
            FleetView()
        case .exceptions:
            ExceptionsView()
        case .shopClosed:
            ShopClosedView()
        case .negotiations:
            SupplierEmptyView(title: "Unavailable", message: "Quantity negotiation is disabled.")
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
        case .payments:
            PaymentsView()
        case .operations:
            OperationsView()
        case .analytics:
            AnalyticsView()
        case .aiRecommendations:
            AIRecommendationsView()
        case .geoReport:
            GeoReportView()
        case .topology:
            TopologyView()
        case .deliveryZones:
            DeliveryZonesView()
        case .supplyLanes:
            SupplyLanesView()
        case .catalog:
            CatalogView()
        case .inventory:
            InventoryView()
        case .promotions:
            PromotionsView()
        case .pricing:
            PricingView()
        case .returns:
            ReturnsView()
        case .reconciliation:
            ReconciliationView()
        case .notifications:
            NotificationInboxView()
        case .earnings:
            EarningsView()
        case .profile:
            ProfileView()
        case .earlyComplete:
            EarlyCompleteView()
        case .orgFleet:
            OrgFleetView()
        case .treasury:
            TreasuryHubView()
        case .retailerOverrides:
            RetailerOverridesView()
        case .chargebacks:
            ChargebacksView()
        case .claims:
            ClaimsView()
        case .claimChargebacks:
            ClaimChargebacksView()
        case .businessSetup:
            BusinessSetupView()
        case .inventoryImport:
            InventoryImportView()
        case .demandForecast:
            DemandHistoryView()
        case .planningBrain:
            PlanningBrainView()
        case .planningSettings:
            PlanningSettingsView()
        case .knowledgeGraph:
            KnowledgeGraphView()
        case .replenishmentPolicies:
            ReplenishmentPoliciesView()
        case .factories:
            FactoriesView()
        case .warehouses:
            WarehousesView()
        case .catalogDetail:
            CatalogDetailView(productId: nil)
        }
    }

    private func startNetworkMonitor() {
        guard pathMonitor == nil else { return }
        let monitor = NWPathMonitor()
        monitor.pathUpdateHandler = { path in
            DispatchQueue.main.async {
                if path.status == .satisfied {
                    if wasOffline {
                        Task { await SupplierSessionReconcile.run() }
                    }
                    wasOffline = false
                } else {
                    wasOffline = true
                }
            }
        }
        monitor.start(queue: DispatchQueue(label: "com.pegasusx.supplier.network"))
        pathMonitor = monitor
    }

    @MainActor
    private func loadClientPolicy() async {
        let version = Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "1.0.0"
        do {
            struct ClientPolicy: Decodable {
                let outdated: Bool
                let forceUpdate: Bool
                let updateDeferred: Bool
                let minimumVersion: String
                let recommendedVersion: String?
                let updateURL: String?
                let deferReason: String?

                enum CodingKeys: String, CodingKey {
                    case outdated
                    case forceUpdate = "force_update"
                    case updateDeferred = "update_deferred"
                    case minimumVersion = "minimum_version"
                    case recommendedVersion = "recommended_version"
                    case updateURL = "update_url"
                    case deferReason = "defer_reason"
                }
            }
            let policy: ClientPolicy = try await APIClient.shared.get(
                "v1/platform/client-policy",
                query: [
                    "role": EnterpriseUpdateConfig.policyRole,
                    "platform": "ios",
                    "version": version,
                    "channel": EnterpriseUpdateConfig.channel,
                ],
            )
            let state = await AutoUpdater.shared.evaluate(
                outdated: policy.outdated,
                forceUpdate: policy.forceUpdate,
                updateDeferred: policy.updateDeferred,
                minimumVersion: policy.minimumVersion,
                recommendedVersion: policy.recommendedVersion,
                deferReason: policy.deferReason,
                updateURL: policy.updateURL,
            )
            clientPolicyMessage = state.message
            clientPolicyForce = state.force
            pendingManifest = state.manifest
            // Force-required: surface install sheet immediately (still dismissible only if deferred).
            if state.force, state.available {
                AutoUpdater.shared.promptInstall(manifest: state.manifest, force: true)
            }
        } catch {
            // Policy fetch is optional on local/dev stacks.
        }
    }
}
