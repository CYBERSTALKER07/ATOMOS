import SwiftUI

struct LedgerView: View {
    @State private var viewModel = LedgerViewModel()

    var body: some View {
        Group {
            if viewModel.loading && viewModel.rows.isEmpty {
                SupplierLoadingView(title: "mobile_supplier.loading.ledger")
            } else if let error = viewModel.error, viewModel.rows.isEmpty {
                SupplierErrorView(message: error) { Task { await viewModel.load() } }
            } else if viewModel.rows.isEmpty {
                SupplierEmptyView(title: "mobile_supplier.empty.no_ledger", message: "mobile_supplier.empty.ledger_message")
            } else {
                ResponsiveGridContentWrapper {
                    ForEach(viewModel.rows) { row in
                        VStack(alignment: .leading, spacing: 4) {
                            Text(row.entryType).font(.headline)
                            Text(row.amountMinor.map { MoneyFormat.minor($0, currency: row.currency) } ?? "—").font(.subheadline)
                            if let orderId = row.orderId { Text(L10n.format("mobile_supplier.ui.order_orderid_2", "\(orderId)")).font(.caption) }
                            Text(row.occurredAt).font(.caption)
                        }
                        .accessibilityElement(children: .combine)
                    }
                }
            }
        }
        .navigationTitle("mobile_supplier.ui.payment_ledger")
        .task { await viewModel.load() }
    }
}
