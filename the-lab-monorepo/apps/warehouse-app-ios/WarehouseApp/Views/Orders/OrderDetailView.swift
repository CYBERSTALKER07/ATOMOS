import SwiftUI

struct OrderDetailView: View {
    let orderId: String
    @State private var order: Order?
    @State private var loading = true
    @State private var error: String?

    var body: some View {
        Group {
            if loading {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let error {
                ContentUnavailableView {
                    Label("Error", systemImage: "exclamationmark.triangle")
                } description: {
                    Text(error)
                } actions: {
                    Button("Retry") { load() }
                }
            } else if let order {
                List {
                    Section("Summary") {
                        LabeledContent("State", value: order.state)
                        LabeledContent("Total", value: "\(order.totalUzs.formatted()) UZS")
                        LabeledContent("Retailer", value: order.retailerName.isEmpty ? "—" : order.retailerName)
                    }
                    Section("Line Items (\(order.lineItems.count))") {
                        ForEach(order.lineItems) { item in
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
                .listStyle(.insetGrouped)
            }
        }
        .navigationTitle("Order Detail")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("Refresh", systemImage: "arrow.clockwise") { load() }
            }
        }
        .task { load() }
    }

    private func load() {
        loading = true
        error = nil
        Task {
            do {
                order = try await WarehouseService.order(id: orderId)
            } catch {
                self.error = error.localizedDescription
            }
            loading = false
        }
    }
}
