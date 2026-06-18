import SwiftUI
import UIKit

/// Shared monochrome design tokens for pegasusX native apps (B&W + status-only accents).
enum PegasusMonochromeTheme {
    static let readableMaxWidth: CGFloat = 960

    static var background: Color { Color(uiColor: .systemGroupedBackground) }
    static var secondaryBackground: Color { Color(uiColor: .secondarySystemGroupedBackground) }
    static var tertiaryBackground: Color { Color(uiColor: .tertiarySystemGroupedBackground) }
    static var card: Color { Color(uiColor: .secondarySystemGroupedBackground) }
    static var fill: Color { Color(uiColor: .systemFill) }
    static var label: Color { Color(uiColor: .label) }
    static var secondaryLabel: Color { Color(uiColor: .secondaryLabel) }
    static var tertiaryLabel: Color { Color(uiColor: .tertiaryLabel) }
    static var separator: Color { Color(uiColor: .separator) }

    static let destructive = Color(red: 1.0, green: 0.23, blue: 0.19)
    static let success = Color(red: 0.20, green: 0.78, blue: 0.35)
    static let warning = Color(red: 1.0, green: 0.58, blue: 0.0)
    static let live = Color(red: 0.0, green: 0.48, blue: 1.0)

    static let spacingXS: CGFloat = 4
    static let spacingSM: CGFloat = 8
    static let spacingMD: CGFloat = 12
    static let spacingLG: CGFloat = 16
    static let spacingXL: CGFloat = 24
    static let spacingXXL: CGFloat = 32

    static let radiusXS: CGFloat = 4
    static let radiusSM: CGFloat = 8
    static let radiusMD: CGFloat = 12
    static let radiusLG: CGFloat = 16
    static let radiusXL: CGFloat = 28

    /// Maps order/ops status strings to semantic badge tints.
    static func statusTint(for status: String) -> Color {
        switch status.uppercased() {
        case "COMPLETED", "DONE", "ACTIVE", "APPROVED", "SEALED", "DISPATCHED", "RECEIVED", "PAID", "OK", "SUCCESS":
            return success
        case "PENDING", "AWAITING_REVIEW", "AWAITING_PAYMENT", "LOADING", "IN_TRANSIT", "OPEN",
             "SUBMITTED", "ACKNOWLEDGED", "URGENT", "DRAFT", "WARN":
            return warning
        case "CANCELLED", "REJECTED", "FAILED", "EXCEPTION", "CRITICAL", "OVERDUE", "FAIL":
            return destructive
        case "LIVE", "TRANSIT":
            return live
        default:
            return secondaryLabel
        }
    }
}

struct PegasusMonochromeCard: ViewModifier {
    func body(content: Content) -> some View {
        content
            .padding(PegasusMonochromeTheme.spacingLG)
            .background(PegasusMonochromeTheme.card)
            .clipShape(RoundedRectangle(cornerRadius: PegasusMonochromeTheme.radiusLG, style: .continuous))
    }
}

extension View {
    func pegasusMonochromeCard() -> some View {
        modifier(PegasusMonochromeCard())
    }

    func pegasusReadableWidth() -> some View {
        frame(maxWidth: PegasusMonochromeTheme.readableMaxWidth)
            .frame(maxWidth: .infinity)
    }
}
