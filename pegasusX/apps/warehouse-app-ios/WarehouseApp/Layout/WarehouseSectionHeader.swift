import SwiftUI

struct WarehouseSectionHeader: View {
    let title: String
    var subtitle: String? = nil

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
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

struct WarehouseStatusBadge: View {
    let text: String
    var tint: Color? = nil

    private var resolvedTint: Color {
        tint ?? LabTheme.statusTint(for: text)
    }

    var body: some View {
        Text(text.replacingOccurrences(of: "_", with: " "))
            .font(.caption.bold())
            .padding(.horizontal, LabTheme.spacingSM)
            .padding(.vertical, LabTheme.spacingXS)
            .foregroundStyle(resolvedTint)
            .background(resolvedTint.opacity(0.14), in: Capsule())
    }
}
