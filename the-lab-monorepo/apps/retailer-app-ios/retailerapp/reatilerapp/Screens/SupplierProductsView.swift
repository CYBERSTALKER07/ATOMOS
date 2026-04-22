import SwiftUI

struct SupplierProductsView: View {
    let supplier: Supplier
    @Environment(CartManager.self) private var cart
    @State private var products: [Product] = []
    @State private var isLoading = false
    @State private var errorMessage: String?
    @State private var selectedProduct: Product?
    @State private var autoOrderSettings: [String: Bool] = [:]
    @State private var supplierAutoOrder = false
    @State private var isMySupplier = false
    @State private var isTogglingMySupplier = false
    @State private var isTogglingSupplierAutoOrder = false
    @State private var togglingProductIds: Set<String> = []

    private let api = APIClient.shared

    private var groupedProducts: [(key: String, value: [Product])] {
        Dictionary(grouping: products) { product in
            product.categoryName ?? supplier.displayCategory ?? "Other"
        }
        .sorted { $0.key.localizedCaseInsensitiveCompare($1.key) == .orderedAscending }
    }

    var body: some View {
        ScrollView {
            VStack(spacing: AppTheme.spacingLG) {
                // Supplier header card
                supplierHeader
                    .slideIn(delay: 0)

                // Auto-order for this supplier
                supplierAutoOrderCard
                    .slideIn(delay: 0.05)

                // Products list
                if isLoading && products.isEmpty {
                    loadingProductsSection
                } else if products.isEmpty {
                    emptyProductsSection
                } else {
                    productsSection
                }
            }
            .padding(.horizontal, AppTheme.spacingLG)
            .padding(.top, AppTheme.spacingSM)
            .padding(.bottom, AppTheme.spacingXXL)
        }
        .scrollIndicators(.hidden)
        .background(AppTheme.background)
        .navigationTitle(supplier.name)
        .navigationBarTitleDisplayMode(.inline)
        .navigationDestination(item: $selectedProduct) { product in
            ProductDetailView(product: product)
        }
        .task {
            cart.supplierIsActive = supplier.isActive
            await loadProducts()
        }
        .refreshable { await loadProducts() }
    }

    // MARK: - Supplier Header

    private var supplierHeader: some View {
        VStack(spacing: AppTheme.spacingMD) {
            HStack(spacing: AppTheme.spacingLG) {
                ZStack {
                    Circle()
                        .fill(AppTheme.surfaceElevated)
                        .frame(width: 64, height: 64)
                    Text(supplier.initials)
                        .font(.system(.title2, design: .rounded, weight: .bold))
                        .foregroundStyle(AppTheme.textSecondary)
                }

                VStack(alignment: .leading, spacing: 4) {
                    Text(supplier.name)
                        .font(.system(.title3, design: .rounded, weight: .bold))
                        .foregroundStyle(AppTheme.textPrimary)
                    if let cat = supplier.displayCategory {
                        Text(cat)
                            .font(.system(.caption, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                    }

                    // OPEN/CLOSED status badge
                    HStack(spacing: 5) {
                        Circle()
                            .fill(supplier.isActive ? AppTheme.success : AppTheme.destructive)
                            .frame(width: 7, height: 7)
                        Text(supplier.isActive ? "OPEN" : "CLOSED")
                            .font(.system(.caption2, design: .rounded, weight: .bold))
                            .foregroundStyle(supplier.isActive ? AppTheme.success : AppTheme.destructive)
                    }
                    .padding(.horizontal, 8)
                    .padding(.vertical, 4)
                    .background((supplier.isActive ? AppTheme.success : AppTheme.destructive).opacity(0.12))
                    .clipShape(.capsule)

                    HStack(spacing: AppTheme.spacingSM) {
                        Label("\(supplier.catalogSubtitle)", systemImage: "shippingbox")
                        if let date = supplier.lastOrderDate {
                            Label(date, systemImage: "calendar")
                        }
                    }
                    .font(.system(.caption2, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
                }

                Spacer()
            }

            if !supplier.operatingCategoryNames.isEmpty {
                ScrollView(.horizontal) {
                    HStack(spacing: AppTheme.spacingSM) {
                        ForEach(supplier.operatingCategoryNames, id: \.self) { categoryName in
                            Text(categoryName)
                                .font(.system(.caption2, design: .rounded, weight: .semibold))
                                .foregroundStyle(AppTheme.textSecondary)
                                .padding(.horizontal, 8)
                                .padding(.vertical, 5)
                                .background(AppTheme.surfaceElevated)
                                .clipShape(.capsule)
                        }
                    }
                }
                .scrollIndicators(.hidden)
            }

            // Add to My Suppliers button
            Button {
                guard !isTogglingMySupplier else { return }
                Haptics.medium()
                withAnimation(AnimationConstants.bouncy) {
                    isMySupplier.toggle()
                }
                Task { await toggleMySupplier() }
            } label: {
                HStack(spacing: AppTheme.spacingSM) {
                    Image(systemName: isMySupplier ? "checkmark.circle.fill" : "plus.circle")
                        .font(.system(size: 16, weight: .semibold))
                        .contentTransition(.symbolEffect(.replace))
                    Text(isMySupplier ? "Added to My Suppliers" : "Add to My Suppliers")
                        .font(.system(.subheadline, design: .rounded, weight: .semibold))
                }
                .foregroundStyle(isMySupplier ? AppTheme.success : AppTheme.textPrimary)
                .frame(maxWidth: .infinity)
                .padding(.vertical, AppTheme.spacingMD)
                .background(isMySupplier ? AppTheme.success.opacity(0.1) : AppTheme.surfaceElevated)
                .clipShape(.rect(cornerRadius: AppTheme.radiusButton))
            }
            .disabled(isTogglingMySupplier)
        }
        .padding(AppTheme.spacingLG)
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
        .shadow(color: AppTheme.shadowColor, radius: 4, y: 2)
    }

    // MARK: - Supplier Auto-Order

    private var supplierAutoOrderCard: some View {
        LabCard {
            HStack(spacing: AppTheme.spacingMD) {
                ZStack {
                    RoundedRectangle(cornerRadius: AppTheme.radiusSM)
                        .fill(AppTheme.surfaceElevated)
                        .frame(width: 36, height: 36)
                    Image(systemName: "arrow.triangle.2.circlepath")
                        .font(.system(size: 14, weight: .semibold))
                        .foregroundStyle(AppTheme.textSecondary)
                }

                VStack(alignment: .leading, spacing: 2) {
                    Text("Auto-Order")
                        .font(.system(.subheadline, design: .rounded, weight: .medium))
                        .foregroundStyle(AppTheme.textPrimary)
                    Text("Auto-order all from \(supplier.name)")
                        .font(.system(.caption, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)
                }

                Spacer()

                Toggle("", isOn: $supplierAutoOrder)
                    .tint(AppTheme.accent)
                    .labelsHidden()
                    .disabled(isTogglingSupplierAutoOrder)
                    .onChange(of: supplierAutoOrder) {
                        Task { await toggleSupplierAutoOrder() }
                    }
            }
            .padding(AppTheme.spacingLG)
        }
    }

    // MARK: - Products List

    private var productsSection: some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
            HStack {
                Text("Products by Category")
                    .font(.system(.headline, design: .rounded))
                    .foregroundStyle(AppTheme.textPrimary)
                Spacer()
                Text("\(products.count) items")
                    .font(.system(.caption, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
            }

            ForEach(Array(groupedProducts.enumerated()), id: \.element.key) { groupIndex, group in
                VStack(alignment: .leading, spacing: AppTheme.spacingSM) {
                    Text(group.key)
                        .font(.system(.subheadline, design: .rounded, weight: .bold))
                        .foregroundStyle(AppTheme.textPrimary)

                    ForEach(Array(group.value.enumerated()), id: \.element.id) { itemIndex, product in
                        productRow(product)
                            .staggeredSlideIn(index: groupIndex + itemIndex)
                    }
                }
            }
        }
    }

    private var loadingProductsSection: some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
            HStack {
                RoundedRectangle(cornerRadius: 6)
                    .fill(AppTheme.surfaceElevated)
                    .frame(width: 150, height: 22)
                Spacer()
                RoundedRectangle(cornerRadius: 6)
                    .fill(AppTheme.surfaceElevated)
                    .frame(width: 70, height: 14)
            }
            .skeleton()

            ForEach(0..<5, id: \.self) { _ in
                HStack(spacing: AppTheme.spacingMD) {
                    RoundedRectangle(cornerRadius: AppTheme.radiusSM)
                        .fill(AppTheme.surfaceElevated)
                        .frame(width: 50, height: 50)
                    VStack(alignment: .leading, spacing: 6) {
                        RoundedRectangle(cornerRadius: 6)
                            .fill(AppTheme.surfaceElevated)
                            .frame(width: 140, height: 14)
                        RoundedRectangle(cornerRadius: 999)
                            .fill(AppTheme.surfaceElevated)
                            .frame(width: 96, height: 18)
                    }
                    Spacer()
                    RoundedRectangle(cornerRadius: 999)
                        .fill(AppTheme.surfaceElevated)
                        .frame(width: 34, height: 22)
                }
                .padding(AppTheme.spacingMD)
                .background(AppTheme.cardBackground)
                .clipShape(.rect(cornerRadius: AppTheme.radiusMD))
                .shadow(color: AppTheme.shadowColor, radius: 2, y: 1)
                .skeleton()
            }
        }
    }

    private var emptyProductsSection: some View {
        VStack(spacing: AppTheme.spacingLG) {
            ZStack {
                Circle()
                    .fill(AppTheme.surfaceElevated)
                    .frame(width: 80, height: 80)
                Image(systemName: "shippingbox")
                    .font(.system(size: 30, weight: .medium))
                    .foregroundStyle(AppTheme.textTertiary)
            }
            Text("No Products Yet")
                .font(.system(.headline, design: .rounded))
                .foregroundStyle(AppTheme.textPrimary)
            Text(errorMessage ?? "This supplier has no active catalog items right now.")
                .font(.system(.subheadline, design: .rounded))
                .foregroundStyle(AppTheme.textTertiary)
                .multilineTextAlignment(.center)
            Button(errorMessage == nil ? "Refresh Catalog" : "Retry") {
                Task { await loadProducts() }
            }
            .font(.system(.subheadline, design: .rounded, weight: .semibold))
            .foregroundStyle(AppTheme.cardBackground)
            .padding(.horizontal, AppTheme.spacingLG)
            .padding(.vertical, AppTheme.spacingMD)
            .background(AppTheme.textPrimary)
            .clipShape(.rect(cornerRadius: AppTheme.radiusButton))
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, AppTheme.spacingXL)
    }

    private func productRow(_ product: Product) -> some View {
        HStack(spacing: AppTheme.spacingMD) {
            // Product icon
            ZStack {
                RoundedRectangle(cornerRadius: AppTheme.radiusSM)
                    .fill(AppTheme.surfaceElevated)
                    .frame(width: 50, height: 50)
                Image(systemName: "leaf.fill")
                    .font(.system(size: 20))
                    .foregroundStyle(AppTheme.textTertiary)
            }

            // Info
            VStack(alignment: .leading, spacing: 3) {
                Text(product.name)
                    .font(.system(.subheadline, design: .rounded, weight: .medium))
                    .foregroundStyle(AppTheme.textPrimary)
                    .lineLimit(1)

                HStack(spacing: AppTheme.spacingSM) {
                    Text(product.displayPrice)
                        .font(.system(.caption, design: .rounded, weight: .bold))
                        .foregroundStyle(AppTheme.textPrimary)

                    if let v = product.defaultVariant {
                        Text(v.size)
                            .font(.system(.caption2, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                            .padding(.horizontal, 4).padding(.vertical, 1)
                            .background(AppTheme.surfaceElevated)
                            .clipShape(.capsule)
                    } else if let merchandisingLabel = product.merchandisingLabel {
                        Text(merchandisingLabel)
                            .font(.system(.caption2, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                            .padding(.horizontal, 4).padding(.vertical, 1)
                            .background(AppTheme.surfaceElevated)
                            .clipShape(.capsule)
                    }
                }
            }

            Spacer()

            // Auto-order toggle
            Toggle("", isOn: Binding(
                get: { autoOrderSettings[product.id] ?? false },
                set: { newVal in
                    guard !togglingProductIds.contains(product.id) else { return }
                    autoOrderSettings[product.id] = newVal
                    Task { await toggleProductAutoOrder(productId: product.id, enabled: newVal) }
                }
            ))
            .tint(AppTheme.accent)
            .labelsHidden()
            .scaleEffect(0.75)
            .disabled(togglingProductIds.contains(product.id))

            // Navigate to detail / Order
            Button {
                Haptics.light()
                selectedProduct = product
            } label: {
                Image(systemName: "chevron.right")
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundStyle(AppTheme.textTertiary.opacity(0.5))
            }
        }
        .padding(AppTheme.spacingMD)
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusMD))
        .shadow(color: AppTheme.shadowColor, radius: 2, y: 1)
    }

    // MARK: - API

    private func loadProducts() async {
        isLoading = true
        errorMessage = nil
        do {
            let result: [Product] = try await api.get(path: "/v1/catalog/products?supplier_id=\(supplier.id)")
            products = result
        } catch {
            products = []
            errorMessage = "Products are unavailable right now. Check your connection and try again."
        }
        isLoading = false
    }

    private func toggleMySupplier() async {
        isTogglingMySupplier = true
        do {
            let path = isMySupplier ? "/v1/retailer/suppliers/\(supplier.id)/add" : "/v1/retailer/suppliers/\(supplier.id)/remove"
            let _: [String: Bool] = try await api.post(path: path, body: ["supplier_id": supplier.id])
        } catch {
            withAnimation(AnimationConstants.express) { isMySupplier.toggle() }
        }
        isTogglingMySupplier = false
    }

    private func toggleSupplierAutoOrder() async {
        isTogglingSupplierAutoOrder = true
        do {
            let _: [String: Bool] = try await api.patch(
                path: "/v1/retailer/settings/auto-order/supplier/\(supplier.id)",
                body: ["auto_order_enabled": supplierAutoOrder]
            )
        } catch {
            withAnimation { supplierAutoOrder.toggle() }
        }
        isTogglingSupplierAutoOrder = false
    }

    private func toggleProductAutoOrder(productId: String, enabled: Bool) async {
        togglingProductIds.insert(productId)
        do {
            let _: [String: Bool] = try await api.patch(
                path: "/v1/retailer/settings/auto-order/product/\(productId)",
                body: ["auto_order_enabled": enabled]
            )
        } catch {
            withAnimation { autoOrderSettings[productId] = !enabled }
        }
        togglingProductIds.remove(productId)
    }
}

#Preview {
    NavigationStack {
        SupplierProductsView(supplier: Supplier.samples[0])
            .environment(CartManager())
    }
}
