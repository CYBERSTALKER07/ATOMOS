import SwiftUI

struct CatalogNoResultsState: View {
    let searchText: String

    var body: some View {
        RetailerEmptyView(
            title: "No Results",
            message: "No products match \"\(searchText)\"",
            systemImage: "magnifyingglass"
        )
        .padding(AppTheme.spacingXL)
    }
}
