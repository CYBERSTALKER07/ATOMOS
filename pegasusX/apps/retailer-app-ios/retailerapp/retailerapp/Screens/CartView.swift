import SwiftUI

struct CartView: View {
    @Environment(CartManager.self) private var cart
    @Environment(\.dismiss) private var dismiss

    @State private var showCheckout = false

    var body: some View {
        VStack(spacing: 0) {
            if cart.isEmpty {
                emptyCartView
            } else {
                ScrollView {
                    VStack(spacing: AppTheme.spacingMD) {
                        HStack {
                            Text(L10n.format("mobile_retailer.ui.totalitems_items_in_your_cart", "\(cart.totalItems)"))
                                .font(.system(.subheadline, design: .rounded))
                                .foregroundStyle(AppTheme.textSecondary)
                            Spacer()
                            Button {
                                Haptics.medium()
                                withAnimation(AnimationConstants.fluid) { cart.clear() }
                            } label: {
                                Text("mobile_retailer.ui.clear_all")
                                    .font(.system(.caption, design: .rounded, weight: .semibold))
                                    .foregroundStyle(AppTheme.destructive)
                                    .pegasusMinimumHitTarget()
                            }
                            .accessibilityLabel("Clear all items from cart")
                        }
                        .padding(.horizontal, AppTheme.spacingLG)
                        .padding(.top, AppTheme.spacingSM)

                        ForEach(Array(cart.items.enumerated()), id: \.element.id) { index, item in
                            CartItemRowView(
                                item: item,
                                onUpdateQuantity: { newVal in
                                    Haptics.selection()
                                    withAnimation(AnimationConstants.express) {
                                        cart.updateQuantity(itemId: item.id, quantity: newVal)
                                    }
                                },
                                onRemove: {
                                    Haptics.medium()
                                    withAnimation(AnimationConstants.fluid) {
                                        cart.remove(itemId: item.id)
                                    }
                                }
                            )
                            .staggeredSlideIn(index: index)
                        }
                    }
                    .padding(.bottom, 140)
                }
                .scrollDismissesKeyboard(.interactively)
                .scrollIndicators(.hidden)

                cartBottomBar
            }
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(AppTheme.background.ignoresSafeArea())
        .fullScreenCover(isPresented: $showCheckout) {
            NavigationStack {
                CheckoutView(supplierIsActive: cart.supplierIsActive)
                    .environment(cart)
            }
        }
        .task {
            await cart.hydrateFromServer()
        }
        .task {
            for await event in RetailerWebSocket.shared.eventStream() {
                if case .cartSyncUpdated = event {
                    await cart.hydrateFromServer()
                }
            }
        }
    }

    // MARK: - Bottom Bar

    private var cartBottomBar: some View {
        VStack(spacing: 0) {
            Rectangle().fill(AppTheme.separator.opacity(0.3)).frame(height: AppTheme.separatorHeight)

            VStack(spacing: AppTheme.spacingMD) {
                HStack {
                    Text("mobile_retailer.ui.subtotal")
                        .font(.system(.subheadline, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)
                    Spacer()
                    Text(cart.displayTotal)
                        .font(.system(.subheadline, design: .rounded, weight: .medium))
                        .foregroundStyle(AppTheme.textPrimary)
                        .contentTransition(.numericText())
                }
                .accessibilityElement(children: .combine)
                .accessibilityLabel("Subtotal \(cart.displayTotal)")

                HStack {
                    Text("mobile_retailer.ui.delivery")
                        .font(.system(.subheadline, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)
                    Spacer()
                    Text("mobile_retailer.ui.free")
                        .font(.system(.subheadline, design: .rounded, weight: .medium))
                        .foregroundStyle(AppTheme.success)
                }
                .accessibilityElement(children: .combine)
                .accessibilityLabel("Delivery Free")

                Rectangle().fill(AppTheme.separator.opacity(0.3)).frame(height: AppTheme.separatorHeight)

                HStack {
                    VStack(alignment: .leading, spacing: 2) {
                        Text("retailer_desktop.pos.text.total")
                            .font(.system(.caption, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                        Text(cart.displayTotal)
                            .font(.system(.title3, design: .rounded, weight: .bold))
                            .foregroundStyle(AppTheme.textPrimary)
                            .contentTransition(.numericText())
                    }
                    .accessibilityElement(children: .combine)
                    .accessibilityLabel("Total \(cart.displayTotal)")

                    Spacer()

                    Button {
                        Haptics.medium()
                        showCheckout = true
                    } label: {
                        HStack(spacing: AppTheme.spacingSM) {
                            Text("mobile_retailer.ui.checkout")
                                .font(.system(.subheadline, design: .rounded, weight: .bold))
                            Image(systemName: "arrow.right")
                                .font(.footnote.weight(.bold))
                        }
                        .foregroundStyle(.white)
                        .padding(.horizontal, AppTheme.spacingXL)
                        .padding(.vertical, AppTheme.spacingMD)
                        .background(AppTheme.accent)
                        .clipShape(.capsule)
                        .pegasusMinimumHitTarget()
                    }
                    .buttonStyle(.pressable)
                    .accessibilityLabel("Proceed to checkout")
                }
            }
            .padding(AppTheme.spacingLG)
            .background(.ultraThinMaterial)
        }
        .onChange(of: cart.items.count) {
            Task { await cart.sync() }
        }
        .onChange(of: cart.totalItems) {
            Task { await cart.sync() }
        }
        .task {
            for await event in RetailerWebSocket.shared.eventStream() {
                if case .promotionChanged = event {
                    await cart.refreshCheckoutQuote()
                }
            }
        }
    }

    // MARK: - Empty Cart

    private var emptyCartView: some View {
        VStack(spacing: AppTheme.spacingXL) {
            Spacer()

            ZStack {
                Circle()
                    .fill(AppTheme.surfaceElevated)
                    .frame(width: 100, height: 100)
                Circle()
                    .fill(AppTheme.surfaceElevated.opacity(0.5))
                    .frame(width: 130, height: 130)
                Image(systemName: "cart")
                    .font(.largeTitle)
                    .foregroundStyle(AppTheme.textTertiary)
            }
            .accessibilityHidden(true)

            VStack(spacing: AppTheme.spacingSM) {
                Text("mobile_retailer.ui.your_cart_is_empty")
                    .font(.system(.title3, design: .rounded, weight: .bold))
                    .foregroundStyle(AppTheme.textPrimary)

                Text("mobile_retailer.ui.browse_the_catalog_and_add_nproducts_to_get_started")
                    .font(.system(.subheadline, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
                    .multilineTextAlignment(.center)
            }
            .accessibilityElement(children: .combine)

            Button {
                Haptics.light()
                dismiss()
            } label: {
                HStack(spacing: AppTheme.spacingSM) {
                    Image(systemName: "square.grid.2x2")
                        .font(.subheadline.weight(.semibold))
                    Text("mobile_retailer.ui.browse_catalog")
                        .font(.system(.subheadline, design: .rounded, weight: .semibold))
                }
                .foregroundStyle(.white)
                .padding(.horizontal, AppTheme.spacingXL)
                .padding(.vertical, AppTheme.spacingMD)
                .background(AppTheme.accent)
                .clipShape(.capsule)
                .pegasusMinimumHitTarget()
            }
            .buttonStyle(.pressable)
            .accessibilityLabel("Browse catalog")

            Spacer()
        }
        .padding(AppTheme.spacingXL)
    }
}

private struct CartItemRowView: View {
    let item: CartItem
    let onUpdateQuantity: (Int) -> Void
    let onRemove: () -> Void

    @State private var quantity: Int

    init(
        item: CartItem,
        onUpdateQuantity: @escaping (Int) -> Void,
        onRemove: @escaping () -> Void
    ) {
        self.item = item
        self.onUpdateQuantity = onUpdateQuantity
        self.onRemove = onRemove
        self._quantity = State(initialValue: item.quantity)
    }

    var body: some View {
        HStack(spacing: AppTheme.spacingMD) {
            ZStack {
                RoundedRectangle(cornerRadius: AppTheme.radiusMD)
                    .fill(AppTheme.surfaceElevated)
                    .frame(width: 72, height: 72)
                Image(systemName: "leaf.fill")
                    .font(.title2)
                    .foregroundStyle(AppTheme.textTertiary)
            }
            .accessibilityHidden(true)

            VStack(alignment: .leading, spacing: 4) {
                Text(item.product.name)
                    .font(.system(.subheadline, design: .rounded, weight: .semibold))
                    .foregroundStyle(AppTheme.textPrimary)
                    .lineLimit(2)

                HStack(spacing: 6) {
                    Text(item.variant.size)
                        .font(.system(.caption2, design: .rounded, weight: .medium))
                        .foregroundStyle(AppTheme.textTertiary)
                        .padding(.horizontal, 6).padding(.vertical, 2)
                        .background(AppTheme.surfaceElevated)
                        .clipShape(.capsule)

                    Text(item.variant.pack)
                        .font(.system(.caption2, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)
                }

                Text(L10n.format("mobile_retailer.ui.formatted_each", "\(item.variant.priceMinor.formatted())"))
                    .font(.system(.caption, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
            }
            .accessibilityElement(children: .combine)
            .accessibilityLabel("\(item.product.name), \(item.variant.size), \(item.variant.pack), \(item.variant.priceMinor.formatted()) each")

            Spacer()

            VStack(alignment: .trailing, spacing: AppTheme.spacingSM) {
                HStack(spacing: 8) {
                    Text("\(item.totalPriceMinor.formatted())")
                        .font(.system(.subheadline, design: .rounded, weight: .bold))
                        .foregroundStyle(AppTheme.textPrimary)
                        .contentTransition(.numericText())

                    Button(action: onRemove) {
                        Image(systemName: "xmark.circle.fill")
                            .font(.title3)
                            .foregroundStyle(AppTheme.textTertiary)
                            .pegasusMinimumHitTarget()
                    }
                    .accessibilityLabel("Remove \(item.product.name)")
                }

                QuantityStepper(
                    quantity: $quantity,
                    compact: true
                )
                .onChange(of: quantity) { _, newVal in
                    guard newVal != item.quantity else { return }
                    onUpdateQuantity(newVal)
                }
                .onChange(of: item.quantity) { _, externalVal in
                    if quantity != externalVal {
                        quantity = externalVal
                    }
                }
            }
        }
        .padding(AppTheme.spacingMD)
        .background(AppTheme.cardBackground)
        .clipShape(RoundedRectangle(cornerRadius: AppTheme.radiusCard))
        .shadow(color: AppTheme.shadowColor, radius: 4, x: 0, y: 2)
        .padding(.horizontal, AppTheme.spacingLG)
    }
}

#Preview {
    CartView()
        .environment(CartManager())
}
