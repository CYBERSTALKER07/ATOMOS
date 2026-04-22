import SwiftUI

struct MySuppliersView: View {
    @State private var suppliers: [Supplier] = []
    @State private var isLoading = false
    @State private var errorMessage: String?
    @State private var autoOrderSettings = SimpleAutoOrderSettings.default

    private let api = APIClient.shared
    private let columns = [GridItem(.adaptive(minimum: 160), spacing: 14)]

    var body: some View {
        ScrollView {
            if isLoading && suppliers.isEmpty {
                skeletonGrid(cardCount: 6)
            } else if suppliers.isEmpty && !isLoading {
                emptyState
            } else {
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
                .padding(.horizontal, AppTheme.spacingLG)
                .padding(.top, AppTheme.spacingSM)
                .padding(.bottom, AppTheme.spacingXXL)
            }
        }
        .scrollIndicators(.hidden)
        .background(AppTheme.background)
        .task { await loadSuppliers() }
        .refreshable { await loadSuppliers() }
    }

    // MARK: - Supplier Card

    private func supplierCard(_ supplier: Supplier) -> some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
            // Avatar
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
                if autoOrderSettings.supplierSettings[supplier.id] == true {
                    Image(systemName: "arrow.triangle.2.circlepath")
                        .font(.system(size: 12, weight: .semibold))
                        .foregroundStyle(AppTheme.success)
                }
            }

            // Info
            Text(supplier.name)
                .font(.system(.subheadline, design: .rounded, weight: .semibold))
                .foregroundStyle(AppTheme.textPrimary)
                .lineLimit(1)

            if let category = supplier.displayCategory {
                Text(category)
                    .font(.system(.caption2, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
            }

            // Stats
            HStack(spacing: AppTheme.spacingXS) {
                Text("\(supplier.orderCount) orders")
                    .font(.system(.caption2, design: .rounded, weight: .medium))
                    .foregroundStyle(AppTheme.textSecondary)
                    .padding(.horizontal, 6).padding(.vertical, 3)
                    .background(AppTheme.surfaceElevated)
                    .clipShape(.capsule)
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
                Image(systemName: "building.2").font(.system(size: 32)).foregroundStyle(AppTheme.textTertiary)
            }
            Text("No Suppliers Yet")
                .font(.system(.headline, design: .rounded))
                .foregroundStyle(AppTheme.textPrimary)
            Text(errorMessage ?? "Suppliers with repeated orders will appear here automatically")
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
            let result: [Supplier] = try await api.get(path: "/v1/retailer/suppliers")
            suppliers = result
        } catch {
            suppliers = []
            errorMessage = "Supplier list could not load. Check your connection and pull to refresh."
        }
        isLoading = false
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
                            .frame(width: 12, height: 12)
                    }
                    RoundedRectangle(cornerRadius: 6)
                        .fill(AppTheme.surfaceElevated)
                        .frame(height: 14)
                    RoundedRectangle(cornerRadius: 6)
                        .fill(AppTheme.surfaceElevated)
                        .frame(width: 84, height: 10)
                    RoundedRectangle(cornerRadius: 999)
                        .fill(AppTheme.surfaceElevated)
                        .frame(width: 92, height: 20)
                }
                .padding(AppTheme.spacingMD)
                .background(AppTheme.cardBackground)
                .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
                .shadow(color: AppTheme.shadowColor, radius: 4, x: 0, y: 2)
                .skeleton()
            }
        }
        .padding(.horizontal, AppTheme.spacingLG)
        .padding(.top, AppTheme.spacingSM)
        .padding(.bottom, AppTheme.spacingXXL)
    }

    private func toggleSupplierAutoOrder(supplierId: String, enabled: Bool) async {
        do {
            let _: [String: Bool] = try await api.patch(
                path: "/v1/retailer/settings/auto-order/supplier/\(supplierId)",
                body: ["auto_order_enabled": enabled]
            )
        } catch {}
    }
}

#Preview {
    NavigationStack {
        MySuppliersView()
    }
}
