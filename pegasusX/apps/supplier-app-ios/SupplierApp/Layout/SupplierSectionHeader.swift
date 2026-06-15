import SwiftUI

struct SupplierSectionHeader: View {
    let title: String
    var subtitle: String? = nil

    var body: some View {
        VStack(alignment: .leading, spacing: SupplierTheme.spacingXS) {
            Text(title)
                .font(.headline)
            if let subtitle, !subtitle.isEmpty {
                Text(subtitle)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }
}

struct SupplierStatusBadge: View {
    let text: String
    var tint: Color? = nil

    private var resolvedTint: Color {
        tint ?? SupplierTheme.statusTint(for: text)
    }

    var body: some View {
        Text(text)
            .font(.caption.bold())
            .padding(.horizontal, SupplierTheme.spacingSM)
            .padding(.vertical, SupplierTheme.spacingXS)
            .foregroundStyle(resolvedTint)
            .background(resolvedTint.opacity(0.14), in: Capsule())
    }
}
