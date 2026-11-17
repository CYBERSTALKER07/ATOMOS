//
//  ContentView.swift
//  retailerapp
//
//  Created by Shakhzod on 3/17/26.
//

import SwiftData
import SwiftUI

// MARK: - Tab Enums

enum AppTab: String, CaseIterable {
    case home, catalog, deliveries, store, more

    var title: String {
        switch self {
        case .home: "Home"
        case .catalog: "Buy"
        case .deliveries: "Incoming"
        case .store: "Store"
        case .more: "More"
        }
    }

    var icon: String {
        switch self {
        case .home: "house"
        case .catalog: "square.grid.2x2"
        case .deliveries: "map"
        case .store: "storefront"
        case .more: "ellipsis.circle"
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
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @Environment(\.modelContext) private var modelContext

    @State private var vm = AppShellViewModel()
    @State private var refreshCenter = RetailerRefreshCenter.shared
    @Namespace private var namespace

    private let ws = RetailerWebSocket.shared

    var body: some View {
        @Bindable var vm = vm
        VStack(spacing: 0) {
            ClientPolicyBanner(
                message: vm.clientPolicyMessage,
                force: vm.clientPolicyForce,
                onUpdate: vm.clientPolicyMessage == nil ? nil : {
                    AutoUpdater.shared.promptInstall(manifest: vm.pendingManifest, force: vm.clientPolicyForce)
                },
                onDismiss: vm.clientPolicyForce ? nil : { vm.clientPolicyMessage = nil }
            )
            RescueWarningBanner(message: vm.rescueWarningMessage) {
                withAnimation(AnimationConstants.fluid) { vm.rescueWarningMessage = nil }
            }
            Group {
                if horizontalSizeClass == .regular {
                    ipadLayout
                } else {
                    iphoneLayout
                }
            }
        }
        .sheet(isPresented: $vm.showFutureDemand) {
            NavigationStack {
                FutureDemandView()
            }
        }
        .sheet(isPresented: $vm.showAutoOrder) {
            NavigationStack {
                AutoOrderView()
            }
        }
        .sheet(isPresented: $vm.showActiveOrderDetail) {
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
                                    .pegasusMinimumHitTarget()
                            }
                            .accessibilityLabel("Delivery Map")
                        }
                        ToolbarItem(placement: .confirmationAction) {
                            Button("Done") { vm.showActiveOrderDetail = false }
                                .font(.system(.subheadline, design: .rounded)).fontWeight(.semibold)
                        }
                    }
            }
            .presentationDetents([.large])
            .presentationCompactAdaptation(.sheet)
        }
        .sheet(isPresented: $vm.showProfile) {
            NavigationStack {
                ProfileView()
                    .toolbar {
                        ToolbarItem(placement: .confirmationAction) {
                            Button("Done") { vm.showProfile = false }
                                .font(.system(.subheadline, design: .rounded)).fontWeight(.semibold)
                        }
                    }
            }
            .presentationDetents([.large])
            .presentationCompactAdaptation(.sheet)
        }
        .sheet(isPresented: $vm.showCart) {
            NavigationStack {
                CartView()
                    .toolbar {
                        ToolbarItem(placement: .confirmationAction) {
                            Button("Done") { vm.showCart = false }
                                .font(.system(.subheadline, design: .rounded)).fontWeight(.semibold)
                        }
                    }
            }
            .presentationDetents([.large])
            .presentationDragIndicator(.visible)
            .presentationCompactAdaptation(.sheet)
        }
        .sheet(isPresented: $vm.showInsights) {
            NavigationStack {
                InsightsView()
            }
            .presentationDetents([.large])
            .presentationDragIndicator(.visible)
            .presentationCompactAdaptation(.sheet)
        }
        .sheet(isPresented: $vm.showProcurement) {
            NavigationStack {
                ProcurementView()
                    .toolbar {
                        ToolbarItem(placement: .confirmationAction) {
                            Button("Done") { vm.showProcurement = false }
                                .font(.system(.subheadline, design: .rounded)).fontWeight(.semibold)
                        }
                    }
            }
            .presentationDetents([.large])
            .presentationDragIndicator(.visible)
            .presentationCompactAdaptation(.sheet)
        }
        .sheet(isPresented: $vm.showCredit) {
            NavigationStack {
                CreditPartnersView()
                    .toolbar {
                        ToolbarItem(placement: .confirmationAction) {
                            Button("Done") { vm.showCredit = false }
                                .font(.system(.subheadline, design: .rounded)).fontWeight(.semibold)
                        }
                    }
            }
            .presentationDetents([.large])
            .presentationDragIndicator(.visible)
            .presentationCompactAdaptation(.sheet)
        }
        .sheet(isPresented: $vm.showHq) {
            NavigationStack {
                HqView()
                    .toolbar {
                        ToolbarItem(placement: .confirmationAction) {
                            Button("Done") { vm.showHq = false }
                                .font(.system(.subheadline, design: .rounded)).fontWeight(.semibold)
                        }
                    }
            }
            .presentationDetents([.large])
            .presentationDragIndicator(.visible)
            .presentationCompactAdaptation(.sheet)
        }
        .sheet(isPresented: $vm.showNotificationInbox) {
            NavigationStack {
                NotificationInboxView()
            }
            .presentationDetents([.large])
            .presentationDragIndicator(.visible)
            .presentationCompactAdaptation(.sheet)
        }
        .onChange(of: vm.showNotificationInbox) { _, isOpen in
            if !isOpen {
                Task { await vm.loadNotificationCount() }
            }
        }
        .onChange(of: cart.totalItems) {
            withAnimation(AnimationConstants.bouncy) { vm.cartBounce = true }
        }
        .onChange(of: vm.selectedTab) { oldTab, newTab in
            vm.handleTabChange(from: oldTab, to: newTab)
        }
        .onChange(of: scenePhase) {
            guard scenePhase == .active else { return }
            Task {
                await vm.refreshOnActive(refreshCenter: refreshCenter)
            }
        }
        .task {
            await vm.bootstrap()
        }
        .task(id: ws.reconnectEpoch) {
            await vm.loadClientPolicy()
        }
        .task {
            await vm.connectWebSocket(modelContext: modelContext, refreshCenter: refreshCenter)
        }
        .sheet(item: $vm.paymentEvent) { event in
            DeliveryPaymentSheetView(event: event) {
                vm.paymentEvent = nil
                Task { await vm.loadActiveOrders() }
            }
            .presentationDetents([.large])
            .interactiveDismissDisabled()
            .presentationCompactAdaptation(.sheet)
        }
        .sheet(item: $vm.shopClosedAlert) { alert in
            NavigationStack {
                ShopClosedBanner(event: alert) { _ in
                    vm.shopClosedAlert = nil
                    Task { await vm.loadActiveOrders() }
                }
                .navigationTitle("mobile_retailer.ui.shop_status")
                .navigationBarTitleDisplayMode(.inline)
            }
            .presentationDetents([.medium, .large])
            .interactiveDismissDisabled()
            .presentationCompactAdaptation(.sheet)
        }
        .animation(AnimationConstants.fluid, value: vm.selectedTab)
    }

    // MARK: - Layouts

    @ViewBuilder
    private var ipadLayout: some View {
        HStack(spacing: 0) {
            // MARK: Collapsible Navigation Rail
            VStack(alignment: .leading, spacing: 0) {
                VStack(alignment: .leading, spacing: 24) {
                    HStack {
                        if vm.isSidebarExpanded {
                            HStack(spacing: 8) {
                                Image(systemName: "leaf.fill")
                                    .font(.title3.weight(.bold))
                                    .foregroundStyle(AppTheme.accent)
                                Text("auth.login.title")
                                    .font(.system(.title3, design: .rounded)).fontWeight(.heavy)
                                    .foregroundStyle(AppTheme.textPrimary)
                            }
                            Spacer()
                        }

                        Button {
                            Haptics.light()
                            withAnimation(.spring(response: 0.4, dampingFraction: 0.7)) {
                                vm.isSidebarExpanded.toggle()
                            }
                        } label: {
                            Image(systemName: "sidebar.left")
                                .font(.title3.weight(.medium))
                                .foregroundStyle(AppTheme.textSecondary)
                                .frame(width: 44, height: 44)
                                .contentShape(Rectangle())
                        }
                        .accessibilityLabel(vm.isSidebarExpanded ? "Collapse sidebar" : "Expand sidebar")
                    }
                    .padding(.horizontal, vm.isSidebarExpanded ? 24 : 22)

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

                        if vm.isSidebarExpanded {
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
                    .padding(.horizontal, vm.isSidebarExpanded ? 24 : 22)
                    .accessibilityElement(children: .combine)
                }
                .padding(.top, 24)
                .padding(.bottom, 24)

                // Navigation Items
                ScrollView(showsIndicators: false) {
                    VStack(alignment: .leading, spacing: 8) {
                        if vm.isSidebarExpanded {
                            Text("MAIN")
                                .font(.system(.caption2, design: .rounded, weight: .bold))
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

                        if vm.isSidebarExpanded {
                            Text("TOOLS")
                                .font(.system(.caption2, design: .rounded, weight: .bold))
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
            .frame(width: vm.isSidebarExpanded ? 280 : 88)
            .background(AppTheme.cardBackground.ignoresSafeArea())
            .clipShape(
                RoundedRectangle(cornerRadius: AppTheme.radiusLG, style: .continuous)
            )
            .shadow(color: AppTheme.shadowColor.opacity(0.08), radius: 12, x: 4, y: 0)
            .zIndex(10)

            // MARK: Detail Content Area
            ZStack(alignment: .bottom) {
                ipadDetailContent
                    .frame(maxWidth: .infinity, maxHeight: .infinity)

                if vm.showFloatingBar(horizontalSizeClass: horizontalSizeClass) {
                    VStack {
                        Spacer()
                        FloatingActiveOrdersBar(activeOrders: vm.activeOrders) {
                            vm.showActiveOrderDetail = true
                        }
                        .retailerReadableWidth()
                        .padding(.horizontal, AppTheme.spacingMD)
                    }
                    .padding(.bottom, 32)
                    .transition(.move(edge: .bottom).combined(with: .opacity))
                    .animation(AnimationConstants.fluid, value: vm.activeOrders.count)
                }
            }
            .background(AppTheme.background.ignoresSafeArea())
            .zIndex(1)
        }
        .animation(.spring(response: 0.4, dampingFraction: 0.75), value: vm.isSidebarExpanded)
    }

    // MARK: - Sidebar Item Component

    @ViewBuilder
    private func sidebarItem(for tab: SideMenuTab) -> some View {
        let isSelected = vm.sideMenuSelection == tab

        Button {
            Haptics.selection()
            withAnimation(.spring(response: 0.3, dampingFraction: 0.7)) {
                if tab == .deliveries {
                    vm.deliveriesHubInitialTab = .map
                }
                vm.sideMenuSelection = tab
            }
        } label: {
            HStack(spacing: 16) {
                ZStack {
                    if isSelected {
                        RoundedRectangle(cornerRadius: AppTheme.radiusSM, style: .continuous)
                            .fill(AppTheme.accentSoft.opacity(0.6))
                            .frame(width: 44, height: 44)
                            .matchedGeometryEffect(id: "sidebar_active_bg", in: namespace)
                    } else {
                        RoundedRectangle(cornerRadius: AppTheme.radiusSM, style: .continuous)
                            .fill(Color.clear)
                            .frame(width: 44, height: 44)
                    }

                    Image(systemName: tab.icon)
                        .font(.title2.weight(isSelected ? .bold : .medium))
                        .foregroundStyle(isSelected ? AppTheme.accent : AppTheme.textSecondary)
                }

                if vm.isSidebarExpanded {
                    Text(tab.title)
                        .font(.system(.body, design: .rounded, weight: isSelected ? .bold : .medium))
                        .foregroundStyle(isSelected ? AppTheme.accent : AppTheme.textPrimary)
                        .transition(.move(edge: .trailing).combined(with: .opacity))

                    Spacer(minLength: 0)
                }
            }
            .padding(.vertical, 8)
            .padding(.horizontal, vm.isSidebarExpanded ? 16 : 22)
            .frame(maxWidth: .infinity, minHeight: AppTheme.minTouchTarget, alignment: .leading)
            .background(Color.white.opacity(0.001))
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .accessibilityLabel(tab.title)
    }

    @ViewBuilder
    private var ipadDetailContent: some View {
        switch vm.sideMenuSelection {
        case .home: tabContent(.home)
        case .catalog: tabContent(.catalog)
        case .orders:
            NavigationStack {
                OrdersView()
                    .toolbar { standardToolbar }
            }
        case .deliveries:
            NavigationStack {
                DeliveriesHubView(initialTab: vm.deliveriesHubInitialTab)
                    .toolbar { standardToolbar }
            }
        case .suppliers:
            NavigationStack {
                MySuppliersView()
                    .toolbar { standardToolbar }
            }
        case .profile:
            NavigationStack {
                ProfileView()
                    .toolbar { standardToolbar }
            }
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
        @Bindable var vm = vm
        ZStack(alignment: .bottom) {
            TabView(selection: $vm.selectedTab) {
                ForEach(AppTab.allCases, id: \.self) { tab in
                    Tab(tab.title, systemImage: tab.icon, value: tab) {
                        tabContent(tab)
                    }
                }
            }
            .sensoryFeedback(.selection, trigger: vm.selectedTab)
            .tint(AppTheme.accent)

            if vm.showFloatingBar(horizontalSizeClass: horizontalSizeClass) && !vm.showSidebar && !vm.activeOrders.isEmpty {
                FloatingActiveOrdersBar(activeOrders: vm.activeOrders) {
                    vm.showActiveOrderDetail = true
                }
                .retailerReadableWidth()
                .padding(.horizontal, AppTheme.spacingMD)
                .padding(.bottom, 52)
                .transition(.move(edge: .bottom).combined(with: .opacity))
                .animation(AnimationConstants.fluid, value: vm.activeOrders.count)
            }

            SidebarMenu(isOpen: $vm.showSidebar) { destination in
                vm.handleSidebarNavigation(destination, horizontalSizeClass: horizontalSizeClass, auth: auth)
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
                    CatalogView(onNavigateToSuppliers: {
                        vm.navigateToSuppliersTab(horizontalSizeClass: horizontalSizeClass)
                    })
                case .deliveries: DeliveriesHubView(initialTab: vm.deliveriesHubInitialTab)
                case .store: StoreStockView()
                case .more: RetailerMoreHubView()
                }
            }
            .toolbar { standardToolbar }
            .toolbarBackground(.ultraThinMaterial, for: .navigationBar)
            .toolbarBackground(.visible, for: .navigationBar)
            .toolbar(vm.showSidebar ? .hidden : .visible, for: .tabBar)
        }
        .sensoryFeedback(.impact(weight: .light), trigger: vm.showSidebar)
        .sensoryFeedback(.impact(weight: .light), trigger: vm.showInsights)
        .sensoryFeedback(.impact(weight: .light), trigger: vm.showActiveOrderDetail)
    }

    // MARK: - Toolbar

    @ToolbarContentBuilder
    private var standardToolbar: some ToolbarContent {
        if horizontalSizeClass != .regular {
            ToolbarItem(placement: .topBarLeading) {
                Button {
                    Haptics.light()
                    withAnimation(AnimationConstants.fluid) { vm.showSidebar.toggle() }
                } label: {
                    ZStack {
                        Circle()
                            .fill(AppTheme.accentGradient)
                            .frame(width: 32, height: 32)
                        Text(String((auth.currentUser?.name ?? "U").prefix(1)))
                            .font(.system(.caption, design: .rounded)).fontWeight(.bold)
                            .foregroundStyle(.white)
                    }
                    .pegasusMinimumHitTarget()
                }
                .accessibilityLabel("Menu")
            }
        }

        ToolbarItem(placement: .principal) {
            HStack(spacing: 6) {
                Image(systemName: "leaf.fill")
                    .font(.footnote)
                    .foregroundStyle(AppTheme.accent)
                Text("auth.login.title")
                    .font(.system(.headline, design: .rounded)).fontWeight(.bold)
                    .foregroundStyle(AppTheme.textPrimary)
            }
        }

        ToolbarItemGroup(placement: .topBarTrailing) {
            Button {
                Haptics.light()
                RetailerRefreshCenter.shared.trigger()
            } label: {
                Image(systemName: "arrow.clockwise")
                    .font(.callout.weight(.medium))
                    .foregroundStyle(AppTheme.textPrimary)
                    .pegasusMinimumHitTarget()
            }
            .accessibilityLabel("Refresh")

            Button {
                Haptics.selection()
                vm.showCart = true
            } label: {
                ZStack(alignment: .topTrailing) {
                    Image(systemName: "cart")
                        .font(.callout.weight(.medium))
                        .foregroundStyle(AppTheme.textPrimary)
                    if cart.totalItems > 0 {
                        Text("\(cart.totalItems)")
                            .font(.system(.caption2, design: .rounded, weight: .black))
                            .foregroundStyle(.white)
                            .frame(minWidth: 16, minHeight: 16)
                            .background(AppTheme.accent)
                            .clipShape(.circle)
                            .offset(x: 8, y: -6)
                    }
                }
                .pegasusMinimumHitTarget()
            }
            .accessibilityLabel("Cart, \(cart.totalItems) items")

            Button {
                Haptics.selection()
                vm.showNotificationInbox = true
            } label: {
                ZStack(alignment: .topTrailing) {
                    Image(systemName: "bell")
                        .font(.callout.weight(.medium))
                        .foregroundStyle(AppTheme.textPrimary)
                    if vm.notificationCount > 0 {
                        Text("\(vm.notificationCount)")
                            .font(.system(.caption2, design: .rounded, weight: .black))
                            .foregroundStyle(.white)
                            .frame(minWidth: 16, minHeight: 16)
                            .background(AppTheme.destructive)
                            .clipShape(.circle)
                            .offset(x: 8, y: -6)
                    }
                }
                .pegasusMinimumHitTarget()
            }
            .accessibilityLabel("Notifications, \(vm.notificationCount) new")
        }
    }
}
