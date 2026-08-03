import SwiftUI


struct CatalogView: View {
    @Environment(CartManager.self) private var cart
    @State private var refreshCenter = RetailerRefreshCenter.shared
    @State private var searchText = ""
    @State private var browseMode: CatalogBrowseMode = .categories
    @State private var categories: [ProductCategory] = []
    @State private var products: [Product] = []
    @State private var selectedProduct: Product?
    @State private var isLoading = false
    @State private var loadError: String?
    @State private var showFullSearch = false

    var onNavigateToSuppliers: () -> Void = {}

    private let api = APIClient.shared

    private var isSearching: Bool {
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

    var body: some View {
        VStack(spacing: 0) {
            // Search Bar
            CatalogSearchBar(searchText: $searchText, showFullSearch: $showFullSearch)
                .padding(.horizontal, AppTheme.spacingLG)
                .padding(.vertical, AppTheme.spacingSM)

            if !isSearching {
                CatalogBrowseChips(browseMode: $browseMode, onNavigateToSuppliers: onNavigateToSuppliers)
                    .padding(.horizontal, AppTheme.spacingLG)
                    .padding(.bottom, AppTheme.spacingSM)
            }

            Rectangle().fill(AppTheme.separator.opacity(0.3)).frame(height: AppTheme.separatorHeight)

            ScrollView {
                if isLoading && categories.isEmpty && products.isEmpty {
                    RetailerLoadingView(
                        title: "Loading catalog",
                        message: "Fetching categories and product listings."
                    )
                } else if let loadError, categories.isEmpty && products.isEmpty {
                    RetailerErrorView(message: loadError) {
                        Task { await loadCategories(); await loadProducts() }
                    }
                } else if isLoading {
                    SkeletonProductGrid()
                } else if isSearching {
                    if filteredProducts.isEmpty {
                        CatalogNoResultsState(searchText: searchText)
                    } else {
                        CatalogSearchResults(filteredProducts: filteredProducts, selectedProduct: $selectedProduct)
                    }
                } else if browseMode == .allProducts {
                    CatalogAllProductsGrid(products: products, selectedProduct: $selectedProduct)
                } else {
                    CatalogBentoGrid(categories: categories)
                }
            }
            .scrollIndicators(.hidden)
        }
        .background(AppTheme.background)
        .navigationDestination(item: $selectedProduct) { product in
            ProductDetailView(product: product)
        }
        .task {
            await loadCategories()
            await loadProducts()
        }
        .task(id: refreshCenter.refreshToken) {
            await loadCategories()
            await loadProducts()
        }
        .refreshable {
            await loadCategories()
            await loadProducts()
        }
        .sheet(isPresented: $showFullSearch) {
            NavigationStack {
                SearchView()
                    .toolbar {
                        ToolbarItem(placement: .cancellationAction) {
                            Button("Close") { showFullSearch = false }
                        }
                    }
            }
        }
    }

    private func loadCategories() async {
        do {
            let result: [ProductCategory] = try await api.get(path: "/v1/catalog/categories")
            categories = result
        } catch {
            categories = []
            loadError = "Could not load categories. Check your connection and retry."
        }
    }

    private func loadProducts() async {
        isLoading = true
        do {
            let result: [Product] = try await api.get(path: "/v1/catalog/products")
            products = result
            loadError = nil
        } catch {
            products = []
            loadError = "Could not load products. Check your connection and retry."
        }
        isLoading = false
    }
}

#Preview {
    NavigationStack { CatalogView().environment(CartManager()) }
}
