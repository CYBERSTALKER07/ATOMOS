import SwiftUI

struct SearchView: View {
    @Environment(CartManager.self) private var cart
    @State private var searchText = ""
    @State private var products: [Product] = []
    @State private var selectedProduct: Product?
    @FocusState private var isSearchFocused: Bool

    private let api = APIClient.shared
    private let columns = [GridItem(.adaptive(minimum: 160), spacing: 14)]

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
            HStack(spacing: AppTheme.spacingMD) {
                Image(systemName: "magnifyingglass")
                    .font(.system(size: 16, weight: .medium))
                    .foregroundStyle(AppTheme.textTertiary)

                TextField("Search all products...", text: $searchText)
                    .font(.system(.body, design: .rounded))
                    .textFieldStyle(.plain)
                    .focused($isSearchFocused)
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
            .padding(AppTheme.spacingMD)
            .padding(.horizontal, AppTheme.spacingSM)

            Rectangle().fill(AppTheme.separator.opacity(0.3)).frame(height: AppTheme.separatorHeight)

            if searchText.isEmpty {
                emptySearchState
            } else if filteredProducts.isEmpty {
                noResultsState
            } else {
                ScrollView {
                    LazyVGrid(columns: columns, spacing: AppTheme.spacingLG) {
                        ForEach(Array(filteredProducts.enumerated()), id: \.element.id) { index, product in
                            ProductCardView(product: product) {
                                selectedProduct = product
                            }
                            .staggeredSlideIn(index: index)
                        }
                    }
                    .padding(AppTheme.spacingLG)
                    .padding(.bottom, AppTheme.spacingXXL)
                }
                .scrollIndicators(.hidden)
            }
        }
        .background(AppTheme.background)
        .navigationDestination(item: $selectedProduct) { product in
            ProductDetailView(product: product)
        }
        .task {
            await loadProducts()
            isSearchFocused = true
        }
    }

    private var emptySearchState: some View {
        VStack(spacing: AppTheme.spacingLG) {
            Spacer()
            ZStack {
                Circle().fill(AppTheme.accentSoft.opacity(0.3)).frame(width: 80, height: 80)
                Image(systemName: "magnifyingglass").font(.system(size: 32)).foregroundStyle(AppTheme.accent.opacity(0.4))
            }
            Text("Search Products")
                .font(.system(.headline, design: .rounded))
                .foregroundStyle(AppTheme.textPrimary)
            Text("Type to find products by name or description")
                .font(.system(.subheadline, design: .rounded))
                .foregroundStyle(AppTheme.textTertiary)
                .multilineTextAlignment(.center)
            Spacer()
        }
        .padding(AppTheme.spacingXL)
    }

    private var noResultsState: some View {
        VStack(spacing: AppTheme.spacingLG) {
            Spacer()
            ZStack {
                Circle().fill(AppTheme.warningSoft.opacity(0.3)).frame(width: 80, height: 80)
                Image(systemName: "exclamationmark.magnifyingglass").font(.system(size: 28)).foregroundStyle(AppTheme.warning.opacity(0.5))
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

    private func loadProducts() async {
        do { let r: [Product] = try await api.get(path: "/v1/products"); products = r }
        catch { products = [] }
    }
}

#Preview {
    NavigationStack { SearchView().environment(CartManager()) }
}
