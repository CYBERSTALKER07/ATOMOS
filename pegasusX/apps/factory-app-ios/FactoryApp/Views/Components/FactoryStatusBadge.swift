import SwiftUI

struct FactoryStatusBadge: View {
    let text: String
    var tint: Color? = nil
    var emphasized: Bool = true

    private var resolvedTint: Color {
        tint ?? LabTheme.statusTint(for: text)
    }

    var body: some View {
        let readable = text.replacingOccurrences(of: "_", with: " ")
        Text(readable)
            .font(emphasized ? .caption.bold() : .caption2.bold())
            .padding(.horizontal, LabTheme.spacingSM)
            .padding(.vertical, LabTheme.spacingXS)
            .foregroundStyle(emphasized ? resolvedTint : LabTheme.secondaryLabel)
            .background(
                emphasized ? resolvedTint.opacity(0.14) : LabTheme.tertiaryBackground,
                in: Capsule()
            )
            .accessibilityLabel("Status: \(readable)")
    }
}
