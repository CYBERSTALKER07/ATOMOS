import SwiftUI

struct CartView: View {
    @Environment(CartManager.self) private var cart

    @State private var showCheckout = false
    @State private var itemToDelete: CartItem?

    var body: some View {
        VStack(spacing: 0) {
            if cart.isEmpty {
                emptyCartView
            } else {
                ScrollView {
                    VStack(spacing: AppTheme.spacingMD) {
                        // Cart count header
                        HStack {
                            Text("\(cart.totalItems) items in your cart")
                                .font(.system(.subheadline, design: .rounded))
                                .foregroundStyle(AppTheme.textSecondary)
                            Spacer()
                            Button {
                                Haptics.medium()
                                withAnimation(AnimationConstants.fluid) { cart.clear() }
                            } label: {
                                Text("Clear All")
                                    .font(.system(.caption, design: .rounded, weight: .semibold))
                                    .foregroundStyle(AppTheme.destructive)
                            }
                        }
                        .padding(.horizontal, AppTheme.spacingLG)
                        .padding(.top, AppTheme.spacingSM)

                        // Cart Items
                        ForEach(Array(cart.items.enumerated()), id: \.element.id) { index, item in
                            cartItemCard(item)
                                .staggeredSlideIn(index: index)
                        }
                    }
                    .padding(.bottom, 140)
                }
                .scrollIndicators(.hidden)

                // Bottom bar
                cartBottomBar
            }
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(AppTheme.background.ignoresSafeArea())
        .fullScreenCover(isPresented: $showCheckout) {
            CheckoutView(supplierIsActive: cart.supplierIsActive)
                .environment(cart)
        }
    }

    // MARK: - Cart Item Card

    private func cartItemCard(_ item: CartItem) -> some View {
        HStack(spacing: AppTheme.spacingMD) {
            // Product image placeholder
            ZStack {
                RoundedRectangle(cornerRadius: AppTheme.radiusMD)
                    .fill(AppTheme.surfaceElevated)
                    .frame(width: 72, height: 72)
                Image(systemName: "leaf.fill")
                    .font(.system(size: 26))
                    .foregroundStyle(AppTheme.textTertiary)
            }

            // Info
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

                Text("\(Int(item.variant.price).formatted()) each")
                    .font(.system(.caption, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
            }

            Spacer()

            // Quantity + Price column
            VStack(alignment: .trailing, spacing: AppTheme.spacingSM) {
                Text("\(Int(item.totalPrice).formatted())")
                    .font(.system(.subheadline, design: .rounded, weight: .bold))
                    .foregroundStyle(AppTheme.textPrimary)
                    .contentTransition(.numericText())

                QuantityStepper(
                    quantity: Binding(
                        get: { item.quantity },
                        set: { newVal in
                            withAnimation(AnimationConstants.express) {
                                cart.updateQuantity(itemId: item.id, quantity: newVal)
                            }
                        }
                    ),
                    compact: true
                )
            }
        }
        .padding(AppTheme.spacingMD)
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
        .shadow(color: AppTheme.shadowColor, radius: 4, x: 0, y: 2)
        .padding(.horizontal, AppTheme.spacingLG)
        .swipeActions(edge: .trailing) {
            Button(role: .destructive) {
                withAnimation(AnimationConstants.fluid) {
                    cart.remove(itemId: item.id)
                }
            } label: {
                Label("Delete", systemImage: "trash")
            }
        }
        // Manual swipe-to-delete via overlay
        .overlay(alignment: .trailing) {
            deleteButton(for: item)
        }
    }

    private func deleteButton(for item: CartItem) -> some View {
        Button {
            Haptics.medium()
            withAnimation(AnimationConstants.fluid) {
                cart.remove(itemId: item.id)
            }
        } label: {
            Image(systemName: "trash")
                .font(.system(size: 12, weight: .semibold))
                .foregroundStyle(AppTheme.destructive)
                .frame(width: 28, height: 28)
                .background(AppTheme.destructive.opacity(0.1))
                .clipShape(.circle)
        }
        .accessibilityLabel("Remove from cart")
        .padding(.trailing, AppTheme.spacingMD + AppTheme.spacingLG)
        .offset(y: -28)
    }

    // MARK: - Bottom Bar

    private var cartBottomBar: some View {
        VStack(spacing: 0) {
            Rectangle().fill(AppTheme.separator.opacity(0.3)).frame(height: AppTheme.separatorHeight)

            VStack(spacing: AppTheme.spacingMD) {
                // Summary rows
                HStack {
                    Text("Subtotal")
                        .font(.system(.subheadline, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)
                    Spacer()
                    Text(cart.displayTotal)
                        .font(.system(.subheadline, design: .rounded, weight: .medium))
                        .foregroundStyle(AppTheme.textPrimary)
                        .contentTransition(.numericText())
                }

                HStack {
                    Text("Delivery")
                        .font(.system(.subheadline, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)
                    Spacer()
                    Text("Free")
                        .font(.system(.subheadline, design: .rounded, weight: .medium))
                        .foregroundStyle(AppTheme.success)
                }

                Rectangle().fill(AppTheme.separator.opacity(0.3)).frame(height: AppTheme.separatorHeight)

                HStack {
                    VStack(alignment: .leading, spacing: 2) {
                        Text("Total")
                            .font(.system(.caption, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                        Text(cart.displayTotal)
                            .font(.system(.title3, design: .rounded, weight: .bold))
                            .foregroundStyle(AppTheme.textPrimary)
                            .contentTransition(.numericText())
                    }

                    Spacer()

                    Button {
                        Haptics.medium()
                        showCheckout = true
                    } label: {
                        HStack(spacing: AppTheme.spacingSM) {
                            Text("Checkout")
                                .font(.system(.subheadline, design: .rounded, weight: .bold))
                            Image(systemName: "arrow.right")
                                .font(.system(size: 13, weight: .bold))
                        }
                        .foregroundStyle(.white)
                        .padding(.horizontal, AppTheme.spacingXL)
                        .padding(.vertical, AppTheme.spacingMD)
                        .background(AppTheme.accent)
                        .clipShape(.capsule)
                    }
                    .pressable()
                }
            }
            .padding(AppTheme.spacingLG)
            .background(.ultraThinMaterial)
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
                    .font(.system(size: 40))
                    .foregroundStyle(AppTheme.textTertiary)
            }

            VStack(spacing: AppTheme.spacingSM) {
                Text("Your cart is empty")
                    .font(.system(.title3, design: .rounded, weight: .bold))
                    .foregroundStyle(AppTheme.textPrimary)

                Text("Browse the catalog and add\nproducts to get started")
                    .font(.system(.subheadline, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
                    .multilineTextAlignment(.center)
            }

            Button {
                // User can tap the Catalog tab
            } label: {
                HStack(spacing: AppTheme.spacingSM) {
                    Image(systemName: "square.grid.2x2")
                        .font(.system(size: 14, weight: .semibold))
                    Text("Browse Catalog")
                        .font(.system(.subheadline, design: .rounded, weight: .semibold))
                }
                .foregroundStyle(.white)
                .padding(.horizontal, AppTheme.spacingXL)
                .padding(.vertical, AppTheme.spacingMD)
                .background(AppTheme.accent)
                .clipShape(.capsule)
            }
            .pressable()

            Spacer()
        }
        .padding(AppTheme.spacingXL)
    }
}

#Preview {
    CartView()
        .environment(CartManager())
}
