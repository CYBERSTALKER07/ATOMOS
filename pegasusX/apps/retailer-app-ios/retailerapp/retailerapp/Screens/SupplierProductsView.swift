import SwiftUI

struct SupplierProductsView: View {
    let supplier: Supplier
    @Environment(CartManager.self) private var cart
    @State private var refreshCenter = RetailerRefreshCenter.shared
    @State private var vm = SupplierProductsViewModel()

    var body: some View {
        @Bindable var vm = vm

        ScrollView {
            VStack(spacing: AppTheme.spacingLG) {
                supplierHeader
                    .slideIn(delay: 0)

                if let errorMessage = vm.errorMessage, !errorMessage.isEmpty {
                    syncStatusBanner(errorMessage)
                }

                if vm.isLoading && vm.products.isEmpty {
                    loadingProductsSection
                } else if vm.products.isEmpty {
                    emptyProductsSection
                } else {
                    productsSection
                }
            }
            .padding(.horizontal, AppTheme.spacingLG)
            .padding(.top, AppTheme.spacingSM)
            .padding(.bottom, AppTheme.spacingXXL)
        }
        .scrollDismissesKeyboard(.interactively)
        .scrollIndicators(.hidden)
        .background(AppTheme.background)
        .navigationTitle(supplier.name)
        .navigationBarTitleDisplayMode(.inline)
        .navigationDestination(item: $vm.selectedProduct) { product in
            ProductDetailView(product: product)
        }
        .task {
            cart.supplierIsActive = supplier.isActive
            await vm.loadProducts(supplierId: supplier.id)
            await vm.loadAutoOrderState(supplierId: supplier.id)
        }
        .task(id: refreshCenter.refreshToken) {
            await vm.loadProducts(supplierId: supplier.id)
            await vm.loadAutoOrderState(supplierId: supplier.id)
        }
        .refreshable {
            await vm.loadProducts(supplierId: supplier.id)
            await vm.loadAutoOrderState(supplierId: supplier.id)
        }
        .task {
            for await event in RetailerWebSocket.shared.eventStream() {
                if case .promotionChanged(let supplierId) = event, supplierId == supplier.id {
                    await vm.loadProducts(supplierId: supplier.id)
                }
            }
        }
    }

    // MARK: - Supplier Header

    private var supplierHeader: some View {
        VStack(spacing: AppTheme.spacingMD) {
            HStack(spacing: AppTheme.spacingLG) {
                ZStack {
                    Circle()
                        .fill(AppTheme.accentGradient.opacity(0.1))
                        .frame(width: 64, height: 64)
                    Text(supplier.initials)
                        .font(.system(.title2, design: .rounded, weight: .bold))
                        .foregroundStyle(AppTheme.accent)
                }

                VStack(alignment: .leading, spacing: 4) {
                    Text(supplier.name)
                        .font(.system(.title3, design: .rounded, weight: .bold))
                        .foregroundStyle(AppTheme.textPrimary)
                    if let cat = supplier.displayCategory {
                        Text(cat.uppercased())
                            .font(.system(.caption2, design: .rounded, weight: .black))
                            .foregroundStyle(AppTheme.textTertiary)
                            .tracking(1)
                    }

                    HStack(spacing: 5) {
                        Circle()
                            .fill(supplier.isActive ? AppTheme.success : AppTheme.destructive)
                            .frame(width: 6, height: 6)
                        Text(supplier.isActive ? "ACTIVE" : "OFFLINE")
                            .font(.system(.caption2, design: .rounded, weight: .bold))
                            .foregroundStyle(supplier.isActive ? AppTheme.success : AppTheme.destructive)
                    }
                    .padding(.horizontal, 10)
                    .padding(.vertical, 4)
                    .background((supplier.isActive ? AppTheme.success : AppTheme.destructive).opacity(0.12))
                    .clipShape(.capsule)

                    HStack(spacing: AppTheme.spacingSM) {
                        Label("\(supplier.catalogSubtitle)", systemImage: "shippingbox.fill")
                        if let date = supplier.lastOrderDate {
                            Label(date, systemImage: "calendar")
                        }
                    }
                    .font(.system(.caption2, design: .rounded, weight: .medium))
                    .foregroundStyle(AppTheme.textTertiary)
                }

                Spacer()
            }
            .accessibilityElement(children: .combine)
            .accessibilityLabel("\(supplier.name), \(supplier.isActive ? "Active" : "Offline"), \(supplier.catalogSubtitle)")

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

            Button {
                Task { await vm.toggleMySupplier(supplierId: supplier.id) }
            } label: {
                HStack(spacing: AppTheme.spacingSM) {
                    Image(systemName: vm.isMySupplier ? "checkmark.circle.fill" : "plus.circle")
                        .font(.callout.weight(.semibold))
                        .contentTransition(.symbolEffect(.replace))
                    Text(vm.isMySupplier ? "Added to My Suppliers" : "Add to My Suppliers")
                        .font(.system(.subheadline, design: .rounded, weight: .semibold))
                }
                .foregroundStyle(vm.isMySupplier ? AppTheme.success : AppTheme.textPrimary)
                .frame(maxWidth: .infinity)
                .padding(.vertical, AppTheme.spacingMD)
                .background(vm.isMySupplier ? AppTheme.success.opacity(0.1) : AppTheme.surfaceElevated)
                .clipShape(.rect(cornerRadius: AppTheme.radiusButton))
                .pegasusMinimumHitTarget()
            }
            .disabled(vm.isTogglingMySupplier)
            .accessibilityLabel(vm.isMySupplier ? "Remove \(supplier.name) from My Suppliers" : "Add \(supplier.name) to My Suppliers")
        }
        .padding(AppTheme.spacingLG)
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
        .shadow(color: AppTheme.shadowColor, radius: 4, y: 2)
    }

    private func syncStatusBanner(_ message: String) -> some View {
        HStack(spacing: AppTheme.spacingSM) {
            Image(systemName: "exclamationmark.triangle.fill")
                .font(.caption.weight(.semibold))
                .foregroundStyle(AppTheme.warning)
            Text(message)
                .font(.system(.caption, design: .rounded, weight: .medium))
                .foregroundStyle(AppTheme.textSecondary)
                .lineLimit(3)
            Spacer()
            Button("common.action.retry") {
                Haptics.light()
                Task {
                    await vm.loadProducts(supplierId: supplier.id)
                    await vm.loadAutoOrderState(supplierId: supplier.id)
                }
            }
            .font(.system(.caption, design: .rounded, weight: .semibold))
            .foregroundStyle(AppTheme.accent)
            .pegasusMinimumHitTarget()
            .accessibilityLabel("Retry loading supplier products")
        }
        .padding(.horizontal, AppTheme.spacingMD)
        .padding(.vertical, AppTheme.spacingSM)
        .background(AppTheme.warning.opacity(0.12))
        .clipShape(.rect(cornerRadius: AppTheme.radiusMD))
        .slideIn(delay: 0.03)
    }

    // MARK: - Products List

    private var productsSection: some View {
        let grouped = vm.groupedProducts(for: supplier)

        return VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
            HStack {
                Text("mobile_retailer.ui.products_by_category")
                    .font(.system(.headline, design: .rounded))
                    .foregroundStyle(AppTheme.textPrimary)
                Spacer()
                Text(L10n.format("mobile_retailer.ui.count_items_2", "\(vm.products.count)"))
                    .font(.system(.caption, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
            }

            ForEach(Array(grouped.enumerated()), id: \.element.key) { groupIndex, group in
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
                    .font(.title.weight(.medium))
                    .foregroundStyle(AppTheme.textTertiary)
            }
            .accessibilityHidden(true)

            Text("mobile_retailer.ui.no_products_yet")
                .font(.system(.headline, design: .rounded))
                .foregroundStyle(AppTheme.textPrimary)
            Text(vm.errorMessage ?? "This supplier has no active catalog items right now.")
                .font(.system(.subheadline, design: .rounded))
                .foregroundStyle(AppTheme.textTertiary)
                .multilineTextAlignment(.center)
            Button(vm.errorMessage == nil ? "Refresh Catalog" : "Retry") {
                Haptics.light()
                Task { await vm.loadProducts(supplierId: supplier.id) }
            }
            .font(.system(.subheadline, design: .rounded, weight: .semibold))
            .foregroundStyle(AppTheme.cardBackground)
            .padding(.horizontal, AppTheme.spacingLG)
            .padding(.vertical, AppTheme.spacingMD)
            .background(AppTheme.textPrimary)
            .clipShape(.rect(cornerRadius: AppTheme.radiusButton))
            .pegasusMinimumHitTarget()
            .accessibilityLabel(vm.errorMessage == nil ? "Refresh catalog" : "Retry loading catalog")
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, AppTheme.spacingXL)
    }

    private func productRow(_ product: Product) -> some View {
        Button {
            Haptics.light()
            vm.selectedProduct = product
        } label: {
            HStack(spacing: AppTheme.spacingMD) {
                ZStack {
                    RoundedRectangle(cornerRadius: AppTheme.radiusSM)
                        .fill(AppTheme.surfaceElevated)
                        .frame(width: 50, height: 50)
                    Image(systemName: "leaf.fill")
                        .font(.title3)
                        .foregroundStyle(AppTheme.textTertiary)
                }

                VStack(alignment: .leading, spacing: 3) {
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
                            .foregroundStyle(product.hasSaleOffer ? AppTheme.success : AppTheme.textPrimary)

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

                Image(systemName: "chevron.right")
                    .font(.caption2.weight(.semibold))
                    .foregroundStyle(AppTheme.textTertiary.opacity(0.5))
            }
            .padding(AppTheme.spacingMD)
            .pegasusMinimumHitTarget()
            .background(AppTheme.cardBackground)
            .clipShape(.rect(cornerRadius: AppTheme.radiusMD))
            .shadow(color: AppTheme.shadowColor, radius: 2, y: 1)
        }
        .buttonStyle(.pressable)
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(product.name), \(product.displayPrice)")
    }
}

#Preview {
    NavigationStack {
        SupplierProductsView(supplier: Supplier(id: "sup-001", name: "Fresh Farms", category: "Produce"))
            .environment(CartManager())
    }
}
