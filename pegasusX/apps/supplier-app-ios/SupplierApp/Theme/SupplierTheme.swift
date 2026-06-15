import SwiftUI

enum SupplierTheme {
    static let background = Color(uiColor: .systemGroupedBackground)
    static let secondaryBackground = Color(uiColor: .secondarySystemGroupedBackground)
    static let tertiaryBackground = Color(uiColor: .tertiarySystemGroupedBackground)
    static let label = Color(uiColor: .label)
    static let secondaryLabel = Color(uiColor: .secondaryLabel)
    static let fill = Color(uiColor: .systemFill)
    static let success = Color.green
    static let warning = Color.orange
    static let destructive = Color.red
    static let live = Color.green

    /// Maps order/ops status strings to semantic badge tints (aligned with factory/warehouse native).
    static func statusTint(for status: String) -> Color {
        switch status.uppercased() {
        case "COMPLETED", "DONE", "ACTIVE", "APPROVED", "SEALED":
            return success
        case "PENDING", "AWAITING_REVIEW", "AWAITING_PAYMENT", "LOADING", "IN_TRANSIT":
            return warning
        case "CANCELLED", "REJECTED", "FAILED", "EXCEPTION":
            return destructive
        default:
            return secondaryLabel
        }
    }

    static let spacingXS: CGFloat = 4
    static let spacingSM: CGFloat = 8
    static let spacingMD: CGFloat = 12
    static let spacingLG: CGFloat = 16
    static let spacingXL: CGFloat = 24
    static let spacingXXL: CGFloat = 32

    static let radiusSM: CGFloat = 8
    static let radiusMD: CGFloat = 12
    static let radiusLG: CGFloat = 16

    /// Keeps dense ops tables readable on iPad without stretching edge-to-edge.
    static let readableMaxWidth: CGFloat = 960
}

enum SupplierAnim {
    static let smooth = Animation.smooth(duration: 0.35)
}

struct SupplierCardModifier: ViewModifier {
    func body(content: Content) -> some View {
        content
            .padding(SupplierTheme.spacingLG)
            .background(SupplierTheme.secondaryBackground, in: RoundedRectangle(cornerRadius: SupplierTheme.radiusMD))
    }
}

extension View {
    func supplierCard() -> some View {
        modifier(SupplierCardModifier())
    }

    func supplierReadableWidth() -> some View {
        frame(maxWidth: SupplierTheme.readableMaxWidth)
            .frame(maxWidth: .infinity)
    }
}
