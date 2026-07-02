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
                    kpiGrid
                        .slideIn(delay: 0)

                    // Hero Service Grid (Yandex Go style)
                    serviceGrid
                        .slideIn(delay: 0.05)

                    // Quick Reorder
                    quickReorderSection
                        .slideIn(delay: 0.1)

                    // AI Prediction Cards
                    aiPredictionSection
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

    // MARK: - KPI Grid

    private var kpiGrid: some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
            RetailerSectionHeader(title: "At a glance", subtitle: "Live retailer KPIs")

            LazyVGrid(
                columns: [GridItem(.adaptive(minimum: kpiGridMin), spacing: AppTheme.spacingMD)],
                spacing: AppTheme.spacingMD
            ) {
                KpiTile(
                    title: "Active Orders",
                    value: "\(activeOrders.count)",
                    systemImage: "shippingbox.fill",
                    tint: AppTheme.accent,
                    chip: activeOrders.isEmpty ? nil : ("LIVE", AppTheme.success)
                )
                KpiTile(
                    title: "Reorder suggestions",
                    value: "\(predictions.count)",
                    systemImage: "sparkles",
                    tint: AppTheme.info,
                    chip: predictions.isEmpty ? nil : ("NEW", AppTheme.warning)
                )
                KpiTile(
                    title: "Quick Reorder",
                    value: "\(reorderProducts.count)",
                    systemImage: "arrow.clockwise",
                    tint: AppTheme.success
                )
            }
        }
    }

    // MARK: - Service Grid (Yandex Go Style)

    private var serviceGrid: some View {
        return VStack(spacing: AppTheme.spacingMD) {
            // Row 1: two big tiles
            HStack(spacing: AppTheme.spacingMD) {
                serviceTileView(title: "Catalog", icon: "bag.fill", subtitle: "Browse products", height: 130)
                serviceTileView(title: "Reorder suggestions", icon: "sparkles", subtitle: "\(predictions.count) items", height: 130)
            }

            // Row 2: one wide + two small
            HStack(spacing: AppTheme.spacingMD) {
                // Left: tall tile
                serviceTileView(title: "Orders", icon: "shippingbox.fill", subtitle: "\(activeOrders.count) active", height: 120)

                // Right: two small stacked
                VStack(spacing: AppTheme.spacingMD) {
                    serviceTileView(title: "Inbox", icon: "tray.fill", subtitle: nil, height: 54)
                    serviceTileView(title: "History", icon: "clock.fill", subtitle: nil, height: 54)
                }
            }

            // Row 3: three equal small tiles
            HStack(spacing: AppTheme.spacingMD) {
                serviceTileSmall(title: "Procurement", icon: "chart.bar.fill")
                serviceTileSmall(title: "Search", icon: "magnifyingglass")
                serviceTileSmall(title: "Profile", icon: "person.fill")
            }
        }
    }

    private func serviceTileView(title: String, icon: String, subtitle: String?, height: Double) -> some View {
        VStack(alignment: .leading, spacing: 0) {
            Spacer()

            Image(systemName: icon)
                .font(.system(size: 28, weight: .semibold)) // Bold icons
                .foregroundStyle(AppTheme.accent)
                .padding(.bottom, AppTheme.spacingSM)

            Text(title)
                .font(.system(.subheadline, design: .rounded, weight: .bold)) // Bold titles
                .foregroundStyle(AppTheme.textPrimary)

            if let subtitle {
                Text(subtitle)
                    .font(.system(.caption2, design: .rounded, weight: .medium)) // Medium weight
                    .foregroundStyle(AppTheme.textTertiary)
                    .padding(.top, 2)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .frame(height: height)
        .padding(AppTheme.spacingMD)
        .background {
            RoundedRectangle(cornerRadius: AppTheme.radiusCard, style: .continuous)
                .fill(AppTheme.cardBackground)
                .overlay {
                    RoundedRectangle(cornerRadius: AppTheme.radiusCard, style: .continuous)
                        .stroke(AppTheme.separator.opacity(0.12), lineWidth: 1)
                }
        }
        .pressable()
    }

    private func serviceTileSmall(title: String, icon: String) -> some View {
        VStack(spacing: AppTheme.spacingSM) {
            Image(systemName: icon)
                .font(.system(size: 20, weight: .semibold)) // Bold icons
                .foregroundStyle(AppTheme.accent)

            Text(title)
                .font(.system(.caption2, design: .rounded, weight: .bold)) // Bold titles
                .foregroundStyle(AppTheme.textSecondary)
        }
        .frame(maxWidth: .infinity)
        .frame(height: 80)
        .background {
            RoundedRectangle(cornerRadius: AppTheme.radiusCard, style: .continuous)
                .fill(AppTheme.cardBackground)
                .overlay {
                    RoundedRectangle(cornerRadius: AppTheme.radiusCard, style: .continuous)
                        .stroke(AppTheme.separator.opacity(0.12), lineWidth: 1)
                }
        }
        .pressable()
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

    // MARK: - Quick Reorder

    private var quickReorderSection: some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
            RetailerSectionHeader(title: "Quick Reorder", icon: "arrow.clockwise")

            ScrollView(.horizontal) {
                HStack(spacing: AppTheme.spacingMD) {
                    ForEach(Array(reorderProducts.prefix(6).enumerated()), id: \.element.id) { index, product in
                        quickReorderItem(product)
                            .staggeredSlideIn(index: index)
                    }
                }
            }
            .scrollIndicators(.hidden)
        }
    }

    private func quickReorderItem(_ product: Product) -> some View {
        Button {
            if let variant = product.defaultVariant {
                Haptics.light()
                withAnimation(AnimationConstants.bouncy) {
                    cart.add(product: product, variant: variant)
                }
            }
        } label: {
            VStack(spacing: AppTheme.spacingSM) {
                ZStack {
                    RoundedRectangle(cornerRadius: AppTheme.radiusMD)
                        .fill(AppTheme.surfaceElevated)
                        .frame(width: 64, height: 64)
                    Image(systemName: "leaf.fill")
                        .font(.system(size: 24))
                        .foregroundStyle(AppTheme.textTertiary)
                }

                Text(product.name)
                    .font(.system(.caption2, design: .rounded, weight: .medium))
                    .foregroundStyle(AppTheme.textPrimary)
                    .lineLimit(1)
                    .frame(width: 70)

                VStack(spacing: 2) {
                    if product.hasSaleOffer, let listPrice = product.displayListPrice {
                        Text(listPrice)
                            .font(.system(.caption2, design: .rounded, weight: .medium))
                            .foregroundStyle(AppTheme.textTertiary)
                            .strikethrough()
                    }
                    Text(product.displayPrice)
                        .font(.system(.caption2, design: .rounded, weight: .bold))
                        .foregroundStyle(product.hasSaleOffer ? AppTheme.success : AppTheme.textPrimary)
                }
            }
            .padding(AppTheme.spacingSM)
        }
        .pressable()
    }

    // MARK: - AI Predictions

    private var aiPredictionSection: some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
            RetailerSectionHeader(title: "AI Predictions", icon: "sparkles", count: predictions.count)

            ForEach(Array(predictions.enumerated()), id: \.element.id) { index, forecast in
                predictionCard(forecast)
                    .staggeredSlideIn(index: index)
            }
        }
    }

    private func predictionCard(_ forecast: DemandForecast) -> some View {
        LabCard {
            HStack(spacing: AppTheme.spacingMD) {
                // Confidence ring
                ZStack {
                    Circle()
                        .stroke(AppTheme.separator.opacity(0.3), lineWidth: 3)
                        .frame(width: 44, height: 44)
                    Circle()
                        .trim(from: 0, to: forecast.confidence)
                        .stroke(confidenceColor(forecast.confidence), style: StrokeStyle(lineWidth: 3, lineCap: .round))
                        .frame(width: 44, height: 44)
                        .rotationEffect(.degrees(-90))
                    Text(forecast.confidencePercent)
                        .font(.system(size: 10, weight: .bold, design: .rounded))
                        .foregroundStyle(confidenceColor(forecast.confidence))
                }

                VStack(alignment: .leading, spacing: 3) {
                    HStack(spacing: 6) {
                        Text(forecast.productName)
                            .font(.system(.subheadline, design: .rounded, weight: .semibold))
                            .foregroundStyle(AppTheme.textPrimary)
                        if forecast.isBlocked {
                            Text("Insufficient history")
                                .font(.system(size: 9, weight: .bold, design: .rounded))
                                .foregroundStyle(AppTheme.warning)
                                .padding(.horizontal, 6)
                                .padding(.vertical, 2)
                                .background(AppTheme.warning.opacity(0.12))
                                .clipShape(Capsule())
                        }
                    }

                    Text(forecast.reasoning)
                        .font(.caption)
                        .foregroundStyle(AppTheme.textTertiary)
                        .lineLimit(2)
                }

                Spacer(minLength: 0)

                VStack(spacing: 6) {
                    Text("\(forecast.predictedQuantity)")
                        .font(.system(.title3, design: .rounded, weight: .bold))
                        .foregroundStyle(AppTheme.textPrimary)
                    Text("units")
                        .font(.system(size: 9, weight: .medium, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)

                    Button {
                        guard preorderingId == nil else { return }
                        Task { await preorder(forecast) }
                    } label: {
                        Group {
                            if preorderingId == forecast.id {
                                ProgressView()
                                    .progressViewStyle(.circular)
                                    .tint(.white)
                            } else {
                                Image(systemName: "cart.badge.plus")
                                    .font(.system(size: 14, weight: .semibold))
                                    .foregroundStyle(.white)
                            }
                        }
                        .frame(width: 32, height: 32)
                        .background(AppTheme.accent)
                        .clipShape(.circle)
                    }
                    .disabled(preorderingId != nil)
                }
            }
            .padding(AppTheme.spacingLG)
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
