//
//  LabTheme.swift
//  driverappios
//

import SwiftUI

// MARK: - Lab Theme — Monochrome, HIG-native

enum LabTheme {

    // MARK: Adaptive Colors (Warm Tactile Minimalist - Sampled from Reference Image)

    /// Primary foreground — deep soot (#111113 light / #F4F4F5 dark)
    static let fg = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 244/255.0, green: 244/255.0, blue: 245/255.0, alpha: 1.0) : UIColor(red: 17/255.0, green: 17/255.0, blue: 19/255.0, alpha: 1.0) })
    /// Secondary foreground — neutral charcoal (#71717A light / #A1A1AA dark)
    static let fgSecondary = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 161/255.0, green: 161/255.0, blue: 170/255.0, alpha: 1.0) : UIColor(red: 113/255.0, green: 113/255.0, blue: 122/255.0, alpha: 1.0) })
    /// Tertiary foreground — muted gray (#A1A1AA light / #71717A dark)
    static let fgTertiary = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 113/255.0, green: 113/255.0, blue: 122/255.0, alpha: 1.0) : UIColor(red: 161/255.0, green: 161/255.0, blue: 170/255.0, alpha: 1.0) })
    /// Card surface — elevated white (#FFFFFF light / #18181B dark)
    static let card = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 24/255.0, green: 24/255.0, blue: 27/255.0, alpha: 1.0) : UIColor.white })
    /// Page background — warm neutral paper (#EDEDF0 light / #0E0E10 dark)
    static let bg = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 14/255.0, green: 14/255.0, blue: 16/255.0, alpha: 1.0) : UIColor(red: 237/255.0, green: 237/255.0, blue: 240/255.0, alpha: 1.0) })
    /// Tactile icon squircle container fill (#E2E2E6 light / #27272A dark)
    static let iconTile = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 39/255.0, green: 39/255.0, blue: 42/255.0, alpha: 1.0) : UIColor(red: 226/255.0, green: 226/255.0, blue: 230/255.0, alpha: 1.0) })
    /// Button foreground on filled buttons
    static let buttonFg = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 17/255.0, green: 17/255.0, blue: 19/255.0, alpha: 1.0) : UIColor.white })
    /// Separator — subtle hairline (#E4E4E7 light / #27272A dark)
    static let separator = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 39/255.0, green: 39/255.0, blue: 42/255.0, alpha: 1.0) : UIColor(red: 228/255.0, green: 228/255.0, blue: 231/255.0, alpha: 1.0) })
    /// Subtle card border
    static let cardBorder = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor.white.withAlphaComponent(0.08) : UIColor.black.withAlphaComponent(0.04) })

    // MARK: Semantic Status (Monochrome + Tactical Accents)

    static let destructive = Color(uiColor: .systemRed)
    static let success     = Color(uiColor: .systemGreen)
    static let warning     = Color(uiColor: .systemOrange)
    static let live        = Color(uiColor: .systemGreen)
    static let offline     = Color(uiColor: .systemRed)
    static let transit     = Color(uiColor: .systemBlue)

    /// Maps delivery/ops status strings to semantic badge tints (aligned with warehouse/payload native).
    static func statusTint(for status: String) -> Color {
        switch status.uppercased() {
        case "COMPLETED", "DONE", "DELIVERED", "ACTIVE", "ON_ROUTE", "IN_TRANSIT",
             "ASSIGNED", "LOADED", "SEALED", "LIVE":
            return success
        case "PENDING", "IDLE", "AWAITING_SEAL", "AWAITING_PAYLOAD", "LOADING",
             "RETURNING", "READY", "READY_TO_DEPART":
            return warning
        case "CANCELLED", "REJECTED", "FAILED", "OFFLINE", "GPS_ERROR", "BLOCKED":
            return destructive
        default:
            return fgTertiary
        }
    }

    /// Keeps dense driver ops readable on iPad without stretching edge-to-edge.
    static let readableMaxWidth: CGFloat = 560

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
