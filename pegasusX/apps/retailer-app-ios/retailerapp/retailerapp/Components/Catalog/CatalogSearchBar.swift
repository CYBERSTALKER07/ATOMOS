import SwiftUI

struct CatalogSearchBar: View {
    @Binding var searchText: String
    @Binding var showFullSearch: Bool

    var body: some View {
        HStack(spacing: AppTheme.spacingMD) {
            Image(systemName: "magnifyingglass")
                .font(.system(size: 15, weight: .bold)) // Bold icon
                .foregroundStyle(AppTheme.accent)

            TextField("mobile_retailer.ui.search_products_suppliers_or_categories", text: $searchText)
                .font(.system(.subheadline, design: .rounded, weight: .medium)) // Medium weight
                .textFieldStyle(.plain)
                .autocorrectionDisabled()

            if !searchText.isEmpty {
                Button {
                    withAnimation(AnimationConstants.express) { searchText = "" }
                } label: {
                    Image(systemName: "xmark.circle.fill")
                        .font(.system(size: 16))
                        .foregroundStyle(AppTheme.textTertiary)
                }
            }

            Button {
                showFullSearch = true
            } label: {
                Image(systemName: "arrow.up.left.and.arrow.down.right")
                    .font(.system(size: 14, weight: .semibold))
                    .foregroundStyle(AppTheme.textSecondary)
            }
            .accessibilityLabel("Open full search")
        }
        .padding(.horizontal, AppTheme.spacingMD)
        .padding(.vertical, AppTheme.spacingMD)
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
