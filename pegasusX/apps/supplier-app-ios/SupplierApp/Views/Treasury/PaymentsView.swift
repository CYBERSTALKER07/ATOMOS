import SwiftUI

struct PaymentsView: View {
    @State private var viewModel = PaymentsViewModel()

    var body: some View {
        Group {
            if viewModel.loading && viewModel.authority == nil {
                SupplierLoadingView(title: "mobile_supplier.loading.settlement")
            } else if let error = viewModel.error, viewModel.authority == nil {
                SupplierErrorView(message: error) { Task { await viewModel.load() } }
            } else if let authority = viewModel.authority {
                ResponsiveGridContentWrapper {
                    Section("mobile_supplier.chrome.scope") {
                        PaymentsKpiRow(label: "mobile_supplier.chrome.supplier_scope", value: authority.supplierId.isEmpty ? "mobile_supplier.chrome.global_scope" : authority.supplierId)
                        PaymentsKpiRow(label: "mobile_supplier.chrome.grouped_rows", value: "\(authority.count)")
                        PaymentsKpiRow(label: "mobile_supplier.chrome.total_entries", value: "\(authority.entryCountTotal)")
                        PaymentsKpiRow(label: "mobile_supplier.chrome.reconciliation_groups", value: "\(viewModel.mismatches.count)")
                    }

                    Section("mobile_supplier.chrome.totals_by_currency") {
                        if authority.totalsByCurrency.isEmpty {
                            Text("mobile_supplier.ui.no_totals_available").foregroundStyle(.secondary)
                        }
                        ForEach(authority.totalsByCurrency, id: \.currency) { row in
                            HStack {
                                Text(row.currency).fontWeight(.medium)
                                Spacer()
                                Text(L10n.format("mobile_supplier.ui.entrycount_entries", "\(row.entryCount)")).font(.caption).foregroundStyle(.secondary)
                                Text(MoneyFormat.minor(row.amountMinorTotal, currency: row.currency))
                            }
                            .font(.subheadline)
                            .accessibilityElement(children: .combine)
                        }
                    }

                    Section("mobile_supplier.chrome.reconciliation_mismatches") {
                        if viewModel.mismatches.isEmpty {
                            Text("mobile_supplier.ui.no_non_zero_mismatches_detected").foregroundStyle(.secondary)
                        }
                        ForEach(viewModel.mismatches) { row in
                            VStack(alignment: .leading, spacing: 4) {
                                HStack {
                                    Text(L10n.format("mobile_supplier.ui.gateway_currency", "\(row.gateway)", "\(row.currency)")).fontWeight(.medium)
                                    Spacer()
                                    Text(row.netAmountMinor.map { MoneyFormat.minor($0, currency: row.currency) } ?? "—")
                                        .foregroundStyle(SupplierTheme.destructive)
                                }
                                Text(L10n.format(
                                    "mobile_supplier.chrome.credit_debit_entries",
                                    row.creditAmountMinorTotal.map { MoneyFormat.minor($0, currency: row.currency) } ?? "—",
                                    row.debitAmountMinorTotal.map { MoneyFormat.minor($0, currency: row.currency) } ?? "—",
                                    "\(row.entryCountTotal)"
                                ))
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                            .font(.subheadline)
                            .accessibilityElement(children: .combine)
                        }
                    }

                    Section("mobile_supplier.chrome.settlement_groups") {
                        if authority.items.isEmpty {
                            Text("mobile_supplier.ui.no_settlement_groups_found").foregroundStyle(.secondary)
                        }
                        ForEach(authority.items) { row in
                            VStack(alignment: .leading, spacing: 4) {
                                HStack {
                                    Text(L10n.format("mobile_supplier.ui.gateway_entrytype", "\(row.gateway)", "\(row.entryType)")).fontWeight(.medium)
                                    Spacer()
                                    Text(MoneyFormat.minor(row.amountMinorTotal, currency: row.currency))
                                }
                                Text(L10n.format("mobile_supplier.ui.currency_entrycount_entries", "\(row.currency)", "\(row.entryCount)"))
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                            .font(.subheadline)
                            .accessibilityElement(children: .combine)
                        }
                    }
                }
            } else {
                SupplierEmptyView(title: "mobile_supplier.empty.no_data", message: "mobile_supplier.empty.payments_message")
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("portal.nav.payments")
        .task { await viewModel.load() }
    }
}

private struct PaymentsKpiRow: View {
    let label: String
    let value: String

    var body: some View {
        HStack {
            Text(LocalizedStringKey(label)).foregroundStyle(.secondary)
            Spacer()
            Text(LocalizedStringKey(value)).fontWeight(.medium)
        }
        .font(.subheadline)
        .accessibilityElement(children: .combine)
    }
}
