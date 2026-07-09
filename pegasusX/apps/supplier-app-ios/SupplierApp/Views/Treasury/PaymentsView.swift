import SwiftUI

struct PaymentsView: View {
    @State private var loading = true
    @State private var error: String?
    @State private var authority: SettlementAuthorityResponse?
    @State private var mismatches: [ReconciliationMismatchRow] = []

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading settlement authority…")
            } else if let error {
                SupplierErrorView(message: error) { Task { await load() } }
            } else if let authority {
                ResponsiveGridContentWrapper {
                    Section("Scope") {
                        PaymentsKpiRow(label: "Supplier scope", value: authority.supplierId.isEmpty ? "(global)" : authority.supplierId)
                        PaymentsKpiRow(label: "Grouped rows", value: "\(authority.count)")
                        PaymentsKpiRow(label: "Total entries", value: "\(authority.entryCountTotal)")
                        PaymentsKpiRow(label: "Reconciliation groups", value: "\(mismatches.count)")
                    }

                    Section("Totals by currency") {
                        if authority.totalsByCurrency.isEmpty {
                            Text("No totals available.").foregroundStyle(.secondary)
                        }
                        ForEach(authority.totalsByCurrency, id: \.currency) { row in
                            HStack {
                                Text(row.currency).fontWeight(.medium)
                                Spacer()
                                Text("\(row.entryCount) entries").font(.caption).foregroundStyle(.secondary)
                                Text(MoneyFormat.minor(row.amountMinorTotal, currency: row.currency))
                            }
                            .font(.subheadline)
                        }
                    }

                    Section("Reconciliation mismatches") {
                        if mismatches.isEmpty {
                            Text("No non-zero mismatches detected.").foregroundStyle(.secondary)
                        }
                        ForEach(mismatches) { row in
                            VStack(alignment: .leading, spacing: 4) {
                                HStack {
                                    Text("\(row.gateway) · \(row.currency)").fontWeight(.medium)
                                    Spacer()
                                    Text(MoneyFormat.minor(row.netAmountMinor, currency: row.currency))
                                        .foregroundStyle(SupplierTheme.destructive)
                                }
                                Text(String(format: "Credit %@ · Debit %@ · %d entries",
                                            MoneyFormat.minor(row.creditAmountMinorTotal, currency: row.currency),
                                            MoneyFormat.minor(row.debitAmountMinorTotal, currency: row.currency),
                                            row.entryCountTotal))
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                            .font(.subheadline)
                        }
                    }

                    Section("Settlement groups") {
                        if authority.items.isEmpty {
                            Text("No settlement groups found.").foregroundStyle(.secondary)
                        }
                        ForEach(authority.items) { row in
                            VStack(alignment: .leading, spacing: 4) {
                                HStack {
                                    Text("\(row.gateway) · \(row.entryType)").fontWeight(.medium)
                                    Spacer()
                                    Text(MoneyFormat.minor(row.amountMinorTotal, currency: row.currency))
                                }
                                Text("\(row.currency) · \(row.entryCount) entries")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                            .font(.subheadline)
                        }
                    }
                }
            } else {
                SupplierEmptyView(title: "No data", message: "No settlement authority data available.")
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("Payments")
        .task { await load() }
    }

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            async let authorityCall = SupplierOperationsService.paymentSettlementAuthority()
            async let mismatchCall = SupplierOperationsService.paymentReconciliationMismatches()
            authority = try await authorityCall
            mismatches = try await mismatchCall.items
        } catch {
            self.error = error.localizedDescription
        }
    }
}

private struct PaymentsKpiRow: View {
    let label: String
    let value: String

    var body: some View {
        HStack {
            Text(label).foregroundStyle(.secondary)
            Spacer()
            Text(value).fontWeight(.medium)
        }
        .font(.subheadline)
    }
}
