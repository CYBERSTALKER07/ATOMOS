import SwiftUI

struct StatsRowView: View {
    var orderCount: Int
    var totalSpent: Int64
    var totalSpentCurrency: String

    private static let spentFormatter: NumberFormatter = {
        let formatter = NumberFormatter()
        formatter.numberStyle = .decimal
        formatter.groupingSeparator = " "
        formatter.maximumFractionDigits = 0
        return formatter
    }()

    private func formatSpent(_ amount: Int64, currency: String) -> String {
        let formatted = Self.spentFormatter.string(from: NSNumber(value: amount)) ?? "\(amount)"
        return "\(formatted) \(currency.isEmpty ? packCurrency(MarketPackStore.pack) : currency)"
    }

    var body: some View {
        LazyVGrid(
            columns: [
                GridItem(.flexible(), spacing: AppTheme.spacingMD),
                GridItem(.flexible(), spacing: AppTheme.spacingMD),
                GridItem(.flexible(), spacing: AppTheme.spacingMD),
            ],
            spacing: AppTheme.spacingMD
        ) {
            KpiTile(title: "Orders", value: "\(orderCount)", systemImage: "shippingbox.fill", tint: AppTheme.accent)
            KpiTile(title: "Spent", value: formatSpent(totalSpent, currency: totalSpentCurrency), systemImage: "dollarsign.circle.fill", tint: AppTheme.success)
            KpiTile(title: "Rating", value: "4.9", systemImage: "star.fill", tint: AppTheme.warning)
        }
    }
}
