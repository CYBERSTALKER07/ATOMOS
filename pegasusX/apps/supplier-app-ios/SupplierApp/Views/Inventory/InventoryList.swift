import SwiftUI

struct InventoryList: View {
    let items: [InventoryItem]
    let adjustingSku: String?
    let onAdjustQuantity: (String, Int64) async -> Void

    @Environment(\.horizontalSizeClass) private var horizontalSizeClass

    var body: some View {
        if horizontalSizeClass == .regular {
            Table(items) {
                TableColumn("SKU") { Text($0.sku).font(.body.monospaced()) }
                TableColumn("Product") { Text($0.productName) }
                TableColumn("Qty") { quantityCell($0) }
            }
            .supplierReadableWidth()
            .padding()
        } else {
            ResponsiveGridContentWrapper {
                ForEach(items) { item in
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
            }
        }
    }

    @ViewBuilder
    private func quantityCell(_ item: InventoryItem) -> some View {
        HStack(spacing: SupplierTheme.spacingSM) {
            Button {
                Task { await onAdjustQuantity(item.sku, -1) }
            } label: {
                Image(systemName: "minus.circle")
            }
            .disabled(adjustingSku == item.sku)

            Text("\(item.quantity)")
                .font(.title3.bold())
                .frame(minWidth: 36)

            Button {
                Task { await onAdjustQuantity(item.sku, 1) }
            } label: {
                Image(systemName: "plus.circle")
            }
            .disabled(adjustingSku == item.sku)
        }
    }
}
