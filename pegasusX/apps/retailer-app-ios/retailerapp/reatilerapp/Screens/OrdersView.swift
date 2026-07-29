import SwiftData
import SwiftUI

// MARK: - Order Tab

enum OrderTab: String, CaseIterable {
    case active, pending, aiPlanned

    var title: String {
        switch self {
        case .active: "Active"
        case .pending: "Pending"
        case .aiPlanned: "AI Planned"
        }
    }

    var icon: String {
        switch self {
        case .active: "bolt.fill"
        case .pending: "clock.fill"
        case .aiPlanned: "sparkles"
        }
    }
}

struct OrdersView: View {
    @Environment(\.modelContext) private var modelContext
    @State private var refreshCenter = RetailerRefreshCenter.shared
    @State private var vm = OrdersViewModel()
    @State private var selectedTab: OrderTab = .active
    @State private var selectedOrder: Order?
    @State private var qrOverlayOrder: Order?
    @State private var deliveryProposalOrder: Order?

    private var activeOrders: [Order] { vm.activeOrders }
    private var pendingOrders: [Order] { vm.pendingOrders }
    private var predictions: [DemandForecast] { vm.predictions }

    var body: some View {
        ZStack {
            VStack(spacing: 0) {
                // Top Tabs
                topTabs

                Rectangle().fill(AppTheme.separator.opacity(0.3)).frame(height: AppTheme.separatorHeight)

                // Tab Content
                TabView(selection: $selectedTab) {
                    activeContent.tag(OrderTab.active)
                    pendingContent.tag(OrderTab.pending)
                    aiPlannedContent.tag(OrderTab.aiPlanned)
                }
                .tabViewStyle(.page(indexDisplayMode: .never))
            }
            .background(AppTheme.background)
            .task { await vm.loadData() }
            .task { await vm.listenWebSocket(modelContext: modelContext) }
            .task { await vm.flushPendingOrders(modelContext: modelContext) }
            .task(id: refreshCenter.refreshToken) {
                await vm.loadData(silent: !vm.allOrders.isEmpty)
                await vm.flushPendingOrders(modelContext: modelContext)
            }
            .refreshable { await vm.loadData() }
            .alert("Failed to Load", isPresented: Binding(
                get: { vm.loadError != nil },
                set: { if !$0 { vm.loadError = nil } }
            )) {
                Button("Retry") { Task { await vm.loadData() } }
                Button("OK", role: .cancel) { vm.loadError = nil }
            } message: {
                Text(vm.loadError ?? "Check your connection and try again.")
            }
            .sheet(item: $selectedOrder) { order in
                OrderDetailSheet(order: order)
                    .presentationDetents([.fraction(0.75)])
                    .presentationDragIndicator(.visible)
            }
            .sheet(item: $deliveryProposalOrder) { order in
                DeliveryProposalReviewSheet(
                    order: order,
                    isPending: vm.orderActionPending,
                    onAccept: {
                        RetailerAsync.run {
                            await vm.acceptDeliveryProposal(order.id)
                            deliveryProposalOrder = nil
                        }
                    },
                    onReject: {
                        RetailerAsync.run {
                            await vm.rejectDeliveryProposal(order.id)
                            deliveryProposalOrder = nil
                        }
                    }
                )
                .presentationDetents([.medium])
                .presentationDragIndicator(.visible)
            }

            // Quick QR Overlay
            if let qrOrder = qrOverlayOrder, qrOrder.status.hasDeliveryToken {
                QROverlay(order: qrOrder) {
                    withAnimation(AnimationConstants.fluid) { qrOverlayOrder = nil }
                }
                .transition(.opacity)
                .zIndex(200)
            }
        }
        .animation(AnimationConstants.fluid, value: qrOverlayOrder?.id)
    }

    // MARK: - Top Tabs

    private var topTabs: some View {
        HStack(spacing: 0) {
            ForEach(OrderTab.allCases, id: \.self) { tab in
                Button {
                    Haptics.light()
                    withAnimation(AnimationConstants.express) {
                        selectedTab = tab
                    }
                } label: {
                    VStack(spacing: AppTheme.spacingSM) {
                        ZStack(alignment: .topTrailing) {
                            Image(systemName: tab.icon)
                                .font(.system(size: 22, weight: selectedTab == tab ? .semibold : .regular))
                            
                            // Badge count
                            let count = badgeCount(for: tab)
                            if count > 0 {
                                Text("\(count)")
                                    .font(.system(size: 10, weight: .bold, design: .rounded))
                                    .foregroundStyle(.white)
                                    .frame(width: 16, height: 16)
                                    .background(AppTheme.destructive)
                                    .clipShape(.circle)
                                    .offset(x: 10, y: -6)
                            }
                        }

                        Text(tab.title)
                            .font(.system(.subheadline, design: .rounded, weight: selectedTab == tab ? .bold : .medium))
                    }
                    .foregroundStyle(selectedTab == tab ? AppTheme.accent : AppTheme.textTertiary.opacity(0.7))
                    .frame(maxWidth: .infinity)
                    .padding(.top, AppTheme.spacingMD)
                    .padding(.bottom, AppTheme.spacingSM)
                    .overlay(alignment: .bottom) {
                        Rectangle()
                            .fill(selectedTab == tab ? AppTheme.accent : .clear)
                            .frame(height: 3)
                    }
                }
            }
        }
        .background(AppTheme.cardBackground)
        .overlay(alignment: .bottom) {
            Rectangle()
                .fill(AppTheme.separator.opacity(0.3))
                .frame(height: 0.5)
        }
    }

    private func badgeCount(for tab: OrderTab) -> Int {
        switch tab {
        case .active: activeOrders.count
        case .pending: pendingOrders.count
        case .aiPlanned: predictions.count
        }
    }

    // MARK: - Active Content

    private var activeContent: some View {
        ScrollView {
            if vm.isLoading && activeOrders.isEmpty {
                SkeletonOrderList()
            } else if activeOrders.isEmpty {
                tabEmptyState(icon: "bolt.slash", title: "No Active Orders", message: "Orders being prepared or en route will appear here")
            } else {
                LazyVStack(spacing: AppTheme.spacingMD) {
                    ForEach(Array(activeOrders.enumerated()), id: \.element.id) { index, order in
                        ActiveOrderCard(order: order, onDetails: { selectedOrder = order }, onQR: { qrOverlayOrder = order })
                            .staggeredSlideIn(index: index)
                    }
                }
                .padding(.horizontal, AppTheme.spacingLG)
                .padding(.top, AppTheme.spacingMD)
                .padding(.bottom, AppTheme.spacingHuge)
            }
        }
        .scrollIndicators(.hidden)
    }

    // MARK: - Pending Content

    private var pendingContent: some View {
        ScrollView {
            if vm.isLoading && pendingOrders.isEmpty {
                SkeletonOrderList()
            } else if pendingOrders.isEmpty {
                tabEmptyState(icon: "clock", title: "No Pending Orders", message: "Orders awaiting confirmation will appear here")
            } else {
                LazyVStack(spacing: AppTheme.spacingMD) {
                    ForEach(Array(pendingOrders.enumerated()), id: \.element.id) { index, order in
                        OrderCardView(
                            order: order,
                            onCancel: order.status.canCancel ? {
                                RetailerAsync.run { await vm.cancelOrder(order) }
                            } : nil,
                            onConfirmAi: order.status == .pendingReview ? {
                                RetailerAsync.run { await vm.confirmAiOrder(order.id) }
                            } : nil,
                            onRejectAi: order.status == .pendingReview ? {
                                RetailerAsync.run { await vm.rejectAiOrder(order.id) }
                            } : nil,
                            onConfirmPreorder: order.needsManualPreorderAction ? {
                                RetailerAsync.run { await vm.confirmPreorder(order.id) }
                            } : nil,
                            onEditPreorder: order.needsManualPreorderAction ? {
                                RetailerAsync.run { await vm.editPreorder(order) }
                            } : nil,
                            onReviewDeliveryProposal: order.needsDeliveryProposalReview ? {
                                deliveryProposalOrder = order
                            } : nil
                        )
                        .staggeredSlideIn(index: index)
                    }
                }
                .padding(.horizontal, AppTheme.spacingLG)
                .padding(.top, AppTheme.spacingMD)
                .padding(.bottom, AppTheme.spacingHuge)
            }
        }
        .scrollIndicators(.hidden)
    }

    // MARK: - AI Planned Content

    private var aiPlannedContent: some View {
        ScrollView {
            if vm.isLoading && predictions.isEmpty {
            } else if predictions.isEmpty {
                tabEmptyState(icon: "sparkles", title: "No AI Predictions", message: "AI-predicted orders based on your history will appear here")
            } else {
                LazyVStack(spacing: AppTheme.spacingMD) {
                    ForEach(Array(predictions.enumerated()), id: \.element.id) { index, forecast in
                        AiPlannedCard(forecast: forecast, onPreorder: { Task { await vm.preorder(forecast) } })
                            .staggeredSlideIn(index: index)
                    }
                }
                .padding(.horizontal, AppTheme.spacingLG)
                .padding(.top, AppTheme.spacingMD)
                .padding(.bottom, AppTheme.spacingHuge)
            }
        }
        .scrollIndicators(.hidden)
    }

    // MARK: - Tab Empty State

    private func tabEmptyState(icon: String, title: String, message: String) -> some View {
        VStack(spacing: AppTheme.spacingLG) {
            Spacer(minLength: 60)
            ZStack {
                Circle().fill(AppTheme.surfaceElevated).frame(width: 72, height: 72)
                Image(systemName: icon).font(.system(size: 28)).foregroundStyle(AppTheme.textTertiary)
            }
            Text(title)
                .font(.system(.headline, design: .rounded))
                .foregroundStyle(AppTheme.textPrimary)
            Text(message)
                .font(.system(.subheadline, design: .rounded))
                .foregroundStyle(AppTheme.textTertiary)
                .multilineTextAlignment(.center)
            Spacer()
        }
        .padding(AppTheme.spacingXL)
    }

#Preview {
    NavigationStack {
        OrdersView()
    }
}
