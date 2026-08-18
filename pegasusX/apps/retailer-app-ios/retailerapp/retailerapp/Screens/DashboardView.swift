import SwiftUI

struct DashboardView: View {
    @Environment(CartManager.self) private var cart
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @State private var refreshCenter = RetailerRefreshCenter.shared
    @State private var activeOrders: [Order] = []
    @State private var predictions: [RetailerAIPrediction] = []
    @State private var reorderProducts: [Product] = []
    @State private var isLoading = false
    @State private var actingId: String?
    @State private var orderActionPending = false
    @State private var loadError: String?
    @State private var pulseEvents: [RetailerPulseEvent] = []
    @State private var pulseLoading = false
    @State private var pulseError: String?
    @State private var commandPulse: ControlTowerPulseWire?
    @State private var commandPulseError: String?
    @State private var socket = RetailerWebSocket.shared
    @State private var commandJump: CommandStatusJump?

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
                    retailerCommandBoard
                        .slideIn(delay: 0)

                    if commandPulseError == nil {
                        KpiGrid(
                            activeOrdersCount: commandPulse?.openOrders ?? activeOrders.count,
                            predictionsCount: predictions.count,
                            reorderProductsCount: reorderProducts.count,
                            horizontalSizeClass: horizontalSizeClass
                        )
                        .slideIn(delay: 0)
                    }

                    PulseStripView(events: pulseEvents, loading: pulseLoading, error: pulseError)
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
                        actingId: $actingId,
                        onConfirm: { item in
                            await confirmAiOrder(item.orderId)
                        },
                        onReject: { item in
                            await rejectAiOrder(item.orderId)
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
        .navigationDestination(item: $commandJump) { jump in
            OrdersView(initialCommandStatus: jump.status, initialSupplierId: jump.supplierId)
        }
        .task {
            while !Task.isCancelled {
                try? await Task.sleep(nanoseconds: 60_000_000_000)
                await loadCommandPulse()
            }
        }
        .onChange(of: socket.reconnectEpoch) { _, _ in
            if orderActionPending {
                orderActionPending = false
                loadError = "Connection restored — verify order status before retrying."
                Task { await loadData(silent: hasCachedDashboardData) }
            }
        }
    }



    private var retailerCommandBoard: some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
            HStack {
                Text("Retailer command")
                    .font(.system(.headline, design: .rounded))
                Spacer()
                if commandPulseError == nil {
                    SourceChip(source: commandPulse?.source ?? "empty")
                }
            }
            if let commandPulseError {
                Text(commandPulseError)
                    .font(.footnote)
                    .foregroundStyle(AppTheme.destructive)
                    .accessibilityIdentifier("gs-u-retailer-command-error")
            } else if commandPulse?.empty == true {
                Text("No live ops signals yet. Empty pulse — not demo tiles.")
                    .font(.footnote)
                    .foregroundStyle(AppTheme.textSecondary)
                    .accessibilityIdentifier("gs-u-retailer-pulse-empty")
            } else if let pulse = commandPulse {
                LazyVGrid(columns: [GridItem(.adaptive(minimum: 140), spacing: 8)], spacing: 8) {
                    commandTile("Open orders", "\(pulse.openOrders)")
                    commandTile("Fulfillment", "\(pulse.activeFulfillments)")
                    commandTile("Dock", "\(pulse.dockPending)")
                    commandTile("POS", "\(pulse.posOpenSessions)")
                }
            }
            if commandPulseError == nil {
                StatusStackView(
                    dictionary: orderStatusFunnel,
                    counts: commandPulse?.ordersByStatus,
                    source: commandPulse?.source,
                    onSelect: { commandJump = CommandStatusJump(status: $0) }
                )
                .accessibilityIdentifier("gs-u-retailer-stack")
                ForEach(Array((commandPulse?.ordersBySupplier ?? []).enumerated()), id: \.offset) { _, facet in
                    VStack(alignment: .leading, spacing: 6) {
                        Text(facet.supplierId.isEmpty ? "missing supplier" : facet.supplierId)
                            .font(.caption)
                            .foregroundStyle(AppTheme.textTertiary)
                        StatusStackView(
                            dictionary: orderStatusFunnel,
                            counts: facet.ordersByStatus,
                            source: commandPulse?.source,
                            onSelect: { commandJump = CommandStatusJump(status: $0, supplierId: facet.supplierId) }
                        )
                    }
                    .accessibilityIdentifier("gs-u-retailer-supplier-facet")
                }
                if commandPulse?.loyalty?.enrolled != true {
                    Text("Not enrolled. No fake Bronze — supplier has not configured a program, or you have no points yet.")
                        .font(.footnote)
                        .foregroundStyle(AppTheme.textSecondary)
                        .accessibilityIdentifier("gs-u-retailer-loyalty")
                }
            }
        }
    }

    private func commandTile(_ label: String, _ value: String) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(label)
                .font(.caption2)
                .foregroundStyle(AppTheme.textTertiary)
            Text(value)
                .font(.system(.headline, design: .rounded))
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(10)
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
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
        pulseError = nil
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
            predictions = try await api.getRetailerAIPredictions()
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
            let result = PulseHonesty.apply(ok: true, incoming: pulse.events, previous: pulseEvents)
            pulseEvents = result.events
            pulseError = result.error
        } catch {
            let result = PulseHonesty.apply(ok: false, incoming: nil, previous: pulseEvents)
            pulseEvents = result.events
            pulseError = result.error
        }
        await loadCommandPulse()
        pulseLoading = false
        if !silent { isLoading = false }
    }

    private func loadCommandPulse() async {
        do {
            let incoming = try await api.getControlTowerPulse()
            let result = PulseHonesty.applyObject(ok: true, incoming: incoming, previous: commandPulse)
            commandPulse = result.value
            commandPulseError = result.error
        } catch {
            let result = PulseHonesty.applyObject(ok: false, incoming: nil, previous: commandPulse)
            commandPulse = result.value
            commandPulseError = result.error
        }
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
        actingId = orderId
        orderActionPending = true
        defer {
            actingId = nil
            orderActionPending = false
        }
        do {
            try await APIClient.shared.confirmAiOrder(orderId: orderId)
            await loadData()
        } catch {
            loadError = "Failed to confirm AI order"
        }
    }

    private func rejectAiOrder(_ orderId: String) async {
        actingId = orderId
        orderActionPending = true
        defer {
            actingId = nil
            orderActionPending = false
        }
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
}

private struct ServiceTile {
    enum Size { case small, regular, large }
    let title: String
    let icon: String
    let subtitle: String?
    let size: Size
}

#Preview {
    NavigationStack {
        DashboardView()
            .environment(CartManager())
    }
}
