//
//  LabTheme.swift
//  driverappios
//

import SwiftUI

// MARK: - Lab Theme — Monochrome, HIG-native

enum LabTheme {

    // MARK: Adaptive Colors (Native Tactical)

    /// Primary foreground — adapts automatically
    static let fg = Color(uiColor: .label)
    /// Secondary foreground
    static let fgSecondary = Color(uiColor: .secondaryLabel)
    /// Tertiary foreground (Strategic label color)
    static let fgTertiary = Color(uiColor: .tertiaryLabel)
    /// Card surface (Pure Apple background)
    static let card = Color(uiColor: .secondarySystemGroupedBackground)
    /// Page background
    static let bg = Color(uiColor: .systemGroupedBackground)
    /// Button foreground on filled buttons
    static let buttonFg = Color(uiColor: .systemBackground)
    /// Separator (Tactical stroke)
    static let separator = Color(uiColor: .separator)

    // MARK: Semantic Status (Monochrome + Tactical Accents)

    static let destructive = Color(uiColor: .systemRed)
    static let success     = Color(uiColor: .systemGreen)
    static let warning     = Color(uiColor: .systemOrange)
    static let live        = Color(uiColor: .systemGreen)
    static let offline     = Color(uiColor: .systemRed)
    static let transit     = Color(uiColor: .systemBlue)

    // MARK: Corner Radii (Native Fluid)

    static let cardRadius: Double   = 24
    static let buttonRadius: Double = 16
    static let pillRadius: Double   = 100
    static let modalRadius: Double  = 30
    static let sheetRadius: Double  = 34

    // MARK: Spacing (H-Scale)

    static let s4: Double  = 4
    static let s8: Double  = 8
    static let s12: Double = 12
    static let s16: Double = 16
    static let s20: Double = 20
    static let s24: Double = 24
    static let s32: Double = 32

    // MARK: Typography helpers

    static let mono: Font = .system(.caption, design: .monospaced, weight: .semibold)
}

// MARK: - Animation Tokens

enum Anim {
    static let snappy      = Animation.spring(response: 0.35, dampingFraction: 0.7)
    static let bouncy      = Animation.spring(response: 0.5, dampingFraction: 0.72)
    static let sheetReveal = Animation.spring(response: 0.55, dampingFraction: 0.78)
    static let settle      = Animation.spring(response: 0.6, dampingFraction: 0.85)
    static let micro       = Animation.spring(response: 0.25, dampingFraction: 0.8)
    static let breathe     = Animation.easeInOut(duration: 1.6).repeatForever(autoreverses: true)

    static func stagger(_ index: Int, base: Animation = .spring(response: 0.4, dampingFraction: 0.75)) -> Animation {
        base.delay(Double(index) * 0.06)
    }
}

// MARK: - Currency Formatter

extension Int {
    var formattedAmount: String {
        let formatted = self.formatted(.number.grouping(.automatic))
        return "\(formatted)"
    }
}
