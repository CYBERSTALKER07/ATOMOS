import SwiftUI

struct ClaimChargebacksView: View {
    @State private var viewModel = ClaimChargebacksViewModel()

    var body: some View {
        @Bindable var vm = viewModel
        List {
            Section {
                TextField("supplier_portal.chargebacks.claims.text.filter_order_id", text: $vm.orderFilter)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                Button("portal.page.orders.action.refresh") { Task { await viewModel.load() } }
                    .frame(minHeight: 44)
                Text(L10n.format("mobile_supplier.ui.count_rows_total_total_minor", "\(viewModel.items.count)", MissingNumber.label(viewModel.total)))
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            if viewModel.loading && viewModel.items.isEmpty {
                Section { ProgressView("mobile_supplier.loading.generic") }
            } else if let error = viewModel.error, viewModel.items.isEmpty {
                Section {
                    Text(error).foregroundStyle(SupplierTheme.destructive)
                    Button("common.action.retry") { Task { await viewModel.load() } }
                }
            } else if viewModel.items.isEmpty {
                Section {
                    Text("mobile_supplier.ui.no_claim_chargebacks_yet_approve_a_claim_to_create_chargeback_cl")
                        .foregroundStyle(.secondary)
                }
            } else {
                Section("portal.nav.claim_chargebacks") {
                    ForEach(viewModel.items) { row in
                        VStack(alignment: .leading, spacing: 4) {
                            Text(Self.formatChargebackRowAmount(row.amountMinor, currency: row.currency))
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
                        .accessibilityElement(children: .combine)
                    }
                }
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .navigationTitle("portal.nav.claim_chargebacks")
        .refreshable { await viewModel.load(silent: true) }
        .task { await viewModel.load() }
    }

    static func formatChargebackRowAmount(_ amountMinor: Int64?, currency: String?) -> String {
        guard let amountMinor else { return MissingNumber.label(nil as Int64?) }
        return MoneyFormat.minor(amountMinor, currency: displayPackCurrency(currency))
    }
}
