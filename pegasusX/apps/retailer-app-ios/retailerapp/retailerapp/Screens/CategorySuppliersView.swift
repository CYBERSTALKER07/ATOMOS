import SwiftUI

struct CategorySuppliersView: View {
    let category: ProductCategory
    @State private var refreshCenter = RetailerRefreshCenter.shared
    @State private var vm = CategorySuppliersViewModel()

    private let columns = [GridItem(.adaptive(minimum: 160), spacing: 14)]

    var body: some View {
        ScrollView {
            if vm.isLoading && vm.suppliers.isEmpty {
                VStack(alignment: .leading, spacing: AppTheme.spacingLG) {
                    headerSkeleton
                    skeletonGrid(cardCount: 4)
                }
                .padding(.horizontal, AppTheme.spacingLG)
                .padding(.top, AppTheme.spacingSM)
                .padding(.bottom, AppTheme.spacingXXL)
            } else if vm.suppliers.isEmpty && !vm.isLoading {
                emptyState
            } else {
                VStack(alignment: .leading, spacing: AppTheme.spacingLG) {
                    HStack(spacing: AppTheme.spacingMD) {
                        ZStack {
                            RoundedRectangle(cornerRadius: AppTheme.radiusMD)
                                .fill(AppTheme.surfaceElevated)
                                .frame(width: 52, height: 52)
                            Image(systemName: category.icon)
                                .font(.title2.weight(.medium))
                                .foregroundStyle(AppTheme.textPrimary)
                        }
                        VStack(alignment: .leading, spacing: 3) {
                            Text(category.name)
                                .font(.system(.title3, design: .rounded, weight: .bold))
                                .foregroundStyle(AppTheme.textPrimary)
                            Text(L10n.format("mobile_retailer.ui.count_suppliers_available", "\(vm.suppliers.count)"))
                                .font(.system(.caption, design: .rounded))
                                .foregroundStyle(AppTheme.textTertiary)
                        }
                        Spacer()
                    }
                    .accessibilityElement(children: .combine)
                    .accessibilityLabel("\(category.name), \(vm.suppliers.count) suppliers available")
                    .slideIn(delay: 0)

                    if let errorMessage = vm.errorMessage, !errorMessage.isEmpty {
                        syncStatusBanner(errorMessage)
                    }

                    LazyVGrid(columns: columns, spacing: AppTheme.spacingLG) {
                        ForEach(Array(vm.suppliers.enumerated()), id: \.element.id) { index, supplier in
                            NavigationLink {
                                SupplierProductsView(supplier: supplier)
                            } label: {
                                supplierCard(supplier)
                            }
                            .buttonStyle(.pressable)
                            .staggeredSlideIn(index: index)
                        }
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
        .task { await vm.loadSuppliers(categoryId: category.id) }
        .task(id: refreshCenter.refreshToken) { await vm.loadSuppliers(categoryId: category.id) }
        .refreshable { await vm.loadSuppliers(categoryId: category.id) }
    }

    // MARK: - Supplier Card

    private func supplierCard(_ supplier: Supplier) -> some View {
        let isAdded = vm.mySupplierIds.contains(supplier.id)

        return VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
            HStack {
                ZStack {
                    RoundedRectangle(cornerRadius: AppTheme.radiusMD)
                        .fill(AppTheme.surfaceElevated)
                        .frame(width: 48, height: 48)
                    Text(supplier.initials)
                        .font(.system(.subheadline, design: .rounded, weight: .bold))
                        .foregroundStyle(AppTheme.textSecondary)
                }
                Spacer()

                Button {
                    Task { await vm.toggleMySupplier(supplier) }
                } label: {
                    Image(systemName: isAdded ? "checkmark.circle.fill" : "plus.circle")
                        .font(.title3.weight(.medium))
                        .foregroundStyle(isAdded ? AppTheme.success : AppTheme.textTertiary)
                        .contentTransition(.symbolEffect(.replace))
                        .pegasusMinimumHitTarget()
                }
                .disabled(vm.togglingIds.contains(supplier.id))
                .accessibilityLabel(isAdded ? "Remove \(supplier.name) from My Suppliers" : "Add \(supplier.name) to My Suppliers")
            }

            VStack(alignment: .leading, spacing: AppTheme.spacingXS) {
                Text(supplier.name)
                    .font(.system(.subheadline, design: .rounded, weight: .semibold))
                    .foregroundStyle(AppTheme.textPrimary)
                    .lineLimit(1)

                if let cat = supplier.displayCategory {
                    Text(cat)
                        .font(.system(.caption2, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)
                }

                HStack(spacing: AppTheme.spacingXS) {
                    Text(supplier.catalogSubtitle)
                        .font(.system(.caption2, design: .rounded, weight: .medium))
                        .foregroundStyle(AppTheme.textSecondary)
                        .padding(.horizontal, 6).padding(.vertical, 3)
                        .background(AppTheme.surfaceElevated)
                        .clipShape(.capsule)

                    if isAdded {
                        Text("mobile_retailer.ui.my_supplier")
                            .font(.system(.caption2, design: .rounded, weight: .bold))
                            .foregroundStyle(AppTheme.success)
                            .padding(.horizontal, 6).padding(.vertical, 3)
                            .background(AppTheme.success.opacity(0.1))
                            .clipShape(.capsule)
                    }
                }
            }
            .accessibilityElement(children: .combine)
            .accessibilityLabel("\(supplier.name), \(supplier.catalogSubtitle)\(isAdded ? ", My Supplier" : "")")
        }
        .padding(AppTheme.spacingMD)
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
        .shadow(color: AppTheme.shadowColor, radius: 4, x: 0, y: 2)
    }

    // MARK: - Empty State & Banner

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
                Task { await vm.loadSuppliers(categoryId: category.id) }
            }
            .font(.system(.caption, design: .rounded, weight: .semibold))
            .foregroundStyle(AppTheme.accent)
            .pegasusMinimumHitTarget()
            .accessibilityLabel("Retry loading category suppliers")
        }
        .padding(.horizontal, AppTheme.spacingMD)
        .padding(.vertical, AppTheme.spacingSM)
        .background(AppTheme.warning.opacity(0.12))
        .clipShape(.rect(cornerRadius: AppTheme.radiusMD))
    }

    private var emptyState: some View {
        VStack(spacing: AppTheme.spacingLG) {
            Spacer(minLength: 100)
            ZStack {
                Circle().fill(AppTheme.surfaceElevated).frame(width: 80, height: 80)
                Image(systemName: category.icon)
                    .font(.title)
                    .foregroundStyle(AppTheme.textTertiary)
            }
            .accessibilityHidden(true)

            Text("mobile_retailer.ui.no_suppliers")
                .font(.system(.headline, design: .rounded))
                .foregroundStyle(AppTheme.textPrimary)
            Text(vm.errorMessage ?? "No suppliers found for \(category.name)")
                .font(.system(.subheadline, design: .rounded))
                .foregroundStyle(AppTheme.textTertiary)
                .multilineTextAlignment(.center)
            Button(vm.errorMessage == nil ? "Refresh" : "Retry") {
                Haptics.light()
                Task { await vm.loadSuppliers(categoryId: category.id) }
            }
            .font(.system(.subheadline, design: .rounded, weight: .semibold))
            .foregroundStyle(AppTheme.cardBackground)
            .padding(.horizontal, AppTheme.spacingLG)
            .padding(.vertical, AppTheme.spacingMD)
            .background(AppTheme.textPrimary)
            .clipShape(.rect(cornerRadius: AppTheme.radiusButton))
            .pegasusMinimumHitTarget()
            .accessibilityLabel(vm.errorMessage == nil ? "Refresh suppliers" : "Retry loading suppliers")
            Spacer()
        }
        .padding(AppTheme.spacingXL)
    }

    private var headerSkeleton: some View {
        HStack(spacing: AppTheme.spacingMD) {
            RoundedRectangle(cornerRadius: AppTheme.radiusMD)
                .fill(AppTheme.surfaceElevated)
                .frame(width: 52, height: 52)
            VStack(alignment: .leading, spacing: 6) {
                RoundedRectangle(cornerRadius: 6)
                    .fill(AppTheme.surfaceElevated)
                    .frame(width: 140, height: 18)
                RoundedRectangle(cornerRadius: 6)
                    .fill(AppTheme.surfaceElevated)
                    .frame(width: 110, height: 12)
            }
            Spacer()
        }
        .skeleton()
    }

    private func skeletonGrid(cardCount: Int) -> some View {
        LazyVGrid(columns: columns, spacing: AppTheme.spacingLG) {
            ForEach(0..<cardCount, id: \.self) { _ in
                VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
                    HStack {
                        RoundedRectangle(cornerRadius: AppTheme.radiusMD)
                            .fill(AppTheme.surfaceElevated)
                            .frame(width: 48, height: 48)
                        Spacer()
                        Circle()
                            .fill(AppTheme.surfaceElevated)
                            .frame(width: 22, height: 22)
                    }
                    RoundedRectangle(cornerRadius: 6)
                        .fill(AppTheme.surfaceElevated)
                        .frame(height: 14)
                    RoundedRectangle(cornerRadius: 6)
                        .fill(AppTheme.surfaceElevated)
                        .frame(width: 88, height: 10)
                    HStack(spacing: AppTheme.spacingXS) {
                        RoundedRectangle(cornerRadius: 999)
                            .fill(AppTheme.surfaceElevated)
                            .frame(width: 90, height: 20)
                        RoundedRectangle(cornerRadius: 999)
                            .fill(AppTheme.surfaceElevated)
                            .frame(width: 72, height: 20)
                    }
                }
                .padding(AppTheme.spacingMD)
                .background(AppTheme.cardBackground)
                .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
                .shadow(color: AppTheme.shadowColor, radius: 4, x: 0, y: 2)
                .skeleton()
            }
        }
    }
}

#Preview {
    NavigationStack {
        CategorySuppliersView(category: ProductCategory(id: "cat-dairy", name: "Dairy", icon: "cup.and.saucer.fill", productCount: 0))
    }
}
