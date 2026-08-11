//
//  ContentView.swift
//  retailerapp
//
//  Created by Shakhzod on 3/17/26.
//

import SwiftUI

// MARK: - Tab Enums

enum AppTab: String, CaseIterable {
    case home, catalog, orders, deliveries, profile, suppliers

    var title: String {
        switch self {
        case .home: "Home"
        case .catalog: "Catalog"
        case .orders: "Orders"
        case .deliveries: "Deliveries"
        case .profile: "Profile"
        case .suppliers: "Suppliers"
        }
    }

    var icon: String {
        switch self {
        case .home: "house"
        case .catalog: "square.grid.2x2"
        case .orders: "shippingbox"
        case .deliveries: "map"
        case .profile: "person.circle"
        case .suppliers: "building.2"
        }
    }
}

enum SideMenuTab: String, Hashable, CaseIterable {
    case home, catalog, orders, deliveries, suppliers
    case insights, procurement, futureDemand, autoOrder, profile

    var title: String {
        switch self {
        case .home: "Home"
        case .catalog: "Catalog"
        case .orders: "Orders"
        case .deliveries: "Deliveries"
        case .suppliers: "Suppliers"
        case .insights: "Insights"
        case .procurement: "Procurement"
        case .futureDemand: "Reorder suggestions"
        case .autoOrder: "Auto Order"
        case .profile: "Profile"
        }
    }

    var icon: String {
        switch self {
        case .home: "house"
        case .catalog: "square.grid.2x2"
        case .orders: "shippingbox"
        case .deliveries: "map"
        case .suppliers: "building.2"
        case .insights: "chart.bar.xaxis"
        case .procurement: "chart.bar"
        case .futureDemand: "waveform.path.ecg"
        case .autoOrder: "arrow.2.squarepath"
        case .profile: "person.crop.circle"
        }
    }
}

// MARK: - Content View

struct ContentView: View {
    @Environment(CartManager.self) private var cart
    @Environment(AuthManager.self) private var auth
    @Environment(\.scenePhase) private var scenePhase

    @State private var selectedTab: AppTab = .home
    @State private var sideMenuSelection: SideMenuTab = .home
    @State private var columnVisibility: NavigationSplitViewVisibility = .automatic
    @State private var isSidebarExpanded: Bool = true
    @Namespace private var namespace

    @State private var showSidebar = false
    @State private var showFutureDemand = false
    @State private var showAutoOrder = false
    @State private var showActiveOrderDetail = false
    @State private var showProfile = false
    @State private var showCart = false
    @State private var showInsights = false
    @State private var showProcurement = false
    @State private var showControlTower = false
    @State private var showNotificationInbox = false
    @State private var notificationCount = 0
    @State private var cartBounce = false
    @State private var activeOrders: [Order] = []
    @State private var paymentEvent: PaymentRequiredEvent?
    @State private var shopClosedAlert: ShopClosedAlertEvent?
    @State private var refreshCenter = RetailerRefreshCenter.shared
    @State private var clientPolicyMessage: String?
    @State private var clientPolicyForce = false
    @State private var pendingManifest: AutoUpdater.Manifest?
    @State private var rescueWarningMessage: String?
    @State private var deliveriesHubInitialTab: DeliveriesHubTab = .map

    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @Environment(\.modelContext) private var modelContext

    private let api = APIClient.shared
    private let ws = RetailerWebSocket.shared

    /// Show floating bar only on main pages
    private var showFloatingBar: Bool {
        if horizontalSizeClass == .regular {
            return sideMenuSelection == .home || sideMenuSelection == .orders || sideMenuSelection == .suppliers || sideMenuSelection == .deliveries
        } else {
            return selectedTab == .home || selectedTab == .orders || selectedTab == .suppliers || selectedTab == .deliveries
        }
    }

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
            RescueWarningBanner(message: rescueWarningMessage) {
                withAnimation(AnimationConstants.fluid) { rescueWarningMessage = nil }
            }
            Group {
                if horizontalSizeClass == .regular {
                    ipadLayout
                } else {
                    iphoneLayout
                }
            }
        }
        .sheet(isPresented: $showFutureDemand) {
            FutureDemandView()
        }
        .sheet(isPresented: $showAutoOrder) {
            AutoOrderView()
        }
        .sheet(isPresented: $showActiveOrderDetail) {
            NavigationStack {
                ActiveDeliveriesView()
                    .navigationTitle("mobile_retailer.ui.active_deliveries")
                    .navigationBarTitleDisplayMode(.inline)
                    .toolbar {
                        ToolbarItem(placement: .topBarLeading) {
                            NavigationLink {
                                DeliveriesHubView()
                            } label: {
                                Image(systemName: "map")
                                    .font(.system(.subheadline, weight: .semibold))
                            }
                        }
                        ToolbarItem(placement: .confirmationAction) {
                            Button("warehouse_portal.kpi_stat_card.text.done") { showActiveOrderDetail = false }
                                .font(.system(.subheadline, design: .rounded)).fontWeight(.semibold)
                        }
                    }
            }
            .presentationDetents([.large])
            .presentationCompactAdaptation(.sheet)
        }
        .sheet(isPresented: $showProfile) {
            NavigationStack {
                ProfileView()
                    .toolbar {
                        ToolbarItem(placement: .confirmationAction) {
                            Button("warehouse_portal.kpi_stat_card.text.done") { showProfile = false }
                                .font(.system(.subheadline, design: .rounded)).fontWeight(.semibold)
                        }
                    }
            }
            .presentationDetents([.large])
            .presentationCompactAdaptation(.sheet)
        }
        .sheet(isPresented: $showCart) {
            NavigationStack {
                CartView()
                    .toolbar {
                        ToolbarItem(placement: .confirmationAction) {
                            Button("warehouse_portal.kpi_stat_card.text.done") { showCart = false }
                                .font(.system(.subheadline, design: .rounded)).fontWeight(.semibold)
                        }
                    }
            }
            .presentationDetents([.large])
            .presentationDragIndicator(.visible)
            .presentationCompactAdaptation(.sheet)
        }
        .sheet(isPresented: $showInsights) {
            NavigationStack {
                InsightsView()
            }
            .presentationDetents([.large])
            .presentationDragIndicator(.visible)
            .presentationCompactAdaptation(.sheet)
        }
        .sheet(isPresented: $showProcurement) {
            NavigationStack {
                ProcurementView()
                    .toolbar {
                        ToolbarItem(placement: .confirmationAction) {
                            Button("warehouse_portal.kpi_stat_card.text.done") { showProcurement = false }
                                .font(.system(.subheadline, design: .rounded)).fontWeight(.semibold)
                        }
                    }
            }
            .presentationDetents([.large])
            .presentationDragIndicator(.visible)
            .presentationCompactAdaptation(.sheet)
        }
        .sheet(isPresented: $showControlTower) {
            NavigationStack {
                ControlTowerView()
                    .toolbar {
                        ToolbarItem(placement: .confirmationAction) {
                            Button("warehouse_portal.kpi_stat_card.text.done") { showControlTower = false }
                                .font(.system(.subheadline, design: .rounded)).fontWeight(.semibold)
                        }
                    }
            }
            .presentationDetents([.large])
            .presentationDragIndicator(.visible)
            .presentationCompactAdaptation(.sheet)
        }
        .sheet(isPresented: $showNotificationInbox) {
            NotificationInboxView()
                .presentationDetents([.large])
                .presentationDragIndicator(.visible)
                .presentationCompactAdaptation(.sheet)
        }
        .onChange(of: showNotificationInbox) { _, isOpen in
            if !isOpen {
                Task { await loadNotificationCount() }
            }
        }
        .onChange(of: cart.totalItems) {
            withAnimation(AnimationConstants.bouncy) { cartBounce = true }
        }
        .onChange(of: scenePhase) {
            guard scenePhase == .active else { return }
            Task {
                await loadActiveOrders()
                await loadPendingPayments()
                await loadNotificationCount()
                refreshCenter.trigger()
            }
        }
        .task {
            await loadActiveOrders()
            await loadPendingPayments()
            await loadNotificationCount()
            await loadClientPolicy()
        }
        .task(id: ws.reconnectEpoch) {
            await loadClientPolicy()
        }
        .task { await connectWebSocket() }
        .sheet(item: $paymentEvent) { event in
            DeliveryPaymentSheetView(event: event) {
                paymentEvent = nil
                Task { await loadActiveOrders() }
            }
            .presentationDetents([.large])
            .interactiveDismissDisabled()
            .presentationCompactAdaptation(.sheet)
        }
        .sheet(item: $shopClosedAlert) { alert in
            NavigationStack {
                ShopClosedBanner(event: alert) { _ in
                    shopClosedAlert = nil
                    Task { await loadActiveOrders() }
                }
                .navigationTitle("mobile_retailer.ui.shop_status")
                .navigationBarTitleDisplayMode(.inline)
            }
            .presentationDetents([.medium, .large])
            .interactiveDismissDisabled()
            .presentationCompactAdaptation(.sheet)
        }
        .animation(AnimationConstants.fluid, value: selectedTab)
    }

    // MARK: - Layouts

    @ViewBuilder
    private var ipadLayout: some View {
        HStack(spacing: 0) {
            // MARK: Collapsible Navigation Rail
            VStack(alignment: .leading, spacing: 0) {
                // Header (App Icon + Toggle)
                VStack(alignment: .leading, spacing: 24) {
                    // Top row: Brand & Menu
                    HStack {
                        if isSidebarExpanded {
                            HStack(spacing: 8) {
                                Image(systemName: "leaf.fill")
                                    .font(.system(size: 20, weight: .bold))
                                    .foregroundColor(AppTheme.accent)
                                Text("auth.login.title")
                                    .font(.system(.title3, design: .rounded)).fontWeight(.heavy)
                                    .foregroundStyle(AppTheme.textPrimary)
                            }
                            Spacer()
                        }

                        Button {
                            withAnimation(.spring(response: 0.4, dampingFraction: 0.7)) {
                                isSidebarExpanded.toggle()
                            }
                        } label: {
                            Image(systemName: "sidebar.left")
                                .font(.system(size: 22, weight: .medium))
                                .foregroundStyle(AppTheme.textSecondary)
                                .frame(width: 44, height: 44)
                                .contentShape(Rectangle())
                        }
                    }
                    .padding(.horizontal, isSidebarExpanded ? 24 : 22)

                    // Profile Row
                    HStack(spacing: 12) {
                        ZStack {
                            Circle()
                                .fill(AppTheme.accentGradient)
                                .frame(width: 44, height: 44)
                            Text(String((auth.currentUser?.name ?? "U").prefix(1)))
                                .font(.system(.title3, design: .rounded)).fontWeight(.bold)
                                .foregroundStyle(.white)
                        }

                        if isSidebarExpanded {
                            VStack(alignment: .leading, spacing: 2) {
                                Text(auth.currentUser?.name ?? "Retailer")
                                    .font(.system(.subheadline, design: .rounded)).fontWeight(.bold)
                                    .foregroundStyle(AppTheme.textPrimary)
                                    .lineLimit(1)
                                Text("mobile_retailer.ui.online")
                                    .font(.system(.caption2, design: .rounded)).fontWeight(.medium)
                                    .foregroundStyle(AppTheme.success)
                                    .lineLimit(1)
                            }
                            .transition(.move(edge: .leading).combined(with: .opacity))
                            Spacer(minLength: 0)
                        }
                    }
                    .padding(.horizontal, isSidebarExpanded ? 24 : 22)
                }
                .padding(.top, 24)
                .padding(.bottom, 24)

                // Navigation Items
                ScrollView(showsIndicators: false) {
                    VStack(alignment: .leading, spacing: 8) {
                        // Main Section
                        if isSidebarExpanded {
                            Text("MAIN")
                                .font(.system(size: 11, weight: .bold, design: .rounded))
                                .foregroundStyle(AppTheme.textTertiary)
                                .padding(.leading, 36)
                                .padding(.bottom, 8)
                                .transition(.opacity)
                        }

                        ForEach([SideMenuTab.home, .catalog, .orders, .deliveries, .suppliers], id: \.self) { tab in
                            sidebarItem(for: tab)
                        }

                        Rectangle()
                            .fill(AppTheme.separator.opacity(0.3))
                            .frame(height: 1)
                            .padding(.vertical, 20)
                            .padding(.horizontal, 24)

                        // Tools Section
                        if isSidebarExpanded {
                            Text("TOOLS")
                                .font(.system(size: 11, weight: .bold, design: .rounded))
                                .foregroundStyle(AppTheme.textTertiary)
                                .padding(.leading, 36)
                                .padding(.bottom, 8)
                                .transition(.opacity)
                        }

                        ForEach([SideMenuTab.insights, .procurement, .futureDemand, .autoOrder, .profile], id: \.self) { tab in
                            sidebarItem(for: tab)
                        }
                    }
                    .padding(.vertical, 8)
                }
                .padding(.bottom, 24)
            }
            .frame(width: isSidebarExpanded ? 280 : 88)
            .background(AppTheme.cardBackground.ignoresSafeArea())
            .clipShape(
                RoundedRectangle(cornerRadius: AppTheme.radiusLG)
            )
            .shadow(color: AppTheme.shadowColor.opacity(0.08), radius: 12, x: 4, y: 0)
            .zIndex(10)

            // MARK: Detail Content Area
            ZStack(alignment: .bottom) {
                ipadDetailContent
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                
                if showFloatingBar {
                    VStack {
                        Spacer()
                        FloatingActiveOrdersBar(activeOrders: activeOrders) {
                            showActiveOrderDetail = true
                        }
                        .padding(.horizontal, AppTheme.spacingMD)
                    }
                    .padding(.bottom, 32)
                    .transition(.move(edge: .bottom).combined(with: .opacity))
                    .animation(AnimationConstants.fluid, value: activeOrders.count)
                }
            }
            .background(AppTheme.background.ignoresSafeArea())
            .zIndex(1)
        }
        .animation(.spring(response: 0.4, dampingFraction: 0.75), value: isSidebarExpanded)
    }

    // MARK: - Sidebar Item Component

    @ViewBuilder
    private func sidebarItem(for tab: SideMenuTab) -> some View {
        let isSelected = sideMenuSelection == tab
        
        Button {
            withAnimation(.spring(response: 0.3, dampingFraction: 0.7)) {
                if tab == .deliveries {
                    deliveriesHubInitialTab = .map
                }
                sideMenuSelection = tab
            }
        } label: {
            HStack(spacing: 16) {
                ZStack {
                    if isSelected {
                        RoundedRectangle(cornerRadius: AppTheme.radiusSM)
                            .fill(AppTheme.accentSoft.opacity(0.6))
                            .frame(width: 44, height: 44)
                            .matchedGeometryEffect(id: "sidebar_active_bg", in: namespace)
                    } else {
                        RoundedRectangle(cornerRadius: AppTheme.radiusSM)
                            .fill(Color.clear)
                            .frame(width: 44, height: 44)
                    }

                    Image(systemName: tab.icon)
                        .font(.system(size: 24, weight: isSelected ? .bold : .medium))
                        .foregroundStyle(isSelected ? AppTheme.accent : AppTheme.textSecondary)
                }
                
                if isSidebarExpanded {
                    Text(tab.title)
                        .font(.system(.body, design: .rounded, weight: isSelected ? .bold : .medium))
                        .foregroundStyle(isSelected ? AppTheme.accent : AppTheme.textPrimary)
                        .transition(.move(edge: .trailing).combined(with: .opacity))
                    
                    Spacer(minLength: 0)
                }
            }
            .padding(.vertical, 8)
            .padding(.horizontal, isSidebarExpanded ? 16 : 22)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(Color.white.opacity(0.001))
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
    }

    @ViewBuilder
    private var ipadDetailContent: some View {
        switch sideMenuSelection {
        case .home: tabContent(.home)
        case .catalog: tabContent(.catalog)
        case .orders: tabContent(.orders)
        case .deliveries:
            NavigationStack {
                DeliveriesHubView(initialTab: deliveriesHubInitialTab)
                    .toolbar { standardToolbar }
            }
        case .suppliers: tabContent(.suppliers)
        case .profile: tabContent(.profile)
        case .insights:
            NavigationStack {
                InsightsView()
                    .toolbar { standardToolbar }
            }
        case .procurement:
            NavigationStack {
                ProcurementView()
                    .toolbar { standardToolbar }
            }
        case .futureDemand:
            NavigationStack {
                FutureDemandView()
                    .toolbar { standardToolbar }
            }
        case .autoOrder:
            NavigationStack {
                AutoOrderView()
                    .toolbar { standardToolbar }
            }
        }
    }

    @ViewBuilder
    private var iphoneLayout: some View {
        ZStack(alignment: .bottom) {
            TabView(selection: tabSelection) {
                ForEach(AppTab.allCases, id: \.self) { tab in
                    Tab(tab.title, systemImage: tab.icon, value: tab) {
                        tabContent(tab)
                    }
                }
            }
            .sensoryFeedback(.selection, trigger: selectedTab)
            .tint(AppTheme.accent)

            // Floating Active Orders Bar
            if showFloatingBar && !showSidebar {
                VStack {
                    Spacer()
                    FloatingActiveOrdersBar(activeOrders: activeOrders) {
                        showActiveOrderDetail = true
                    }
                    .padding(.horizontal, AppTheme.spacingMD)
                }
                .padding(.bottom, 52)
                .transition(.move(edge: .bottom).combined(with: .opacity))
                .animation(AnimationConstants.fluid, value: activeOrders.count)
            }

            // Sidebar Overlay
            SidebarMenu(isOpen: $showSidebar) { destination in
                handleSidebarNavigation(destination)
            }
            .zIndex(100)
        }
    }

    // MARK: - Tab Content

    @ViewBuilder
    private func tabContent(_ tab: AppTab) -> some View {
        NavigationStack {
            Group {
                switch tab {
                case .home: DashboardView()
                case .catalog:
                    CatalogView(onNavigateToSuppliers: navigateToSuppliersTab)
                case .orders: OrdersView()
                case .deliveries: DeliveriesHubView(initialTab: deliveriesHubInitialTab)
                case .profile: ProfileView()
                case .suppliers: MySuppliersView()
                }
            }
            .toolbar { standardToolbar }
            .toolbarBackground(.ultraThinMaterial, for: .navigationBar)
            .toolbarBackground(.visible, for: .navigationBar)
            .toolbar(showSidebar ? .hidden : .visible, for: .tabBar)
        }
        .sensoryFeedback(.impact(weight: .light), trigger: showSidebar)
        .sensoryFeedback(.impact(weight: .light), trigger: showInsights)
        .sensoryFeedback(.impact(weight: .light), trigger: showActiveOrderDetail)
    }

    // MARK: - Toolbar

    @ToolbarContentBuilder
    private var standardToolbar: some ToolbarContent {
        if horizontalSizeClass != .regular {
            ToolbarItem(placement: .topBarLeading) {
                Button {
                    withAnimation(AnimationConstants.fluid) { showSidebar.toggle() }
                } label: {
                    ZStack {
                        Circle()
                            .fill(AppTheme.accentGradient)
                            .frame(width: 32, height: 32)
                        Text(String((auth.currentUser?.name ?? "U").prefix(1)))
                            .font(.system(.caption, design: .rounded)).fontWeight(.bold)
                            .foregroundStyle(.white)
                    }
                }
                .accessibilityLabel("Menu")
            }
        }

        ToolbarItem(placement: .principal) {
            HStack(spacing: 6) {
                Image(systemName: "leaf.fill")
                    .font(.system(size: 13))
                    .foregroundStyle(AppTheme.accent)
                Text("auth.login.title")
                    .font(.system(.headline, design: .rounded)).fontWeight(.bold)
                    .foregroundStyle(AppTheme.textPrimary)
            }
        }

        ToolbarItemGroup(placement: .topBarTrailing) {
            Button {
                RetailerRefreshCenter.shared.trigger()
            } label: {
                Image(systemName: "arrow.clockwise")
                    .font(.system(size: 16, weight: .medium))
                    .foregroundStyle(AppTheme.textPrimary)
            }
            .accessibilityLabel("Refresh")

            Button {
                showCart = true
            } label: {
                ZStack(alignment: .topTrailing) {
                    Image(systemName: "cart")
                        .font(.system(size: 16, weight: .medium))
                        .foregroundStyle(AppTheme.textPrimary)
                    if cart.totalItems > 0 {
                        Text("\(cart.totalItems)")
                            .font(.system(size: 9, weight: .black, design: .rounded))
                            .foregroundStyle(.white)
                            .frame(width: 16, height: 16)
                            .background(AppTheme.accent)
                            .clipShape(.circle)
                            .offset(x: 8, y: -6)
                    }
                }
            }
            .accessibilityLabel("Cart, \(cart.totalItems) items")

            Button {
                showNotificationInbox = true
            } label: {
                ZStack(alignment: .topTrailing) {
                    Image(systemName: "bell")
                        .font(.system(size: 16, weight: .medium))
                        .foregroundStyle(AppTheme.textPrimary)
                    if notificationCount > 0 {
                        Text("\(notificationCount)")
                            .font(.system(size: 9, weight: .black, design: .rounded))
                            .foregroundStyle(.white)
                            .frame(width: 16, height: 16)
                            .background(AppTheme.destructive)
                            .clipShape(.circle)
                            .offset(x: 8, y: -6)
                    }
                }
            }
            .accessibilityLabel("Notifications, \(notificationCount) new")
        }
    }

    // MARK: - Tab Navigation

    private func navigateToSuppliersTab() {
        if horizontalSizeClass == .regular {
            withAnimation(.spring(response: 0.3, dampingFraction: 0.7)) {
                sideMenuSelection = .suppliers
            }
        } else {
            selectedTab = .suppliers
        }
    }

    // MARK: - Sidebar Navigation

    private var tabSelection: Binding<AppTab> {
        Binding(
            get: { selectedTab },
            set: { newTab in
                if newTab == .deliveries && selectedTab != .deliveries {
                    deliveriesHubInitialTab = .map
                }
                selectedTab = newTab
            }
        )
    }

    private func handleSidebarNavigation(_ destination: SidebarDestination) {
        switch destination {
        case .dashboard: selectedTab = .home
        case .procurement:
            if horizontalSizeClass == .regular {
                withAnimation(.spring(response: 0.3, dampingFraction: 0.7)) {
                    sideMenuSelection = .procurement
                }
            } else {
                showProcurement = true
            }
        case .autoOrder: showAutoOrder = true
        case .futureDemand: showFutureDemand = true
        case .dock:
            deliveriesHubInitialTab = .dock
            if horizontalSizeClass == .regular {
                withAnimation(.spring(response: 0.3, dampingFraction: 0.7)) {
                    sideMenuSelection = .deliveries
                }
            } else {
                selectedTab = .deliveries
            }
        case .inbox: showNotificationInbox = true
        case .profile: selectedTab = .profile
        case .insights: showInsights = true
        case .controlTower: showControlTower = true
        case .settings: selectedTab = .profile
        case .logout: auth.logout()
        }
    }

    // MARK: - API

    private func loadActiveOrders() async {
        let rid = AuthManager.shared.currentUser?.id ?? ""
        do {
            let result: [Order] = try await api.get(path: "/v1/retailers/\(rid)/orders")
            activeOrders = result.filter { $0.status.isActive }
        } catch {
            activeOrders = []
        }
    }

    private func loadNotificationCount() async {
        do {
            let resp: NotificationsResponse = try await api.get(
                path: "/v1/user/notifications?limit=1&offset=0"
            )
            notificationCount = resp.unreadCount
        } catch {
            // Keep last known badge; inbox sheet refreshes on open.
        }
    }

    private func loadPendingPayments(reconcile: Bool = false) async {
        if !reconcile {
            guard paymentEvent == nil else { return }
        }
        do {
            let response = try await api.getPendingPayments()
            guard let session = response.pendingPayments.first else {
                if reconcile {
                    paymentEvent = nil
                }
                return
            }
            paymentEvent = PaymentRequiredEvent(
                type: "PAYMENT_REQUIRED",
                orderId: session.orderId,
                invoiceId: session.invoiceId ?? "",
                sessionId: session.sessionId,
                amountUzs: session.lockedAmount,
                originalAmountUzs: session.lockedAmount,
                availableCardGateways: session.gateway == "CASH" ? [] : [session.gateway],
                message: "Pending payment requires completion.",
                paymentMethod: session.gateway == "CASH" ? "CASH" : "CARD"
            )
        } catch {
            // WebSocket delivery remains the primary realtime path.
        }
    }

    private func loadClientPolicy() async {
        let version = Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "1.0.0"
        do {
            struct ClientPolicy: Decodable {
                let outdated: Bool?
                let forceUpdate: Bool?
                let updateDeferred: Bool?
                let minimumVersion: String?
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
            let role = EnterpriseUpdateConfig.policyRole
            let channel = EnterpriseUpdateConfig.channel
            let policy: ClientPolicy = try await api.get(
                path: "/v1/platform/client-policy?role=\(role)&platform=ios&version=\(version)&channel=\(channel)"
            )
            let state = await AutoUpdater.shared.evaluate(
                outdated: policy.outdated == true,
                forceUpdate: policy.forceUpdate == true,
                updateDeferred: policy.updateDeferred == true,
                minimumVersion: policy.minimumVersion,
                recommendedVersion: policy.recommendedVersion,
                deferReason: policy.deferReason,
                updateURL: policy.updateURL,
            )
            clientPolicyMessage = state.message
            clientPolicyForce = state.force
            pendingManifest = state.manifest
            if state.force, state.available {
                await MainActor.run {
                    AutoUpdater.shared.promptInstall(manifest: state.manifest, force: true)
                }
            }
        } catch {
            clientPolicyMessage = nil
            clientPolicyForce = false
            pendingManifest = nil
        }
    }

    // MARK: - WebSocket

    private func connectWebSocket() async {
        let rid = AuthManager.shared.currentUser?.id ?? ""
        guard !rid.isEmpty else { return }
        ws.connect(retailerId: rid)
        for await event in ws.eventStream() {
            switch event {
            case .paymentRequired(let payload):
                paymentEvent = payload
            case .driverApproaching:
                await loadActiveOrders()
            case .orderCompleted, .fiscalSucceeded:
                await loadActiveOrders()
            case .fiscalizing:
                await loadActiveOrders()
            case .paymentSettled:
                await loadActiveOrders()
            case .paymentFailed, .paymentExpired:
                await loadActiveOrders()
            case .orderStatusChanged:
                await loadActiveOrders()
            case .orderReassigned(let orderId, let licensePlate):
                withAnimation(AnimationConstants.fluid) {
                    rescueWarningMessage = "Your order #\(orderId.suffix(4)) has been reassigned to rescue truck (\(licensePlate)) due to a breakdown."
                }
                await loadActiveOrders()
            case .preOrderAutoAccepted, .preOrderConfirmed, .preOrderEdited, .preOrderNudge, .preOrderConfirmationPush,
                 .preOrderDateProposed, .preOrderDateAccepted, .preOrderDateRejected, .preOrderCancelled:
                await loadActiveOrders()
            case .shopClosedAlert(let alert):
                shopClosedAlert = alert
                await loadActiveOrders()
            case .cartSyncUpdated:
                break
            case .promotionChanged:
                break
            case .transportReconnected:
                await RetailerSessionReconcile.run(api: api)
                await loadActiveOrders()
                await loadPendingPayments(reconcile: true)
                await loadClientPolicy()
                await PendingOrderReplayer.flush(modelContext: modelContext, api: api)
            }
            refreshCenter.trigger()
        }
    }
}

#Preview {
    ContentView()
        .environment(CartManager())
        .environment(AuthManager.shared)
}
