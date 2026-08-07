import SwiftUI

struct MySuppliersView: View {
    @State private var refreshCenter = RetailerRefreshCenter.shared
    @State private var suppliers: [Supplier] = []
    @State private var isLoading = false
    @State private var errorMessage: String?
    @State private var autoOrderSettings = SimpleAutoOrderSettings.default
    @State private var showConnectSheet = false
    @State private var supplierPendingRemoval: Supplier?

    private let api = APIClient.shared
    private let columns = [GridItem(.adaptive(minimum: 160), spacing: 14)]

    var body: some View {
        ScrollView {
            if isLoading && suppliers.isEmpty {
                skeletonGrid(cardCount: 6)
            } else if suppliers.isEmpty && !isLoading {
                emptyState
            } else {
                VStack(alignment: .leading, spacing: AppTheme.spacingLG) {
                    if let errorMessage, !errorMessage.isEmpty {
                        syncStatusBanner(errorMessage)
                    }

                    LazyVGrid(columns: columns, spacing: AppTheme.spacingLG) {
                        ForEach(Array(suppliers.enumerated()), id: \.element.id) { index, supplier in
                            NavigationLink {
                                SupplierProductsView(supplier: supplier)
                            } label: {
                                supplierCard(supplier)
                            }
                            .buttonStyle(.plain)
                            .contextMenu {
                                Button("mobile_retailer.ui.remove_vendor", role: .destructive) {
                                    supplierPendingRemoval = supplier
                                }
                            }
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
        .navigationTitle("mobile_retailer.ui.my_suppliers")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("mobile_retailer.ui.connect") { showConnectSheet = true }
            }
        }
        .sheet(isPresented: $showConnectSheet) {
            ConnectSupplierSheet(existingSuppliers: suppliers) {
                await loadSuppliers()
            }
        }
        .confirmationDialog(
            "Remove vendor?",
            isPresented: Binding(
                get: { supplierPendingRemoval != nil },
                set: { if !$0 { supplierPendingRemoval = nil } }
            ),
            titleVisibility: .visible
        ) {
            if let supplier = supplierPendingRemoval {
                Button(L10n.format("mobile_retailer.ui.remove_name", "\(supplier.name)"), role: .destructive) {
                    Task { await removeSupplier(supplier.id) }
                }
            }
            Button("common.action.cancel", role: .cancel) { supplierPendingRemoval = nil }
        } message: {
            if let supplier = supplierPendingRemoval {
                Text(L10n.format("mobile_retailer.ui.name_will_be_removed_from_your_connected_suppliers", "\(supplier.name)"))
            }
        }
        .task { await loadSuppliers() }
        .task(id: refreshCenter.refreshToken) { await loadSuppliers() }
        .refreshable { await loadSuppliers() }
    }

    private func supplierCard(_ supplier: Supplier) -> some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
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

            Text(supplier.name)
                .font(.system(.subheadline, design: .rounded, weight: .semibold))
                .foregroundStyle(AppTheme.textPrimary)
                .lineLimit(1)

            if let category = supplier.displayCategory {
                Text(category)
                    .font(.system(.caption2, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
            }

            HStack(spacing: AppTheme.spacingXS) {
                Text(L10n.format("mobile_retailer.ui.ordercount_orders_2", "\(supplier.orderCount)"))
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

    private func syncStatusBanner(_ message: String) -> some View {
        HStack(spacing: AppTheme.spacingSM) {
            Image(systemName: "exclamationmark.triangle.fill")
                .font(.system(size: 12, weight: .semibold))
                .foregroundStyle(AppTheme.warning)
            Text(message)
                .font(.system(.caption, design: .rounded, weight: .medium))
                .foregroundStyle(AppTheme.textSecondary)
                .lineLimit(3)
            Spacer()
            Button("common.action.retry") {
                Task { await loadSuppliers() }
            }
            .font(.system(.caption, design: .rounded, weight: .semibold))
            .foregroundStyle(AppTheme.accent)
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
                Image(systemName: "building.2").font(.system(size: 32)).foregroundStyle(AppTheme.textTertiary)
            }
            Text("mobile_retailer.ui.no_suppliers_yet")
                .font(.system(.headline, design: .rounded))
                .foregroundStyle(AppTheme.textPrimary)
            Text(errorMessage ?? "Connect vendors from the network or browse category catalogs.")
                .font(.system(.subheadline, design: .rounded))
                .foregroundStyle(AppTheme.textTertiary)
                .multilineTextAlignment(.center)
            Button(errorMessage == nil ? "Connect vendor" : "Retry") {
                if errorMessage == nil {
                    showConnectSheet = true
                } else {
                    Task { await loadSuppliers() }
                }
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

    private func loadSuppliers() async {
        isLoading = true
        errorMessage = nil
        do {
            let result: [Supplier] = try await api.get(path: "/v1/retailer/suppliers")
            suppliers = result
        } catch {
            errorMessage = RetailerErrorSupport.message(
                for: error,
                restricted: "My suppliers access is restricted for this account.",
                offline: "Offline mode active. Showing latest supplier list.",
                fallback: "Supplier list could not load. Check your connection and pull to refresh.",
            )
        }
        isLoading = false
    }

    private func removeSupplier(_ supplierId: String) async {
        do {
            try await RetailerSupplierDiscoveryService.removeSupplier(api: api, supplierId: supplierId)
            supplierPendingRemoval = nil
            await loadSuppliers()
        } catch {
            errorMessage = error.localizedDescription
            supplierPendingRemoval = nil
        }
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
}

#Preview {
    NavigationStack {
        MySuppliersView()
    }
}
