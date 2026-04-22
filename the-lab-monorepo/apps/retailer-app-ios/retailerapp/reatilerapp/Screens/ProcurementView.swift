import SwiftUI

struct ProcurementView: View {
    @Environment(CartManager.self) private var cart
    @State private var forecasts: [DemandForecast] = []
    @State private var selectedItems: Set<String> = []
    @State private var quantities: [String: Int] = [:]
    @State private var isSubmitting = false
    @State private var showSuccess = false
    @State private var showError = false
    @State private var errorMessage = ""
    @State private var isLoading = false
    @State private var products: [Product] = []

    private let api = APIClient.shared

    var body: some View {
        VStack(spacing: 0) {
            ScrollView {
                VStack(spacing: AppTheme.spacingLG) {
                    headerSection.slideIn(delay: 0)
                    suggestionsSection.slideIn(delay: 0.05)

                    if !selectedItems.isEmpty {
                        selectedSummary.slideIn(delay: 0.1)
                    }
                }
                .padding(AppTheme.spacingLG)
                .padding(.bottom, selectedItems.isEmpty ? AppTheme.spacingXXL : 100)
            }
            .scrollIndicators(.hidden)

            if !selectedItems.isEmpty {
                actionBar
            }
        }
        .background(AppTheme.background)
        .task { await loadPredictions() }
        .alert("Order Created", isPresented: $showSuccess) {
            Button("OK") { selectedItems.removeAll(); quantities.removeAll() }
        } message: {
            Text("Your procurement order has been submitted successfully.")
        }
        .alert("Order Failed", isPresented: $showError) {
            Button("Retry") { Task { await createOrder() } }
            Button("Cancel", role: .cancel) {}
        } message: {
            Text(errorMessage)
        }
    }

    // MARK: - Header

    private var headerSection: some View {
        GradientHeaderCard(title: "AI Procurement", subtitle: "Smart suggestions based on demand analysis", icon: "sparkles") {
            HStack(spacing: AppTheme.spacingXL) {
                VStack(spacing: 3) {
                    Text("\(forecasts.count)")
                        .font(.system(.headline, design: .rounded, weight: .bold))
                        .foregroundStyle(AppTheme.textPrimary)
                    Text("Suggestions")
                        .font(.system(.caption2, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)
                }
                .frame(maxWidth: .infinity)

                VStack(spacing: 3) {
                    Text("\(selectedItems.count)")
                        .font(.system(.headline, design: .rounded, weight: .bold))
                        .foregroundStyle(AppTheme.accent)
                        .contentTransition(.numericText())
                    Text("Selected")
                        .font(.system(.caption2, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)
                }
                .frame(maxWidth: .infinity)
            }
        }
    }

    // MARK: - Suggestions

    private var suggestionsSection: some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
            HStack {
                HStack(spacing: AppTheme.spacingSM) {
                    Image(systemName: "lightbulb.fill")
                        .font(.system(size: 14, weight: .semibold))
                        .foregroundStyle(AppTheme.accent)
                    Text("Suggestions")
                        .font(.system(.headline, design: .rounded))
                        .foregroundStyle(AppTheme.textPrimary)
                }
                Spacer()
                Button {
                    withAnimation(AnimationConstants.express) {
                        if selectedItems.count == forecasts.count {
                            selectedItems.removeAll()
                        } else {
                            selectedItems = Set(forecasts.map(\.id))
                            for f in forecasts { quantities[f.id] = f.predictedQuantity }
                        }
                    }
                    Haptics.light()
                } label: {
                    Text(selectedItems.count == forecasts.count ? "Deselect All" : "Select All")
                        .font(.system(.caption, design: .rounded, weight: .bold))
                        .foregroundStyle(AppTheme.accent)
                }
            }

            ForEach(Array(forecasts.enumerated()), id: \.element.id) { index, forecast in
                suggestionCard(forecast)
                    .staggeredSlideIn(index: index)
            }
        }
    }

    private func suggestionCard(_ forecast: DemandForecast) -> some View {
        let isSelected = selectedItems.contains(forecast.id)
        let qty = quantities[forecast.id] ?? forecast.predictedQuantity

        return LabCard {
            VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
                HStack {
                    Button {
                        withAnimation(AnimationConstants.express) {
                            if isSelected { selectedItems.remove(forecast.id) }
                            else { selectedItems.insert(forecast.id); quantities[forecast.id] = forecast.predictedQuantity }
                        }
                        Haptics.light()
                    } label: {
                        ZStack {
                            RoundedRectangle(cornerRadius: 6)
                                .fill(isSelected ? AppTheme.accent : AppTheme.separator.opacity(0.3))
                                .frame(width: 24, height: 24)
                            if isSelected {
                                Image(systemName: "checkmark")
                                    .font(.system(size: 12, weight: .bold))
                                    .foregroundStyle(.white)
                            }
                        }
                    }

                    VStack(alignment: .leading, spacing: 2) {
                        Text(forecast.productName)
                            .font(.system(.subheadline, design: .rounded, weight: .semibold))
                            .foregroundStyle(AppTheme.textPrimary)
                        Text("Confidence: \(forecast.confidencePercent)")
                            .font(.system(.caption, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                    }

                    Spacer()

                    if isSelected {
                        QuantityStepper(quantity: Binding(get: { qty }, set: { quantities[forecast.id] = $0 }), compact: true)
                    } else {
                        Text("\(forecast.predictedQuantity) units")
                            .font(.system(.caption, design: .rounded, weight: .bold))
                            .foregroundStyle(AppTheme.accent)
                    }
                }

                Text(forecast.reasoning)
                    .font(.system(.caption, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
                    .lineLimit(2)
            }
            .padding(AppTheme.spacingLG)
        }
        .overlay {
            if isSelected {
                RoundedRectangle(cornerRadius: AppTheme.radiusCard)
                    .strokeBorder(AppTheme.accent.opacity(0.4), lineWidth: 2)
            }
        }
    }

    // MARK: - Summary

    private var selectedSummary: some View {
        LabCard {
            HStack {
                HStack(spacing: AppTheme.spacingSM) {
                    Image(systemName: "checkmark.circle.fill")
                        .foregroundStyle(AppTheme.accent)
                    Text("\(selectedItems.count) items selected")
                        .font(.system(.subheadline, design: .rounded, weight: .medium))
                        .foregroundStyle(AppTheme.textPrimary)
                }
                Spacer()
                AnimatedNumberText(
                    value: selectedItems.reduce(0) { $0 + (quantities[$1] ?? 0) },
                    font: .system(.subheadline, design: .rounded, weight: .bold),
                    color: AppTheme.accent
                )
                Text("units")
                    .font(.system(.caption, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
            }
            .padding(AppTheme.spacingLG)
        }
    }

    // MARK: - Action Bar

    private var actionBar: some View {
        VStack(spacing: 0) {
            Rectangle().fill(AppTheme.separator.opacity(0.3)).frame(height: AppTheme.separatorHeight)
            HStack(spacing: AppTheme.spacingMD) {
                LabButton("Create Order", icon: "cart.badge.plus") {
                    Task { await createOrder() }
                }
                .opacity(isSubmitting ? 0.5 : 1).disabled(isSubmitting)

                LabButton("Add to Cart", variant: .secondary, icon: "cart") {
                    addToCart()
                }
            }
            .padding(AppTheme.spacingLG)
            .background(.ultraThinMaterial)
        }
        .transition(.move(edge: .bottom).combined(with: .opacity))
    }

    // MARK: - Actions

    private func addToCart() {
        for forecast in forecasts where selectedItems.contains(forecast.id) {
            let qty = quantities[forecast.id] ?? forecast.predictedQuantity
            if let product = products.first(where: { $0.id == forecast.productId }),
               let variant = product.defaultVariant {
                cart.add(product: product, variant: variant, quantity: qty)
            }
        }
        Haptics.success()
        withAnimation(AnimationConstants.fluid) { selectedItems.removeAll() }
    }

    private func createOrder() async {
        isSubmitting = true
        let rid = AuthManager.shared.currentUser?.id ?? ""
        do {
            let orderItems = forecasts.filter { selectedItems.contains($0.id) }.map {
                ProcurementOrderRequest.Item(productId: $0.productId, quantity: quantities[$0.id] ?? $0.predictedQuantity)
            }
            let body = ProcurementOrderRequest(retailerId: rid, items: orderItems)
            let _: ProcurementOrderResponse = try await api.post(path: "/v1/order/create", body: body)
            isSubmitting = false
            showSuccess = true
        } catch {
            isSubmitting = false
            errorMessage = "Failed to submit order. Please try again."
            showError = true
        }
    }

    private func loadPredictions() async {
        let rid = AuthManager.shared.currentUser?.id ?? ""
        isLoading = true
        do { let r: [DemandForecast] = try await api.get(path: "/v1/ai/predictions?retailer_id=\(rid)"); forecasts = r }
        catch { forecasts = [] }
        do { let p: [Product] = try await api.get(path: "/v1/products"); products = p }
        catch { products = [] }
        isLoading = false
    }
}

private struct ProcurementOrderRequest: Codable {
    let retailerId: String
    let items: [Item]
    struct Item: Codable {
        let productId: String
        let quantity: Int
        enum CodingKeys: String, CodingKey { case productId = "product_id"; case quantity }
    }
    enum CodingKeys: String, CodingKey { case retailerId = "retailer_id"; case items }
}

private struct ProcurementOrderResponse: Codable {
    let status: String
    let orderId: String
    let total: Int64?
    enum CodingKeys: String, CodingKey {
        case status
        case orderId = "order_id"
        case total = "total"
    }
}

#Preview {
    NavigationStack { ProcurementView().environment(CartManager()) }
}
