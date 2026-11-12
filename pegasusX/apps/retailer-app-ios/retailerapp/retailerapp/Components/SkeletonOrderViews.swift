import SwiftUI

struct SkeletonOrderCard: View {
    var body: some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
            HStack(alignment: .top) {
                Circle()
                    .fill(AppTheme.separator.opacity(0.25))
                    .frame(width: 42, height: 42)

                VStack(alignment: .leading, spacing: 3) {
                    Text("mobile_retailer.ui.order_abc")
                        .font(.system(.subheadline, design: .rounded, weight: .bold))
                        .foregroundStyle(AppTheme.separator)

                    Text("mobile_retailer.ui.3_items_0_00")
                        .font(.caption)
                        .foregroundStyle(AppTheme.separator)
                }

                Spacer()

                Text("PENDING")
                    .font(.caption2)
                    .padding(.horizontal, 10)
                    .padding(.vertical, 5)
                    .background(AppTheme.separator.opacity(0.2))
                    .clipShape(.capsule)
            }

            RoundedRectangle(cornerRadius: 4, style: .continuous)
                .fill(AppTheme.separator.opacity(0.2))
                .frame(height: 6)
        }
        .padding(AppTheme.spacingMD)
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
        .redacted(reason: .placeholder)
    }
}

struct SkeletonOrderList: View {
    var count: Int = 4

    var body: some View {
        VStack(spacing: AppTheme.spacingMD) {
            ForEach(0..<count, id: \.self) { _ in
                SkeletonOrderCard()
            }
        }
        .padding(.horizontal, AppTheme.spacingLG)
        .padding(.top, AppTheme.spacingMD)
    }
}
