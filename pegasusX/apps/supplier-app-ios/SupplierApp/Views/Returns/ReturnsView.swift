import SwiftUI

struct ReturnsView: View {
    @State private var orders: [SupplierOrder] = []
    @State private var loading = true
    @State private var error: String?

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading returns…")
            } else if let error {
                SupplierErrorView(message: error) { Task { await load() } }
            } else if orders.isEmpty {
                SupplierEmptyView(
                    title: "No returns",
                    message: "No cancelled or rejected orders in the current window."
                )
            } else {
                List(orders) { order in
                    VStack(alignment: .leading, spacing: 4) {
                        Text(order.orderId).font(.headline)
                        Text(order.status).font(.subheadline)
                        if let decision = order.decision {
                            Text(decision).font(.caption).foregroundStyle(.secondary)
                        }
                        if let note = order.note {
                            Text(note).font(.caption)
                        }
                    }
                }
                .listStyle(.insetGrouped)
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("Returns")
        .task { await load() }
    }

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let resp = try await SupplierOperationsService.orders(filter: "RETURNS", limit: 100, offset: 0)
            orders = resp.orders
        } catch {
            self.error = error.localizedDescription
        }
    }
}
