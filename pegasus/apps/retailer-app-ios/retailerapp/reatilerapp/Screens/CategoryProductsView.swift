import SwiftUI

struct CategoryProductsView: View {
    let category: ProductCategory
    @Environment(CartManager.self) private var cart
    @State private var products: [Product] = []
    @State private var isLoading = false
    @State private var selectedProduct: Product?
    @State private var autoOrderSettings: [String: Bool] = [:]
    @State private var togglingIds: Set<String> = []

    private let api = APIClient.shared
    private let columns = [GridItem(.adaptive(minimum: 160), spacing: 14)]

    // Group products by a simulated supplier (in production, backend returns supplier_name)
    private var groupedProducts: [(supplier: String, products: [Product])] {
        // Group by first word of description as a proxy for supplier grouping
        let grouped = Dictionary(grouping: products) { product in
            String(product.name.split(separator: " ").last ?? "Other")
        }
        return grouped.sorted { $0.key < $1.key }.map { (supplier: $0.key, products: $0.value) }
    }

    var body: some View {
        ScrollView {
            if products.isEmpty && !isLoading {
                emptyState
            } else {
                LazyVStack(alignment: .leading, spacing: AppTheme.spacingXL) {
                    ForEach(Array(groupedProducts.enumerated()), id: \.element.supplier) { sectionIndex, group in
                        VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
                            // Supplier header
                            HStack(spacing: AppTheme.spacingSM) {
                                ZStack {
                                    RoundedRectangle(cornerRadius: AppTheme.radiusSM)
                                        .fill(AppTheme.surfaceElevated)
                                        .frame(width: 28, height: 28)
                                    Image(systemName: "building.2")
                                        .font(.system(size: 12, weight: .semibold))
                                        .foregroundStyle(AppTheme.textSecondary)
                                }
                                Text(group.supplier)
                                    .font(.system(.subheadline, design: .rounded, weight: .bold))
                                    .foregroundStyle(AppTheme.textPrimary)
                                Spacer()
                                Text("\(group.products.count) items")
                                    .font(.system(.caption2, design: .rounded))
                                    .foregroundStyle(AppTheme.textTertiary)
                            }

                            // Products
                            ForEach(group.products) { product in
                                productRow(product)
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
        .scrollIndicators(.hidden)
        .background(AppTheme.background)
        .navigationTitle(category.name)
        .navigationBarTitleDisplayMode(.inline)
        .navigationDestination(item: $selectedProduct) { product in
            ProductDetailView(product: product)
        }
        .task { await loadProducts() }
    }

    // MARK: - Product Row with Auto-Order Toggle

    private func productRow(_ product: Product) -> some View {
        HStack(spacing: AppTheme.spacingMD) {
            // Product icon
            ZStack {
                RoundedRectangle(cornerRadius: AppTheme.radiusSM)
                    .fill(AppTheme.surfaceElevated)
                    .frame(width: 44, height: 44)
                Image(systemName: "leaf.fill")
                    .font(.system(size: 18))
                    .foregroundStyle(AppTheme.textTertiary)
            }

            // Info
            VStack(alignment: .leading, spacing: 2) {
                Text(product.name)
                    .font(.system(.subheadline, design: .rounded, weight: .medium))
                    .foregroundStyle(AppTheme.textPrimary)
                    .lineLimit(1)

                Text(product.displayPrice)
                    .font(.system(.caption, design: .rounded, weight: .bold))
                    .foregroundStyle(AppTheme.textSecondary)
            }

            Spacer()

            // Auto-order toggle (Product Level — Empathy Engine)
            Toggle("", isOn: Binding(
                get: { autoOrderSettings[product.id] ?? false },
                set: { newVal in
                    guard !togglingIds.contains(product.id) else { return }
                    autoOrderSettings[product.id] = newVal
                    Task { await toggleProductAutoOrder(productId: product.id, enabled: newVal) }
                }
            ))
            .tint(AppTheme.accent)
            .labelsHidden()
            .scaleEffect(0.8)
            .disabled(togglingIds.contains(product.id))

            // Navigate to detail
            Button {
                Haptics.light()
                selectedProduct = product
            } label: {
                Image(systemName: "chevron.right")
                    .font(.system(size: 12, weight: .semibold))
                    .foregroundStyle(AppTheme.textTertiary.opacity(0.5))
            }
        }
        .padding(AppTheme.spacingMD)
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusMD))
        .shadow(color: AppTheme.shadowColor, radius: 2, y: 1)
    }

    // MARK: - Empty State

    private var emptyState: some View {
        VStack(spacing: AppTheme.spacingLG) {
            Spacer(minLength: 80)
            ZStack {
                Circle().fill(AppTheme.surfaceElevated).frame(width: 80, height: 80)
                Image(systemName: category.icon).font(.system(size: 32)).foregroundStyle(AppTheme.textTertiary)
            }
            Text("No Products")
                .font(.system(.headline, design: .rounded))
                .foregroundStyle(AppTheme.textPrimary)
            Text("No products found in \(category.name)")
                .font(.system(.subheadline, design: .rounded))
                .foregroundStyle(AppTheme.textTertiary)
            Spacer()
        }
        .padding(AppTheme.spacingXL)
    }

    // MARK: - API

    private func loadProducts() async {
        isLoading = true
        do {
            let result: [Product] = try await api.get(path: "/v1/catalog/products?category_id=\(category.id)")
            products = result
        } catch {
            products = []
        }
        isLoading = false
    }

    private func toggleProductAutoOrder(productId: String, enabled: Bool) async {
        togglingIds.insert(productId)
        do {
            let _: [String: Bool] = try await api.patch(
                path: "/v1/retailer/settings/auto-order/product/\(productId)",
                body: ["auto_order_enabled": enabled]
            )
        } catch {
            withAnimation { autoOrderSettings[productId] = !enabled }
        }
        togglingIds.remove(productId)
    }
}

#Preview {
    NavigationStack {
        CategoryProductsView(category: ProductCategory.samples[0])
            .environment(CartManager())
    }
}
