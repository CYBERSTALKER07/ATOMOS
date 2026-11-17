import SwiftUI

// MARK: - Aggregated Model for Shelf vs. Backroom

struct AggregatedStockItem: Identifiable, Equatable {
    var id: String { sku }
    let sku: String
    var floorOnHand: Int64 = 0
    var floorAvailable: Int64 = 0
    var backroomOnHand: Int64 = 0
    var backroomAvailable: Int64 = 0

    var totalOnHand: Int64 { floorOnHand + backroomOnHand }
    var totalAvailable: Int64 { floorAvailable + backroomAvailable }
    var isLowShelf: Bool { floorOnHand < 10 && backroomOnHand > 0 }
    var hasBackroom: Bool { backroomOnHand > 0 }

    var shelfRatio: Double {
        let totalCount = Double(totalOnHand)
        guard totalCount > 0 else { return 0 }
        return Double(floorOnHand) / totalCount
    }
}

enum StockFilterTab: String, CaseIterable, Identifiable {
    case all = "All Items"
    case lowShelf = "Low on Shelf"
    case backroomReserve = "In Backroom"

    var id: String { rawValue }
}

// MARK: - Row Data Model

struct StoreStockRow: Identifiable, Equatable {
    let id: String
    let sku: String
    let bin: String
    let onHand: Int64
    let available: Int64
}

extension TrackingOrder {
    func asClaimOrder() -> Order {
        let status = OrderStatus(rawValue: state) ?? .completed
        let lineItems = items.map { item in
            OrderLineItem(
                id: item.productId,
                productId: item.productId,
                productName: item.productName,
                variantId: item.productId,
                variantSize: "",
                quantity: item.quantity,
                unitPrice: item.unitPrice,
                totalPrice: item.lineTotal
            )
        }
        return Order(
            id: orderId,
            retailerId: "",
            supplierId: supplierId,
            supplierName: supplierName,
            status: status,
            items: lineItems,
            totalAmount: Int64(totalAmount),
            orderSource: orderSource,
            createdAt: createdAt,
            updatedAt: createdAt,
            estimatedDelivery: nil,
            qrCode: deliveryToken.isEmpty ? nil : deliveryToken
        )
    }
}

// MARK: - Main StoreStockView

struct StoreStockView: View {
    @State private var vm = StoreStockViewModel()

    var body: some View {
        @Bindable var vm = vm
        ScrollView {
            VStack(spacing: 16) {
                if let banner = vm.banner {
                    bannerView(banner)
                }

                if let loadError = vm.loadError, vm.rawRows.isEmpty {
                    RetailerErrorView(message: loadError) {
                        Task { await vm.load() }
                    }
                }

                kpiOverviewGrid
                actionShortcutsBar
                filterAndSearchBar

                if vm.filteredItems.isEmpty {
                    emptyStateView
                } else {
                    LazyVStack(spacing: 12) {
                        ForEach(vm.filteredItems) { item in
                            stockItemCard(item)
                        }
                    }
                }
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 12)
        }
        .scrollDismissesKeyboard(.interactively)
        .background(Color(.systemGroupedBackground))
        .navigationTitle("Store Inventory")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Menu {
                    Button {
                        Haptics.selection()
                        vm.showReceiveSheet = true
                    } label: {
                        Label("Receive Delivery", systemImage: "arrow.down.box")
                    }
                    Button {
                        Haptics.selection()
                        vm.selectedSkuForAction = nil
                        vm.showPutawaySheet = true
                    } label: {
                        Label("Putaway to Shelf", systemImage: "arrow.up.and.down.and.arrow.left.and.right")
                    }
                    Button {
                        Haptics.selection()
                        vm.selectedSkuForAction = nil
                        vm.showAdjustSheet = true
                    } label: {
                        Label("Adjust Stock Count", systemImage: "slider.horizontal.3")
                    }
                    Divider()
                    Button {
                        vm.openOrderPicker(skuHint: nil)
                    } label: {
                        Label("File Return / Claim", systemImage: "exclamationmark.triangle")
                    }
                } label: {
                    Image(systemName: "ellipsis.circle")
                        .font(.body)
                        .pegasusMinimumHitTarget()
                }
                .accessibilityLabel("Inventory actions")
            }
        }
        .task { await vm.load() }
        .refreshable { await vm.load() }
        .sheet(isPresented: $vm.showReceiveSheet) {
            ReceiveDeliverySheet(locationId: vm.locationId) {
                Task { await vm.load() }
            }
        }
        .sheet(isPresented: $vm.showPutawaySheet) {
            PutawaySheet(
                locationId: vm.locationId,
                initialSku: vm.selectedSkuForAction ?? "",
                availableSkus: vm.aggregatedItems.filter(\.hasBackroom).map(\.sku)
            ) {
                Task { await vm.load() }
            }
        }
        .sheet(isPresented: $vm.showAdjustSheet) {
            AdjustStockSheet(
                locationId: vm.locationId,
                initialSku: vm.selectedSkuForAction ?? "",
                allSkus: vm.aggregatedItems.map(\.sku)
            ) {
                Task { await vm.load() }
            }
        }
        .sheet(isPresented: $vm.showOrderPicker) {
            NavigationStack {
                StoreStockOrderPickerSheet(vm: vm)
            }
        }
        .sheet(item: $vm.claimOrder) { order in
            FileClaimView(order: order, preferredSku: vm.claimPreferredSku)
        }
    }

    // MARK: - Subviews

    private func bannerView(_ message: String) -> some View {
        HStack {
            Image(systemName: "checkmark.circle.fill")
                .foregroundStyle(.green)
            Text(message)
                .font(.subheadline)
                .foregroundStyle(AppTheme.textPrimary)
            Spacer()
            Button {
                Haptics.light()
                vm.banner = nil
            } label: {
                Image(systemName: "xmark")
                    .font(.caption2)
                    .foregroundStyle(AppTheme.textSecondary)
                    .pegasusMinimumHitTarget()
            }
            .accessibilityLabel("Dismiss banner")
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 6)
        .background(Color(.secondarySystemGroupedBackground), in: .rect(cornerRadius: 10))
    }

    private var kpiOverviewGrid: some View {
        LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 12) {
            kpiCard(title: "On Sales Floor", value: "\(vm.totalFloorOnHand)", subtitle: "Ready for shoppers", icon: "storefront", color: .blue)
            kpiCard(title: "In Backroom", value: "\(vm.totalBackroomOnHand)", subtitle: "Reserve storage", icon: "shippingbox", color: .indigo)
            kpiCard(title: "Total Units", value: "\(vm.totalStoreUnits)", subtitle: "\(vm.aggregatedItems.count) total SKUs", icon: "archivebox", color: .secondary)
            kpiCard(title: "Low on Shelf", value: "\(vm.lowShelfCount)", subtitle: "Need restock", icon: "exclamationmark.triangle", color: vm.lowShelfCount > 0 ? .orange : .green)
        }
    }

    private func kpiCard(title: String, value: String, subtitle: String, icon: String, color: Color) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack {
                Text(title)
                    .font(.caption.weight(.medium))
                    .foregroundStyle(AppTheme.textSecondary)
                Spacer()
                Image(systemName: icon)
                    .font(.caption)
                    .foregroundStyle(color)
            }
            Text(value)
                .font(.title2.weight(.bold))
                .foregroundStyle(AppTheme.textPrimary)
            Text(subtitle)
                .font(.caption2)
                .foregroundStyle(AppTheme.textSecondary)
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Color(.secondarySystemGroupedBackground), in: .rect(cornerRadius: 12))
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(title): \(value), \(subtitle)")
    }

    private var actionShortcutsBar: some View {
        HStack(spacing: 10) {
            Button {
                Haptics.selection()
                vm.showReceiveSheet = true
            } label: {
                Label("Receive", systemImage: "arrow.down.box")
                    .font(.subheadline.weight(.semibold))
                    .frame(maxWidth: .infinity, minHeight: AppTheme.minTouchTarget)
            }
            .buttonStyle(.borderedProminent)
            .tint(.accentColor)

            Button {
                Haptics.selection()
                vm.selectedSkuForAction = nil
                vm.showPutawaySheet = true
            } label: {
                Label("Putaway", systemImage: "arrow.up.right")
                    .font(.subheadline.weight(.medium))
                    .frame(maxWidth: .infinity, minHeight: AppTheme.minTouchTarget)
            }
            .buttonStyle(.bordered)

            Button {
                vm.openOrderPicker(skuHint: nil)
            } label: {
                Label("Return", systemImage: "arrow.counterclockwise")
                    .font(.subheadline.weight(.medium))
                    .frame(maxWidth: .infinity, minHeight: AppTheme.minTouchTarget)
            }
            .buttonStyle(.bordered)
        }
    }

    private var filterAndSearchBar: some View {
        @Bindable var vm = vm
        return VStack(spacing: 10) {
            HStack {
                Image(systemName: "magnifyingglass")
                    .foregroundStyle(.secondary)
                TextField("Search by SKU or name...", text: $vm.searchQuery)
                    .font(.subheadline)
                if !vm.searchQuery.isEmpty {
                    Button {
                        Haptics.light()
                        vm.searchQuery = ""
                    } label: {
                        Image(systemName: "xmark.circle.fill")
                            .foregroundStyle(.secondary)
                            .pegasusMinimumHitTarget()
                    }
                    .accessibilityLabel("Clear search")
                }
            }
            .padding(.horizontal, 12)
            .frame(minHeight: AppTheme.minTouchTarget)
            .background(Color(.secondarySystemGroupedBackground), in: .rect(cornerRadius: 10))

            HStack(spacing: 8) {
                ForEach(StockFilterTab.allCases) { tab in
                    Button {
                        Haptics.selection()
                        withAnimation(.snappy) {
                            vm.selectedFilter = tab
                        }
                    } label: {
                        Text(tab.rawValue)
                            .font(.caption.weight(vm.selectedFilter == tab ? .semibold : .regular))
                            .padding(.horizontal, 14)
                            .frame(minHeight: AppTheme.minTouchTarget)
                            .background(
                                vm.selectedFilter == tab ? Color.accentColor : Color(.secondarySystemGroupedBackground),
                                in: .capsule
                            )
                            .foregroundStyle(vm.selectedFilter == tab ? .white : AppTheme.textPrimary)
                    }
                    .buttonStyle(.plain)
                    .accessibilityLabel("Filter: \(tab.rawValue)")
                }
                Spacer()
            }
        }
    }

    private func stockItemCard(_ item: AggregatedStockItem) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            VStack(alignment: .leading, spacing: 10) {
                HStack {
                    VStack(alignment: .leading, spacing: 2) {
                        Text(item.sku)
                            .font(.headline)
                            .foregroundStyle(AppTheme.textPrimary)
                        Text("Total: \(item.totalOnHand) units")
                            .font(.caption2)
                            .foregroundStyle(AppTheme.textSecondary)
                    }
                    Spacer()
                    if item.isLowShelf {
                        HStack(spacing: 4) {
                            Image(systemName: "exclamationmark.circle.fill")
                            Text("Low Shelf")
                        }
                        .font(.caption2.weight(.bold))
                        .padding(.horizontal, 8)
                        .padding(.vertical, 4)
                        .background(Color.orange.opacity(0.15), in: .capsule)
                        .foregroundStyle(.orange)
                    }
                }

                VStack(spacing: 4) {
                    GeometryReader { geo in
                        HStack(spacing: 2) {
                            Rectangle()
                                .fill(Color.blue)
                                .frame(width: max(0, geo.size.width * CGFloat(item.shelfRatio)))
                            Rectangle()
                                .fill(Color.indigo.opacity(0.7))
                        }
                        .clipShape(.rect(cornerRadius: 4))
                    }
                    .frame(height: 8)

                    HStack {
                        HStack(spacing: 4) {
                            Circle().fill(Color.blue).frame(width: 6, height: 6)
                            Text("Shelf: \(item.floorOnHand)")
                                .font(.caption.weight(.semibold))
                                .foregroundStyle(AppTheme.textPrimary)
                        }
                        Spacer()
                        HStack(spacing: 4) {
                            Circle().fill(Color.indigo.opacity(0.7)).frame(width: 6, height: 6)
                            Text("Backroom: \(item.backroomOnHand)")
                                .font(.caption.weight(.semibold))
                                .foregroundStyle(AppTheme.textPrimary)
                        }
                    }
                }
            }
            .accessibilityElement(children: .combine)
            .accessibilityLabel("\(item.sku), Shelf \(item.floorOnHand), Backroom \(item.backroomOnHand), Total \(item.totalOnHand) units\(item.isLowShelf ? ", Low Shelf" : "")")

            Divider()

            HStack {
                if item.backroomOnHand > 0 {
                    Button {
                        let qty = min(10, item.backroomOnHand)
                        Task { await vm.quickPutaway(sku: item.sku, qtyToMove: qty) }
                    } label: {
                        HStack(spacing: 4) {
                            Image(systemName: "plus.circle.fill")
                            Text("+\(min(10, item.backroomOnHand)) to Shelf")
                        }
                        .font(.caption.weight(.semibold))
                        .frame(minHeight: AppTheme.minTouchTarget)
                    }
                    .buttonStyle(.bordered)
                    .tint(.blue)
                    .disabled(vm.busy)
                } else {
                    Text("No backroom reserve")
                        .font(.caption2)
                        .foregroundStyle(AppTheme.textSecondary)
                }

                Spacer()

                Menu {
                    Button {
                        Haptics.selection()
                        vm.selectedSkuForAction = item.sku
                        vm.showPutawaySheet = true
                    } label: {
                        Label("Custom Putaway...", systemImage: "arrow.up.and.down.and.arrow.left.and.right")
                    }
                    Button {
                        Haptics.selection()
                        vm.selectedSkuForAction = item.sku
                        vm.showAdjustSheet = true
                    } label: {
                        Label("Adjust Count...", systemImage: "slider.horizontal.3")
                    }
                    Button {
                        vm.openOrderPicker(skuHint: item.sku)
                    } label: {
                        Label("Return / Claim...", systemImage: "exclamationmark.triangle")
                    }
                } label: {
                    Image(systemName: "ellipsis")
                        .font(.subheadline)
                        .foregroundStyle(AppTheme.textSecondary)
                        .pegasusMinimumHitTarget()
                }
                .accessibilityLabel("More actions for \(item.sku)")
            }
        }
        .padding(14)
        .background(Color(.secondarySystemGroupedBackground), in: .rect(cornerRadius: 14))
    }

    private var emptyStateView: some View {
        VStack(spacing: 12) {
            Image(systemName: "shippingbox.fill")
                .font(.largeTitle)
                .foregroundStyle(.secondary)
            Text("No Inventory Matches")
                .font(.headline)
            Text("Try clearing your search filter or receive a new incoming delivery order.")
                .font(.subheadline)
                .foregroundStyle(.secondary)
                .multilineTextAlignment(.center)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 40)
        .accessibilityElement(children: .combine)
    }
}
