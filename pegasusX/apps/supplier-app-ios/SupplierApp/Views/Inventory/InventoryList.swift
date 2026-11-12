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
                TableColumn("mobile_supplier.chrome.product") { Text($0.productName) }
                TableColumn("mobile_supplier.chrome.qty") { quantityCell($0) }
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
            Button("mobile_supplier.chrome.decrease_quantity", systemImage: "minus.circle") {
                PegasusHaptics.light()
                Task { await onAdjustQuantity(item.sku, -1) }
            }
            .labelStyle(.iconOnly)
            .frame(minWidth: 44, minHeight: 44)
            .contentShape(Rectangle())
            .disabled(adjustingSku == item.sku)

            Text("\(item.quantity)")
                .font(.title3.bold())
                .frame(minWidth: 36)

            Button("mobile_supplier.chrome.increase_quantity", systemImage: "plus.circle") {
                PegasusHaptics.light()
                Task { await onAdjustQuantity(item.sku, 1) }
            }
            .labelStyle(.iconOnly)
            .frame(minWidth: 44, minHeight: 44)
            .contentShape(Rectangle())
            .disabled(adjustingSku == item.sku)
        }
    }
}
