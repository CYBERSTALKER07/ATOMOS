import SwiftUI

struct ClaimChargebacksView: View {
    @State private var items: [PaymentLedgerEntry] = []
    @State private var orderFilter = ""
    @State private var loading = true
    @State private var error: String?

    private var total: Int64 {
        items.reduce(0) { $0 + $1.amountMinor }
    }

    var body: some View {
        List {
            Section {
                TextField("supplier_portal.chargebacks.claims.text.filter_order_id", text: $orderFilter)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                Button("portal.page.orders.action.refresh") { Task { await load() } }
                Text(L10n.format("mobile_supplier.ui.count_rows_total_total_minor", "\(items.count)", "\(total)"))
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            if loading {
                Section { ProgressView("Loading…") }
            } else if let error {
                Section {
                    Text(error).foregroundStyle(.red)
                    Button("common.action.retry") { Task { await load() } }
                }
            } else if items.isEmpty {
                Section {
                    Text("mobile_supplier.ui.no_claim_chargebacks_yet_approve_a_claim_to_create_chargeback_cl")
                        .foregroundStyle(.secondary)
                }
            } else {
                Section("portal.nav.claim_chargebacks") {
                    ForEach(items) { row in
                        VStack(alignment: .leading, spacing: 4) {
                            Text("\(row.amountMinor) \(displayPackCurrency(row.currency))")
                                .font(.subheadline.weight(.semibold))
                            Text(L10n.format("mobile_supplier.ui.order_orderid_2", "\(row.orderId ?? "—")"))
                                .font(.caption)
                            if let ref = row.referenceId {
                                Text(ref).font(.caption2.monospaced()).foregroundStyle(.secondary)
                            }
                            if let source = row.source, !source.isEmpty {
                                Text(source).font(.caption2).foregroundStyle(.secondary)
                            }
                            Text(row.occurredAt.isEmpty ? (row.createdAt ?? "—") : row.occurredAt)
                                .font(.caption2)
                                .foregroundStyle(.tertiary)
                        }
                        .padding(.vertical, 2)
                    }
                }
            }
        }
        .navigationTitle("portal.nav.claim_chargebacks")
        .refreshable { await load() }
        .task { await load() }
    }

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let resp = try await SupplierOperationsService.listClaimChargebacks(
                limit: 100,
                orderId: orderFilter.trimmingCharacters(in: .whitespacesAndNewlines)
            )
            items = resp.items
        } catch {
            self.error = error.localizedDescription
            items = []
        }
    }
}
