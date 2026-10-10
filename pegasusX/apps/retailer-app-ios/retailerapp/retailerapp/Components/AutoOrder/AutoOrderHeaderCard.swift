import SwiftUI

struct AutoOrderHeaderCard: View {
    let supplierCount: Int
    let categoryCount: Int
    let productCount: Int
    let predictionCount: Int

    var body: some View {
        GradientHeaderCard(title: "Empathy Engine", subtitle: "Auto-order intelligence with 5-level control", icon: "wand.and.stars") {
            HStack(spacing: AppTheme.spacingXL) {
                miniStat(value: "\(supplierCount)", label: "Suppliers")
                miniStat(value: "\(categoryCount)", label: "Categories")
                miniStat(value: "\(productCount)", label: "Products")
                miniStat(value: "\(predictionCount)", label: "Predictions")
            }
        }
    }
    
    private func miniStat(value: String, label: String) -> some View {
        VStack(spacing: 3) {
            Text(value)
                .font(.system(.headline, design: .rounded, weight: .bold))
                .foregroundStyle(AppTheme.textPrimary)
            Text(label)
                .font(.system(.caption2, design: .rounded))
                .foregroundStyle(AppTheme.textTertiary)
        }
        .frame(maxWidth: .infinity)
    }
}
