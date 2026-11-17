import SwiftUI

struct PayoutsView: View {
    @State private var viewModel = PayoutsViewModel()

    var body: some View {
        @Bindable var vm = viewModel
        Form {
            if viewModel.loading && viewModel.batches.isEmpty && viewModel.rail == nil {
                Section { ProgressView("mobile_supplier.loading.payouts") }
            } else if let error = viewModel.error, viewModel.batches.isEmpty, viewModel.rail == nil {
                Section {
                    SupplierErrorView(message: error) { Task { await viewModel.load() } }
                }
            } else {
                Section("mobile_supplier.chrome.rail") {
                    Text("mobile_supplier.chrome.bank_file_rail")
                        .font(.footnote)
                    Text(LocalizedStringKey(viewModel.rail?.isLive == true ? "mobile_supplier.chrome.live_rail_yes" : "mobile_supplier.chrome.live_rail_no"))
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                Section("mobile_supplier.chrome.payout_policy") {
                    if viewModel.policyKnown, let policy = viewModel.policy, !policy.payoutMode.isEmpty {
                        Text(L10n.format("mobile_supplier.chrome.payout_mode_line", policy.payoutMode, policy.source))
                            .font(.footnote)
                    }
                    if let policyError = viewModel.policyError {
                        Text(policyError)
                            .font(.caption)
                            .foregroundStyle(SupplierTheme.destructive)
                    }
                    Picker("mobile_supplier.chrome.mode", selection: $vm.draftMode) {
                        Text("HQ_SUPPLIER").tag("HQ_SUPPLIER")
                        Text("WAREHOUSE_LOCAL").tag("WAREHOUSE_LOCAL")
                    }
                    TextField("mobile_supplier.chrome.reason_required", text: $vm.policyReason)
                    Button("mobile_supplier.chrome.save_mode") {
                        PegasusHaptics.light()
                        Task { await viewModel.savePolicy() }
                    }
                    .frame(minHeight: 44)
                    .disabled(viewModel.busy || viewModel.policyReason.isEmpty || !viewModel.policyKnown || (viewModel.policy?.payoutMode.isEmpty ?? true))
                }
                Section("mobile_supplier.chrome.generate_period") {
                    TextField("mobile_supplier.chrome.period_start", text: $vm.periodStart)
                    TextField("mobile_supplier.chrome.period_end", text: $vm.periodEnd)
                    Button("mobile_supplier.chrome.generate_batch") {
                        PegasusHaptics.light()
                        Task { await viewModel.generate() }
                    }
                    .frame(minHeight: 44)
                    .disabled(viewModel.busy || viewModel.periodStart.isEmpty || viewModel.periodEnd.isEmpty)
                }
                Section("mobile_supplier.chrome.batches") {
                    if viewModel.batches.isEmpty {
                        Text("mobile_supplier.chrome.no_batches")
                            .foregroundStyle(.secondary)
                    } else {
                        ForEach(viewModel.batches) { batch in
                            VStack(alignment: .leading, spacing: 4) {
                                Text(Self.formatBatchPayout(status: batch.status, netPayoutMinor: batch.netPayoutMinor, currency: batch.currency))
                                Text("\(batch.periodStart) → \(batch.periodEnd)")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                                HStack {
                                    Button("mobile_supplier.chrome.export_csv") {
                                        PegasusHaptics.light()
                                        Task { await viewModel.exportCSV(batch.batchId) }
                                    }
                                    .frame(minHeight: 44)
                                    .disabled(viewModel.busy)

                                    Button("mobile_supplier.chrome.mark_paid") {
                                        PegasusHaptics.light()
                                        Task { await viewModel.markPaid(batch.batchId) }
                                    }
                                    .frame(minHeight: 44)
                                    .disabled(viewModel.busy)

                                    Button("mobile_supplier.chrome.dispatch_live") {
                                        PegasusHaptics.light()
                                        Task { await viewModel.dispatchLive(batch.batchId) }
                                    }
                                    .frame(minHeight: 44)
                                    .disabled(viewModel.busy)
                                }
                                .font(.caption)
                            }
                        }
                    }
                }
                if let status = viewModel.status {
                    Section { Text(status).font(.footnote) }
                }
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .background(SupplierTheme.background)
        .navigationTitle("portal.nav.payouts")
        .task { await viewModel.load() }
        .refreshable { await viewModel.load(silent: true) }
    }

    static func formatBatchPayout(status: String, netPayoutMinor: Int64?, currency: String) -> String {
        let amount = netPayoutMinor.map { MoneyFormat.minor($0, currency: currency) } ?? MissingNumber.label(nil as Int64?)
        if status.isEmpty {
            return amount
        }
        return "\(status) · \(amount)"
    }
}
