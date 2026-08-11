import SwiftUI

struct StatsRowView: View {
    var orderCount: Int
    var totalSpent: Int64
    var totalSpentCurrency: String

    private func formatSpent(_ amount: Int64, currency: String) -> String {
        let formatter = NumberFormatter()
        formatter.numberStyle = .decimal
        formatter.groupingSeparator = " "
        formatter.maximumFractionDigits = 0
        let formatted = formatter.string(from: NSNumber(value: amount)) ?? "\(amount)"
        return "\(formatted) \(currency.isEmpty ? "UZS" : currency)"
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
