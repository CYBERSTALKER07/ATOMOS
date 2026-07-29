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
                TextField("Filter order id", text: $orderFilter)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                Button("Refresh") { Task { await load() } }
                Text("\(items.count) rows · total \(total) minor")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            if loading {
                Section { ProgressView("Loading…") }
            } else if let error {
                Section {
                    Text(error).foregroundStyle(.red)
                    Button("Retry") { Task { await load() } }
                }
            } else if items.isEmpty {
                Section {
                    Text("No claim chargebacks yet. Approve a claim to create chargeback_clm_* ledger rows.")
                        .foregroundStyle(.secondary)
                }
            } else {
                Section("Claim chargebacks") {
                    ForEach(items) { row in
                        VStack(alignment: .leading, spacing: 4) {
                            Text("\(row.amountMinor) \(row.currency.isEmpty ? "UZS" : row.currency)")
                                .font(.subheadline.weight(.semibold))
                            Text("Order \(row.orderId ?? "—")")
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
        .navigationTitle("Claim chargebacks")
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
