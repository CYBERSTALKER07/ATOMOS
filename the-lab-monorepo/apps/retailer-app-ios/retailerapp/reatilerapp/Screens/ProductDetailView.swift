import SwiftUI

struct ProductDetailView: View {
    let product: Product
    @Environment(CartManager.self) private var cart
    @Environment(\.dismiss) private var dismiss

    @State private var selectedVariant: Variant?
    @State private var quantity: Int = 1
    @State private var addedToCart = false
    @State private var imageScale: Double = 0.95
    @State private var variantAutoOrder: [String: Bool] = [:]
    @State private var variantHasHistory: [String: Bool] = [:]
    @State private var productAutoOrderEnabled: Bool = false
    @State private var productHasHistory: Bool = false
    @State private var pendingVariantTarget: String? // skuId pending history dialog
    @State private var pendingProductEnable: Bool = false

    private let api = APIClient.shared

    var body: some View {
        ScrollView {
            VStack(spacing: 0) {
                // Hero Image
                heroImage.slideIn(delay: 0)

                // Product Info
                VStack(spacing: AppTheme.spacingLG) {
                    productInfo.slideIn(delay: 0.05)
                    variantPicker.slideIn(delay: 0.1)
                    quantitySection.slideIn(delay: 0.15)
                    nutritionSection.slideIn(delay: 0.2)
                    variantAutoOrderSection.slideIn(delay: 0.25)
                }
                .padding(.horizontal, AppTheme.spacingLG)
                .padding(.top, AppTheme.spacingLG)
                .padding(.bottom, 120)
            }
        }
        .scrollIndicators(.hidden)
        .background(AppTheme.background)
        .safeAreaInset(edge: .bottom) { addToCartBar }
        .navigationTitle(product.name)
        .navigationBarTitleDisplayMode(.inline)
        .onAppear {
            selectedVariant = product.defaultVariant
            withAnimation(AnimationConstants.hero) { imageScale = 1.0 }
            Task { await loadAutoOrderState() }
        }
        .sensoryFeedback(.success, trigger: addedToCart)
        .alert("Use Previous Analytics?", isPresented: Binding(
            get: { pendingVariantTarget != nil || pendingProductEnable },
            set: { if !$0 { pendingVariantTarget = nil; pendingProductEnable = false } }
        ), actions: {
            Button("Use History") {
                Task { await confirmProductOrVariantEnable(useHistory: true) }
            }
            Button("Start Fresh", role: .destructive) {
                Task { await confirmProductOrVariantEnable(useHistory: false) }
            }
            Button("Cancel", role: .cancel) {
                if pendingProductEnable { productAutoOrderEnabled = false }
                pendingVariantTarget = nil
                pendingProductEnable = false
            }
        }, message: {
            Text("Enable auto-order using existing history, or start fresh? Starting fresh requires at least 2 orders before predictions begin.")
        })
    }

    // MARK: - Hero

    private var heroImage: some View {
        ZStack {
            AppTheme.accentSoft.opacity(0.15)
            if let urlStr = product.imageURL, let url = URL(string: urlStr) {
                AsyncImage(url: url) { phase in
                    switch phase {
                    case .success(let image):
                        image
                            .resizable()
                            .aspectRatio(contentMode: .fill)
                    default:
                        heroPlaceholder
                    }
                }
            } else {
                heroPlaceholder
            }
        }
        .frame(height: 260)
        .clipped()
        .clipShape(UnevenRoundedRectangle(bottomLeadingRadius: AppTheme.radiusXL, bottomTrailingRadius: AppTheme.radiusXL))
        .scaleEffect(imageScale)
    }

    private var heroPlaceholder: some View {
        VStack(spacing: AppTheme.spacingMD) {
            Image(systemName: "leaf.fill")
                .font(.system(size: 52))
                .foregroundStyle(AppTheme.accent.opacity(0.3))
            Text(product.name)
                .font(.system(.title3, design: .rounded, weight: .bold))
                .foregroundStyle(AppTheme.accent.opacity(0.4))
        }
    }

    // MARK: - Info

    private var productInfo: some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingSM) {
            HStack(alignment: .top) {
                VStack(alignment: .leading, spacing: AppTheme.spacingXS) {
                    Text(product.name)
                        .font(.system(.title2, design: .rounded, weight: .bold))
                        .foregroundStyle(AppTheme.textPrimary)

                    Text(product.description)
                        .font(.system(.body, design: .rounded))
                        .foregroundStyle(AppTheme.textSecondary)
                }

                Spacer()

                if let variant = selectedVariant {
                    VStack(alignment: .trailing, spacing: 2) {
                        Text("\(Int(variant.price).formatted())")
                            .font(.system(.title3, design: .rounded, weight: .bold))
                            .foregroundStyle(AppTheme.accent)
                        if variant.packCount > 1 {
                            Text("/ \(variant.pack)")
                                .font(.system(.caption, design: .rounded))
                                .foregroundStyle(AppTheme.textTertiary)
                        }
                    }
                }
            }
        }
    }

    // MARK: - Variant Picker

    private var variantPicker: some View {
        LabCardWithHeader(title: "Select Variant", icon: "cube") {
            VStack(spacing: AppTheme.spacingSM) {
                ForEach(product.variants) { variant in
                    let isSelected = selectedVariant?.id == variant.id

                    Button {
                        withAnimation(AnimationConstants.express) {
                            selectedVariant = variant
                            quantity = 1
                        }
                        Haptics.light()
                    } label: {
                        HStack(spacing: AppTheme.spacingMD) {
                            ZStack {
                                Circle()
                                    .fill(isSelected ? AppTheme.accent : AppTheme.separator.opacity(0.3))
                                    .frame(width: 22, height: 22)
                                if isSelected {
                                    Circle()
                                        .fill(.white)
                                        .frame(width: 8, height: 8)
                                }
                            }

                            VStack(alignment: .leading, spacing: 2) {
                                Text("\(variant.size) — \(variant.pack)")
                                    .font(.system(.subheadline, design: .rounded, weight: .medium))
                                    .foregroundStyle(AppTheme.textPrimary)
                                Text(variant.weightPerUnit)
                                    .font(.system(.caption, design: .rounded))
                                    .foregroundStyle(AppTheme.textTertiary)
                            }

                            Spacer()

                            Text("\(Int(variant.price).formatted())")
                                .font(.system(.subheadline, design: .rounded, weight: .bold))
                                .foregroundStyle(isSelected ? AppTheme.accent : AppTheme.textSecondary)
                        }
                        .padding(AppTheme.spacingMD)
                        .background(isSelected ? AppTheme.accentSoft.opacity(0.2) : .clear)
                        .clipShape(.rect(cornerRadius: AppTheme.radiusMD))
                        .overlay {
                            if isSelected {
                                RoundedRectangle(cornerRadius: AppTheme.radiusMD)
                                    .strokeBorder(AppTheme.accent.opacity(0.3), lineWidth: 1.5)
                            }
                        }
                    }
                }
            }
        }
    }

    // MARK: - Quantity

    private var quantitySection: some View {
        LabCard {
            HStack {
                VStack(alignment: .leading, spacing: 2) {
                    Text("Quantity")
                        .font(.system(.subheadline, design: .rounded, weight: .semibold))
                        .foregroundStyle(AppTheme.textPrimary)
                    if let v = selectedVariant {
                        Text("Total: \(Int(Double(quantity) * v.price).formatted())")
                            .font(.system(.caption, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                            .contentTransition(.numericText())
                            .animation(.snappy, value: quantity)
                    }
                }
                Spacer()
                QuantityStepper(quantity: $quantity)
            }
            .padding(AppTheme.spacingLG)
        }
    }

    // MARK: - Nutrition

    private var nutritionSection: some View {
        LabCardWithHeader(title: "Nutrition", icon: "leaf") {
            Text(product.nutrition)
                .font(.system(.subheadline, design: .rounded))
                .foregroundStyle(AppTheme.textSecondary)
        }
    }

    // MARK: - Variant Auto-Order

    private var variantAutoOrderSection: some View {
        VStack(spacing: AppTheme.spacingMD) {
            // Product-level toggle
            LabCard {
                HStack(spacing: AppTheme.spacingMD) {
                    ZStack {
                        RoundedRectangle(cornerRadius: AppTheme.radiusSM)
                            .fill(productAutoOrderEnabled ? AppTheme.accent.opacity(0.15) : AppTheme.surfaceElevated)
                            .frame(width: 40, height: 40)
                        Image(systemName: "arrow.triangle.2.circlepath")
                            .font(.system(size: 16, weight: .semibold))
                            .foregroundStyle(productAutoOrderEnabled ? AppTheme.accent : AppTheme.textSecondary)
                    }
                    VStack(alignment: .leading, spacing: 2) {
                        Text("Auto-Order – This Product")
                            .font(.system(.subheadline, design: .rounded, weight: .semibold))
                            .foregroundStyle(AppTheme.textPrimary)
                        Text("Product-level override")
                            .font(.system(.caption, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                    }
                    Spacer()
                    Toggle("", isOn: $productAutoOrderEnabled)
                        .tint(AppTheme.accent)
                        .labelsHidden()
                        .onChange(of: productAutoOrderEnabled) { _, newVal in
                            if newVal && productHasHistory {
                                pendingProductEnable = true
                            } else {
                                Task { await patchProductAutoOrder(enabled: newVal, useHistory: false) }
                            }
                        }
                }
                .padding(AppTheme.spacingLG)
            }

            // Variant-level toggles
            LabCardWithHeader(title: "Auto-Order per Variant", icon: "cube") {
                VStack(spacing: AppTheme.spacingSM) {
                    ForEach(product.variants) { variant in
                        HStack(spacing: AppTheme.spacingMD) {
                            VStack(alignment: .leading, spacing: 2) {
                                Text("\(variant.size) — \(variant.pack)")
                                    .font(.system(.subheadline, design: .rounded, weight: .medium))
                                    .foregroundStyle(AppTheme.textPrimary)
                                Text(variant.weightPerUnit)
                                    .font(.system(.caption2, design: .rounded))
                                    .foregroundStyle(AppTheme.textTertiary)
                            }

                            Spacer()

                            Toggle("", isOn: Binding(
                                get: { variantAutoOrder[variant.id] ?? false },
                                set: { newVal in
                                    if newVal && (variantHasHistory[variant.id] ?? false) {
                                        pendingVariantTarget = variant.id
                                    } else {
                                        variantAutoOrder[variant.id] = newVal
                                        Task { await toggleVariantAutoOrder(skuId: variant.id, enabled: newVal, useHistory: false) }
                                    }
                                }
                            ))
                            .tint(AppTheme.accent)
                            .labelsHidden()
                            .scaleEffect(0.8)
                        }
                        .padding(.vertical, 2)

                        if variant.id != product.variants.last?.id {
                            Rectangle()
                                .fill(AppTheme.separator.opacity(0.15))
                                .frame(height: AppTheme.separatorHeight)
                        }
                    }
                }
            }
        }
    }

    private func toggleVariantAutoOrder(skuId: String, enabled: Bool, useHistory: Bool) async {
        do {
            var body: [String: Any] = ["auto_order_enabled": enabled]
            if enabled { body["use_history"] = useHistory }
            let _: [String: Bool] = try await api.patch(
                path: "/v1/retailer/settings/auto-order/variant/\(skuId)",
                body: AnyCodable(body)
            )
        } catch {
            variantAutoOrder[skuId] = !enabled // revert on failure
        }
    }

    private func patchProductAutoOrder(enabled: Bool, useHistory: Bool) async {
        do {
            var body: [String: Any] = ["auto_order_enabled": enabled]
            if enabled { body["use_history"] = useHistory }
            let _: [String: Bool] = try await api.patch(
                path: "/v1/retailer/settings/auto-order/product/\(product.id)",
                body: AnyCodable(body)
            )
        } catch {
            productAutoOrderEnabled = !enabled
        }
    }

    private func loadAutoOrderState() async {
        do {
            let s: AutoOrderSettings = try await api.get(path: "/v1/retailer/settings/auto-order")
            // product-level
            if let pov = s.productOverrides.first(where: { $0.productID == product.id }) {
                productAutoOrderEnabled = pov.enabled
                productHasHistory = pov.hasHistory
            }
            // variant-level
            for variant in product.variants {
                if let vov = s.variantOverrides.first(where: { $0.skuID == variant.id }) {
                    variantAutoOrder[variant.id] = vov.enabled
                    variantHasHistory[variant.id] = vov.hasHistory
                }
            }
        } catch {}
    }

    private func confirmProductOrVariantEnable(useHistory: Bool) async {
        if let skuId = pendingVariantTarget {
            pendingVariantTarget = nil
            variantAutoOrder[skuId] = true
            await toggleVariantAutoOrder(skuId: skuId, enabled: true, useHistory: useHistory)
        } else if pendingProductEnable {
            pendingProductEnable = false
            await patchProductAutoOrder(enabled: true, useHistory: useHistory)
        }
    }

    // MARK: - Bottom Bar

    private var addToCartBar: some View {
        VStack(spacing: 0) {
            Rectangle().fill(AppTheme.separator.opacity(0.3)).frame(height: AppTheme.separatorHeight)

            HStack(spacing: AppTheme.spacingLG) {
                VStack(alignment: .leading, spacing: 2) {
                    Text("Total")
                        .font(.system(.caption, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)
                    if let variant = selectedVariant {
                        AnimatedCurrencyText(value: Double(quantity) * variant.price, font: .system(.title3, design: .rounded, weight: .bold))
                    }
                }

                Spacer()

                LabButton("Add to Cart", icon: "cart.badge.plus") {
                    guard let variant = selectedVariant else { return }
                    cart.add(product: product, variant: variant, quantity: quantity)
                    addedToCart.toggle()
                    quantity = 1
                }
            }
            .padding(AppTheme.spacingLG)
            .background(.ultraThinMaterial)
        }
    }
}

#Preview {
    NavigationStack {
        ProductDetailView(product: Product.samples[0])
            .environment(CartManager())
    }
}
