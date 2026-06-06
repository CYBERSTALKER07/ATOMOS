//
//  SkeletonViews.swift
//  reatilerapp
//

import SwiftUI

// MARK: - Skeleton Product Card

struct SkeletonProductCard: View {
    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            // Image placeholder
            Rectangle()
                .fill(AppTheme.separator.opacity(0.25))
                .frame(height: 140)
                .clipShape(UnevenRoundedRectangle(
                    topLeadingRadius: AppTheme.radiusCard,
                    topTrailingRadius: AppTheme.radiusCard
                ))

            // Text area
            VStack(alignment: .leading, spacing: AppTheme.spacingSM) {
                Text("Product name placeholder")
                    .font(.system(.subheadline, design: .rounded, weight: .semibold))
                    .foregroundStyle(AppTheme.separator)

                Text("Short product description goes here")
                    .font(.caption)
                    .foregroundStyle(AppTheme.separator)
                    .lineLimit(2)
                    .frame(minHeight: 30, alignment: .top)

                HStack(spacing: 6) {
                    Text("500ml")
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

// MARK: - Skeleton Product Grid

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

// MARK: - Skeleton Order Card

struct SkeletonOrderCard: View {
    var body: some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
            // Header row
            HStack(alignment: .top) {
                Circle()
                    .fill(AppTheme.separator.opacity(0.25))
                    .frame(width: 42, height: 42)

                VStack(alignment: .leading, spacing: 3) {
                    Text("Order #ABC")
                        .font(.system(.subheadline, design: .rounded, weight: .bold))
                        .foregroundStyle(AppTheme.separator)

                    Text("3 items · $0.00")
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

            // Progress bar placeholder
            RoundedRectangle(cornerRadius: 4)
                .fill(AppTheme.separator.opacity(0.2))
                .frame(height: 6)
        }
        .padding(AppTheme.spacingMD)
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
        .redacted(reason: .placeholder)
    }
}

// MARK: - Skeleton Order List

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
