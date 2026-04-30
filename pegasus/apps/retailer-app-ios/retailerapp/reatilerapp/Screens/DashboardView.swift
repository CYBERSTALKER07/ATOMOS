import SwiftUI

struct DashboardView: View {
    @Environment(CartManager.self) private var cart
    @State private var activeOrders: [Order] = []
    @State private var predictions: [DemandForecast] = []
    @State private var reorderProducts: [Product] = []
    @State private var isLoading = false
    @State private var preorderingId: String?

    private let api = APIClient.shared

    var body: some View {
        ScrollView {
            VStack(spacing: AppTheme.spacingXL) {
                // Hero Service Grid (Yandex Go style)
                serviceGrid
                    .slideIn(delay: 0)

                // Quick Reorder
                quickReorderSection
                    .slideIn(delay: 0.1)

                // AI Prediction Cards
                aiPredictionSection
                    .slideIn(delay: 0.15)
            }
            .padding(.horizontal, AppTheme.spacingLG)
            .padding(.bottom, AppTheme.spacingHuge)
        }
        .scrollIndicators(.hidden)
        .background(AppTheme.background)
        .task {
            await loadData()
        }
        .refreshable {
            await loadData()
        }
    }

    // MARK: - Service Grid (Yandex Go Style)

    private var serviceGrid: some View {
        return VStack(spacing: AppTheme.spacingMD) {
            // Row 1: two big tiles
            HStack(spacing: AppTheme.spacingMD) {
                serviceTileView(title: "Catalog", icon: "bag.fill", subtitle: "Browse products", height: 130)
                serviceTileView(title: "AI Insights", icon: "sparkles", subtitle: "\(predictions.count) predictions", height: 130)
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
                .font(.system(size: 28, weight: .medium))
                .foregroundStyle(AppTheme.textPrimary)
                .padding(.bottom, AppTheme.spacingSM)

            Text(title)
                .font(.system(.subheadline, design: .rounded, weight: .semibold))
                .foregroundStyle(AppTheme.textPrimary)

            if let subtitle {
                Text(subtitle)
                    .font(.system(.caption2, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
                    .padding(.top, 1)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .frame(height: height)
        .padding(AppTheme.spacingMD)
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
        .shadow(color: AppTheme.shadowColor, radius: 4, x: 0, y: 2)
        .pressable()
    }

    private func serviceTileSmall(title: String, icon: String) -> some View {
        VStack(spacing: AppTheme.spacingSM) {
            Image(systemName: icon)
                .font(.system(size: 22, weight: .medium))
                .foregroundStyle(AppTheme.textPrimary)

            Text(title)
                .font(.system(.caption2, design: .rounded, weight: .medium))
                .foregroundStyle(AppTheme.textSecondary)
        }
        .frame(maxWidth: .infinity)
        .frame(height: 80)
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
        .shadow(color: AppTheme.shadowColor, radius: 4, x: 0, y: 2)
        .pressable()
    }

    // MARK: - Active Deliveries

    private var activeDeliveriesSection: some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
            sectionHeader(title: "Active Deliveries", icon: "shippingbox.fill", count: activeOrders.count)

            if activeOrders.isEmpty {
                emptyState(icon: "shippingbox", title: "All clear!", message: "No active deliveries right now")
            } else {
                ForEach(Array(activeOrders.enumerated()), id: \.element.id) { index, order in
                    OrderCardView(order: order) {
                        Task { await cancelOrder(order.id) }
                    }
                    .staggeredSlideIn(index: index)
                }
            }
        }
    }

    // MARK: - Quick Reorder

    private var quickReorderSection: some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
            sectionHeader(title: "Quick Reorder", icon: "arrow.clockwise", count: nil)

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

                Text(product.displayPrice)
                    .font(.system(.caption2, design: .rounded, weight: .bold))
                    .foregroundStyle(AppTheme.textPrimary)
            }
            .padding(AppTheme.spacingSM)
        }
        .pressable()
    }

    // MARK: - AI Predictions

    private var aiPredictionSection: some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
            sectionHeader(title: "AI Predictions", icon: "sparkles", count: predictions.count)

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
                    Text(forecast.productName)
                        .font(.system(.subheadline, design: .rounded, weight: .semibold))
                        .foregroundStyle(AppTheme.textPrimary)

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

    private func sectionHeader(title: String, icon: String, count: Int?) -> some View {
        HStack(spacing: AppTheme.spacingSM) {
            Image(systemName: icon)
                .font(.system(size: 14, weight: .semibold))
                .foregroundStyle(AppTheme.textSecondary)

            Text(title)
                .font(.system(.headline, design: .rounded))
                .foregroundStyle(AppTheme.textPrimary)

            if let count {
                Text("\(count)")
                    .font(.system(.caption2, design: .rounded, weight: .bold))
                    .foregroundStyle(.white)
                    .frame(width: 20, height: 20)
                    .background(AppTheme.accent)
                    .clipShape(.circle)
            }

            Spacer()
        }
    }

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

    private func loadData() async {
        let rid = AuthManager.shared.currentUser?.id ?? ""
        isLoading = true
        do {
            let orders: [Order] = try await api.get(path: "/v1/retailers/\(rid)/orders")
            activeOrders = orders.filter { $0.status.isActive }
        } catch {
            activeOrders = []
        }

        do {
            let forecasts: [DemandForecast] = try await api.get(path: "/v1/ai/predictions?retailer_id=\(rid)")
            predictions = forecasts
        } catch {
            predictions = []
        }

        do {
            let products: [Product] = try await api.get(path: "/v1/products")
            reorderProducts = products
        } catch {
            reorderProducts = []
        }
        isLoading = false
    }

    private func cancelOrder(_ orderId: String) async {
        let retailerId = AuthManager.shared.currentUser?.id ?? ""
        do {
            let _: [String: String] = try await api.post(path: "/v1/order/cancel", body: [
                "order_id": orderId,
                "retailer_id": retailerId,
            ])
            withAnimation(AnimationConstants.fluid) {
                activeOrders.removeAll { $0.id == orderId }
            }
        } catch {}
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
