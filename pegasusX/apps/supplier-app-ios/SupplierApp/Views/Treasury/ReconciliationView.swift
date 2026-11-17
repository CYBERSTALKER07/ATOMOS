import SwiftUI

struct ReconciliationView: View {
    @State private var viewModel = ReconciliationViewModel()

    var body: some View {
        Group {
            if viewModel.loading && !viewModel.hasLoaded {
                SupplierLoadingView(title: "mobile_supplier.loading.reconciliation")
            } else if let error = viewModel.error, !viewModel.hasLoaded {
                SupplierErrorView(message: error) { Task { await viewModel.load() } }
            } else {
                ResponsiveGridContentWrapper {
                    Section {
                        VStack(alignment: .leading, spacing: 4) {
                            Text("supplier_portal.residual.text.settlement_net_authority").font(.caption).foregroundStyle(.secondary)
                            Text(Self.formatSettlementNet(netMinor: viewModel.netMinor, currency: viewModel.currency)).font(.title2.bold())
                        }
                        .accessibilityElement(children: .combine)
                    }
                    Section {
                        VStack(alignment: .leading, spacing: 4) {
                            Text("supplier_portal.residual.text.open_mismatches").font(.caption).foregroundStyle(.secondary)
                            Text("\(viewModel.mismatchCount)").font(.title2.bold())
                        }
                        .accessibilityElement(children: .combine)
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
        .task { await viewModel.load() }
        .refreshable { await viewModel.load(silent: true) }
    }

    static func formatSettlementNet(netMinor: Int64?, currency: String) -> String {
        guard let netMinor else { return MissingNumber.label(nil as Int64?) }
        return MoneyFormat.minor(netMinor, currency: currency)
    }
}
