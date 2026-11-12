import SwiftUI
import Network

struct FocusedWarehouseSectionKey: FocusedValueKey {
    typealias Value = Binding<WarehouseSection?>
}

extension FocusedValues {
    var warehouseSection: Binding<WarehouseSection?>? {
        get { self[FocusedWarehouseSectionKey.self] }
        set { self[FocusedWarehouseSectionKey.self] = newValue }
    }
}

struct WarehouseAdaptiveShell: View {
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @Environment(WarehouseRealtimeHub.self) private var realtimeHub
    @Environment(TokenStore.self) private var tokenStore

    @State private var sidebarSelection: WarehouseSection? = .dashboard
    @State private var columnVisibility: NavigationSplitViewVisibility = .all
    @State private var compactTab: WarehouseCompactTab = .dashboard
    @State private var showInspector: Bool = false
    @State private var searchFilter: String = ""
    @FocusState private var isSidebarFilterFocused: Bool
    private var network = NetworkLifecycleEngine.shared
    private var wasOffline: Bool { !network.isOnline }

    var body: some View {
        Group {
            if horizontalSizeClass == .regular {
                regularControlTowerShell
            } else {
                compactFloorSuiteShell
            }
        }
        .tint(TacticalTheme.accentCobalt)
        .focusedSceneValue(\.warehouseSection, $sidebarSelection)
    }

    // MARK: - iPad Control Tower 3-Column Split View
    private var regularControlTowerShell: some View {
        NavigationSplitView(columnVisibility: $columnVisibility) {
            sidebarView
                .navigationSplitViewColumnWidth(min: 240, ideal: 270, max: 320)
        } content: {
            NavigationStack {
                contentColumnView
            }
            .navigationSplitViewColumnWidth(min: 340, ideal: 400, max: 500)
        } detail: {
            NavigationStack {
                detailWorkbenchView
            }
        }
        .inspector(isPresented: $showInspector) {
            TacticalInspectorView(
                title: sidebarSelection?.rawValue ?? "Warehouse Depot",
                subtitle: "ZONE UZ-TAS-01 · DOCK BAY 3",
                status: "ACTIVE"
            )
            .inspectorColumnWidth(min: 300, ideal: 340, max: 420)
        }
    }

    // MARK: - Sidebar Navigation Rail
    private var sidebarView: some View {
        VStack(spacing: 0) {
            // Tactical Brand Header
            VStack(alignment: .leading, spacing: 6) {
                HStack(spacing: 8) {
                    Image(systemName: "shippingbox.and.arrow.backward.fill")
                        .font(.system(.callout, weight: .bold))
                        .foregroundStyle(TacticalTheme.accentCobalt)

                    Text("PEGASUS")
                        .font(.system(.subheadline, weight: .heavy))
                        .tracking(1.5)
                        .foregroundStyle(TacticalTheme.textPrimary)

                    Text("WH")
                        .font(.system(.caption2, design: .monospaced, weight: .bold))
                        .padding(.horizontal, 5)
                        .padding(.vertical, 2)
                        .foregroundStyle(TacticalTheme.accentCobalt)
                        .background(TacticalTheme.accentCobalt.opacity(0.15))
                        .clipShape(RoundedRectangle(cornerRadius: 4, style: .continuous))

                    Spacer()

                    Circle()
                        .fill(wasOffline ? TacticalTheme.statusDanger : TacticalTheme.statusSuccess)
                        .frame(width: 8, height: 8)
                        .overlay(
                            Circle()
                                .stroke((wasOffline ? TacticalTheme.statusDanger : TacticalTheme.statusSuccess).opacity(0.4), lineWidth: 2)
                        )
                        .accessibilityLabel(wasOffline ? "Network offline" : "Network online")
                }

                Text("DEPOT: UZ-TAS-01 (TASHKENT)")
                    .tacticalMicroLabel()
            }
            .padding(.horizontal, 16)
            .padding(.top, 16)
            .padding(.bottom, 12)

            // Search Filter
            HStack(spacing: 8) {
                Image(systemName: "magnifyingglass")
                    .font(.caption)
                    .foregroundStyle(isSidebarFilterFocused ? TacticalTheme.accentCobalt : TacticalTheme.textTertiary)
                TextField("Filter sections...", text: $searchFilter)
                    .font(.footnote)
                    .focused($isSidebarFilterFocused)
                    .submitLabel(.search)
                if !searchFilter.isEmpty {
                    Button {
                        PegasusHaptics.light()
                        searchFilter = ""
                        isSidebarFilterFocused = true
                    } label: {
                        Image(systemName: "xmark.circle.fill")
                            .font(.subheadline)
                            .foregroundStyle(TacticalTheme.textTertiary)
                            .frame(minWidth: 44, minHeight: 44)
                            .contentShape(Rectangle())
                    }
                    .buttonStyle(.plain)
                    .accessibilityLabel("Clear search filter")
                }
            }
            .padding(.horizontal, 10)
            .padding(.vertical, 7)
            .background(TacticalTheme.surfaceSunken)
            .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
            .overlay(
                RoundedRectangle(cornerRadius: 8, style: .continuous)
                    .stroke(isSidebarFilterFocused ? TacticalTheme.accentCobalt : TacticalTheme.border, lineWidth: isSidebarFilterFocused ? 1 : 0.5)
            )
            .padding(.horizontal, 14)
            .padding(.bottom, 10)

            Divider()
                .background(TacticalTheme.border)

            // Grouped Navigation List
            List(selection: $sidebarSelection) {
                ForEach(filteredSidebarGroups, id: \.title) { group in
                    Section(header: Text(group.title).tacticalMicroLabel()) {
                        ForEach(group.items, id: \.self) { section in
                            NavigationLink(value: section) {
                                HStack(spacing: 10) {
                                    Image(systemName: section.icon)
                                        .font(.system(.footnote, weight: .semibold))
                                        .frame(width: 20)
                                        .foregroundStyle(
                                            sidebarSelection == section
                                                ? TacticalTheme.accentCobalt
                                                : TacticalTheme.textSecondary
                                        )

                                    Text(section.rawValue)
                                        .font(.system(.footnote, weight: sidebarSelection == section ? .bold : .medium))
                                        .foregroundStyle(
                                            sidebarSelection == section
                                                ? TacticalTheme.textPrimary
                                                : TacticalTheme.textSecondary
                                        )

                                    Spacer()
                                }
                                .padding(.vertical, 2)
                                .frame(minHeight: 44)
                            }
                            .listRowBackground(
                                sidebarSelection == section
                                    ? TacticalTheme.surfaceSubtle
                                    : Color.clear
                            )
                        }
                    }
                }
            }
            .listStyle(.sidebar)
            .scrollDismissesKeyboard(.interactively)
            .scrollContentBackground(.hidden)
            .background(TacticalTheme.surface)
            .onChange(of: sidebarSelection) { _, _ in
                PegasusHaptics.selection()
            }

            // Operator Footer Capsule
            Divider()
                .background(TacticalTheme.border)

            HStack(spacing: 10) {
                Image(systemName: "person.crop.circle.fill")
                    .font(.title)
                    .foregroundStyle(TacticalTheme.accentCobalt)

                VStack(alignment: .leading, spacing: 2) {
                    Text("DEPOT SUPERVISOR")
                        .font(.system(.caption, weight: .bold))
                        .foregroundStyle(TacticalTheme.textPrimary)
                    Text("ON-DUTY SHIFT A")
                        .tacticalMicroLabel()
                }

                Spacer()

                Button {
                    PegasusHaptics.warning()
                    tokenStore.clear()
                } label: {
                    Image(systemName: "rectangle.portrait.and.arrow.right")
                        .font(.subheadline)
                        .foregroundStyle(TacticalTheme.textTertiary)
                        .frame(width: 44, height: 44)
                        .background(TacticalTheme.surfaceSunken)
                        .clipShape(Circle())
                }
                .buttonStyle(.plain)
                .accessibilityLabel("Sign out")
            }
            .padding(14)
            .background(TacticalTheme.surfaceRaised)
        }
        .background(TacticalTheme.surface)
        .navigationTitle("Command Rail")
        .navigationBarTitleDisplayMode(.inline)
    }

    // MARK: - Content Operations Feed
    private var contentColumnView: some View {
        Group {
            if let section = sidebarSelection {
                sectionView(section)
            } else {
                ContentUnavailableView("Select a Section", systemImage: "sidebar.left")
            }
        }
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    withAnimation(.spring(response: 0.35, dampingFraction: 0.8)) {
                        showInspector.toggle()
                    }
                } label: {
                    Image(systemName: showInspector ? "sidebar.right" : "sidebar.right")
                        .foregroundStyle(showInspector ? TacticalTheme.accentCobalt : TacticalTheme.textSecondary)
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                }
                .accessibilityLabel(showInspector ? "Hide Inspector" : "Show Inspector")
            }
        }
    }

    // MARK: - Detail Workbench
    private var detailWorkbenchView: some View {
        TacticalWorkbenchView(
            section: sidebarSelection ?? .dashboard,
            onInspect: {
                withAnimation(.spring(response: 0.35, dampingFraction: 0.8)) {
                    showInspector.toggle()
                }
            }
        )
    }

    // MARK: - iPhone Floor Suite Shell
    private var compactFloorSuiteShell: some View {
        TabView(selection: $compactTab) {
            NavigationStack {
                sectionView(.dashboard)
            }
            .tabItem {
                Label("Command", systemImage: "antenna.radiowaves.left.and.right")
            }
            .tag(WarehouseCompactTab.dashboard)

            NavigationStack {
                sectionView(.manifests)
            }
            .tabItem {
                Label("Inbound", systemImage: "doc.text")
            }
            .tag(WarehouseCompactTab.inbound)

            NavigationStack {
                sectionView(.inventory)
            }
            .tabItem {
                Label("Floor", systemImage: "archivebox")
            }
            .tag(WarehouseCompactTab.floor)

            NavigationStack {
                sectionView(.dispatch)
            }
            .tabItem {
                Label("Dispatch", systemImage: "paperplane")
            }
            .tag(WarehouseCompactTab.dispatch)

            NavigationStack {
                MoreHubView()
            }
            .tabItem {
                Label("Hub", systemImage: "ellipsis.circle")
            }
            .tag(WarehouseCompactTab.more)
        }
    }

    // MARK: - Filtered Groups for Sidebar
    private var sidebarGroups: [(title: String, items: [WarehouseSection])] {
        [
            ("Primary Command", WarehouseSection.primarySections),
            ("Fulfillment & Staging", WarehouseSection.fulfillmentSections),
            ("Inventory & Bins", WarehouseSection.inventorySections),
            ("Operations Control", WarehouseSection.operationsSections),
            ("Portal Administration", WarehouseSection.portalSections)
        ]
    }

    private var filteredSidebarGroups: [(title: String, items: [WarehouseSection])] {
        if searchFilter.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            return sidebarGroups
        }
        let query = searchFilter.lowercased()
        return sidebarGroups.compactMap { group in
            let filteredItems = group.items.filter { $0.rawValue.lowercased().contains(query) }
            return filteredItems.isEmpty ? nil : (title: group.title, items: filteredItems)
        }
    }

    // MARK: - Section View Resolver
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
            DispatchSettingsView()
        case .fleetLiveMap:
            FleetLiveMapView()
        case .transferActions:
            TransferActionsView()
        case .products:
            ProductsView()
        case .supplyRequests:
            SupplyRequestsHubView()
        case .preorders:
            PreordersView()
        case .stockCommitments:
            StockCommitmentsView()
        case .tomorrowBoard:
            TomorrowBoardView()
        case .replenishment:
            ReplenishmentView()
        case .demandForecast:
            DemandForecastView()
        case .retailers:
            CRMView()
        case .returns:
            ReturnsView()
        case .laborCapacity:
            LaborCapacityView()
        case .exceptions:
            ExceptionsView()
        case .cashier:
            CashierBlindCountView(onBack: nil)
        case .yardGate:
            YardGateView(onBack: nil)
        case .controlTower:
            WarehouseScoredExceptionsView()
        case .claims:
            ClaimsView()
        case .rescues:
            RescuesView()
        case .coverage:
            CoverageView()
        case .paymentConfig:
            PaymentConfigView()
        case .opsSettings:
            OpsSettingsView()
        case .returnPolicy:
            ReturnPolicySettingsView()
        case .notifications:
            NotificationInboxView()
        case .portalSetup, .portalProfile, .portalSearch:
            if let feature = section.portalFeature {
                PortalHandoffView(feature: feature)
            }
        }
    }
}
