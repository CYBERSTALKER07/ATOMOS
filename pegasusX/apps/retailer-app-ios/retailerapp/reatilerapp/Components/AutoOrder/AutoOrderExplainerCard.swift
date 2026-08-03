import SwiftUI

struct AutoOrderExplainerCard: View {
    var body: some View {
        LabCard {
            VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
                HStack(spacing: AppTheme.spacingSM) {
                    Image(systemName: "info.circle.fill")
                        .font(.system(size: 14))
                        .foregroundStyle(AppTheme.accent)
                    Text("How It Works")
                        .font(.system(.subheadline, design: .rounded, weight: .semibold))
                        .foregroundStyle(AppTheme.textPrimary)
                }

                VStack(alignment: .leading, spacing: AppTheme.spacingSM) {
                    explainerRow(num: "1", text: "The AI analyzes your purchase patterns even when auto-order is off")
                    explainerRow(num: "2", text: "When you enable, choose to use your history or start fresh")
                    explainerRow(num: "3", text: "Starting fresh requires at least 2 orders per product")
                    explainerRow(num: "4", text: "Overrides: Variant > Product > Category > Supplier > Global")
                }
            }
            .padding(AppTheme.spacingLG)
        }
    }

    private func explainerRow(num: String, text: String) -> some View {
        HStack(alignment: .top, spacing: AppTheme.spacingSM) {
            Text(num)
                .font(.system(.caption, design: .rounded, weight: .bold))
                .foregroundStyle(.white)
                .frame(width: 20, height: 20)
                .background(AppTheme.accent)
                .clipShape(.circle)
            Text(text)
                .font(.system(.caption, design: .rounded))
                .foregroundStyle(AppTheme.textSecondary)
        }
    }
}
