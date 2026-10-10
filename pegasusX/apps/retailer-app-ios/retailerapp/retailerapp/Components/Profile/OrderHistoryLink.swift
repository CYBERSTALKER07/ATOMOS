import SwiftUI

struct OrderHistoryLink: View {
    var orderCount: Int

    var body: some View {
        NavigationLink {
            HistoryView()
        } label: {
            HStack(spacing: AppTheme.spacingMD) {
                ZStack {
                    RoundedRectangle(cornerRadius: AppTheme.radiusSM)
                        .fill(AppTheme.surfaceElevated)
                        .frame(width: 36, height: 36)
                    Image(systemName: "clock.fill")
                        .font(.system(size: 14, weight: .semibold))
                        .foregroundStyle(AppTheme.textSecondary)
                }

                Text("mobile_retailer.ui.order_history")
                    .font(.system(.subheadline, design: .rounded, weight: .medium))
                    .foregroundStyle(AppTheme.textPrimary)

                Spacer()

                Text(L10n.format("mobile_retailer.ui.ordercount_orders_2", "\(orderCount)"))
                    .font(.system(.caption, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)

                Image(systemName: "chevron.right")
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundStyle(AppTheme.textTertiary.opacity(0.5))
            }
            .padding(AppTheme.spacingLG)
            .background(AppTheme.cardBackground)
            .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
            .shadow(color: AppTheme.shadowColor, radius: 4, y: 2)
        }
    }
}
