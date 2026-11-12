import SwiftUI

struct CatalogBentoGrid: View {
    let categories: [ProductCategory]

    var body: some View {
        VStack(spacing: AppTheme.spacingMD) {
            RetailerSectionHeader(
                title: "Categories",
                subtitle: "\(categories.count) types",
                icon: "square.grid.2x2"
            )
            .padding(.horizontal, AppTheme.spacingLG)
            .padding(.top, AppTheme.spacingMD)

            if categories.count >= 2 {
                HStack(spacing: AppTheme.spacingMD) {
                    bentoBig(categories[0], height: 150)
                        .staggeredSlideIn(index: 0)
                    bentoBig(categories[1], height: 150)
                        .staggeredSlideIn(index: 1)
                }
                .padding(.horizontal, AppTheme.spacingLG)
            }

            if categories.count >= 4 {
                HStack(spacing: AppTheme.spacingMD) {
                    bentoWide(categories[2], height: 130)
                        .staggeredSlideIn(index: 2)

                    VStack(spacing: AppTheme.spacingMD) {
                        bentoSmall(categories[3])
                            .staggeredSlideIn(index: 3)
                        if categories.count >= 5 {
                            bentoSmall(categories[4])
                                .staggeredSlideIn(index: 4)
                        }
                    }
                }
                .padding(.horizontal, AppTheme.spacingLG)
            }

            if categories.count >= 7 {
                HStack(spacing: AppTheme.spacingMD) {
                    bentoCompact(categories[5])
                        .staggeredSlideIn(index: 5)
                    bentoCompact(categories[6])
                        .staggeredSlideIn(index: 6)
                    if categories.count >= 8 {
                        bentoCompact(categories[7])
                            .staggeredSlideIn(index: 7)
                    }
                }
                .padding(.horizontal, AppTheme.spacingLG)
            }

            if categories.count > 8 {
                let remaining = Array(categories.dropFirst(8))
                let cols = [GridItem(.adaptive(minimum: 160), spacing: AppTheme.spacingMD)]
                LazyVGrid(columns: cols, spacing: AppTheme.spacingMD) {
                    ForEach(Array(remaining.enumerated()), id: \.element.id) { idx, cat in
                        bentoBig(cat, height: 120)
                            .staggeredSlideIn(index: idx + 8)
                    }
                }
                .padding(.horizontal, AppTheme.spacingLG)
            }
        }
        .padding(.bottom, AppTheme.spacingHuge)
    }

    func bentoBig(_ cat: ProductCategory, height: Double) -> some View {
        NavigationLink {
            CategorySuppliersView(category: cat)
        } label: {
            VStack(alignment: .leading, spacing: 0) {
                Spacer()
                Image(systemName: cat.icon)
                    .font(.system(.title, weight: .bold))
                    .foregroundStyle(AppTheme.accent)
                    .padding(.bottom, AppTheme.spacingSM)
                Text(cat.name)
                    .font(.system(.subheadline, design: .rounded, weight: .bold))
                    .foregroundStyle(AppTheme.textPrimary)
                if let count = cat.productCount {
                    Text(L10n.format("mobile_retailer.ui.count_items_2", "\(count)"))
                        .font(.system(.caption2, design: .rounded, weight: .medium))
                        .foregroundStyle(AppTheme.textTertiary)
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .frame(minHeight: height)
            .padding(AppTheme.spacingMD)
            .background {
                RoundedRectangle(cornerRadius: AppTheme.radiusCard, style: .continuous)
                    .fill(AppTheme.cardBackground)
                    .overlay {
                        RoundedRectangle(cornerRadius: AppTheme.radiusCard, style: .continuous)
                            .stroke(AppTheme.separator.opacity(0.12), lineWidth: 1)
                    }
            }
            .accessibilityElement(children: .combine)
            .accessibilityLabel(cat.productCount.map { "\(cat.name), \($0) items" } ?? cat.name)
        }
        .buttonStyle(.pressable)
    }

    func bentoWide(_ cat: ProductCategory, height: Double) -> some View {
        NavigationLink {
            CategorySuppliersView(category: cat)
        } label: {
            HStack(spacing: AppTheme.spacingMD) {
                Image(systemName: cat.icon)
                    .font(.system(.largeTitle, weight: .bold))
                    .foregroundStyle(AppTheme.accent)
                VStack(alignment: .leading, spacing: 3) {
                    Text(cat.name)
                        .font(.system(.headline, design: .rounded, weight: .bold))
                        .foregroundStyle(AppTheme.textPrimary)
                    if let count = cat.productCount {
                        Text(L10n.format("mobile_retailer.ui.count_items_2", "\(count)"))
                            .font(.system(.caption, design: .rounded, weight: .medium))
                            .foregroundStyle(AppTheme.textTertiary)
                    }
                }
                Spacer()
            }
            .frame(minHeight: height)
            .padding(AppTheme.spacingMD)
            .background {
                RoundedRectangle(cornerRadius: AppTheme.radiusCard, style: .continuous)
                    .fill(AppTheme.cardBackground)
                    .overlay {
                        RoundedRectangle(cornerRadius: AppTheme.radiusCard, style: .continuous)
                            .stroke(AppTheme.separator.opacity(0.12), lineWidth: 1)
                    }
            }
            .accessibilityElement(children: .combine)
            .accessibilityLabel(cat.productCount.map { "\(cat.name), \($0) items" } ?? cat.name)
        }
        .buttonStyle(.pressable)
    }

    func bentoSmall(_ cat: ProductCategory) -> some View {
        NavigationLink {
            CategorySuppliersView(category: cat)
        } label: {
            HStack(spacing: AppTheme.spacingSM) {
                Image(systemName: cat.icon)
                    .font(.system(.headline, weight: .bold))
                    .foregroundStyle(AppTheme.accent)
                Text(cat.name)
                    .font(.system(.caption, design: .rounded, weight: .bold))
                    .foregroundStyle(AppTheme.textPrimary)
                    .lineLimit(1)
                Spacer()
            }
            .frame(minHeight: 54)
            .padding(.horizontal, AppTheme.spacingMD)
            .background {
                RoundedRectangle(cornerRadius: AppTheme.radiusMD, style: .continuous)
                    .fill(AppTheme.cardBackground)
                    .overlay {
                        RoundedRectangle(cornerRadius: AppTheme.radiusMD, style: .continuous)
                            .stroke(AppTheme.separator.opacity(0.12), lineWidth: 1)
                    }
            }
            .accessibilityElement(children: .combine)
            .accessibilityLabel(cat.name)
        }
        .buttonStyle(.pressable)
    }

    func bentoCompact(_ cat: ProductCategory) -> some View {
        NavigationLink {
            CategorySuppliersView(category: cat)
        } label: {
            VStack(spacing: AppTheme.spacingSM) {
                Image(systemName: cat.icon)
                    .font(.system(.title2, weight: .bold))
                    .foregroundStyle(AppTheme.accent)
                Text(cat.name)
                    .font(.system(.caption2, design: .rounded, weight: .bold))
                    .foregroundStyle(AppTheme.textSecondary)
            }
            .frame(maxWidth: .infinity)
            .frame(minHeight: 80)
            .background {
                RoundedRectangle(cornerRadius: AppTheme.radiusMD, style: .continuous)
                    .fill(AppTheme.cardBackground)
                    .overlay {
                        RoundedRectangle(cornerRadius: AppTheme.radiusMD, style: .continuous)
                            .stroke(AppTheme.separator.opacity(0.12), lineWidth: 1)
                    }
            }
            .accessibilityElement(children: .combine)
            .accessibilityLabel(cat.name)
        }
        .buttonStyle(.pressable)
    }
}
