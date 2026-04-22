//
//  ContentView.swift
//  reatilerapp
//
//  Created by Shakhzod on 3/17/26.
//

import SwiftUI

// MARK: - Tab Enums

enum AppTab: String, CaseIterable {
    case home, catalog, orders, profile, suppliers

    var title: String {
        switch self {
        case .home: "Home"
        case .catalog: "Catalog"
        case .orders: "Orders"
        case .profile: "Profile"
        case .suppliers: "Suppliers"
        }
    }

    var icon: String {
        switch self {
        case .home: "house"
        case .catalog: "square.grid.2x2"
        case .orders: "shippingbox"
        case .profile: "person.circle"
        case .suppliers: "building.2"
        }
    }
}

enum SideMenuTab: String, Hashable, CaseIterable {
    case home, catalog, orders, suppliers
    case insights, futureDemand, autoOrder, profile

    var title: String {
        switch self {
        case .home: "Home"
        case .catalog: "Catalog"
        case .orders: "Orders"
        case .suppliers: "Suppliers"
        case .insights: "Insights"
        case .futureDemand: "Future Demand"
        case .autoOrder: "Auto Order"
        case .profile: "Profile"
        }
    }

    var icon: String {
        switch self {
        case .home: "house"
        case .catalog: "square.grid.2x2"
        case .orders: "shippingbox"
        case .suppliers: "building.2"
        case .insights: "chart.bar.xaxis"
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
    @State private var showNotificationInbox = false
    @State private var notificationCount = 3
    @State private var cartBounce = false
    @State private var activeOrders: [Order] = []
    @State private var paymentEvent: PaymentRequiredEvent?

    @Environment(\.horizontalSizeClass) private var horizontalSizeClass

    private let api = APIClient.shared
    private let ws = RetailerWebSocket.shared

    /// Show floating bar only on main pages
    private var showFloatingBar: Bool {
        if horizontalSizeClass == .regular {
            return sideMenuSelection == .home || sideMenuSelection == .orders || sideMenuSelection == .suppliers
        } else {
            return selectedTab == .home || selectedTab == .orders || selectedTab == .suppliers
        }
    }

    var body: some View {
        Group {
            if horizontalSizeClass == .regular {
                ipadLayout
            } else {
                iphoneLayout
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
                    .navigationTitle("Active Deliveries")
                    .navigationBarTitleDisplayMode(.inline)
                    .toolbar {
                        ToolbarItem(placement: .topBarLeading) {
                            NavigationLink {
                                DeliveryMapView()
                            } label: {
                                Image(systemName: "map")
                                    .font(.system(.subheadline, weight: .semibold))
                            }
                        }
                        ToolbarItem(placement: .confirmationAction) {
                            Button("Done") { showActiveOrderDetail = false }
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
                            Button("Done") { showProfile = false }
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
                            Button("Done") { showCart = false }
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
        .sheet(isPresented: $showNotificationInbox) {
            NotificationInboxView()
                .presentationDetents([.large])
                .presentationDragIndicator(.visible)
                .presentationCompactAdaptation(.sheet)
        }
        .onChange(of: cart.totalItems) {
            withAnimation(AnimationConstants.bouncy) { cartBounce = true }
        }
        .task { await loadActiveOrders() }
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
                                Text("The Lab")
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
                                Text("Online")
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

                        ForEach([SideMenuTab.home, .catalog, .orders, .suppliers], id: \.self) { tab in
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

                        ForEach([SideMenuTab.insights, .futureDemand, .autoOrder, .profile], id: \.self) { tab in
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
        case .suppliers: tabContent(.suppliers)
        case .profile: tabContent(.profile)
        case .insights:
            NavigationStack {
                InsightsView()
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
            TabView(selection: $selectedTab) {
                ForEach(AppTab.allCases, id: \.self) { tab in
                    Tab(tab.title, systemImage: tab.icon, value: tab) {
                        tabContent(tab)
                    }
                }
            }
            .sensoryFeedback(.selection, trigger: selectedTab)
            .tint(AppTheme.accent)

            // Floating Active Orders Bar
            if showFloatingBar {
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
            if showSidebar {
                SidebarMenu(isOpen: $showSidebar) { destination in
                    handleSidebarNavigation(destination)
                }
                .zIndex(100)
            }
        }
    }

    // MARK: - Tab Content

    @ViewBuilder
    private func tabContent(_ tab: AppTab) -> some View {
        NavigationStack {
            Group {
                switch tab {
                case .home: DashboardView()
                case .catalog: CatalogView()
                case .orders: OrdersView()
                case .profile: ProfileView()
                case .suppliers: MySuppliersView()
                }
            }
            .toolbar { standardToolbar }
            .toolbarBackground(.ultraThinMaterial, for: .navigationBar)
            .toolbarBackground(.visible, for: .navigationBar)
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
                Text("The Lab")
                    .font(.system(.headline, design: .rounded)).fontWeight(.bold)
                    .foregroundStyle(AppTheme.textPrimary)
            }
        }

        ToolbarItemGroup(placement: .topBarTrailing) {
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

    // MARK: - Sidebar Navigation

    private func handleSidebarNavigation(_ destination: SidebarDestination) {
        switch destination {
        case .dashboard: selectedTab = .home
        case .procurement: selectedTab = .home
        case .autoOrder: showAutoOrder = true
        case .futureDemand: showFutureDemand = true
        case .inbox: showNotificationInbox = true
        case .profile: selectedTab = .profile
        case .insights: showInsights = true
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

    // MARK: - WebSocket

    private func connectWebSocket() async {
        let rid = AuthManager.shared.currentUser?.id ?? ""
        guard !rid.isEmpty else { return }
        ws.connect(retailerId: rid)
        for await event in ws.events {
            switch event {
            case .paymentRequired(let payload):
                paymentEvent = payload
            case .driverApproaching:
                await loadActiveOrders()
            case .orderCompleted:
                await loadActiveOrders()
            case .paymentSettled:
                await loadActiveOrders()
            case .paymentFailed, .paymentExpired:
                await loadActiveOrders()
            case .orderStatusChanged:
                await loadActiveOrders()
            }
        }
    }
}

#Preview {
    ContentView()
        .environment(CartManager())
        .environment(AuthManager.shared)
}
