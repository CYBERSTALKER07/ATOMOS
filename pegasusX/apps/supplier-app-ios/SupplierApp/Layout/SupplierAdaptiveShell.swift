import Network
import SwiftUI

enum CompactTab: Hashable {
    case dashboard
    case orders
    case dispatch
    case plan
    case more
}

struct FocusedSupplierSectionKey: FocusedValueKey {
    typealias Value = Binding<SupplierSection?>
}

extension FocusedValues {
    var supplierSection: Binding<SupplierSection?>? {
        get { self[FocusedSupplierSectionKey.self] }
        set { self[FocusedSupplierSectionKey.self] = newValue }
    }
}

struct SupplierAdaptiveShell: View {
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @Environment(TokenStore.self) private var tokenStore
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    @State private var vm = SupplierShellViewModel()
    private var network = NetworkLifecycleEngine.shared
    private var statusCenter = SupplierStatusCenter.shared

    var body: some View {
        @Bindable var vm = vm
        VStack(spacing: 0) {
            ClientPolicyBanner(
                message: vm.clientPolicyMessage,
                force: vm.clientPolicyForce,
                onUpdate: vm.clientPolicyMessage == nil ? nil : {
                    vm.triggerUpdateInstall()
                },
                onDismiss: vm.clientPolicyForce ? nil : { vm.dismissBanner() }
            )
            if let notice = statusCenter.notice, !notice.isEmpty {
                SupplierRefreshNotice(message: notice) {
                    statusCenter.clear()
                }
            }
            Group {
                if horizontalSizeClass == .regular {
                    regularShell
                } else {
                    compactShell
                }
            }
        }
        .focusedSceneValue(\.supplierSection, $vm.sidebarSelection)
        .onChange(of: network.syncEpoch) { _, _ in
            realtimeHub.bumpReconnect()
            Task { await SupplierSessionReconcile.run() }
        }
        .task(id: realtimeHub.reconnectEpoch) {
            await vm.loadClientPolicy()
        }
    }

    private var regularShell: some View {
        @Bindable var vm = vm
        return CollapsibleSidebar(
            title: "Pegasus Supplier",
            isExpanded: $vm.isSidebarExpanded,
            selection: $vm.sidebarSelection,
            groups: sidebarGroups
        ) {
            if let section = vm.sidebarSelection {
                NavigationStack {
                    sectionView(section)
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                        .background(PegasusMonochromeTheme.background)
                }
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
        @Bindable var vm = vm
        return TabView(selection: $vm.compactTab) {
            NavigationStack {
                sectionView(.dashboard)
            }
            .tabItem { Label("portal.nav.dashboard", systemImage: SupplierSection.dashboard.icon) }
            .tag(CompactTab.dashboard)

            NavigationStack {
                sectionView(.orders)
            }
            .tabItem { Label("portal.nav.orders", systemImage: SupplierSection.orders.icon) }
            .tag(CompactTab.orders)

            NavigationStack {
                sectionView(.dispatchPreview)
            }
            .tabItem { Label("portal.nav.dispatch", systemImage: SupplierSection.dispatchPreview.icon) }
            .tag(CompactTab.dispatch)

            NavigationStack {
                sectionView(.planningBrain)
            }
            .tabItem { Label("portal.nav.planning", systemImage: SupplierSection.planningBrain.icon) }
            .tag(CompactTab.plan)

            NavigationStack {
                MoreHubView()
            }
            .tabItem { Label("mobile_supplier.ui.more", systemImage: "ellipsis.circle") }
            .tag(CompactTab.more)
        }
        .onChange(of: vm.compactTab) { _, _ in
            PegasusHaptics.selection()
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
        case .compliance:
            ComplianceAuditView()
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
        case .returnPolicy:
            ReturnPolicySettingsView()
        case .knowledgeGraph:
            KnowledgeGraphView()
        case .replenishmentPolicies:
            ReplenishmentPoliciesView()
        case .factories:
            FactoriesView()
        case .warehouses:
            WarehousesView()
        case .crm:
            CRMView()
        case .loyalty:
            LoyaltyProgramView()
        case .entityResolution:
            EntityResolutionView()
        case .payouts:
            PayoutsView()
        case .catalogDetail:
            CatalogDetailView(productId: nil)
        case .controlTower:
            ScoredExceptionsView()
        case .playbooks:
            PlaybooksView()
        case .segmentation:
            JSONFeedView(title: "Segmentation", path: "v1/supplier/segmentation/retailers")
        case .taxRegimes:
            JSONFeedView(title: "Tax regimes", path: "v1/admin/tax-regimes", query: ["country": "UZ"])
        case .creditPolicy:
            JSONFeedView(title: "Credit policy", path: "v1/supplier/credit-program")
        case .creditAdminDisable:
            CreditAdminDisableView()
        case .flywheel:
            JSONFeedView(title: "POS flywheel", path: "v1/supplier/analytics/demand/flywheel", query: ["days": "7"])
        case .paydayCalendar:
            JSONFeedView(title: "Payday calendar", path: "v1/demand/signals", query: ["type": "PAYDAY"])
        }
    }
}
