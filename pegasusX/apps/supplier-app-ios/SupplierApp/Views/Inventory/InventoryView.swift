import SwiftUI

struct InventoryView: View {
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    @State private var vm = InventoryViewModel()

    var body: some View {
        Group {
            if vm.loading && vm.items.isEmpty {
                SupplierLoadingView(title: "mobile_supplier.loading.inventory")
            } else if let error = vm.error, vm.items.isEmpty {
                SupplierErrorView(message: error) { Task { await vm.load() } }
            } else if vm.filtered.isEmpty {
                SupplierEmptyView(
                    title: "mobile_supplier.empty.no_skus",
                    message: vm.query.isEmpty
                        ? "mobile_supplier.empty.inventory_empty"
                        : L10n.format("mobile_supplier.empty.no_products_match", vm.query)
                )
            } else {
                InventoryList(
                    items: vm.filtered,
                    adjustingSku: vm.adjustingSku,
                    onAdjustQuantity: { sku, delta in
                        await vm.adjustQuantity(sku: sku, delta: delta)
                    }
                )
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .background(SupplierTheme.background)
        .navigationTitle("portal.nav.inventory")
        .searchable(text: $vm.query, prompt: "mobile_supplier.chrome.sku_or_product")
        .task { await vm.load() }
        .refreshable { await loadSilent() }
        .silentRealtimeRefresh(
            refreshEpoch: realtimeHub.refreshEpoch,
            reconnectEpoch: realtimeHub.reconnectEpoch
        ) { silent in
            Task { await vm.load(silent: silent) }
        }
    }

    private func loadSilent() async {
        PegasusHaptics.light()
        await vm.load(silent: true)
    }
}

