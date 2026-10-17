import SwiftUI

struct CatalogSearchResults: View {
    let filteredProducts: [Product]
    @Binding var selectedProduct: Product?

    var body: some View {
        let cols = [GridItem(.adaptive(minimum: 160), spacing: 14)]
        return LazyVGrid(columns: cols, spacing: AppTheme.spacingLG) {
            ForEach(Array(filteredProducts.enumerated()), id: \.element.id) { index, product in
                ProductCardView(product: product) { selectedProduct = product }
                    .staggeredSlideIn(index: index)
            }
        }
        .padding(AppTheme.spacingLG)
        .padding(.bottom, AppTheme.spacingXXL)
    }
}
