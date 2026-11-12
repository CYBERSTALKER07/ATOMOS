import SwiftUI

struct CreditProfilesView: View {
    @State private var viewModel = CreditProfilesViewModel()

    var body: some View {
        List {
            if viewModel.loading && viewModel.rows.isEmpty {
                ProgressView("mobile_supplier.loading.generic")
            } else if let error = viewModel.error, viewModel.rows.isEmpty {
                Text(error).foregroundStyle(SupplierTheme.destructive)
                Button("common.action.retry") { Task { await viewModel.load() } }
            } else if viewModel.rows.isEmpty {
                Text("mobile_supplier.ui.no_credit_profiles_for_this_supplier")
                    .foregroundStyle(.secondary)
            } else {
                ForEach(viewModel.rows) { row in
                    VStack(alignment: .leading, spacing: 6) {
                        Text(row.retailerId).font(.caption.monospaced())
                        Text(L10n.format("mobile_supplier.ui.status_risk_risktier", "\(row.status)", "\(row.riskTier.isEmpty ? "—" : row.riskTier)"))
                        Text(Self.formatCreditProfileSummary(
                            limitMinor: row.creditLimitMinor,
                            balanceMinor: row.currentBalanceMinor,
                            availMinor: row.availableCreditMinor
                        ))
                        HStack {
                            if row.status.uppercased() == "ACTIVE" {
                                Button("mobile_supplier.ui.freeze") {
                                    PegasusHaptics.light()
                                    Task { await viewModel.setStatus(row, status: "FROZEN") }
                                }
                                .frame(minHeight: 44)
                                .disabled(viewModel.busyId == row.retailerId)
                            } else if row.status.uppercased() == "FROZEN" {
                                Button("mobile_supplier.ui.unfreeze") {
                                    PegasusHaptics.light()
                                    Task { await viewModel.setStatus(row, status: "ACTIVE") }
                                }
                                .buttonStyle(.borderedProminent)
                                .frame(minHeight: 44)
                                .disabled(viewModel.busyId == row.retailerId)
                            }
                        }
                    }
                    .padding(.vertical, 4)
                    .accessibilityElement(children: .combine)
                    .accessibilityLabel("Retailer \(row.retailerId), status \(row.status), risk \(row.riskTier)")
                }
            }
        }
        .navigationTitle("mobile_supplier.ui.credit_profiles")
        .refreshable {
            PegasusHaptics.light()
            await viewModel.load(silent: true)
        }
        .task { await viewModel.load() }
    }

    static func formatCreditProfileSummary(limitMinor: Int64?, balanceMinor: Int64?, availMinor: Int64?) -> String {
        let limit = limitMinor.map { formatMinorUnits($0, fractionDigits: 2, trimTrailingZeros: true) } ?? MissingNumber.label(nil as Int64?)
        let bal = balanceMinor.map { formatMinorUnits($0, fractionDigits: 2, trimTrailingZeros: true) } ?? MissingNumber.label(nil as Int64?)
        let avail = availMinor.map { formatMinorUnits($0, fractionDigits: 2, trimTrailingZeros: true) } ?? MissingNumber.label(nil as Int64?)
        return L10n.format("mobile_supplier.ui.limit_creditlimitminor_bal_currentbalanceminor_avail_availablecreditmino", limit, bal, avail)
    }
}
