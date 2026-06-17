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
    @State private var clientPolicyMessage: String?

    private var effectiveRefreshEpoch: Int { refreshEpoch + realtimeHub.refreshEpoch }
    @State private var pathMonitor: NWPathMonitor?
    @State private var wasOffline = false

    var body: some View {
        VStack(spacing: 0) {
            ClientPolicyBanner(message: clientPolicyMessage)
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
        .task(id: effectiveRefreshEpoch) {
            await loadClientPolicy()
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
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                    .background(SupplierTheme.background)
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
        case .businessSetup:
            BusinessSetupView()
        case .inventoryImport:
            InventoryImportView()
        case .demandForecast:
            DemandHistoryView()
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

    @MainActor
    private func loadClientPolicy() async {
        let version = Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "1.0.0"
        do {
            struct ClientPolicy: Decodable {
                let outdated: Bool
                let forceUpdate: Bool
                let minimumVersion: String
                let deferReason: String?

                enum CodingKeys: String, CodingKey {
                    case outdated
                    case forceUpdate = "force_update"
                    case minimumVersion = "minimum_version"
                    case deferReason = "defer_reason"
                }
            }
            let policy: ClientPolicy = try await APIClient.shared.get(
                "v1/platform/client-policy",
                query: [
                    "role": "ADMIN",
                    "platform": "ios",
                    "version": version,
                    "channel": "production",
                ],
            )
            if policy.outdated || policy.forceUpdate {
                var message = policy.forceUpdate ? "Update required" : "Update available"
                if !policy.minimumVersion.isEmpty {
                    message += " — minimum version \(policy.minimumVersion)"
                }
                if let deferReason = policy.deferReason, !deferReason.isEmpty {
                    message += ". \(deferReason)"
                }
                clientPolicyMessage = message
            } else {
                clientPolicyMessage = nil
            }
        } catch {
            // Policy fetch is optional on local/dev stacks.
        }
    }
}
