import SwiftUI

struct CatalogBrowseChips: View {
    @Binding var browseMode: CatalogBrowseMode
    var onNavigateToSuppliers: () -> Void

    var body: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: AppTheme.spacingSM) {
                browseChip("Categories", selected: browseMode == .categories) {
                    browseMode = .categories
                }
                browseChip("All products", selected: browseMode == .allProducts) {
                    browseMode = .allProducts
                }
                browseChip("Suppliers", selected: false) {
                    onNavigateToSuppliers()
                }
            }
        }
    }

    func browseChip(_ title: String, selected: Bool, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            Text(title)
                .font(.system(.subheadline, design: .rounded, weight: .semibold))
                .foregroundStyle(selected ? AppTheme.cardBackground : AppTheme.textPrimary)
                .padding(.horizontal, AppTheme.spacingMD)
                .padding(.vertical, AppTheme.spacingSM)
                .background(selected ? AppTheme.textPrimary : AppTheme.cardBackground)
                .clipShape(.capsule)
                .overlay {
                    Capsule()
                        .stroke(AppTheme.separator.opacity(0.2), lineWidth: selected ? 0 : 1)
                }
        }
        .buttonStyle(.plain)
    }

}
