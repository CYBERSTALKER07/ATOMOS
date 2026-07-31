import SwiftUI
import UIKit

struct ComplianceAuditView: View {
    @State private var loading = true
    @State private var error: String?
    @State private var summary: ComplianceSummary?
    @State private var forceCompletes: [ComplianceForceCompleteRow] = []
    @State private var openFiscal: [ComplianceFiscalOpenRow] = []

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading compliance audit…")
            } else if let error {
                SupplierErrorView(message: error) { Task { await load() } }
            } else if let summary {
                List {
                    Section("Summary") {
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
                    if !openFiscal.isEmpty {
                        Section("Open fiscal") {
                            ForEach(openFiscal) { row in
                                VStack(alignment: .leading, spacing: 4) {
                                    Text(row.orderId).font(.subheadline.monospaced())
                                    Text("\(row.status) · \(row.fiscalStatus)")
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                    Button("View receipt") {
                                        Task { await openReceipt(orderId: row.orderId) }
                                    }
                                    .font(.caption.weight(.semibold))
                                }
                            }
                        }
                    }
                    if !forceCompletes.isEmpty {
                        Section("Force-completes") {
                            ForEach(forceCompletes) { row in
                                VStack(alignment: .leading, spacing: 4) {
                                    Text(row.orderId).font(.subheadline.monospaced())
                                    Text("Reason \(row.reasonCode.isEmpty ? "—" : row.reasonCode)")
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                    Button("View receipt") {
                                        Task { await openReceipt(orderId: row.orderId) }
                                    }
                                    .font(.caption.weight(.semibold))
                                }
                            }
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

    private func openReceipt(orderId: String) async {
        do {
            let meta: OrderReceiptMeta = try await APIClient.shared.get(
                "v1/supplier/orders/\(orderId)/receipt",
                query: ["format": "json"]
            )
            let raw = [meta.htmlUrl, meta.qrUrl, meta.pdfUrl]
                .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
                .first { !$0.isEmpty }
            if let raw, let url = URL(string: raw) {
                await MainActor.run {
                    UIApplication.shared.open(url)
                }
            }
        } catch {
            // Receipt may not exist yet for open fiscal rows.
        }
    }

    private func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        defer { if !silent { loading = false } }
        do {
            let response = try await SupplierOperationsService.complianceDashboard()
            summary = response.summary
            openFiscal = response.openFiscal
            forceCompletes = response.forceCompletes
        } catch {
            if !silent { self.error = error.localizedDescription }
        }
    }
}
