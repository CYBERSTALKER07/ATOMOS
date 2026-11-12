import SwiftUI

struct ComplianceAuditView: View {
    @State private var viewModel = ComplianceAuditViewModel()

    var body: some View {
        Group {
            if viewModel.loading && viewModel.summary == nil {
                SupplierLoadingView(title: "mobile_supplier.loading.compliance")
            } else if let error = viewModel.error, viewModel.summary == nil {
                SupplierErrorView(message: error) { Task { await viewModel.load() } }
            } else if let summary = viewModel.summary {
                List {
                    Section("mobile_supplier.chrome.summary") {
                        metricRow("mobile_supplier.chrome.open_fiscal", value: summary.openFiscalCount)
                        metricRow("mobile_supplier.chrome.force_completes_count", value: summary.forceCompleteCount)
                        metricRow("mobile_supplier.chrome.claim_mismatches", value: summary.claimMismatchCount)
                        metricRow("mobile_supplier.chrome.credit_freezes", value: summary.creditFreezeCount)
                        if !summary.generatedAt.isEmpty {
                            Text(L10n.format("mobile_supplier.ui.generated_generatedat", "\(summary.generatedAt)"))
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                    }
                    if !viewModel.openFiscal.isEmpty {
                        Section("mobile_supplier.chrome.open_fiscal") {
                            ForEach(viewModel.openFiscal) { row in
                                VStack(alignment: .leading, spacing: 4) {
                                    Text(row.orderId).font(.subheadline.monospaced())
                                    Text(L10n.format("mobile_supplier.ui.status_fiscalstatus", "\(row.status)", "\(row.fiscalStatus)"))
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                    Button("mobile_supplier.ui.view_receipt") {
                                        Task { await viewModel.openReceipt(orderId: row.orderId) }
                                    }
                                    .font(.caption.weight(.semibold))
                                    .frame(minHeight: 44)
                                }
                            }
                        }
                    }
                    if !viewModel.forceCompletes.isEmpty {
                        Section("mobile_supplier.chrome.force_completes") {
                            ForEach(viewModel.forceCompletes) { row in
                                VStack(alignment: .leading, spacing: 4) {
                                    Text(row.orderId).font(.subheadline.monospaced())
                                    Text(L10n.format("mobile_supplier.ui.reason_reasoncode", "\(row.reasonCode.isEmpty ? "—" : row.reasonCode)"))
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                    Button("mobile_supplier.ui.view_receipt") {
                                        Task { await viewModel.openReceipt(orderId: row.orderId) }
                                    }
                                    .font(.caption.weight(.semibold))
                                    .frame(minHeight: 44)
                                }
                            }
                        }
                    }
                }
            } else {
                SupplierEmptyView(
                    title: "mobile_supplier.empty.no_compliance",
                    message: "mobile_supplier.empty.compliance_message"
                )
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("portal.nav.compliance_audit")
        .task { await viewModel.load() }
        .refreshable { await viewModel.load(silent: true) }
    }

    private func metricRow(_ label: String, value: Int) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(LocalizedStringKey(label)).font(.caption).foregroundStyle(.secondary)
            Text("\(value)").font(.title2.bold())
        }
        .accessibilityElement(children: .combine)
    }
}

enum ReceiptOpen {
    static func shouldStayQuiet(_ error: Error) -> Bool {
        if case APIError.httpError(404) = error { return true }
        if case APIError.server(let message) = error {
            let lowered = message.lowercased()
            return lowered.contains("not_found") || lowered.contains("not found")
        }
        return false
    }
}
