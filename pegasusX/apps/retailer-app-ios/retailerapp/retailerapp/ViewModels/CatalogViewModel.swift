import Foundation
import SwiftUI
import Observation

@Observable
@MainActor
final class CatalogViewModel {
    var searchText = ""
    var browseMode: CatalogBrowseMode = .categories
    var categories: [ProductCategory] = []
    var products: [Product] = []
    var selectedProduct: Product?
    var isLoading = false
    var loadError: String?
    var showFullSearch = false

    private let api = APIClient.shared

    var isSearching: Bool {
        searchText.trimmingCharacters(in: .whitespacesAndNewlines).count >= 2
    }

    var filteredProducts: [Product] {
        guard isSearching else { return [] }
        let query = searchText.trimmingCharacters(in: .whitespacesAndNewlines)
        return products.filter { product in
            product.name.localizedCaseInsensitiveContains(query) ||
            product.description.localizedCaseInsensitiveContains(query) ||
            (product.supplierName?.localizedCaseInsensitiveContains(query) ?? false) ||
            (product.categoryName?.localizedCaseInsensitiveContains(query) ?? false)
        }
    }

    func loadCategories() async {
        do {
            let result: [ProductCategory] = try await api.get(path: "/v1/catalog/categories")
            categories = result
        } catch {
            categories = []
        }
    }

    func loadProducts() async {
        isLoading = true
        do {
            let result: [Product] = try await api.get(path: "/v1/catalog/products")
            products = result
            loadError = nil
        } catch {
            products = []
            loadError = error.localizedDescription
        }
        isLoading = false
    }
}
