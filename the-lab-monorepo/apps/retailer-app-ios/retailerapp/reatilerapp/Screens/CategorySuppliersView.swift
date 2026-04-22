import SwiftUI

struct CategorySuppliersView: View {
    let category: ProductCategory
    @State private var suppliers: [Supplier] = []
    @State private var isLoading = false
    @State private var errorMessage: String?
    @State private var mySupplierIds: Set<String> = []
    @State private var togglingIds: Set<String> = []

    private let api = APIClient.shared
    private let columns = [GridItem(.adaptive(minimum: 160), spacing: 14)]

    var body: some View {
        ScrollView {
            if isLoading && suppliers.isEmpty {
                VStack(alignment: .leading, spacing: AppTheme.spacingLG) {
                    headerSkeleton
                    skeletonGrid(cardCount: 4)
                }
                .padding(.horizontal, AppTheme.spacingLG)
                .padding(.top, AppTheme.spacingSM)
                .padding(.bottom, AppTheme.spacingXXL)
            } else if suppliers.isEmpty && !isLoading {
                emptyState
            } else {
                VStack(alignment: .leading, spacing: AppTheme.spacingLG) {
                    // Header
                    HStack(spacing: AppTheme.spacingMD) {
                        ZStack {
                            RoundedRectangle(cornerRadius: AppTheme.radiusMD)
                                .fill(AppTheme.surfaceElevated)
                                .frame(width: 52, height: 52)
                            Image(systemName: category.icon)
                                .font(.system(size: 24, weight: .medium))
                                .foregroundStyle(AppTheme.textPrimary)
                        }
                        VStack(alignment: .leading, spacing: 3) {
                            Text(category.name)
                                .font(.system(.title3, design: .rounded, weight: .bold))
                                .foregroundStyle(AppTheme.textPrimary)
                            Text("\(suppliers.count) suppliers available")
                                .font(.system(.caption, design: .rounded))
                                .foregroundStyle(AppTheme.textTertiary)
                        }
                        Spacer()
                    }
                    .slideIn(delay: 0)

                    // Supplier Cards
                    LazyVGrid(columns: columns, spacing: AppTheme.spacingLG) {
                        ForEach(Array(suppliers.enumerated()), id: \.element.id) { index, supplier in
                            NavigationLink {
                                SupplierProductsView(supplier: supplier)
                            } label: {
                                supplierCard(supplier)
                            }
                            .buttonStyle(.plain)
                            .staggeredSlideIn(index: index)
                        }
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
        .task { await loadSuppliers() }
        .refreshable { await loadSuppliers() }
    }

    // MARK: - Supplier Card

    private func supplierCard(_ supplier: Supplier) -> some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
            // Avatar + Add button
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

                // Add to My Suppliers
                Button {
                    guard !togglingIds.contains(supplier.id) else { return }
                    Haptics.medium()
                    withAnimation(AnimationConstants.bouncy) {
                        if mySupplierIds.contains(supplier.id) {
                            mySupplierIds.remove(supplier.id)
                        } else {
                            mySupplierIds.insert(supplier.id)
                        }
                    }
                    Task { await toggleMySupplier(supplier) }
                } label: {
                    Image(systemName: mySupplierIds.contains(supplier.id) ? "checkmark.circle.fill" : "plus.circle")
                        .font(.system(size: 22, weight: .medium))
                        .foregroundStyle(mySupplierIds.contains(supplier.id) ? AppTheme.success : AppTheme.textTertiary)
                        .contentTransition(.symbolEffect(.replace))
                }
                .disabled(togglingIds.contains(supplier.id))
            }

            // Name + category
            Text(supplier.name)
                .font(.system(.subheadline, design: .rounded, weight: .semibold))
                .foregroundStyle(AppTheme.textPrimary)
                .lineLimit(1)

            if let cat = supplier.displayCategory {
                Text(cat)
                    .font(.system(.caption2, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
            }

            // Stats
            HStack(spacing: AppTheme.spacingXS) {
                Text(supplier.catalogSubtitle)
                    .font(.system(.caption2, design: .rounded, weight: .medium))
                    .foregroundStyle(AppTheme.textSecondary)
                    .padding(.horizontal, 6).padding(.vertical, 3)
                    .background(AppTheme.surfaceElevated)
                    .clipShape(.capsule)

                if mySupplierIds.contains(supplier.id) {
                    Text("My Supplier")
                        .font(.system(.caption2, design: .rounded, weight: .bold))
                        .foregroundStyle(AppTheme.success)
                        .padding(.horizontal, 6).padding(.vertical, 3)
                        .background(AppTheme.success.opacity(0.1))
                        .clipShape(.capsule)
                }
            }
        }
        .padding(AppTheme.spacingMD)
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
        .shadow(color: AppTheme.shadowColor, radius: 4, x: 0, y: 2)
        .pressable()
    }

    // MARK: - Empty State

    private var emptyState: some View {
        VStack(spacing: AppTheme.spacingLG) {
            Spacer(minLength: 100)
            ZStack {
                Circle().fill(AppTheme.surfaceElevated).frame(width: 80, height: 80)
                Image(systemName: category.icon).font(.system(size: 32)).foregroundStyle(AppTheme.textTertiary)
            }
            Text("No Suppliers")
                .font(.system(.headline, design: .rounded))
                .foregroundStyle(AppTheme.textPrimary)
            Text(errorMessage ?? "No suppliers found for \(category.name)")
                .font(.system(.subheadline, design: .rounded))
                .foregroundStyle(AppTheme.textTertiary)
                .multilineTextAlignment(.center)
            Button(errorMessage == nil ? "Refresh" : "Retry") {
                Task { await loadSuppliers() }
            }
            .font(.system(.subheadline, design: .rounded, weight: .semibold))
            .foregroundStyle(AppTheme.cardBackground)
            .padding(.horizontal, AppTheme.spacingLG)
            .padding(.vertical, AppTheme.spacingMD)
            .background(AppTheme.textPrimary)
            .clipShape(.rect(cornerRadius: AppTheme.radiusButton))
            Spacer()
        }
        .padding(AppTheme.spacingXL)
    }

    // MARK: - API

    private func loadSuppliers() async {
        isLoading = true
        errorMessage = nil
        do {
            let result: [Supplier] = try await api.get(path: "/v1/catalog/categories/\(category.id)/suppliers")
            suppliers = result
        } catch {
            suppliers = []
            errorMessage = "Suppliers are unavailable right now. Check your connection and try again."
        }
        isLoading = false
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

    private func toggleMySupplier(_ supplier: Supplier) async {
        togglingIds.insert(supplier.id)
        let wasAdded = mySupplierIds.contains(supplier.id)
        do {
            let _: [String: Bool] = try await api.post(
                path: "/v1/retailer/suppliers/\(supplier.id)/\(wasAdded ? "add" : "remove")",
                body: ["supplier_id": supplier.id]
            )
        } catch {
            withAnimation(AnimationConstants.express) {
                if wasAdded { mySupplierIds.remove(supplier.id) }
                else { mySupplierIds.insert(supplier.id) }
            }
        }
        togglingIds.remove(supplier.id)
    }
}

#Preview {
    NavigationStack {
        CategorySuppliersView(category: ProductCategory.samples[0])
    }
}
