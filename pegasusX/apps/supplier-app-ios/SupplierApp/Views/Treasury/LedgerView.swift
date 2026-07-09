import SwiftUI

struct LedgerView: View {
    @State private var rows: [PaymentLedgerEntry] = []
    @State private var loading = true
    @State private var error: String?

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading ledger…")
            } else if let error {
                SupplierErrorView(message: error) { Task { await load() } }
            } else if rows.isEmpty {
                SupplierEmptyView(title: "No ledger entries", message: "Payment movements will appear here.")
            } else {
                ResponsiveGridContentWrapper {
                    ForEach(rows) { row in
                    VStack(alignment: .leading, spacing: 4) {
                        Text(row.entryType).font(.headline)
                        Text(MoneyFormat.minor(row.amountMinor, currency: row.currency)).font(.subheadline)
                        if let orderId = row.orderId { Text("Order \(orderId)").font(.caption) }
                        Text(row.occurredAt).font(.caption)
                    }
                }
            }
        }
        .navigationTitle("Payment ledger")
        .task { await load() }
    }

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let resp = try await SupplierOperationsService.paymentLedger()
            rows = resp.items
        } catch {
            self.error = error.localizedDescription
        }
    }
}
