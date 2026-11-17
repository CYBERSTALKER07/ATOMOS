import SwiftUI

struct CreditNotesListView: View {
    @State private var viewModel = CreditNotesListViewModel()

    var body: some View {
        List {
            if viewModel.loading && viewModel.rows.isEmpty {
                ProgressView("mobile_supplier.loading.generic")
            } else if let error = viewModel.error, viewModel.rows.isEmpty {
                Text(error).foregroundStyle(SupplierTheme.destructive)
                Button("common.action.retry") { Task { await viewModel.load() } }
            } else if viewModel.rows.isEmpty {
                Text("mobile_supplier.ui.no_draft_credit_notes")
                    .foregroundStyle(.secondary)
            } else {
                ForEach(viewModel.rows) { row in
                    VStack(alignment: .leading, spacing: 6) {
                        Text(row.creditNoteId).font(.caption.monospaced())
                        Text(L10n.format("mobile_supplier.ui.order_orderid_status", "\(row.orderId)", "\(row.status)"))
                        Text(L10n.format("mobile_supplier.ui.totalgrossminor_minor", MissingNumber.label(row.totalGrossMinor)))
                        if row.status.uppercased() == "DRAFT" {
                            Button("mobile_supplier.ui.issue") {
                                PegasusHaptics.light()
                                Task { await viewModel.issue(row.creditNoteId) }
                            }
                            .buttonStyle(.borderedProminent)
                            .frame(minHeight: 44)
                            .disabled(viewModel.busyId == row.creditNoteId)
                        }
                    }
                    .padding(.vertical, 4)
                    .accessibilityElement(children: .combine)
                    .accessibilityLabel("Credit note \(row.creditNoteId), order \(row.orderId), status \(row.status)")
                }
            }
        }
        .navigationTitle("portal.nav.credit_notes")
        .refreshable {
            PegasusHaptics.light()
            await viewModel.load(silent: true)
        }
        .task { await viewModel.load() }
    }
}
