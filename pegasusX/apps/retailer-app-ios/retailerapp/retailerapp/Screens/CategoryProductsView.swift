import SwiftUI

struct CategoryProductsView: View {
    let category: ProductCategory
    @Environment(CartManager.self) private var cart
    @State private var refreshCenter = RetailerRefreshCenter.shared
    @State private var vm = CategoryProductsViewModel()

    var body: some View {
        @Bindable var vm = vm

        ScrollView {
            if vm.products.isEmpty && !vm.isLoading {
                emptyState
            } else {
                LazyVStack(alignment: .leading, spacing: AppTheme.spacingXL) {
                    ForEach(Array(vm.groupedProducts.enumerated()), id: \.element.supplier) { sectionIndex, group in
                        VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
                            HStack(spacing: AppTheme.spacingSM) {
                                ZStack {
                                    RoundedRectangle(cornerRadius: AppTheme.radiusSM)
                                        .fill(AppTheme.surfaceElevated)
                                        .frame(width: 28, height: 28)
                                    Image(systemName: "building.2")
                                        .font(.caption.weight(.semibold))
                                        .foregroundStyle(AppTheme.textSecondary)
                                }
                                Text(group.supplier)
                                    .font(.system(.subheadline, design: .rounded, weight: .bold))
                                    .foregroundStyle(AppTheme.textPrimary)
                                Spacer()
                                Text(L10n.format("mobile_retailer.ui.count_items_2", "\(group.products.count)"))
                                    .font(.system(.caption2, design: .rounded))
                                    .foregroundStyle(AppTheme.textTertiary)
                            }
                            .accessibilityElement(children: .combine)
                            .accessibilityLabel("\(group.supplier), \(group.products.count) items")

                            ForEach(group.products) { product in
                                CategoryProductRowView(
                                    product: product,
                                    externalEnabled: vm.autoOrderSettings[product.id] ?? false,
                                    isToggling: vm.togglingIds.contains(product.id),
                                    onToggleAutoOrder: { newVal in
                                        Task {
                                            await vm.toggleProductAutoOrder(productId: product.id, enabled: newVal)
                                        }
                                    },
                                    onSelect: {
                                        Haptics.light()
                                        vm.selectedProduct = product
                                    }
                                )
                            }
                        }
                        .staggeredSlideIn(index: sectionIndex)
                    }
                }
                .padding(.horizontal, AppTheme.spacingLG)
                .padding(.top, AppTheme.spacingSM)
                .padding(.bottom, AppTheme.spacingXXL)
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .scrollIndicators(.hidden)
        .background(AppTheme.background)
        .navigationTitle(category.name)
        .navigationBarTitleDisplayMode(.inline)
        .navigationDestination(item: $vm.selectedProduct) { product in
            ProductDetailView(product: product)
        }
        .task { await vm.loadProducts(categoryId: category.id) }
        .task(id: refreshCenter.refreshToken) { await vm.loadProducts(categoryId: category.id) }
        .refreshable { await vm.loadProducts(categoryId: category.id) }
        .task {
            for await event in RetailerWebSocket.shared.eventStream() {
                if case .promotionChanged = event {
                    await vm.loadProducts(categoryId: category.id)
                }
            }
        }
    }

    // MARK: - Empty State

    private var emptyState: some View {
        VStack(spacing: AppTheme.spacingLG) {
            Spacer(minLength: 80)
            ZStack {
                Circle().fill(AppTheme.surfaceElevated).frame(width: 80, height: 80)
                Image(systemName: category.icon)
                    .font(.title)
                    .foregroundStyle(AppTheme.textTertiary)
            }
            .accessibilityHidden(true)

            Text("mobile_retailer.ui.no_products")
                .font(.system(.headline, design: .rounded))
                .foregroundStyle(AppTheme.textPrimary)
            Text(L10n.format("mobile_retailer.ui.no_products_found_in_name", "\(category.name)"))
                .font(.system(.subheadline, design: .rounded))
                .foregroundStyle(AppTheme.textTertiary)
            Spacer()
        }
        .padding(AppTheme.spacingXL)
        .accessibilityElement(children: .combine)
    }
}

private struct CategoryProductRowView: View {
    let product: Product
    let externalEnabled: Bool
    let isToggling: Bool
    let onToggleAutoOrder: (Bool) -> Void
    let onSelect: () -> Void

    @State private var isAutoOrderEnabled: Bool

    init(
        product: Product,
        externalEnabled: Bool,
        isToggling: Bool,
        onToggleAutoOrder: @escaping (Bool) -> Void,
        onSelect: @escaping () -> Void
    ) {
        self.product = product
        self.externalEnabled = externalEnabled
        self.isToggling = isToggling
        self.onToggleAutoOrder = onToggleAutoOrder
        self.onSelect = onSelect
        self._isAutoOrderEnabled = State(initialValue: externalEnabled)
    }

    var body: some View {
        HStack(spacing: AppTheme.spacingMD) {
            Button(action: onSelect) {
                HStack(spacing: AppTheme.spacingMD) {
                    ZStack {
                        RoundedRectangle(cornerRadius: AppTheme.radiusSM)
                            .fill(AppTheme.surfaceElevated)
                            .frame(width: 44, height: 44)
                        Image(systemName: "leaf.fill")
                            .font(.title3)
                            .foregroundStyle(AppTheme.textTertiary)
                    }

                    VStack(alignment: .leading, spacing: 2) {
                        Text(product.name)
                            .font(.system(.subheadline, design: .rounded, weight: .medium))
                            .foregroundStyle(AppTheme.textPrimary)
                            .lineLimit(1)

                        HStack(spacing: AppTheme.spacingSM) {
                            if product.hasSaleOffer, let listPrice = product.displayListPrice {
                                Text(listPrice)
                                    .font(.system(.caption2, design: .rounded, weight: .medium))
                                    .foregroundStyle(AppTheme.textTertiary)
                                    .strikethrough()
                            }
                            Text(product.displayPrice)
                                .font(.system(.caption, design: .rounded, weight: .bold))
                                .foregroundStyle(product.hasSaleOffer ? AppTheme.success : AppTheme.textSecondary)
                        }
                    }

                    Spacer()
                }
                .pegasusMinimumHitTarget()
            }
            .buttonStyle(.plain)
            .accessibilityElement(children: .combine)
            .accessibilityLabel("\(product.name), \(product.displayPrice)")

            Toggle("", isOn: $isAutoOrderEnabled)
                .tint(AppTheme.accent)
                .labelsHidden()
                .scaleEffect(0.8)
                .disabled(isToggling)
                .accessibilityLabel("Auto-order \(product.name)")
                .onChange(of: isAutoOrderEnabled) { _, newVal in
                    guard newVal != externalEnabled else { return }
                    if isToggling {
                        isAutoOrderEnabled = externalEnabled
                        return
                    }
                    onToggleAutoOrder(newVal)
                }
                .onChange(of: externalEnabled) { _, updated in
                    if isAutoOrderEnabled != updated {
                        isAutoOrderEnabled = updated
                    }
                }

            Button(action: onSelect) {
                Image(systemName: "chevron.right")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(AppTheme.textTertiary.opacity(0.5))
                    .pegasusMinimumHitTarget()
            }
            .accessibilityLabel("View details for \(product.name)")
        }
        .padding(AppTheme.spacingMD)
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusMD))
        .shadow(color: AppTheme.shadowColor, radius: 2, y: 1)
    }
}

#Preview {
    NavigationStack {
        CategoryProductsView(category: ProductCategory(id: "cat-dairy", name: "Dairy", icon: "cup.and.saucer.fill", productCount: 0))
            .environment(CartManager())
    }
}
