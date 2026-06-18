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
                    inventoryTable
                }
            }
            .background(SupplierTheme.background)
            .navigationTitle("Inventory")
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

    @ViewBuilder
    private var inventoryTable: some View {
        if horizontalSizeClass == .regular {
            Table(vm.filtered) {
                TableColumn("SKU") { Text($0.sku).font(.body.monospaced()) }
                TableColumn("Product") { Text($0.productName) }
                TableColumn("Qty") { quantityCell($0) }
            }
            .supplierReadableWidth()
            .padding()
        } else {
            List(vm.filtered) { item in
                HStack {
                    VStack(alignment: .leading) {
                        Text(item.productName)
                            .font(.headline)
                        Text(item.sku)
                            .font(.caption.monospaced())
                            .foregroundStyle(.secondary)
                    }
                    Spacer()
                    quantityCell(item)
                }
            }
            .listStyle(.insetGrouped)
        }
    }

    @ViewBuilder
    private func quantityCell(_ item: InventoryItem) -> some View {
        HStack(spacing: SupplierTheme.spacingSM) {
            Button {
                Task { await vm.adjustQuantity(sku: item.sku, delta: -1) }
            } label: {
                Image(systemName: "minus.circle")
            }
            .disabled(vm.adjustingSku == item.sku)

            Text("\(item.quantity)")
                .font(.title3.bold())
                .frame(minWidth: 36)

            Button {
                Task { await vm.adjustQuantity(sku: item.sku, delta: 1) }
            } label: {
                Image(systemName: "plus.circle")
            }
            .disabled(vm.adjustingSku == item.sku)
        }
    }
}
