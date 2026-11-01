import SwiftUI
import UIKit

// MARK: - Tactical Color Extensions
extension Color {
    init(hex: UInt32, opacity: Double = 1.0) {
        let red = Double((hex >> 16) & 0xFF) / 255.0
        let green = Double((hex >> 8) & 0xFF) / 255.0
        let blue = Double(hex & 0xFF) / 255.0
        self.init(.sRGB, red: red, green: green, blue: blue, opacity: opacity)
    }

    init(lightHex: UInt32, darkHex: UInt32) {
        self.init(UIColor { trait in
            trait.userInterfaceStyle == .dark
                ? UIColor(
                    red: CGFloat((darkHex >> 16) & 0xFF) / 255.0,
                    green: CGFloat((darkHex >> 8) & 0xFF) / 255.0,
                    blue: CGFloat(darkHex & 0xFF) / 255.0,
                    alpha: 1.0
                )
                : UIColor(
                    red: CGFloat((lightHex >> 16) & 0xFF) / 255.0,
                    green: CGFloat((lightHex >> 8) & 0xFF) / 255.0,
                    blue: CGFloat(lightHex & 0xFF) / 255.0,
                    alpha: 1.0
                )
        })
    }
}

// MARK: - Tactical Design System Master Tokens (Warm Minimalist Monochrome from Reference Image)
enum TacticalTheme {
    // Canvas & Surfaces
    static let canvas = Color(lightHex: 0xEDEDF0, darkHex: 0x0E0E10)
    static let surface = Color(lightHex: 0xFFFFFF, darkHex: 0x18181B)
    static let surfaceSubtle = Color(lightHex: 0xF6F6F8, darkHex: 0x202024)
    static let surfaceSunken = Color(lightHex: 0xEAEAEB, darkHex: 0x131316)
    static let surfaceRaised = Color(lightHex: 0xFFFFFF, darkHex: 0x1F1F23)
    static let iconTile = Color(lightHex: 0xE2E2E6, darkHex: 0x27272A)

    // Borders (1px Hairline Precision)
    static let border = Color(lightHex: 0xE4E4E7, darkHex: 0x27272A)
    static let borderStrong = Color(lightHex: 0xD4D4D8, darkHex: 0x3F3F46)

    // Typography
    static let textPrimary = Color(lightHex: 0x111113, darkHex: 0xF4F4F5)
    static let textSecondary = Color(lightHex: 0x71717A, darkHex: 0xA1A1AA)
    static let textTertiary = Color(lightHex: 0xA1A1AA, darkHex: 0x71717A)

    // Tactical Accents
    static let accentCobalt = Color(lightHex: 0x2563EB, darkHex: 0x3B82F6)
    static let accentOrange = Color(lightHex: 0xF97316, darkHex: 0xFF7A1A)
    static let accentLime = Color(lightHex: 0x65A30D, darkHex: 0xE2FD52)
    static let accentCyan = Color(lightHex: 0x0284C7, darkHex: 0x06B6D4)
    static let accentPurple = Color(lightHex: 0x7C3AED, darkHex: 0x8B5CF6)

    // Status Semantics
    static let statusSuccess = Color(lightHex: 0x059669, darkHex: 0x10B981)
    static let statusWarning = Color(lightHex: 0xD97706, darkHex: 0xF59E0B)
    static let statusDanger = Color(lightHex: 0xDC2626, darkHex: 0xEF4444)

    // Radius
    static let radiusSM: CGFloat = 8
    static let radiusMD: CGFloat = 12
    static let radiusLG: CGFloat = 16
    static let radiusCard: CGFloat = 24
    static let radiusXL: CGFloat = 28
}

// MARK: - Tactical Card Modifier
struct TacticalCardModifier: ViewModifier {
    var isRaised: Bool = false
    var padding: CGFloat = 16

    func body(content: Content) -> some View {
        content
            .padding(padding)
            .background(isRaised ? TacticalTheme.surfaceRaised : TacticalTheme.surface)
            .clipShape(RoundedRectangle(cornerRadius: TacticalTheme.radiusMD, style: .continuous))
            .overlay(
                RoundedRectangle(cornerRadius: TacticalTheme.radiusMD, style: .continuous)
                    .stroke(TacticalTheme.border, lineWidth: 1)
            )
    }
}

// MARK: - View Modifiers
extension View {
    func tacticalCard(isRaised: Bool = false, padding: CGFloat = 16) -> some View {
        modifier(TacticalCardModifier(isRaised: isRaised, padding: padding))
    }

    func tacticalMicroLabel() -> some View {
        self
            .font(.system(size: 11, weight: .bold, design: .default))
            .textCase(.uppercase)
            .tracking(1.1)
            .foregroundStyle(TacticalTheme.textTertiary)
    }

    func tacticalMonospaceValue(size: CGFloat = 28) -> some View {
        self
            .font(.system(size: size, weight: .bold, design: .monospaced))
            .monospacedDigit()
            .foregroundStyle(TacticalTheme.textPrimary)
    }
}
