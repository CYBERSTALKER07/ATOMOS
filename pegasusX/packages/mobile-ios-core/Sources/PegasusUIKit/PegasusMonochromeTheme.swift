import SwiftUI
#if canImport(UIKit)
import UIKit
#endif

/// Shared monochrome design tokens for pegasusX native apps (B&W + status-only accents).
public enum PegasusMonochromeTheme {
    public static let readableMaxWidth: CGFloat = 960

    #if canImport(UIKit)
    public static var background: Color { Color(uiColor: .systemGroupedBackground) }
    public static var secondaryBackground: Color { Color(uiColor: .secondarySystemGroupedBackground) }
    public static var tertiaryBackground: Color { Color(uiColor: .tertiarySystemGroupedBackground) }
    public static var card: Color { Color(uiColor: .secondarySystemGroupedBackground) }
    public static var fill: Color { Color(uiColor: .systemFill) }
    public static var label: Color { Color(uiColor: .label) }
    public static var secondaryLabel: Color { Color(uiColor: .secondaryLabel) }
    public static var tertiaryLabel: Color { Color(uiColor: .tertiaryLabel) }
    public static var separator: Color { Color(uiColor: .separator) }
    #else
    public static var background: Color { Color(NSColor.windowBackgroundColor) }
    public static var secondaryBackground: Color { Color(NSColor.controlBackgroundColor) }
    public static var tertiaryBackground: Color { Color(NSColor.underPageBackgroundColor) }
    public static var card: Color { Color(NSColor.controlBackgroundColor) }
    public static var fill: Color { Color.gray.opacity(0.15) }
    public static var label: Color { Color.primary }
    public static var secondaryLabel: Color { Color.secondary }
    public static var tertiaryLabel: Color { Color.secondary.opacity(0.7) }
    public static var separator: Color { Color.gray.opacity(0.2) }
    #endif

    public static let destructive = Color(red: 1.0, green: 0.23, blue: 0.19)
    public static let success = Color(red: 0.20, green: 0.78, blue: 0.35)
    public static let warning = Color(red: 1.0, green: 0.58, blue: 0.0)
    public static let live = Color(red: 0.0, green: 0.48, blue: 1.0)

    public static let spacingXS: CGFloat = 4
    public static let spacingSM: CGFloat = 8
    public static let spacingMD: CGFloat = 12
    public static let spacingLG: CGFloat = 16
    public static let spacingXL: CGFloat = 24
    public static let spacingXXL: CGFloat = 32

    public static let radiusXS: CGFloat = 4
    public static let radiusSM: CGFloat = 8
    public static let radiusMD: CGFloat = 12
    public static let radiusLG: CGFloat = 16
    public static let radiusXL: CGFloat = 28

    /// Maps order/ops status strings to semantic badge tints.
    public static func statusTint(for status: String) -> Color {
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

public struct PegasusMonochromeCard: ViewModifier {
    public init() {}
    public func body(content: Content) -> some View {
        content
            .padding(PegasusMonochromeTheme.spacingLG)
            .background(PegasusMonochromeTheme.card)
            .clipShape(RoundedRectangle(cornerRadius: PegasusMonochromeTheme.radiusLG, style: .continuous))
            .overlay(
                RoundedRectangle(cornerRadius: PegasusMonochromeTheme.radiusLG, style: .continuous)
                    .stroke(PegasusMonochromeTheme.separator.opacity(0.3), lineWidth: 0.5)
            )
    }
}

/// HIG minimum touch target modifier (44x44pt on iOS).
public struct MinimumHitTargetModifier: ViewModifier {
    public var minSize: CGFloat

    public init(minSize: CGFloat = 44) {
        self.minSize = minSize
    }

    public func body(content: Content) -> some View {
        content
            .frame(minWidth: minSize, minHeight: minSize)
            .contentShape(Rectangle())
    }
}

public extension View {
    func pegasusMonochromeCard() -> some View {
        modifier(PegasusMonochromeCard())
    }

    func pegasusReadableWidth() -> some View {
        frame(maxWidth: PegasusMonochromeTheme.readableMaxWidth)
            .frame(maxWidth: .infinity)
    }

    /// Enforces HIG minimum touch targets (default 44x44pt) with accessible content bounds.
    func pegasusMinimumHitTarget(_ minSize: CGFloat = 44) -> some View {
        modifier(MinimumHitTargetModifier(minSize: minSize))
    }
}

/// Production tactile haptic feedback provider for pegasusX mobile clients adhering to Apple HIG.
public enum PegasusHaptics {
    public static func light() {
        #if os(iOS)
        let generator = UIImpactFeedbackGenerator(style: .light)
        generator.prepare()
        generator.impactOccurred()
        #endif
    }

    public static func medium() {
        #if os(iOS)
        let generator = UIImpactFeedbackGenerator(style: .medium)
        generator.prepare()
        generator.impactOccurred()
        #endif
    }

    public static func heavy() {
        #if os(iOS)
        let generator = UIImpactFeedbackGenerator(style: .heavy)
        generator.prepare()
        generator.impactOccurred()
        #endif
    }

    public static func selection() {
        #if os(iOS)
        let generator = UISelectionFeedbackGenerator()
        generator.prepare()
        generator.selectionChanged()
        #endif
    }

    public static func success() {
        #if os(iOS)
        let generator = UINotificationFeedbackGenerator()
        generator.prepare()
        generator.notificationOccurred(.success)
        #endif
    }

    public static func warning() {
        #if os(iOS)
        let generator = UINotificationFeedbackGenerator()
        generator.prepare()
        generator.notificationOccurred(.warning)
        #endif
    }

    public static func error() {
        #if os(iOS)
        let generator = UINotificationFeedbackGenerator()
        generator.prepare()
        generator.notificationOccurred(.error)
        #endif
    }
}
