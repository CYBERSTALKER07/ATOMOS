import SwiftUI

struct ComplianceAuditView: View {
    @State private var loading = true
    @State private var error: String?
    @State private var summary: ComplianceSummary?

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading compliance audit…")
            } else if let error {
                SupplierErrorView(message: error) { Task { await load() } }
            } else if let summary {
                ResponsiveGridContentWrapper {
                    VStack(alignment: .leading, spacing: 8) {
                        metricRow("Open fiscal", value: summary.openFiscalCount)
                        metricRow("Force completes", value: summary.forceCompleteCount)
                        metricRow("Claim mismatches", value: summary.claimMismatchCount)
                        metricRow("Credit freezes", value: summary.creditFreezeCount)
                        if !summary.generatedAt.isEmpty {
                            Text("Generated \(summary.generatedAt)")
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                    }
                }
            } else {
                SupplierEmptyView(
                    title: "No compliance data",
                    message: "Compliance audit metrics will appear here."
                )
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("Compliance audit")
        .task { await load() }
        .refreshable { await load(silent: true) }
    }

    private func metricRow(_ label: String, value: Int) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(label).font(.caption).foregroundStyle(.secondary)
            Text("\(value)").font(.title2.bold())
        }
    }

    private func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        defer { if !silent { loading = false } }
        do {
            let response = try await SupplierOperationsService.complianceDashboard()
            summary = response.summary
        } catch {
            if !silent { self.error = error.localizedDescription }
        }
    }
}
