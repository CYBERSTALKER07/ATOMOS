import SwiftUI

struct ProductCardView: View {
    let product: Product
    var onTap: (() -> Void)?

    @State private var isPressed = false
    @State private var appeared = false

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            // Product Image
            ZStack(alignment: .topTrailing) {
                AsyncImage(url: URL(string: product.imageURL ?? "")) { phase in
                    switch phase {
                    case .success(let image):
                        image
                            .resizable()
                            .aspectRatio(contentMode: .fill)
                    default:
                        productPlaceholder
                    }
                }
                .frame(height: 140)
                .clipShape(UnevenRoundedRectangle(
                    topLeadingRadius: AppTheme.radiusCard,
                    topTrailingRadius: AppTheme.radiusCard
                ))

                // Price tag
                if product.displayPrice != "—" {
                    Text(product.displayPrice)
                        .font(.system(.caption, design: .rounded, weight: .heavy))
                        .foregroundStyle(.white)
                        .padding(.horizontal, 10)
                        .padding(.vertical, 5)
                        .background(AppTheme.accentGradient)
                        .clipShape(.capsule)
                        .padding(AppTheme.spacingSM)
                        .shadow(color: AppTheme.accent.opacity(0.3), radius: 4, y: 2)
                }
            }

            // Product Info
            VStack(alignment: .leading, spacing: AppTheme.spacingSM) {
                Text(product.name)
                    .font(.system(.subheadline, design: .rounded, weight: .semibold))
                    .foregroundStyle(AppTheme.textPrimary)
                    .lineLimit(1)

                Text(product.description)
                    .font(.caption)
                    .foregroundStyle(AppTheme.textTertiary)
                    .lineLimit(2)
                    .frame(minHeight: 30, alignment: .top)

                // Variant tags
                if let variant = product.defaultVariant {
                    HStack(spacing: 6) {
                        tagPill(variant.size, color: AppTheme.accentSoft, textColor: AppTheme.accent)

                        if variant.packCount > 1 {
                            tagPill(variant.pack, color: AppTheme.successSoft, textColor: AppTheme.success)
                        }
                    }
                } else if let merchandisingLabel = product.merchandisingLabel {
                    tagPill(merchandisingLabel, color: AppTheme.accentSoft, textColor: AppTheme.accent)
                }
            }
            .padding(.horizontal, AppTheme.spacingMD)
            .padding(.vertical, AppTheme.spacingMD)
        }
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
        .shadow(color: AppTheme.shadowColor, radius: isPressed ? 2 : AppTheme.shadowRadius, x: 0, y: isPressed ? 1 : AppTheme.shadowOffsetY)
        .scaleEffect(isPressed ? 0.96 : 1.0)
        .animation(AnimationConstants.express, value: isPressed)
        .onTapGesture {
            Haptics.light()
            onTap?()
        }
        .onLongPressGesture(minimumDuration: .infinity, pressing: { pressing in
            isPressed = pressing
        }, perform: {})
        .opacity(appeared ? 1 : 0)
        .offset(y: appeared ? 0 : 20)
        .onAppear {
            withAnimation(AnimationConstants.fluid) {
                appeared = true
            }
        }
    }

    private var productPlaceholder: some View {
        ZStack {
            AppTheme.accentSoft.opacity(0.2)
            VStack(spacing: 6) {
                Image(systemName: "leaf.fill")
                    .font(.system(size: 28))
                    .foregroundStyle(AppTheme.accent.opacity(0.4))
                Text(String(product.name.prefix(1)))
                    .font(.system(.title, design: .rounded, weight: .bold))
                    .foregroundStyle(AppTheme.accent.opacity(0.3))
            }
        }
    }

    private func tagPill(_ text: String, color: Color, textColor: Color) -> some View {
        Text(text)
            .font(.system(size: 10, weight: .semibold, design: .rounded))
            .foregroundStyle(textColor)
            .padding(.horizontal, 8)
            .padding(.vertical, 3)
            .background(color.opacity(0.6))
            .clipShape(.capsule)
    }
}

#Preview {
    ScrollView {
        LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 16) {
            ForEach(Product.samples) { product in
                ProductCardView(product: product)
            }
        }
        .padding()
    }
}
