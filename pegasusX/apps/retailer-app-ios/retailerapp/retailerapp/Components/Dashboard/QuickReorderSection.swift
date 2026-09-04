import SwiftUI

struct QuickReorderSection: View {
    let reorderProducts: [Product]

    var body: some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
            RetailerSectionHeader(title: "Quick Reorder", icon: "arrow.clockwise")

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

struct QuickReorderItem: View {
    let product: Product
    @Environment(CartManager.self) private var cart
    
    var body: some View {
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
}
