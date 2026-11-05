import AppKit
import SwiftUI

// MARK: - Color Hex Helpers

public extension Color {
    init(hex: UInt32, alpha: Double = 1.0) {
        let r = Double((hex >> 16) & 0xFF) / 255.0
        let g = Double((hex >> 8) & 0xFF) / 255.0
        let b = Double(hex & 0xFF) / 255.0
        self.init(.sRGB, red: r, green: g, blue: b, opacity: alpha)
    }

    init(lightHex: UInt32, darkHex: UInt32) {
        self.init(nsColor: NSColor(name: nil, dynamicProvider: { appearance in
            let isDark = appearance.bestMatch(from: [.darkAqua, .aqua]) == .darkAqua
            let hex = isDark ? darkHex : lightHex
            let r = CGFloat((hex >> 16) & 0xFF) / 255.0
            let g = CGFloat((hex >> 8) & 0xFF) / 255.0
            let b = CGFloat(hex & 0xFF) / 255.0
            return NSColor(srgbRed: r, green: g, blue: b, alpha: 1.0)
        }))
    }
}

// MARK: - Control Tower Design System (Sampled from Global Switch Reference Palette)
/// Color palette sampled from reference image:
/// - Primary Dark Canvas: #101010 / #0D0D0D (Obsidian Black)
/// - Card Surface: #161717 / #181919 (Deep Charcoal)
/// - Raised Surface / Wells: #252525 / #292929
/// - High-Precision Borders: #2D3130 / #3A3A3A
/// - Warm Stone / Titanium Muted Text: #B6AAA3 / #B2A69E
/// - Signature Accent: #1CC29F / #2DCAA6 (Electric Mint / Teal Glow)
/// - Supporting Telemetry Slate: #466766 / #3A5757
/// - Supporting Warm Copper / Terracotta: #BA8372 / #D9A897
public enum ControlTowerTheme {
    // MARK: - Backgrounds & Surfaces
    public static var canvas: Color {
        Color(lightHex: 0xF4F4F6, darkHex: 0x101010)
    }

    public static var surface: Color {
        Color(lightHex: 0xFFFFFF, darkHex: 0x161717)
    }

    public static var surfaceSubtle: Color {
        Color(lightHex: 0xF8F8FA, darkHex: 0x1D1F1F)
    }

    public static var surfaceElevated: Color {
        Color(lightHex: 0xFFFFFF, darkHex: 0x242626)
    }

    public static var inputWell: Color {
        Color(lightHex: 0xEEEEF1, darkHex: 0x252525)
    }

    // MARK: - Borders & Dividers
    public static var border: Color {
        Color(lightHex: 0xE2E2E6, darkHex: 0x2D3130)
    }

    public static var borderStrong: Color {
        Color(lightHex: 0xCBCBD0, darkHex: 0x404443)
    }

    // MARK: - Typography
    public static var textPrimary: Color {
        Color(lightHex: 0x111827, darkHex: 0xFFFFFF)
    }

    public static var textSecondary: Color {
        Color(lightHex: 0x5E564F, darkHex: 0xB6AAA3)
    }

    public static var textTertiary: Color {
        Color(lightHex: 0x8C847E, darkHex: 0x7E7670)
    }

    // MARK: - Accents & Brand
    public static var accentTeal: Color {
        Color(lightHex: 0x008A6F, darkHex: 0x2DCAA6)
    }

    public static var accentTealHover: Color {
        Color(lightHex: 0x00775F, darkHex: 0x3CDAB6)
    }

    public static var accentTealSoft: Color {
        Color(lightHex: 0xE0F7F2, darkHex: 0x162E29)
    }

    public static var slateTeal: Color {
        Color(lightHex: 0x3E5E5D, darkHex: 0x466766)
    }

    public static var warmCopper: Color {
        Color(lightHex: 0xA36653, darkHex: 0xBA8372)
    }

    public static var warmCopperSoft: Color {
        Color(lightHex: 0xFBEFEA, darkHex: 0x372822)
    }

    // MARK: - Status Indicators
    public static var statusNominal: Color {
        Color(lightHex: 0x008A6F, darkHex: 0x2DCAA6)
    }

    public static var statusWarning: Color {
        Color(lightHex: 0xC26A1B, darkHex: 0xE8A058)
    }

    public static var statusCritical: Color {
        Color(lightHex: 0xD92D20, darkHex: 0xFF5449)
    }

    public static var statusInfo: Color {
        Color(lightHex: 0x1E5AC8, darkHex: 0x5894FF)
    }
}

// MARK: - View Modifiers for Tactical Card Styling
public struct ControlTowerCardModifier: ViewModifier {
    var borderColor: Color
    var backgroundColor: Color

    public init(borderColor: Color = ControlTowerTheme.border, backgroundColor: Color = ControlTowerTheme.surface) {
        self.borderColor = borderColor
        self.backgroundColor = backgroundColor
    }

    public func body(content: Content) -> some View {
        content
            .background(backgroundColor)
            .overlay(
                RoundedRectangle(cornerRadius: 12, style: .continuous)
                    .stroke(borderColor, lineWidth: 1)
            )
            .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
            .shadow(color: Color.black.opacity(0.12), radius: 6, x: 0, y: 2)
    }
}

public extension View {
    func controlTowerCard(borderColor: Color = ControlTowerTheme.border, backgroundColor: Color = ControlTowerTheme.surface) -> some View {
        modifier(ControlTowerCardModifier(borderColor: borderColor, backgroundColor: backgroundColor))
    }
}
