import SwiftUI

struct CatalogAllProductsGrid: View {
    let products: [Product]
    @Binding var selectedProduct: Product?

    var body: some View {
        let cols = [GridItem(.adaptive(minimum: 160), spacing: 14)]
        return VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
            RetailerSectionHeader(
                title: "All products",
                subtitle: "\(products.count) items",
                icon: "bag.fill"
            )
            .padding(.horizontal, AppTheme.spacingLG)
            .padding(.top, AppTheme.spacingMD)

            LazyVGrid(columns: cols, spacing: AppTheme.spacingLG) {
                ForEach(Array(products.enumerated()), id: \.element.id) { index, product in
                    ProductCardView(product: product) { selectedProduct = product }
                        .staggeredSlideIn(index: index)
                }
            }
            .padding(.horizontal, AppTheme.spacingLG)
        }
        .padding(.bottom, AppTheme.spacingHuge)
    }
}
