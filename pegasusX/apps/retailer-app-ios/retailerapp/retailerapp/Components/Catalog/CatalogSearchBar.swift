import SwiftUI

struct CatalogSearchBar: View {
    @Binding var searchText: String
    @Binding var showFullSearch: Bool

    var body: some View {
        HStack(spacing: AppTheme.spacingMD) {
            Image(systemName: "magnifyingglass")
                .font(.system(.subheadline, weight: .bold))
                .foregroundStyle(AppTheme.accent)

            TextField("mobile_retailer.ui.search_products_suppliers_or_categories", text: $searchText)
                .font(.system(.subheadline, design: .rounded, weight: .medium))
                .textFieldStyle(.plain)
                .autocorrectionDisabled()

            if !searchText.isEmpty {
                Button {
                    Haptics.light()
                    withAnimation(AnimationConstants.express) { searchText = "" }
                } label: {
                    Image(systemName: "xmark.circle.fill")
                        .font(.callout)
                        .foregroundStyle(AppTheme.textTertiary)
                        .pegasusMinimumHitTarget()
                }
                .accessibilityLabel("Clear search text")
            }

            Button {
                Haptics.light()
                showFullSearch = true
            } label: {
                Image(systemName: "arrow.up.left.and.arrow.down.right")
                    .font(.system(.subheadline, weight: .semibold))
                    .foregroundStyle(AppTheme.textSecondary)
                    .pegasusMinimumHitTarget()
            }
            .accessibilityLabel("Open full search")
        }
        .padding(.horizontal, AppTheme.spacingMD)
        .padding(.vertical, AppTheme.spacingSM)
        .frame(minHeight: 44)
        .background {
            RoundedRectangle(cornerRadius: AppTheme.radiusButton, style: .continuous)
                .fill(AppTheme.cardBackground)
                .overlay {
                    RoundedRectangle(cornerRadius: AppTheme.radiusButton, style: .continuous)
                        .stroke(AppTheme.separator.opacity(0.12), lineWidth: 1)
                }
        }
    }
}
