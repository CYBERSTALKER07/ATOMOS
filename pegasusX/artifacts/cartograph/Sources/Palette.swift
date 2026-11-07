import SwiftUI

/// Ink on paper — black and white only.
enum Ink {
    static let bg = Color.black
    static let fg = Color.white
    static let dim = Color.white.opacity(0.55)
    static let faint = Color.white.opacity(0.22)
    static let hair = Color.white.opacity(0.12)
    static let fill1 = Color.white.opacity(0.06)
    static let fill2 = Color.white.opacity(0.10)
    static let panel = Color.black
    static let invert = Color.black

    static func gray(_ n: Double) -> Color {
        Color(white: max(0, min(1, n)))
    }

    static func verdict(_ v: Verdict) -> Color { fg }
}

enum Typeface {
    static func display(_ size: CGFloat, weight: Font.Weight = .semibold) -> Font {
        .custom("Avenir Next Condensed", size: size).weight(weight)
    }

    static func serif(_ size: CGFloat) -> Font {
        .custom("Iowan Old Style", size: size)
    }

    static func mono(_ size: CGFloat) -> Font {
        .system(size: size, design: .monospaced)
    }
}

enum PanelTab: String, CaseIterable, Identifiable {
    case path = "Path"
    case hall = "Hall"
    case key = "Key"
    var id: String { rawValue }
}
