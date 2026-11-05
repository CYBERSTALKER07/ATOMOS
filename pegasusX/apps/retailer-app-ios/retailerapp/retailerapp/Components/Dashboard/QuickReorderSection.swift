import SwiftUI

struct QuickReorderSection: View {
    let reorderProducts: [Product]

    var body: some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
            RetailerSectionHeader(title: "Quick Reorder", icon: "arrow.clockwise")

            if reorderProducts.isEmpty {
                HStack(spacing: AppTheme.spacingMD) {
                    Image(systemName: "cart.badge.plus")
                        .font(.system(size: 20))
                        .foregroundStyle(AppTheme.accentWarm)
                    VStack(alignment: .leading, spacing: 2) {
                        Text("No reorders yet")
                            .font(.system(.subheadline, design: .rounded, weight: .semibold))
                            .foregroundStyle(AppTheme.textPrimary)
                        Text("Past orders and popular replenishment items appear here.")
                            .font(.caption)
                            .foregroundStyle(AppTheme.textSecondary)
                    }
                    Spacer()
                    NavigationLink {
                        CatalogView()
                    } label: {
                        Text("Browse")
                            .font(.caption.bold())
                            .foregroundStyle(AppTheme.cardBackground)
                            .padding(.horizontal, 12)
                            .padding(.vertical, 6)
                            .background(AppTheme.accent)
                            .clipShape(Capsule())
                    }
                    .buttonStyle(.pressable)
                }
                .padding(AppTheme.spacingMD)
                .background(AppTheme.cardBackground)
                .clipShape(RoundedRectangle(cornerRadius: AppTheme.radiusCard, style: .continuous))
            } else {
                ScrollView(.horizontal) {
                    HStack(spacing: AppTheme.spacingMD) {
                        ForEach(Array(reorderProducts.prefix(6).enumerated()), id: \.element.id) { index, product in
                            QuickReorderItem(product: product)
                                .staggeredSlideIn(index: index)
                        }
                    }
                }
                .scrollIndicators(.hidden)
            }
        }
    }
}

struct QuickReorderItem: View {
    let product: Product
    @Environment(CartManager.self) private var cart
    @State private var addedToast = false
    
    var body: some View {
        ZStack(alignment: .topTrailing) {
            NavigationLink {
                ProductDetailView(product: product)
            } label: {
                VStack(spacing: AppTheme.spacingSM) {
                    ZStack {
                        RoundedRectangle(cornerRadius: AppTheme.radiusMD)
                            .fill(AppTheme.surfaceElevated)
                            .frame(width: 68, height: 68)
                        Image(systemName: "leaf.fill")
                            .font(.system(size: 24))
                            .foregroundStyle(AppTheme.textTertiary)
                    }

                    Text(product.name)
                        .font(.system(.caption2, design: .rounded, weight: .medium))
                        .foregroundStyle(AppTheme.textPrimary)
                        .lineLimit(1)
                        .frame(width: 72)

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
            .buttonStyle(.pressable)

            // Independent 44x44pt Quick Add Button overlay
            Button {
                if let variant = product.defaultVariant {
                    Haptics.light()
                    withAnimation(AnimationConstants.bouncy) {
                        cart.add(product: product, variant: variant)
                        addedToast = true
                    }
                    DispatchQueue.main.asyncAfter(deadline: .now() + 1.2) {
                        addedToast = false
                    }
                }
            } label: {
                Image(systemName: addedToast ? "checkmark.circle.fill" : "plus.circle.fill")
                    .font(.system(size: 20, weight: .semibold))
                    .foregroundStyle(addedToast ? AppTheme.success : AppTheme.accent)
                    .frame(width: 44, height: 44)
                    .contentShape(Rectangle())
            }
            .buttonStyle(.plain)
            .offset(x: 4, y: -4)
        }
    }
}
