import SwiftUI


struct CatalogView: View {
    @Environment(CartManager.self) private var cart
    @State private var refreshCenter = RetailerRefreshCenter.shared
    @State private var vm = CatalogViewModel()

    var onNavigateToSuppliers: () -> Void = {}

    private var categories: [ProductCategory] { vm.categories }
    private var products: [Product] { vm.products }
    private var isLoading: Bool { vm.isLoading }
    private var loadError: String? { vm.loadError }
    private var isSearching: Bool { vm.isSearching }
    private var filteredProducts: [Product] { vm.filteredProducts }

    var body: some View {
        @Bindable var vm = vm
        VStack(spacing: 0) {
            // Search Bar
            CatalogSearchBar(searchText: $vm.searchText, showFullSearch: $vm.showFullSearch)
                .padding(.horizontal, AppTheme.spacingLG)
                .padding(.vertical, AppTheme.spacingSM)

            if !isSearching {
                CatalogBrowseChips(browseMode: $vm.browseMode, onNavigateToSuppliers: onNavigateToSuppliers)
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
                        CatalogNoResultsState(searchText: vm.searchText)
                    } else {
                        CatalogSearchResults(filteredProducts: filteredProducts, selectedProduct: $vm.selectedProduct)
                    }
                } else if vm.browseMode == .allProducts {
                    CatalogAllProductsGrid(products: products, selectedProduct: $vm.selectedProduct)
                } else {
                    CatalogBentoGrid(categories: categories)
                }
            }
            .scrollIndicators(.hidden)
        }
        .background(AppTheme.background)
        .navigationDestination(item: $vm.selectedProduct) { product in
            ProductDetailView(product: product)
        }
        .scrollDismissesKeyboard(.interactively)
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
        .sheet(isPresented: $vm.showFullSearch) {
            NavigationStack {
                SearchView()
                    .toolbar {
                        ToolbarItem(placement: .cancellationAction) {
                            Button("common.action.close") { vm.showFullSearch = false }
                        }
                    }
            }
        }
    }

    private func loadCategories() async {
        await vm.loadCategories()
    }

    private func loadProducts() async {
        await vm.loadProducts()
    }
}

#Preview {
    NavigationStack { CatalogView().environment(CartManager()) }
}
