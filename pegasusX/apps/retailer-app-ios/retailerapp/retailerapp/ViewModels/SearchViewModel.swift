import Foundation
import Observation

@Observable
@MainActor
final class SearchViewModel {
    var searchText = ""
    var products: [Product] = []
    var selectedProduct: Product?
    var isLoading = false

    private let api = APIClient.shared

    var filteredProducts: [Product] {
        let query = searchText.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !query.isEmpty else { return [] }
        return products.filter {
            $0.name.localizedCaseInsensitiveContains(query) ||
            $0.description.localizedCaseInsensitiveContains(query)
        }
    }

    func loadProducts() async {
        isLoading = true
        do {
            let result: [Product] = try await api.get(path: "/v1/catalog/products")
            products = result
        } catch {
            products = []
        }
        isLoading = false
    }

    func clearSearch() {
        Haptics.light()
        searchText = ""
    }
}
