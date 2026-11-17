import SwiftUI

struct SkeletonProductCard: View {
    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            Rectangle()
                .fill(AppTheme.separator.opacity(0.25))
                .frame(height: 140)
                .clipShape(UnevenRoundedRectangle(
                    topLeadingRadius: AppTheme.radiusCard,
                    topTrailingRadius: AppTheme.radiusCard
                ))

            VStack(alignment: .leading, spacing: AppTheme.spacingSM) {
                Text("mobile_retailer.ui.product_name_placeholder")
                    .font(.system(.subheadline, design: .rounded, weight: .semibold))
                    .foregroundStyle(AppTheme.separator)

                Text("mobile_retailer.ui.short_product_description_goes_here")
                    .font(.caption)
                    .foregroundStyle(AppTheme.separator)
                    .lineLimit(2)
                    .frame(minHeight: 30, alignment: .top)

                HStack(spacing: 6) {
                    Text("mobile_retailer.ui.500ml")
                        .font(.caption2)
                        .padding(.horizontal, 8)
                        .padding(.vertical, 4)
                        .background(AppTheme.separator.opacity(0.2))
                        .clipShape(.capsule)
                    Spacer()
                }

                Spacer().frame(height: AppTheme.spacingSM)
            }
            .padding(AppTheme.spacingMD)
        }
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
        .redacted(reason: .placeholder)
    }
}

struct SkeletonProductGrid: View {
    var columns: Int = 2
    var count: Int = 6

    private var gridColumns: [GridItem] {
        [GridItem(.adaptive(minimum: 160), spacing: AppTheme.spacingMD)]
    }

    var body: some View {
        LazyVGrid(columns: gridColumns, spacing: AppTheme.spacingMD) {
            ForEach(0..<count, id: \.self) { _ in
                SkeletonProductCard()
            }
        }
        .padding(.horizontal, AppTheme.spacingLG)
        .padding(.top, AppTheme.spacingMD)
    }
}
