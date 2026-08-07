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
                Text(L10n.format("mobile_retailer.ui.sale_count_salecount_2", "\(saleCount)"))
                Text(L10n.format("mobile_retailer.ui.on_hand_skus_onhand_2", "\(onHand)"))
                Text(L10n.format("mobile_retailer.ui.low_stock_bins_lowstock_2", "\(lowStock)"))
                Text(L10n.format("mobile_retailer.ui.top_sku_topline_2", "\(topLine)"))
            }
        }
        .navigationTitle("portal.nav.reports_pro")
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
