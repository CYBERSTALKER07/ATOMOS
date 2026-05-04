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
    @State private var selectedTab: OrderTab = .active
    @State private var allOrders: [Order] = []
    @State private var predictions: [DemandForecast] = []
    @State private var isLoading = false
    @State private var loadError = false
    @State private var selectedOrder: Order?
    @State private var qrOverlayOrder: Order?

    private let api = APIClient.shared

    var activeOrders: [Order] {
        allOrders.filter { $0.status == .loaded || $0.status == .dispatched || $0.status == .inTransit || $0.status == .arrived }
    }

    var pendingOrders: [Order] {
        allOrders.filter { $0.status == .pending }
    }

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
            .task { await loadData() }
            .task { await listenWebSocket() }
            .task { await flushPendingOrders() }
            .refreshable { await loadData() }
            .alert("Failed to Load", isPresented: $loadError) {
                Button("Retry") { Task { await loadData() } }
                Button("OK", role: .cancel) {}
            } message: {
                Text("Check your connection and try again.")
            }
            .sheet(item: $selectedOrder) { order in
                OrderDetailSheet(order: order)
                    .presentationDetents([.fraction(0.75)])
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
            if isLoading {
                SkeletonOrderList()
            } else if activeOrders.isEmpty {
                tabEmptyState(icon: "bolt.slash", title: "No Active Orders", message: "Orders being prepared or en route will appear here")
            } else {
                LazyVStack(spacing: AppTheme.spacingMD) {
                    ForEach(Array(activeOrders.enumerated()), id: \.element.id) { index, order in
                        activeOrderCard(order)
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
            if isLoading {
                SkeletonOrderList()
            } else if pendingOrders.isEmpty {
                tabEmptyState(icon: "clock", title: "No Pending Orders", message: "Orders awaiting confirmation will appear here")
            } else {
                LazyVStack(spacing: AppTheme.spacingMD) {
                    ForEach(Array(pendingOrders.enumerated()), id: \.element.id) { index, order in
                        pendingOrderCard(order)
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
            if isLoading {
                SkeletonOrderList(count: 3)
            } else if predictions.isEmpty {
                tabEmptyState(icon: "sparkles", title: "No AI Predictions", message: "AI-predicted orders based on your history will appear here")
            } else {
                LazyVStack(spacing: AppTheme.spacingMD) {
                    ForEach(Array(predictions.enumerated()), id: \.element.id) { index, forecast in
                        aiPlannedCard(forecast)
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

    // MARK: - Active Order Card

    private func activeOrderCard(_ order: Order) -> some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
            HStack(alignment: .top) {
                ZStack {
                    Circle().fill(AppTheme.success.opacity(0.1)).frame(width: 40, height: 40)
                    Image(systemName: "shippingbox.fill").font(.system(size: 15, weight: .semibold)).foregroundStyle(AppTheme.success)
                }

                VStack(alignment: .leading, spacing: 2) {
                    Text("Order #\(order.id.suffix(3))")
                        .font(.system(.subheadline, design: .rounded, weight: .bold))
                        .foregroundStyle(AppTheme.textPrimary)
                    Text("\(order.itemCount) items · \(order.displayTotal)")
                        .font(.system(.caption, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)
                }

                Spacer()

                HStack(spacing: 4) {
                    Circle().fill(AppTheme.success).frame(width: 6, height: 6)
                    Text(order.status.displayName).font(.system(size: 11, weight: .bold, design: .rounded))
                }
                .foregroundStyle(AppTheme.success)
                .padding(.horizontal, 10).padding(.vertical, 5)
                .background(AppTheme.success.opacity(0.08))
                .clipShape(.capsule)
            }

            // Order Status Timeline
            OrderStatusTimeline(currentStep: order.status.timelineStepIndex)

            if let eta = order.estimatedDelivery {
                HStack(spacing: AppTheme.spacingSM) {
                    Image(systemName: "clock").font(.system(size: 12, weight: .semibold)).foregroundStyle(AppTheme.textSecondary)
                    CountdownText(targetISO: eta, font: .system(.caption, design: .monospaced, weight: .bold), color: AppTheme.textPrimary)
                    Spacer()
                }
                .padding(AppTheme.spacingSM)
                .background(AppTheme.surfaceElevated)
                .clipShape(.rect(cornerRadius: AppTheme.radiusSM))
            }

            Rectangle().fill(AppTheme.separator.opacity(0.2)).frame(height: AppTheme.separatorHeight)

            HStack(spacing: AppTheme.spacingMD) {
                Button {
                    Haptics.light()
                    selectedOrder = order
                } label: {
                    HStack(spacing: 4) {
                        Image(systemName: "doc.text").font(.system(size: 12, weight: .semibold))
                        Text("Details").font(.system(.caption, design: .rounded, weight: .semibold))
                    }
                    .foregroundStyle(AppTheme.textPrimary)
                    .padding(.horizontal, AppTheme.spacingMD).padding(.vertical, AppTheme.spacingSM)
                    .background(AppTheme.surfaceElevated)
                    .clipShape(.capsule)
                }

                if order.status.hasDeliveryToken {
                    Button {
                        Haptics.light()
                        qrOverlayOrder = order
                    } label: {
                        HStack(spacing: 4) {
                            Image(systemName: "qrcode").font(.system(size: 12, weight: .semibold))
                            Text("Show QR").font(.system(.caption, design: .rounded, weight: .semibold))
                        }
                        .foregroundStyle(.white)
                        .padding(.horizontal, AppTheme.spacingMD).padding(.vertical, AppTheme.spacingSM)
                        .background(AppTheme.accent)
                        .clipShape(.capsule)
                    }
                } else {
                    HStack(spacing: 4) {
                        Image(systemName: "qrcode").font(.system(size: 12, weight: .semibold))
                        Text("Awaiting Dispatch").font(.system(.caption, design: .rounded, weight: .semibold))
                    }
                    .foregroundStyle(AppTheme.textTertiary)
                    .padding(.horizontal, AppTheme.spacingMD).padding(.vertical, AppTheme.spacingSM)
                    .background(AppTheme.surfaceElevated)
                    .clipShape(.capsule)
                }
                Spacer()
            }
        }
        .padding(AppTheme.spacingLG)
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
        .shadow(color: AppTheme.shadowColor, radius: 4, y: 2)
    }

    // MARK: - Pending Order Card

    private func pendingOrderCard(_ order: Order) -> some View {
        HStack(spacing: AppTheme.spacingMD) {
            ZStack {
                Circle().fill(AppTheme.warning.opacity(0.1)).frame(width: 40, height: 40)
                Image(systemName: "clock.fill").font(.system(size: 15, weight: .semibold)).foregroundStyle(AppTheme.warning)
            }

            VStack(alignment: .leading, spacing: 2) {
                Text("Order #\(order.id.suffix(3))")
                    .font(.system(.subheadline, design: .rounded, weight: .semibold))
                    .foregroundStyle(AppTheme.textPrimary)
                Text("\(order.itemCount) items · \(order.displayTotal)")
                    .font(.system(.caption, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
            }

            Spacer()

            Button {
                Haptics.light()
                selectedOrder = order
            } label: {
                Text("View")
                    .font(.system(.caption, design: .rounded, weight: .bold))
                    .foregroundStyle(AppTheme.textPrimary)
                    .padding(.horizontal, AppTheme.spacingMD).padding(.vertical, 6)
                    .background(AppTheme.surfaceElevated)
                    .clipShape(.capsule)
            }
        }
        .padding(AppTheme.spacingMD)
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
        .shadow(color: AppTheme.shadowColor, radius: 3, y: 1)
    }

    // MARK: - AI Planned Card

    private func aiPlannedCard(_ forecast: DemandForecast) -> some View {
        HStack(spacing: AppTheme.spacingMD) {
            ZStack {
                Circle().stroke(AppTheme.separator.opacity(0.3), lineWidth: 3).frame(width: 40, height: 40)
                Circle()
                    .trim(from: 0, to: forecast.confidence)
                    .stroke(confidenceColor(forecast.confidence), style: StrokeStyle(lineWidth: 3, lineCap: .round))
                    .frame(width: 40, height: 40)
                    .rotationEffect(.degrees(-90))
                Text(forecast.confidencePercent)
                    .font(.system(size: 9, weight: .bold, design: .rounded))
                    .foregroundStyle(confidenceColor(forecast.confidence))
            }

            VStack(alignment: .leading, spacing: 2) {
                Text(forecast.productName)
                    .font(.system(.subheadline, design: .rounded, weight: .semibold))
                    .foregroundStyle(AppTheme.textPrimary)
                    .lineLimit(1)
                Text("Order by \(forecast.suggestedOrderDate)")
                    .font(.system(.caption2, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
            }

            Spacer()

            VStack(alignment: .trailing, spacing: 4) {
                Text("\(forecast.predictedQuantity) units")
                    .font(.system(.caption, design: .rounded, weight: .bold))
                    .foregroundStyle(AppTheme.textPrimary)

                Button {
                    Haptics.medium()
                    Task { await preorder(forecast) }
                } label: {
                    Text("Pre-Order")
                        .font(.system(size: 11, weight: .bold, design: .rounded))
                        .foregroundStyle(.white)
                        .padding(.horizontal, 10).padding(.vertical, 5)
                        .background(AppTheme.accent)
                        .clipShape(.capsule)
                }
            }
        }
        .padding(AppTheme.spacingMD)
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
        .shadow(color: AppTheme.shadowColor, radius: 3, y: 1)
    }

    // MARK: - Helpers

    private func confidenceColor(_ confidence: Double) -> Color {
        if confidence >= 0.8 { return AppTheme.success }
        if confidence >= 0.6 { return AppTheme.warning }
        return AppTheme.destructive
    }

    // MARK: - API

    private func listenWebSocket() async {
        let rid = AuthManager.shared.currentUser?.id ?? ""
        guard !rid.isEmpty else { return }
        RetailerWebSocket.shared.connect(retailerId: rid)
        for await event in RetailerWebSocket.shared.events {
            switch event {
            case .paymentRequired, .driverApproaching, .orderCompleted, .paymentSettled,
                 .paymentFailed, .paymentExpired, .orderStatusChanged,
                 .preOrderAutoAccepted, .preOrderConfirmed, .preOrderEdited:
                await loadData()
            }
        }
    }

    private func flushPendingOrders() async {
        let descriptor = FetchDescriptor<PendingOrder>(sortBy: [SortDescriptor(\.createdAt)])
        guard let pending = try? modelContext.fetch(descriptor), !pending.isEmpty else { return }
        for order in pending {
            guard let data = order.payloadJson.data(using: .utf8),
                  let payload = try? JSONDecoder().decode(UnifiedCheckoutPayload.self, from: data) else {
                modelContext.delete(order)
                continue
            }
            do {
                let _: CheckoutResponse = try await api.post(
                    path: "/v1/checkout/unified",
                    body: payload,
                    headers: ["Idempotency-Key": "retailer-checkout-pending:\(Int64(order.createdAt.timeIntervalSince1970 * 1000))"]
                )
                modelContext.delete(order)
            } catch {
                order.retryCount += 1
            }
        }
        try? modelContext.save()
    }

    private func loadData() async {
        let rid = AuthManager.shared.currentUser?.id ?? ""
        isLoading = true
        do {
            let orders: [Order] = try await api.get(path: "/v1/retailers/\(rid)/orders")
            allOrders = orders
        } catch {
            allOrders = []
            loadError = true
        }
        do {
            let forecasts: [DemandForecast] = try await api.get(path: "/v1/ai/predictions?retailer_id=\(rid)")
            predictions = forecasts
        } catch {
            predictions = []
        }
        isLoading = false
    }

    private func preorder(_ forecast: DemandForecast) async {
        do {
            struct PreorderBody: Codable {
                let productId: String
                let quantity: Int
                enum CodingKeys: String, CodingKey { case productId = "product_id"; case quantity }
            }
            let _: Order = try await api.post(
                path: "/v1/ai/preorder",
                body: PreorderBody(productId: forecast.productId, quantity: forecast.predictedQuantity),
                headers: ["Idempotency-Key": "retailer-ai-preorder:\(forecast.id):\(forecast.predictedQuantity)"]
            )
            Haptics.success()
        } catch {
            Haptics.error()
        }
    }
}

// MARK: - Order Status Timeline

private struct OrderStatusTimeline: View {
    let currentStep: Int

    private let steps = OrderStatus.timelineSteps

    var body: some View {
        HStack(spacing: 0) {
            ForEach(Array(steps.enumerated()), id: \.offset) { index, label in
                let isCompleted = index < currentStep
                let isActive = index == currentStep

                VStack(spacing: 4) {
                    Circle()
                        .fill(dotColor(isCompleted: isCompleted, isActive: isActive))
                        .frame(width: isActive ? 10 : 8, height: isActive ? 10 : 8)

                    Text(label)
                        .font(.system(size: 8, weight: isActive ? .bold : .medium, design: .rounded))
                        .foregroundStyle(labelColor(isCompleted: isCompleted, isActive: isActive))
                        .lineLimit(1)
                }
                .frame(maxWidth: .infinity)
            }
        }
        .padding(.horizontal, 10)
        .padding(.vertical, 10)
        .background(AppTheme.surfaceElevated.opacity(0.5))
        .clipShape(.rect(cornerRadius: AppTheme.radiusSM))
    }

    private func dotColor(isCompleted: Bool, isActive: Bool) -> Color {
        if isCompleted { return AppTheme.success }
        if isActive { return .teal }
        return AppTheme.textTertiary.opacity(0.4)
    }

    private func labelColor(isCompleted: Bool, isActive: Bool) -> Color {
        if isCompleted { return AppTheme.textSecondary }
        if isActive { return .teal }
        return AppTheme.textTertiary.opacity(0.5)
    }
}

#Preview {
    NavigationStack {
        OrdersView()
    }
}
