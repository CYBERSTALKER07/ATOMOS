import SwiftUI

struct OrderLineItems: View {
    let lineItems: [LineItem]
    
    var body: some View {
        if !lineItems.isEmpty {
            Section("Line Items (\(lineItems.count))") {
                ForEach(lineItems) { item in
                    HStack {
                        VStack(alignment: .leading) {
                            Text(item.productName.isEmpty ? "Product" : item.productName)
                                .font(.headline)
                            Text("Qty: \(item.quantity)")
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                        Spacer()
                        Text("\(item.unitPrice.formatted()) UZS")
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                    }
                }
            }
        }
    }
}
