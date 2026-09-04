import SwiftUI

struct InventoryView: View {
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    @State private var vm = InventoryViewModel()

    var body: some View {
        NavigationStack {
            Group {
                if vm.loading && vm.items.isEmpty {
                    SupplierLoadingView(title: "Loading inventory…")
                } else if let error = vm.error {
                    SupplierErrorView(message: error) { Task { await vm.load() } }
                } else if vm.filtered.isEmpty {
                    SupplierEmptyView(
                        title: "No SKUs",
                        message: vm.query.isEmpty ? "Inventory will appear when stock is registered." : "No matches for \"\(vm.query)\"."
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
            .background(SupplierTheme.background)
            .navigationTitle("portal.nav.inventory")
            .searchable(text: $vm.query, prompt: "SKU or product")
            .task { await vm.load() }
            .refreshable { await vm.load(silent: true) }
            .silentRealtimeRefresh(
                refreshEpoch: realtimeHub.refreshEpoch,
                reconnectEpoch: realtimeHub.reconnectEpoch
            ) { silent in
                Task { await vm.load(silent: silent) }
            }
        }
    }
}

