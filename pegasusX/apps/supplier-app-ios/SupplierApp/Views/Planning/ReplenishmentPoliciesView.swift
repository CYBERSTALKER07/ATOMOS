import SwiftUI

struct ReplenishmentPoliciesView: View {
    @State private var policy: SupplierReplenishmentPolicy?
    @State private var loading = true
    @State private var error: String?

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading policies…")
            } else if let error {
                SupplierErrorView(message: error) { Task { await load() } }
            } else if let policy {
                List {
                    Section("Auto-approval") {
                        LabeledContent("Stable replenishment", value: policy.autoApproveStable ? "Enabled" : "Disabled")
                        LabeledContent("Predictive push", value: policy.autoApprovePredictivePush ? "Enabled" : "Disabled")
                    }
                    Section("Thresholds") {
                        LabeledContent("Max daily transfer units", value: "\(policy.maxDailyTransferUnits)")
                        LabeledContent("Min confidence score", value: String(format: "%.1f", policy.minConfidenceScore))
                    }
                    Section {
                        Text("Supplier: \(policy.supplierId)")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }
                .listStyle(.insetGrouped)
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("Replenishment policies")
        .task { await load() }
        .refreshable { await load(silent: true) }
    }

    @MainActor
    private func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        defer { loading = false }
        do {
            policy = try await SupplierOperationsService.replenishmentPolicies()
        } catch {
            if !silent { self.error = error.localizedDescription }
        }
    }
}
