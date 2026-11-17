import SwiftUI

struct ReplenishmentPoliciesView: View {
    @State private var viewModel = ReplenishmentPoliciesViewModel()

    var body: some View {
        Group {
            if viewModel.loading && viewModel.policy == nil {
                SupplierLoadingView(title: "mobile_supplier.loading.policies")
            } else if let error = viewModel.error, viewModel.policy == nil {
                SupplierErrorView(message: error) { Task { await viewModel.load() } }
            } else if let policy = viewModel.policy {
                ResponsiveGridContentWrapper {
                    Section("mobile_supplier.chrome.auto_approval") {
                        LabeledContent("mobile_supplier.chrome.stable_replenishment", value: policy.autoApproveStable ? L10n.format("mobile_supplier.chrome.enabled") : L10n.format("mobile_supplier.chrome.disabled"))
                        LabeledContent("mobile_supplier.chrome.predictive_push", value: policy.autoApprovePredictivePush ? L10n.format("mobile_supplier.chrome.enabled") : L10n.format("mobile_supplier.chrome.disabled"))
                    }
                    Section("mobile_supplier.chrome.thresholds") {
                        LabeledContent("mobile_supplier.chrome.max_daily_transfer", value: "\(policy.maxDailyTransferUnits)")
                        LabeledContent("mobile_supplier.chrome.min_confidence_score", value: String(format: "%.1f", policy.minConfidenceScore))
                    }
                    Section("mobile_supplier.chrome.safety_stock") {
                        LabeledContent("mobile_supplier.chrome.target_service_level", value: String(format: "%.0f%%", policy.targetServiceLevel * 100))
                        LabeledContent("mobile_supplier.chrome.lead_time_days", value: "\(policy.leadTimeDays)")
                        LabeledContent("mobile_supplier.chrome.lead_sigma_days", value: String(format: "%.1f", policy.leadTimeSigmaDays))
                        Text("mobile_supplier.ui.lead_is_assumed_until_10_transfers_have_receivedat_history")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                    Section {
                        Text(L10n.format("mobile_supplier.ui.supplier_supplierid", "\(policy.supplierId)"))
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("supplier_portal.operations.replenishment_policies.text.replenishment_policies")
        .task { await viewModel.load() }
        .refreshable { await viewModel.load(silent: true) }
    }
}
