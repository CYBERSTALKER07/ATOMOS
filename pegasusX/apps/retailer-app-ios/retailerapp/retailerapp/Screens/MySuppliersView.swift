import SwiftUI

struct MySuppliersView: View {
    @State private var refreshCenter = RetailerRefreshCenter.shared
    @State private var vm = MySuppliersViewModel()

    private let columns = [GridItem(.adaptive(minimum: 160), spacing: 14)]

    var body: some View {
        @Bindable var vm = vm

        ScrollView {
            if vm.isLoading && vm.suppliers.isEmpty {
                skeletonGrid(cardCount: 6)
            } else if vm.suppliers.isEmpty && !vm.isLoading {
                emptyState
            } else {
                VStack(alignment: .leading, spacing: AppTheme.spacingLG) {
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
                            .contextMenu {
                                Button("mobile_retailer.ui.remove_vendor", role: .destructive) {
                                    vm.promptRemoval(for: supplier)
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
        .scrollDismissesKeyboard(.interactively)
        .scrollIndicators(.hidden)
        .background(AppTheme.background)
        .navigationTitle("mobile_retailer.ui.my_suppliers")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("mobile_retailer.ui.connect") {
                    Haptics.light()
                    vm.showConnectSheet = true
                }
                .pegasusMinimumHitTarget()
                .accessibilityLabel("Connect supplier")
            }
        }
        .sheet(isPresented: $vm.showConnectSheet) {
            NavigationStack {
                ConnectSupplierSheet(existingSuppliers: vm.suppliers) {
                    await vm.loadSuppliers()
                }
            }
        }
        .confirmationDialog(
            "Remove vendor?",
            isPresented: $vm.showRemoveConfirmation,
            titleVisibility: .visible
        ) {
            if let supplier = vm.supplierPendingRemoval {
                Button(L10n.format("mobile_retailer.ui.remove_name", "\(supplier.name)"), role: .destructive) {
                    Task { await vm.removeSupplier(supplier.id) }
                }
            }
            Button("common.action.cancel", role: .cancel) {
                vm.cancelRemoval()
            }
        } message: {
            if let supplier = vm.supplierPendingRemoval {
                Text(L10n.format("mobile_retailer.ui.name_will_be_removed_from_your_connected_suppliers", "\(supplier.name)"))
            }
        }
        .onChange(of: vm.showRemoveConfirmation) { _, isPresented in
            if !isPresented {
                vm.supplierPendingRemoval = nil
            }
        }
        .task { await vm.loadSuppliers() }
        .task(id: refreshCenter.refreshToken) { await vm.loadSuppliers() }
        .refreshable { await vm.loadSuppliers() }
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
                if vm.autoOrderSettings.supplierSettings[supplier.id] == true {
                    Image(systemName: "arrow.triangle.2.circlepath")
                        .font(.caption.weight(.semibold))
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
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(supplier.name)\(supplier.displayCategory.map { ", \($0)" } ?? ""), \(supplier.orderCount) orders")
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
                Task { await vm.loadSuppliers() }
            }
            .font(.system(.caption, design: .rounded, weight: .semibold))
            .foregroundStyle(AppTheme.accent)
            .pegasusMinimumHitTarget()
            .accessibilityLabel("Retry loading suppliers")
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
                Image(systemName: "building.2")
                    .font(.title)
                    .foregroundStyle(AppTheme.textTertiary)
            }
            .accessibilityHidden(true)

            Text("mobile_retailer.ui.no_suppliers_yet")
                .font(.system(.headline, design: .rounded))
                .foregroundStyle(AppTheme.textPrimary)
            Text(vm.errorMessage ?? "Connect vendors from the network or browse category catalogs.")
                .font(.system(.subheadline, design: .rounded))
                .foregroundStyle(AppTheme.textTertiary)
                .multilineTextAlignment(.center)
            Button(vm.errorMessage == nil ? "Connect vendor" : "Retry") {
                Haptics.medium()
                if vm.errorMessage == nil {
                    vm.showConnectSheet = true
                } else {
                    Task { await vm.loadSuppliers() }
                }
            }
            .font(.system(.subheadline, design: .rounded, weight: .semibold))
            .foregroundStyle(AppTheme.cardBackground)
            .padding(.horizontal, AppTheme.spacingLG)
            .padding(.vertical, AppTheme.spacingMD)
            .background(AppTheme.textPrimary)
            .clipShape(.rect(cornerRadius: AppTheme.radiusButton))
            .pegasusMinimumHitTarget()
            .accessibilityLabel(vm.errorMessage == nil ? "Connect vendor" : "Retry loading suppliers")
            Spacer()
        }
        .padding(AppTheme.spacingXL)
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
