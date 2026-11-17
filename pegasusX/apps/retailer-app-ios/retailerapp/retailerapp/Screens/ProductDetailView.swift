import SwiftUI

struct ProductDetailView: View {
    let product: Product
    @Environment(CartManager.self) private var cart
    @Environment(\.dismiss) private var dismiss

    @State private var vm = ProductDetailViewModel()

    var body: some View {
        @Bindable var vm = vm

        ScrollView {
            VStack(spacing: 0) {
                heroImage.slideIn(delay: 0)

                VStack(spacing: AppTheme.spacingLG) {
                    productInfo.slideIn(delay: 0.05)
                    variantPicker.slideIn(delay: 0.1)
                    quantitySection(quantityBinding: $vm.quantity).slideIn(delay: 0.15)
                    nutritionSection.slideIn(delay: 0.2)
                }
                .padding(.horizontal, AppTheme.spacingLG)
                .padding(.top, AppTheme.spacingLG)
                .padding(.bottom, 120)
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .scrollIndicators(.hidden)
        .background(AppTheme.background)
        .safeAreaInset(edge: .bottom) { addToCartBar }
        .navigationTitle(product.name)
        .navigationBarTitleDisplayMode(.inline)
        .task(id: product.id) {
            vm.configure(for: product)
            withAnimation(AnimationConstants.hero) {
                vm.imageScale = 1.0
            }
            await vm.loadAutoOrderState(for: product)
        }
        .sensoryFeedback(.success, trigger: vm.addedToCart)
        .alert("Use Previous Analytics?", isPresented: $vm.showHistoryAlert, actions: {
            Button("mobile_retailer.ui.use_history") {
                Task { await vm.confirmProductOrVariantEnable(productId: product.id, useHistory: true) }
            }
            Button("mobile_retailer.ui.start_fresh", role: .destructive) {
                Task { await vm.confirmProductOrVariantEnable(productId: product.id, useHistory: false) }
            }
            Button("common.action.cancel", role: .cancel) {
                vm.cancelPendingEnable()
            }
        }, message: {
            Text("mobile_retailer.ui.enable_auto_order_using_existing_history_or_start_fresh_starting")
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
        .clipShape(UnevenRoundedRectangle(bottomLeadingRadius: AppTheme.radiusXXL, bottomTrailingRadius: AppTheme.radiusXXL))
        .scaleEffect(vm.imageScale)
        .accessibilityHidden(true)
    }

    private var heroPlaceholder: some View {
        VStack(spacing: AppTheme.spacingMD) {
            Image(systemName: "leaf.fill")
                .font(.largeTitle)
                .imageScale(.large)
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

                if let variant = vm.selectedVariant {
                    VStack(alignment: .trailing, spacing: 2) {
                        if product.hasSaleOffer, let listPrice = product.displayListPrice {
                            Text(listPrice)
                                .font(.system(.caption, design: .rounded, weight: .medium))
                                .foregroundStyle(AppTheme.textTertiary)
                                .strikethrough()
                        }
                        Text(product.displayPrice)
                            .font(.system(.title3, design: .rounded, weight: .bold))
                            .foregroundStyle(product.hasSaleOffer ? AppTheme.success : AppTheme.accent)
                        if variant.packCount > 1 {
                            Text(L10n.format("mobile_retailer.ui.pack", "\(variant.pack)"))
                                .font(.system(.caption, design: .rounded))
                                .foregroundStyle(AppTheme.textTertiary)
                        }
                    }
                }
            }
        }
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(product.name), \(product.displayPrice). \(product.description)")
    }

    // MARK: - Variant Picker

    private var variantPicker: some View {
        LabCardWithHeader(title: "Select Variant", icon: "cube") {
            VStack(spacing: AppTheme.spacingSM) {
                ForEach(product.variants) { variant in
                    let isSelected = vm.selectedVariant?.id == variant.id

                    Button {
                        vm.selectVariant(variant)
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
                                Text(L10n.format("mobile_retailer.ui.size_pack", "\(variant.size)", "\(variant.pack)"))
                                    .font(.system(.subheadline, design: .rounded, weight: .medium))
                                    .foregroundStyle(AppTheme.textPrimary)
                                Text(variant.weightPerUnit)
                                    .font(.system(.caption, design: .rounded))
                                    .foregroundStyle(AppTheme.textTertiary)
                            }

                            Spacer()

                            Text("\(variant.priceMinor.formatted())")
                                .font(.system(.subheadline, design: .rounded, weight: .bold))
                                .foregroundStyle(isSelected ? AppTheme.accent : AppTheme.textSecondary)
                        }
                        .padding(AppTheme.spacingMD)
                        .pegasusMinimumHitTarget()
                        .background(isSelected ? AppTheme.accentSoft.opacity(0.2) : .clear)
                        .clipShape(.rect(cornerRadius: AppTheme.radiusMD))
                        .overlay {
                            if isSelected {
                                RoundedRectangle(cornerRadius: AppTheme.radiusMD)
                                    .strokeBorder(AppTheme.accent.opacity(0.3), lineWidth: 1.5)
                            }
                        }
                    }
                    .buttonStyle(.plain)
                    .accessibilityElement(children: .combine)
                    .accessibilityLabel("Variant \(variant.size), \(variant.pack), \(variant.priceMinor.formatted())\(isSelected ? ", selected" : "")")
                }
            }
        }
    }

    // MARK: - Quantity

    private func quantitySection(quantityBinding: Binding<Int>) -> some View {
        LabCard {
            HStack {
                VStack(alignment: .leading, spacing: 2) {
                    Text("warehouse_portal.inventory.inventory_stock_list.text.quantity")
                        .font(.system(.subheadline, design: .rounded, weight: .semibold))
                        .foregroundStyle(AppTheme.textPrimary)
                    if let v = vm.selectedVariant {
                        Text(L10n.format("mobile_retailer.ui.total_formatted", "\((Int64(vm.quantity) * v.priceMinor).formatted())"))
                            .font(.system(.caption, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                            .contentTransition(.numericText())
                            .animation(.snappy, value: vm.quantity)
                    }
                }
                .accessibilityElement(children: .combine)

                Spacer()
                QuantityStepper(quantity: quantityBinding)
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
        .accessibilityElement(children: .combine)
    }

    // MARK: - Bottom Bar

    private var addToCartBar: some View {
        VStack(spacing: 0) {
            Rectangle().fill(AppTheme.separator.opacity(0.3)).frame(height: AppTheme.separatorHeight)

            HStack(spacing: AppTheme.spacingLG) {
                VStack(alignment: .leading, spacing: 2) {
                    Text("retailer_desktop.pos.text.total")
                        .font(.system(.caption, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)
                    if let variant = vm.selectedVariant {
                        AnimatedCurrencyText(
                            value: Int64(vm.quantity) * variant.priceMinor,
                            font: .system(.title3, design: .rounded, weight: .bold)
                        )
                    }
                }
                .accessibilityElement(children: .combine)

                Spacer()

                LabButton("Add to Cart", icon: "cart.badge.plus") {
                    vm.addToCart(product: product, cart: cart)
                }
                .disabled(product.blocksAddToCart || vm.selectedVariant == nil)
                .accessibilityLabel("Add \(vm.quantity) \(product.name) to cart")
            }
            .padding(AppTheme.spacingLG)
            .background(.ultraThinMaterial)
        }
    }
}

#Preview {
    NavigationStack {
        ProductDetailView(product: Product(
            id: "preview-1",
            name: "Organic Whole Milk",
            description: "Fresh milk",
            nutrition: "",
            imageURL: nil,
            variants: [Variant(id: "v-1", size: "1L", pack: "Single", packCount: 1, weightPerUnit: "1kg", priceMinor: 12_500)]
        ))
        .environment(CartManager())
    }
}
