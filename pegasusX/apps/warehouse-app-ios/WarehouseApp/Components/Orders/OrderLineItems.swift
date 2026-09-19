import SwiftUI

struct OrderLineItems: View {
    let lineItems: [LineItem]
    
    var body: some View {
        if !lineItems.isEmpty {
            Section(L10n.format("mobile_warehouse.ui.line_items_count", "\(lineItems.count)")) {
                ForEach(lineItems) { item in
                    HStack {
                        VStack(alignment: .leading) {
                            Text(item.productName.isEmpty ? "Product" : item.productName)
                                .font(.headline)
                            Text(L10n.format("mobile_warehouse.ui.qty_quantity", "\(item.quantity)"))
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                        Spacer()
                        Text(L10n.format("mobile_warehouse.ui.formatted_uzs", "\(item.unitPrice.formatted())"))
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                    }
                }
            }
        }
    }
}
