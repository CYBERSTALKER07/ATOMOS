import SwiftUI

struct CatalogView: View {
    @Environment(CartManager.self) private var cart
    @State private var searchText = ""
    @State private var categories: [ProductCategory] = []
    @State private var products: [Product] = []
    @State private var selectedProduct: Product?
    @State private var isLoading = false
    @State private var loadError = false

    private let api = APIClient.shared

    var filteredProducts: [Product] {
        guard !searchText.isEmpty else { return [] }
        return products.filter {
            $0.name.localizedCaseInsensitiveContains(searchText) ||
            $0.description.localizedCaseInsensitiveContains(searchText)
        }
    }

    var body: some View {
        VStack(spacing: 0) {
            // Search Bar
            searchBar
                .padding(.horizontal, AppTheme.spacingLG)
                .padding(.vertical, AppTheme.spacingSM)

            Rectangle().fill(AppTheme.separator.opacity(0.3)).frame(height: AppTheme.separatorHeight)

            ScrollView {
                if isLoading {
                    SkeletonProductGrid()
                } else if searchText.isEmpty {
                    bentoGrid
                } else if filteredProducts.isEmpty {
                    noResultsState
                } else {
                    searchResults
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
        .alert("Failed to Load", isPresented: $loadError) {
            Button("Retry") { Task { await loadCategories(); await loadProducts() } }
            Button("OK", role: .cancel) {}
        } message: {
            Text("Could not load catalog. Check your connection.")
        }
    }

    // MARK: - Search Bar

    private var searchBar: some View {
        HStack(spacing: AppTheme.spacingMD) {
            Image(systemName: "magnifyingglass")
                .font(.system(size: 15, weight: .medium))
                .foregroundStyle(AppTheme.textTertiary)

            TextField("Search products...", text: $searchText)
                .font(.system(.subheadline, design: .rounded))
                .textFieldStyle(.plain)
                .autocorrectionDisabled()

            if !searchText.isEmpty {
                Button {
                    withAnimation(AnimationConstants.express) { searchText = "" }
                } label: {
                    Image(systemName: "xmark.circle.fill")
                        .font(.system(size: 16))
                        .foregroundStyle(AppTheme.textTertiary)
                }
            }
        }
        .padding(.horizontal, AppTheme.spacingMD)
        .padding(.vertical, AppTheme.spacingMD)
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusButton))
        .shadow(color: AppTheme.shadowColor, radius: 4, x: 0, y: 2)
    }

    // MARK: - Bento Grid

    private var bentoGrid: some View {
        VStack(spacing: AppTheme.spacingMD) {
            // Section title
            HStack {
                Text("Categories")
                    .font(.system(.headline, design: .rounded))
                    .foregroundStyle(AppTheme.textPrimary)
                Spacer()
                Text("\(categories.count) types")
                    .font(.system(.caption, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
            }
            .padding(.horizontal, AppTheme.spacingLG)
            .padding(.top, AppTheme.spacingMD)

            // Row 1: 1 big + 1 big
            if categories.count >= 2 {
                HStack(spacing: AppTheme.spacingMD) {
                    bentoBig(categories[0], height: 150)
                        .staggeredSlideIn(index: 0)
                    bentoBig(categories[1], height: 150)
                        .staggeredSlideIn(index: 1)
                }
                .padding(.horizontal, AppTheme.spacingLG)
            }

            // Row 2: 1 wide + 2 small stacked
            if categories.count >= 4 {
                HStack(spacing: AppTheme.spacingMD) {
                    bentoWide(categories[2], height: 130)
                        .staggeredSlideIn(index: 2)

                    VStack(spacing: AppTheme.spacingMD) {
                        bentoSmall(categories[3])
                            .staggeredSlideIn(index: 3)
                        if categories.count >= 5 {
                            bentoSmall(categories[4])
                                .staggeredSlideIn(index: 4)
                        }
                    }
                }
                .padding(.horizontal, AppTheme.spacingLG)
            }

            // Row 3: 3 equal
            if categories.count >= 7 {
                HStack(spacing: AppTheme.spacingMD) {
                    bentoCompact(categories[5])
                        .staggeredSlideIn(index: 5)
                    bentoCompact(categories[6])
                        .staggeredSlideIn(index: 6)
                    if categories.count >= 8 {
                        bentoCompact(categories[7])
                            .staggeredSlideIn(index: 7)
                    }
                }
                .padding(.horizontal, AppTheme.spacingLG)
            }

            // Remaining categories in adaptive grid
            if categories.count > 8 {
                let remaining = Array(categories.dropFirst(8))
                let cols = [GridItem(.adaptive(minimum: 160), spacing: AppTheme.spacingMD)]
                LazyVGrid(columns: cols, spacing: AppTheme.spacingMD) {
                    ForEach(Array(remaining.enumerated()), id: \.element.id) { idx, cat in
                        bentoBig(cat, height: 120)
                            .staggeredSlideIn(index: idx + 8)
                    }
                }
                .padding(.horizontal, AppTheme.spacingLG)
            }
        }
        .padding(.bottom, AppTheme.spacingHuge)
    }

    // MARK: - Bento Cards

    private func bentoBig(_ cat: ProductCategory, height: Double) -> some View {
        NavigationLink {
            CategorySuppliersView(category: cat)
        } label: {
            VStack(alignment: .leading, spacing: 0) {
                Spacer()
                Image(systemName: cat.icon)
                    .font(.system(size: 36, weight: .medium))
                    .foregroundStyle(AppTheme.textPrimary)
                    .padding(.bottom, AppTheme.spacingSM)
                Text(cat.name)
                    .font(.system(.subheadline, design: .rounded, weight: .bold))
                    .foregroundStyle(AppTheme.textPrimary)
                if let count = cat.productCount {
                    Text("\(count) items")
                        .font(.system(.caption2, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .frame(height: height)
            .padding(AppTheme.spacingMD)
            .background(AppTheme.cardBackground)
            .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
            .shadow(color: AppTheme.shadowColor, radius: 4, x: 0, y: 2)
        }
        .buttonStyle(.plain)
        .pressable()
    }

    private func bentoWide(_ cat: ProductCategory, height: Double) -> some View {
        NavigationLink {
            CategorySuppliersView(category: cat)
        } label: {
            HStack(spacing: AppTheme.spacingMD) {
                Image(systemName: cat.icon)
                    .font(.system(size: 42, weight: .medium))
                    .foregroundStyle(AppTheme.textPrimary)
                VStack(alignment: .leading, spacing: 3) {
                    Text(cat.name)
                        .font(.system(.headline, design: .rounded))
                        .foregroundStyle(AppTheme.textPrimary)
                    if let count = cat.productCount {
                        Text("\(count) items")
                            .font(.system(.caption, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                    }
                }
                Spacer()
            }
            .frame(height: height)
            .padding(AppTheme.spacingMD)
            .background(AppTheme.cardBackground)
            .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
            .shadow(color: AppTheme.shadowColor, radius: 4, x: 0, y: 2)
        }
        .buttonStyle(.plain)
        .pressable()
    }

    private func bentoSmall(_ cat: ProductCategory) -> some View {
        NavigationLink {
            CategorySuppliersView(category: cat)
        } label: {
            HStack(spacing: AppTheme.spacingSM) {
                Image(systemName: cat.icon)
                    .font(.system(size: 18, weight: .medium))
                    .foregroundStyle(AppTheme.textPrimary)
                Text(cat.name)
                    .font(.system(.caption, design: .rounded, weight: .semibold))
                    .foregroundStyle(AppTheme.textPrimary)
                    .lineLimit(1)
                Spacer()
            }
            .frame(height: 54)
            .padding(.horizontal, AppTheme.spacingMD)
            .background(AppTheme.cardBackground)
            .clipShape(.rect(cornerRadius: AppTheme.radiusMD))
            .shadow(color: AppTheme.shadowColor, radius: 3, x: 0, y: 1)
        }
        .buttonStyle(.plain)
        .pressable()
    }

    private func bentoCompact(_ cat: ProductCategory) -> some View {
        NavigationLink {
            CategorySuppliersView(category: cat)
        } label: {
            VStack(spacing: AppTheme.spacingSM) {
                Image(systemName: cat.icon)
                    .font(.system(size: 24, weight: .medium))
                    .foregroundStyle(AppTheme.textPrimary)
                Text(cat.name)
                    .font(.system(.caption2, design: .rounded, weight: .medium))
                    .foregroundStyle(AppTheme.textSecondary)
            }
            .frame(maxWidth: .infinity)
            .frame(height: 80)
            .background(AppTheme.cardBackground)
            .clipShape(.rect(cornerRadius: AppTheme.radiusMD))
            .shadow(color: AppTheme.shadowColor, radius: 3, x: 0, y: 1)
        }
        .buttonStyle(.plain)
        .pressable()
    }

    // MARK: - Search Results

    private var searchResults: some View {
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

    // MARK: - No Results

    private var noResultsState: some View {
        VStack(spacing: AppTheme.spacingLG) {
            Spacer(minLength: 60)
            ZStack {
                Circle().fill(AppTheme.surfaceElevated).frame(width: 80, height: 80)
                Image(systemName: "magnifyingglass").font(.system(size: 32)).foregroundStyle(AppTheme.textTertiary)
            }
            Text("No Results")
                .font(.system(.headline, design: .rounded))
                .foregroundStyle(AppTheme.textPrimary)
            Text("No products match \"\(searchText)\"")
                .font(.system(.subheadline, design: .rounded))
                .foregroundStyle(AppTheme.textTertiary)
            Spacer()
        }
        .padding(AppTheme.spacingXL)
    }

    // MARK: - API

    private func loadCategories() async {
        do {
            let result: [ProductCategory] = try await api.get(path: "/v1/catalog/categories")
            categories = result
        } catch { categories = []; loadError = true }
    }

    private func loadProducts() async {
        isLoading = true
        do {
            let result: [Product] = try await api.get(path: "/v1/products")
            products = result
        } catch { products = []; loadError = true }
        isLoading = false
    }
}

#Preview {
    NavigationStack { CatalogView().environment(CartManager()) }
}
