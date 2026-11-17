import SwiftUI

struct CashReconciliationsView: View {
    @State private var viewModel = CashReconciliationsViewModel()

    var body: some View {
        List {
            if viewModel.loading && viewModel.rows.isEmpty {
                ProgressView("mobile_supplier.loading.generic")
            } else if let error = viewModel.error, viewModel.rows.isEmpty {
                Text(error).foregroundStyle(SupplierTheme.destructive)
                Button("common.action.retry") { Task { await viewModel.load() } }
            } else if viewModel.rows.isEmpty {
                Text("mobile_supplier.ui.no_open_cash_discrepancies")
                    .foregroundStyle(.secondary)
            } else {
                ForEach(viewModel.rows) { row in
                    VStack(alignment: .leading, spacing: 6) {
                        Text(row.reconciliationId).font(.caption.monospaced())
                        Text(L10n.format("mobile_supplier.ui.driver_driverid_status", "\(row.driverId)", "\(row.status)"))
                        Text(L10n.format("mobile_supplier.ui.diff_differenceminor_minor", "\(row.differenceMinor)"))
                        let open = row.status.uppercased() == "PENDING" || row.status.uppercased() == "DISPUTED"
                        if open {
                            Button("mobile_supplier.ui.accept") {
                                PegasusHaptics.light()
                                Task { await viewModel.accept(row.reconciliationId) }
                            }
                            .buttonStyle(.borderedProminent)
                            .frame(minHeight: 44)
                            .disabled(viewModel.busyId == row.reconciliationId)
                        }
                    }
                    .padding(.vertical, 4)
                    .accessibilityElement(children: .combine)
                    .accessibilityLabel("Reconciliation \(row.reconciliationId), driver \(row.driverId), status \(row.status), difference \(row.differenceMinor)")
                }
            }
        }
        .navigationTitle("portal.nav.cash_reconciliations")
        .refreshable {
            PegasusHaptics.light()
            await viewModel.load(silent: true)
        }
        .task { await viewModel.load() }
    }
}
