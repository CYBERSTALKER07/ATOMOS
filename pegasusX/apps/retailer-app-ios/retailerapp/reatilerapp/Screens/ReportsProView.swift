import SwiftUI

struct ReportsProView: View {
    @State private var salesMinor: Int64 = 0
    @State private var saleCount = 0
    @State private var onHand = 0
    @State private var lowStock = 0
    @State private var topLine = "—"
    @State private var banner: String?
    private let api = APIClient.shared

    var body: some View {
        List {
            if let banner { Section { Text(banner).font(.caption).foregroundStyle(AppTheme.accent) } }
            Section("Last 7 days") {
                Text(String(format: "Sales %.2f", Double(salesMinor) / 100.0))
                Text("Sale count: \(saleCount)")
                Text("On-hand SKUs: \(onHand)")
                Text("Low stock bins: \(lowStock)")
                Text("Top SKU: \(topLine)")
            }
        }
        .navigationTitle("Reports Pro")
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
        .refreshable { await load() }
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
            banner = "REPORTS_PRO auto-enabled if needed"
        } catch {
            banner = error.localizedDescription
        }
    }
}
