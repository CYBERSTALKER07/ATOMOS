import SwiftUI

struct SearchView: View {
    @Environment(CartManager.self) private var cart
    @State private var refreshCenter = RetailerRefreshCenter.shared
    @State private var vm = SearchViewModel()
    @FocusState private var isSearchFocused: Bool

    private let columns = [GridItem(.adaptive(minimum: 160), spacing: 14)]

    var body: some View {
        @Bindable var vm = vm

        VStack(spacing: 0) {
            HStack(spacing: AppTheme.spacingMD) {
                Image(systemName: "magnifyingglass")
                    .font(.callout.weight(.medium))
                    .foregroundStyle(AppTheme.textTertiary)
                    .accessibilityHidden(true)

                TextField("mobile_retailer.ui.search_all_products", text: $vm.searchText)
                    .font(.system(.body, design: .rounded))
                    .textFieldStyle(.plain)
                    .focused($isSearchFocused)
                    .submitLabel(.search)
                    .autocorrectionDisabled()
                    .accessibilityLabel("Search all products")

                if !vm.searchText.isEmpty {
                    Button {
                        withAnimation(AnimationConstants.express) { vm.clearSearch() }
                        isSearchFocused = true
                    } label: {
                        Image(systemName: "xmark.circle.fill")
                            .font(.callout)
                            .foregroundStyle(AppTheme.textTertiary)
                            .pegasusMinimumHitTarget()
                    }
                    .accessibilityLabel("Clear search query")
                }
            }
            .padding(.horizontal, AppTheme.spacingMD)
            .padding(.vertical, AppTheme.spacingXS)
            .defaultFocus($isSearchFocused, true)

            Rectangle().fill(AppTheme.separator.opacity(0.3)).frame(height: AppTheme.separatorHeight)

            if vm.searchText.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                emptySearchState
            } else if vm.filteredProducts.isEmpty {
                noResultsState
            } else {
                ScrollView {
                    LazyVGrid(columns: columns, spacing: AppTheme.spacingLG) {
                        ForEach(Array(vm.filteredProducts.enumerated()), id: \.element.id) { index, product in
                            ProductCardView(product: product) {
                                Haptics.light()
                                vm.selectedProduct = product
                            }
                            .accessibilityElement(children: .combine)
                            .accessibilityLabel("\(product.name), \(product.displayPrice)")
                            .staggeredSlideIn(index: index)
                        }
                    }
                    .padding(AppTheme.spacingLG)
                    .padding(.bottom, AppTheme.spacingXXL)
                }
                .scrollDismissesKeyboard(.interactively)
                .scrollIndicators(.hidden)
            }
        }
        .background(AppTheme.background)
        .navigationDestination(item: $vm.selectedProduct) { product in
            ProductDetailView(product: product)
        }
        .task {
            await vm.loadProducts()
            isSearchFocused = true
        }
        .task(id: refreshCenter.refreshToken) { await vm.loadProducts() }
        .refreshable { await vm.loadProducts() }
    }

    private var emptySearchState: some View {
        VStack(spacing: AppTheme.spacingLG) {
            Spacer()
            ZStack {
                Circle().fill(AppTheme.accentSoft.opacity(0.3)).frame(width: 80, height: 80)
                Image(systemName: "magnifyingglass")
                    .font(.title)
                    .foregroundStyle(AppTheme.accent.opacity(0.4))
            }
            .accessibilityHidden(true)

            Text("mobile_retailer.ui.search_products")
                .font(.system(.headline, design: .rounded))
                .foregroundStyle(AppTheme.textPrimary)
            Text("mobile_retailer.ui.type_to_find_products_by_name_or_description")
                .font(.system(.subheadline, design: .rounded))
                .foregroundStyle(AppTheme.textTertiary)
                .multilineTextAlignment(.center)
            Spacer()
        }
        .padding(AppTheme.spacingXL)
        .accessibilityElement(children: .combine)
    }

    private var noResultsState: some View {
        VStack(spacing: AppTheme.spacingLG) {
            Spacer()
            ZStack {
                Circle().fill(AppTheme.warningSoft.opacity(0.3)).frame(width: 80, height: 80)
                Image(systemName: "exclamationmark.magnifyingglass")
                    .font(.title2)
                    .foregroundStyle(AppTheme.warning.opacity(0.5))
            }
            .accessibilityHidden(true)

            Text("mobile_retailer.ui.no_results")
                .font(.system(.headline, design: .rounded))
                .foregroundStyle(AppTheme.textPrimary)
            Text(L10n.format("mobile_retailer.ui.no_products_found_in_name", vm.searchText))
                .font(.system(.subheadline, design: .rounded))
                .foregroundStyle(AppTheme.textTertiary)
            Spacer()
        }
        .padding(AppTheme.spacingXL)
        .accessibilityElement(children: .combine)
    }
}

#Preview {
    NavigationStack { SearchView().environment(CartManager()) }
}
