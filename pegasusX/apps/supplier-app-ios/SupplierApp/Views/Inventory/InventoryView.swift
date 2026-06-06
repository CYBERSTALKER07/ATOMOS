import SwiftUI

struct InventoryView: View {
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @State private var items: [InventoryItem] = []
    @State private var loading = true
    @State private var error: String?
    @State private var query = ""

    private var filtered: [InventoryItem] {
        guard !query.isEmpty else { return items }
        return items.filter {
            $0.sku.localizedCaseInsensitiveContains(query)
                || $0.productName.localizedCaseInsensitiveContains(query)
        }
    }

    var body: some View {
        NavigationStack {
            Group {
                if loading {
                    SupplierLoadingView(title: "Loading inventory…")
                } else if let error {
                    SupplierErrorView(message: error) { Task { await load() } }
                } else if filtered.isEmpty {
                    SupplierEmptyView(
                        title: "No SKUs",
                        message: query.isEmpty ? "Inventory will appear when stock is registered." : "No matches for \"\(query)\"."
                    )
                } else {
                    inventoryTable
                }
            }
            .background(SupplierTheme.background)
            .navigationTitle("Inventory")
            .searchable(text: $query, prompt: "SKU or product")
            .task { await load() }
            .refreshable { await load(silent: true) }
        }
    }

    @ViewBuilder
    private var inventoryTable: some View {
        if horizontalSizeClass == .regular {
            Table(filtered) {
                TableColumn("SKU") { Text($0.sku).font(.body.monospaced()) }
                TableColumn("Product") { Text($0.productName) }
                TableColumn("Qty") { Text("\($0.quantity)") }
            }
            .supplierReadableWidth()
            .padding()
        } else {
            List(filtered) { item in
                HStack {
                    VStack(alignment: .leading) {
                        Text(item.productName)
                            .font(.headline)
                        Text(item.sku)
                            .font(.caption.monospaced())
                            .foregroundStyle(.secondary)
                    }
                    Spacer()
                    Text("\(item.quantity)")
                        .font(.title3.bold())
                }
            }
            .listStyle(.insetGrouped)
        }
    }

    @MainActor
    private func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        do {
            items = try await SupplierService.inventory()
        } catch {
            if !silent { self.error = error.localizedDescription }
        }
        loading = false
    }
}
