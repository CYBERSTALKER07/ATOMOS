import SwiftUI
import UIKit

struct ReportsProView: View {
    @State private var salesMinor: Int64 = 0
    @State private var saleCount = 0
    @State private var onHand = 0
    @State private var lowStock = 0
    @State private var topLine = "—"
    @State private var banner: String?
    @State private var loadError: String?
    @State private var summaryReady = false
    @State private var exportURL: URL?
    @State private var exporting = false
    @State private var showShare = false
    private let api = APIClient.shared

    var body: some View {
        List {
            if let loadError {
                Section { Text(loadError).font(.caption).foregroundStyle(AppTheme.destructive) }
            } else if let banner {
                Section { Text(banner).font(.caption).foregroundStyle(AppTheme.accent) }
            }
            if summaryReady && loadError == nil {
                Section("Last 7 days") {
                    Text(String(format: "Sales %.2f", Double(salesMinor) / 100.0))
                    Text(L10n.format("mobile_retailer.ui.sale_count_salecount_2", "\(saleCount)"))
                    Text(L10n.format("mobile_retailer.ui.on_hand_skus_onhand_2", "\(onHand)"))
                    Text(L10n.format("mobile_retailer.ui.low_stock_bins_lowstock_2", "\(lowStock)"))
                    Text(L10n.format("mobile_retailer.ui.top_sku_topline_2", "\(topLine)"))
                }
            }
            Section {
                Button {
                    Task { await exportCSV() }
                } label: {
                    HStack {
                        Text(exporting ? "Exporting…" : "Export sales CSV")
                        Spacer()
                        if exporting {
                            ProgressView()
                        }
                    }
                }
                .disabled(exporting)
            }
        }
        .navigationTitle("portal.nav.reports_pro")
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
        .refreshable { await load() }
        .sheet(isPresented: $showShare) {
            if let exportURL {
                ActivityView(activityItems: [exportURL])
            }
        }
    }

    private func load() async {
        do {
            let s = try await api.getReportsSummary()
            salesMinor = s.salesMinor ?? 0
            saleCount = s.saleCount ?? 0
            onHand = s.onHandSkuCount ?? 0
            lowStock = s.lowStockCount ?? 0
            if let first = s.topSkus?.first {
                topLine = "\(first.sku ?? "?") · \(Double(first.salesMinor ?? 0) / 100.0)"
            }
            summaryReady = true
            loadError = nil
            banner = "REPORTS_PRO auto-enabled if needed"
        } catch {
            loadError = "reports_failed"
        }
    }

    private func exportCSV() async {
        guard !exporting else { return }
        exporting = true
        defer { exporting = false }
        do {
            let data = try await api.exportReportsCSV(report: "sales")
            let url = FileManager.default.temporaryDirectory.appendingPathComponent("sales.csv")
            try data.write(to: url, options: .atomic)
            exportURL = url
            showShare = true
            banner = "Sales CSV ready to share"
        } catch {
            banner = error.localizedDescription
        }
    }
}

/// UIKit share sheet bridge for CSV files.
private struct ActivityView: UIViewControllerRepresentable {
    let activityItems: [Any]

    func makeUIViewController(context: Context) -> UIActivityViewController {
        UIActivityViewController(activityItems: activityItems, applicationActivities: nil)
    }

    func updateUIViewController(_ uiViewController: UIActivityViewController, context: Context) {}
}
