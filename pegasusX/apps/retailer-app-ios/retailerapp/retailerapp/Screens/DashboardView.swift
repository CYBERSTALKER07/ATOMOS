import SwiftUI

struct DashboardView: View {
    @Environment(CartManager.self) private var cart
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @State private var refreshCenter = RetailerRefreshCenter.shared
    @State private var activeOrders: [Order] = []
    @State private var predictions: [DemandForecast] = []
    @State private var reorderProducts: [Product] = []
    @State private var isLoading = false
    @State private var preorderingId: String?
    @State private var orderActionPending = false
    @State private var loadError: String?
    @State private var pulseEvents: [RetailerPulseEvent] = []
    @State private var pulseLoading = false
    @State private var socket = RetailerWebSocket.shared

    private let api = APIClient.shared

    private var kpiGridMin: CGFloat {
        horizontalSizeClass == .regular ? 160 : 140
    }

    var body: some View {
        ScrollView {
            VStack(spacing: AppTheme.spacingXL) {
                if isLoading && activeOrders.isEmpty && predictions.isEmpty && reorderProducts.isEmpty {
                    RetailerLoadingView(
                        title: "Loading home",
                        message: "Fetching active orders, AI predictions, and quick reorder picks."
                    )
                } else if let loadError, activeOrders.isEmpty && predictions.isEmpty {
                    RetailerErrorView(message: loadError) {
                        Task { await loadData() }
                    }
                } else {
                    KpiGrid(
                        activeOrdersCount: activeOrders.count,
                        predictionsCount: predictions.count,
                        reorderProductsCount: reorderProducts.count,
                        horizontalSizeClass: horizontalSizeClass
                    )
                        .slideIn(delay: 0)

                    PulseStripView(events: pulseEvents, loading: pulseLoading)
                        .slideIn(delay: 0.02)

                    // Hero Service Grid (Yandex Go style)
                    ServiceGrid(
                        activeOrdersCount: activeOrders.count,
                        predictionsCount: predictions.count
                    )
                        .slideIn(delay: 0.05)

                    // Quick Reorder
                    QuickReorderSection(reorderProducts: reorderProducts)
                        .slideIn(delay: 0.1)

                    // AI Prediction Cards
                    AiPredictionSection(
                        predictions: predictions,
                        preorderingId: $preorderingId,
                        onPreorder: { forecast in
                            await preorder(forecast)
                        }
                    )
                        .slideIn(delay: 0.15)
                }
            }
            .padding(.horizontal, AppTheme.spacingLG)
            .padding(.bottom, AppTheme.spacingHuge)
            .retailerReadableWidth()
        }
        .scrollIndicators(.hidden)
        .background(AppTheme.background)
        .task {
            await loadData()
        }
        .task(id: refreshCenter.refreshToken) {
            await loadData(silent: hasCachedDashboardData)
        }
        .task {
            for await event in RetailerWebSocket.shared.eventStream() {
                if case .promotionChanged = event {
                    await loadData(silent: hasCachedDashboardData)
                }
            }
        }
        .refreshable {
            await loadData()
        }
        .onChange(of: socket.reconnectEpoch) { _, _ in
            if orderActionPending {
                orderActionPending = false
                loadError = "Connection restored — verify order status before retrying."
                Task { await loadData(silent: hasCachedDashboardData) }
            }
        }
    }



    // MARK: - Active Deliveries

    private var activeDeliveriesSection: some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
            RetailerSectionHeader(title: "Active Deliveries", icon: "shippingbox.fill", count: activeOrders.count)

            if activeOrders.isEmpty {
                emptyState(icon: "shippingbox", title: "All clear!", message: "No active deliveries right now")
            } else {
                ForEach(Array(activeOrders.enumerated()), id: \.element.id) { index, order in
                    OrderCardView(
                        order: order,
                        onCancel: {
                            Task { await cancelOrder(order.id) }
                        },
                        onConfirmAi: {
                            Task { await confirmAiOrder(order.id) }
                        },
                        onRejectAi: {
                            Task { await rejectAiOrder(order.id) }
                        },
                        onConfirmPreorder: {
                            Task { await confirmPreorder(order.id) }
                        },
                        onEditPreorder: {
                            Task { await editPreorder(order.id) }
                        }
                    )
                    .staggeredSlideIn(index: index)
                }
            }
        }
    }




    // MARK: - Helpers

    private func emptyState(icon: String, title: String, message: String) -> some View {
        VStack(spacing: AppTheme.spacingMD) {
            ZStack {
                Circle()
                    .fill(AppTheme.surfaceElevated)
                    .frame(width: 64, height: 64)
                Image(systemName: icon)
                    .font(.system(size: 24))
                    .foregroundStyle(AppTheme.textTertiary)
            }
            Text(title)
                .font(.system(.subheadline, design: .rounded, weight: .semibold))
                .foregroundStyle(AppTheme.textPrimary)
            Text(message)
                .font(.caption)
                .foregroundStyle(AppTheme.textTertiary)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, AppTheme.spacingXXL)
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
        .shadow(color: AppTheme.shadowColor, radius: AppTheme.shadowRadius, x: 0, y: AppTheme.shadowOffsetY)
    }

    private func confidenceColor(_ confidence: Double) -> Color {
        if confidence >= 0.8 { return AppTheme.success }
        if confidence >= 0.6 { return AppTheme.warning }
        return AppTheme.destructive
    }

    // MARK: - API

    private var hasCachedDashboardData: Bool {
        !activeOrders.isEmpty || !predictions.isEmpty || !reorderProducts.isEmpty
    }

    private func loadData(silent: Bool = false) async {
        let rid = AuthManager.shared.currentUser?.id ?? ""
        if !silent { isLoading = true }
        if !silent { loadError = nil }
        pulseLoading = true
        do {
            let orders: [Order] = try await api.get(path: "/v1/retailers/\(rid)/orders")
            activeOrders = orders.filter { $0.status.isActive }
        } catch {
            if !silent {
                activeOrders = []
                loadError = "Could not load dashboard data. Pull to refresh or try again."
            }
        }

        do {
            let forecasts: [DemandForecast] = try await api.get(path: "/v1/ai/predictions?retailer_id=\(rid)")
            predictions = forecasts
        } catch {
            if !silent { predictions = [] }
        }

        do {
            let products: [Product] = try await api.get(path: "/v1/catalog/products")
            reorderProducts = Array(products.prefix(6))
        } catch {
            if !silent { reorderProducts = [] }
        }

        do {
            let pulse = try await api.getRetailerPulse()
            pulseEvents = pulse.events
        } catch {
            pulseEvents = []
        }
        pulseLoading = false
        if !silent { isLoading = false }
    }

    private func cancelOrder(_ orderId: String) async {
        let retailerId = AuthManager.shared.currentUser?.id ?? ""
        do {
            let _: [String: String] = try await api.post(
                path: "/v1/order/cancel",
                body: [
                    "order_id": orderId,
                    "retailer_id": retailerId,
                ],
                headers: ["Idempotency-Key": RetailerIdempotency.cancel(orderId: orderId)]
            )
            withAnimation(AnimationConstants.fluid) {
                activeOrders.removeAll { $0.id == orderId }
            }
        } catch {}
    }

    private func confirmAiOrder(_ orderId: String) async {
        orderActionPending = true
        defer { orderActionPending = false }
        do {
            try await APIClient.shared.confirmAiOrder(orderId: orderId)
            await loadData()
        } catch {
            loadError = "Failed to confirm AI order"
        }
    }

    private func rejectAiOrder(_ orderId: String) async {
        orderActionPending = true
        defer { orderActionPending = false }
        do {
            try await APIClient.shared.rejectAiOrder(orderId: orderId, reason: "Retailer rejected")
            await loadData()
        } catch {
            loadError = "Failed to reject AI order"
        }
    }

    private func confirmPreorder(_ orderId: String) async {
        orderActionPending = true
        defer { orderActionPending = false }
        do {
            try await APIClient.shared.confirmPreorder(orderId: orderId)
            await loadData()
        } catch {
            loadError = "Failed to confirm preorder"
        }
    }

    private func editPreorder(_ orderId: String) async {
        orderActionPending = true
        defer { orderActionPending = false }
        guard let order = await findOrder(orderId) else { return }
        let deliveryDate = order.deliverBefore ?? order.autoConfirmAt ?? order.estimatedDelivery ?? ""
        guard !deliveryDate.isEmpty else { return }
        let lineItems = order.items.map { item in
            APIClient.EditPreorderItem(
                sku: item.productId.isEmpty ? item.id : item.productId,
                name: item.productName,
                quantity: Int64(item.quantity),
                unitPriceMinor: Int64(item.unitPrice.rounded())
            )
        }
        guard !lineItems.isEmpty else { return }
        do {
            try await APIClient.shared.editPreorder(orderId: orderId, deliveryDate: deliveryDate, items: lineItems)
            await loadData()
        } catch {
            print("Failed to edit preorder")
        }
    }

    private func findOrder(_ orderId: String) async -> Order? {
        if let order = activeOrders.first(where: { $0.id == orderId }) {
            return order
        }
        let retailerId = AuthManager.shared.currentUser?.id ?? ""
        guard !retailerId.isEmpty else { return nil }
        let orders: [Order] = (try? await api.get(path: "/v1/retailers/\(retailerId)/orders")) ?? []
        return orders.first { $0.id == orderId }
    }

    private func preorder(_ forecast: DemandForecast) async {
        preorderingId = forecast.id
        do {
            let body = PreorderRequest(productId: forecast.productId, quantity: forecast.predictedQuantity)
            let _: [String: String] = try await api.post(path: "/v1/ai/preorder", body: body)
            Haptics.success()
        } catch {
            Haptics.error()
        }
        preorderingId = nil
    }
}

private struct ServiceTile {
    enum Size { case small, regular, large }
    let title: String
    let icon: String
    let subtitle: String?
    let size: Size
}

private struct PreorderRequest: Codable {
    let productId: String
    let quantity: Int
    enum CodingKeys: String, CodingKey {
        case productId = "product_id"
        case quantity
    }
}

#Preview {
    NavigationStack {
        DashboardView()
            .environment(CartManager())
    }
}
