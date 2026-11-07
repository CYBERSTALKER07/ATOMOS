import SwiftUI
#if canImport(UIKit)
import UIKit

public extension UIColor {
    convenience init(lightHex: UInt32, darkHex: UInt32) {
        self.init { trait in
            let hex = trait.userInterfaceStyle == .dark ? darkHex : lightHex
            let r = CGFloat((hex >> 16) & 0xFF) / 255.0
            let g = CGFloat((hex >> 8) & 0xFF) / 255.0
            let b = CGFloat(hex & 0xFF) / 255.0
            return UIColor(red: r, green: g, blue: b, alpha: 1.0)
        }
    }
}
#endif

#if canImport(AppKit) && !canImport(UIKit)
import AppKit

public extension NSColor {
    convenience init(lightHex: UInt32, darkHex: UInt32) {
        self.init(name: nil) { appearance in
            let isDark = appearance.bestMatch(from: [.darkAqua, .aqua]) == .darkAqua
            let hex = isDark ? darkHex : lightHex
            let r = CGFloat((hex >> 16) & 0xFF) / 255.0
            let g = CGFloat((hex >> 8) & 0xFF) / 255.0
            let b = CGFloat(hex & 0xFF) / 255.0
            return NSColor(srgbRed: r, green: g, blue: b, alpha: 1.0)
        }
    }
}
#endif

/// Shared warm monochrome design tokens for pegasusX native apps (sampled from tactile minimal reference image).
public enum PegasusMonochromeTheme {
    public static let readableMaxWidth: CGFloat = 960

    #if canImport(UIKit)
    /// Warm neutral canvas background (#EDEDF0 light / #0E0E10 dark).
    public static var background: Color { Color(uiColor: UIColor(lightHex: 0xEDEDF0, darkHex: 0x0E0E10)) }
    /// Crisp elevated card surface (#FFFFFF light / #18181B dark).
    public static var card: Color { Color(uiColor: UIColor(lightHex: 0xFFFFFF, darkHex: 0x18181B)) }
    /// Secondary container surface (#F6F6F8 light / #202024 dark).
    public static var secondaryBackground: Color { Color(uiColor: UIColor(lightHex: 0xF6F6F8, darkHex: 0x202024)) }
    /// Tertiary sunken background (#EAEAEB light / #27272A dark).
    public static var tertiaryBackground: Color { Color(uiColor: UIColor(lightHex: 0xEAEAEB, darkHex: 0x27272A)) }
    /// Tactile squircle icon tile container fill (#E2E2E6 light / #27272A dark).
    public static var iconTile: Color { Color(uiColor: UIColor(lightHex: 0xE2E2E6, darkHex: 0x27272A)) }
    public static var fill: Color { iconTile }
    /// Deep soot primary text & chart bars (#111113 light / #F4F4F5 dark).
    public static var label: Color { Color(uiColor: UIColor(lightHex: 0x111113, darkHex: 0xF4F4F5)) }
    /// Sophisticated neutral charcoal secondary label (#71717A light / #A1A1AA dark).
    public static var secondaryLabel: Color { Color(uiColor: UIColor(lightHex: 0x71717A, darkHex: 0xA1A1AA)) }
    /// Tertiary timestamps & axis labels (#A1A1AA light / #71717A dark).
    public static var tertiaryLabel: Color { Color(uiColor: UIColor(lightHex: 0xA1A1AA, darkHex: 0x71717A)) }
    /// Subtle hairline separator (#E4E4E7 light / #27272A dark).
    public static var separator: Color { Color(uiColor: UIColor(lightHex: 0xE4E4E7, darkHex: 0x27272A)) }
    /// Ultra-diffuse card border (0.04 black light / 0.08 white dark).
    public static var cardBorder: Color { Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor.white.withAlphaComponent(0.08) : UIColor.black.withAlphaComponent(0.04) }) }
    /// High-contrast primary action button / FAB (#111113 light / #FFFFFF dark).
    public static var primaryAction: Color { Color(uiColor: UIColor(lightHex: 0x111113, darkHex: 0xFFFFFF)) }
    public static var onPrimaryAction: Color { Color(uiColor: UIColor(lightHex: 0xFFFFFF, darkHex: 0x111113)) }
    /// Warm terracotta accent sampled from reference image (#BC7F70 light / #D2B39B dark).
    public static var accentWarm: Color { Color(uiColor: UIColor(lightHex: 0xBC7F70, darkHex: 0xD2B39B)) }
    public static var accentWarmSoft: Color { Color(uiColor: UIColor(lightHex: 0xF5ECE9, darkHex: 0x362723)) }
    public static var accentBronze: Color { Color(uiColor: UIColor(lightHex: 0x624D45, darkHex: 0xE4BFA2)) }
    /// Solid monochrome chart bars.
    public static var barPrimary: Color { primaryAction }
    public static var barMuted: Color { Color(uiColor: UIColor(lightHex: 0xD4D4D8, darkHex: 0x3F3F46)) }
    #else
    public static var background: Color { Color(nsColor: NSColor(lightHex: 0xEDEDF0, darkHex: 0x0E0E10)) }
    public static var card: Color { Color(nsColor: NSColor(lightHex: 0xFFFFFF, darkHex: 0x18181B)) }
    public static var secondaryBackground: Color { Color(nsColor: NSColor(lightHex: 0xF6F6F8, darkHex: 0x202024)) }
    public static var tertiaryBackground: Color { Color(nsColor: NSColor(lightHex: 0xEAEAEB, darkHex: 0x27272A)) }
    public static var iconTile: Color { Color(nsColor: NSColor(lightHex: 0xE2E2E6, darkHex: 0x27272A)) }
    public static var fill: Color { iconTile }
    public static var label: Color { Color(nsColor: NSColor(lightHex: 0x111113, darkHex: 0xF4F4F5)) }
    public static var secondaryLabel: Color { Color(nsColor: NSColor(lightHex: 0x71717A, darkHex: 0xA1A1AA)) }
    public static var tertiaryLabel: Color { Color(nsColor: NSColor(lightHex: 0xA1A1AA, darkHex: 0x71717A)) }
    public static var separator: Color { Color(nsColor: NSColor(lightHex: 0xE4E4E7, darkHex: 0x27272A)) }
    public static var cardBorder: Color { Color(nsColor: NSColor(name: nil) { $0.bestMatch(from: [.darkAqua, .aqua]) == .darkAqua ? NSColor.white.withAlphaComponent(0.08) : NSColor.black.withAlphaComponent(0.04) }) }
    public static var primaryAction: Color { Color(nsColor: NSColor(lightHex: 0x111113, darkHex: 0xFFFFFF)) }
    public static var onPrimaryAction: Color { Color(nsColor: NSColor(lightHex: 0xFFFFFF, darkHex: 0x111113)) }
    public static var accentWarm: Color { Color(nsColor: NSColor(lightHex: 0xBC7F70, darkHex: 0xD2B39B)) }
    public static var accentWarmSoft: Color { Color(nsColor: NSColor(lightHex: 0xF5ECE9, darkHex: 0x362723)) }
    public static var accentBronze: Color { Color(nsColor: NSColor(lightHex: 0x624D45, darkHex: 0xE4BFA2)) }
    public static var barPrimary: Color { primaryAction }
    public static var barMuted: Color { Color(nsColor: NSColor(lightHex: 0xD4D4D8, darkHex: 0x3F3F46)) }
    #endif

    public static let destructive = Color(red: 0.93, green: 0.27, blue: 0.27)
    public static let success = Color(red: 0.13, green: 0.77, blue: 0.37)
    public static let warning = Color(red: 0.96, green: 0.62, blue: 0.04)
    public static let live = Color(red: 0.23, green: 0.51, blue: 0.96)

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
    public static let radiusCard: CGFloat = 24
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
            .clipShape(RoundedRectangle(cornerRadius: PegasusMonochromeTheme.radiusCard, style: .continuous))
            .overlay(
                RoundedRectangle(cornerRadius: PegasusMonochromeTheme.radiusCard, style: .continuous)
                    .stroke(PegasusMonochromeTheme.cardBorder, lineWidth: 1)
            )
            .shadow(color: Color.black.opacity(0.03), radius: 6, x: 0, y: 2)
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

/// HIG-compliant fluid press animation style for buttons & navigation links.
public struct PegasusPressableButtonStyle: ButtonStyle {
    public init() {}
    public func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .scaleEffect(configuration.isPressed ? 0.97 : 1.0)
            .opacity(configuration.isPressed ? 0.88 : 1.0)
            .animation(.spring(response: 0.22, dampingFraction: 0.82), value: configuration.isPressed)
    }
}

public extension ButtonStyle where Self == PegasusPressableButtonStyle {
    static var pegasusPressable: PegasusPressableButtonStyle { PegasusPressableButtonStyle() }
}
