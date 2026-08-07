import SwiftUI

struct ReconciliationView: View {
    @State private var loading = true
    @State private var error: String?
    @State private var netMinor: Int64 = 0
    @State private var currency = "UZS"
    @State private var mismatchCount = 0

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading reconciliation…")
            } else if let error {
                SupplierErrorView(message: error) { Task { await load() } }
            } else {
                ResponsiveGridContentWrapper {
                    Section {
                        VStack(alignment: .leading, spacing: 4) {
                            Text("supplier_portal.residual.text.settlement_net_authority").font(.caption).foregroundStyle(.secondary)
                            Text(MoneyFormat.minor(netMinor, currency: currency)).font(.title2.bold())
                        }
                    }
                    Section {
                        VStack(alignment: .leading, spacing: 4) {
                            Text("supplier_portal.residual.text.open_mismatches").font(.caption).foregroundStyle(.secondary)
                            Text("\(mismatchCount)").font(.title2.bold())
                        }
                    }
                    Section {
                        Text("mobile_supplier.ui.full_ledger_detail_is_on_payment_ledger")
                            .font(.footnote)
                            .foregroundStyle(.secondary)
                    }
                }
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("portal.nav.reconciliation")
        .task { await load() }
    }

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            async let authority = SupplierOperationsService.paymentSettlementAuthority()
            async let mismatches = SupplierOperationsService.paymentReconciliationMismatches()
            let authorityResp = try await authority
            let mismatchResp = try await mismatches
            if let primary = authorityResp.totalsByCurrency.first {
                currency = primary.currency
                netMinor = primary.amountMinorTotal
            }
            mismatchCount = mismatchResp.items.count
        } catch {
            self.error = error.localizedDescription
        }
    }
}
