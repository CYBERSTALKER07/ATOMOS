import SwiftUI
#if canImport(AppKit)
import AppKit
#endif
#if canImport(UIKit)
import UIKit
#endif

/// Swiss CAD Dark Theme tokens matching the Pegasus web design system.
/// Designed for Apple Human Interface Guidelines and UI/UX Pro Max standards.
public enum SwissCADTheme {
    // MARK: - Color Palette (WCAG AAA Compliance on Pitch Black)
    public static let pitchBlack = Color(red: 0, green: 0, blue: 0)
    public static let panelDark = Color(red: 0.05, green: 0.05, blue: 0.06)
    public static let cardDark = Color(red: 0.08, green: 0.08, blue: 0.09)
    public static let cardHover = Color(red: 0.12, green: 0.12, blue: 0.14)
    public static let inputDark = Color(red: 0.03, green: 0.03, blue: 0.04)

    // MARK: - CAD Accent Hues
    public static let electricCyan = Color(red: 169/255, green: 235/255, blue: 249/255)
    public static let voltLime = Color(red: 251/255, green: 255/255, blue: 99/255)
    public static let emeraldGreen = Color(red: 141/255, green: 220/255, blue: 150/255)
    public static let laserOrange = Color(red: 254/255, green: 89/255, blue: 52/255)
    public static let techGray = Color(white: 0.6)
    public static let mutedGray = Color(white: 0.35)

    // MARK: - Architectural Strokes
    public static let borderStroke = Color.white.opacity(0.14)
    public static let borderStrong = Color.white.opacity(0.3)
    public static let borderCyan = electricCyan.opacity(0.5)

    // MARK: - Status Badge Helpers
    public static func statusColor(for status: String) -> Color {
        switch status.lowercased() {
        case "new":
            return voltLime
        case "in_review":
            return electricCyan
        case "qualified":
            return emeraldGreen
        case "contacted":
            return Color(red: 100/255, green: 180/255, blue: 255/255)
        case "closed":
            return mutedGray
        default:
            return techGray
        }
    }

    public static func statusBackground(for status: String) -> Color {
        statusColor(for: status).opacity(0.14)
    }

    public static func statusBorder(for status: String) -> Color {
        statusColor(for: status).opacity(0.4)
    }

    // MARK: - Haptic Feedback Helper (Apple HIG)
    public static func triggerHaptic() {
        #if os(iOS)
        let generator = UIImpactFeedbackGenerator(style: .medium)
        generator.prepare()
        generator.impactOccurred()
        #endif
    }

    // MARK: - Pegasus Emblem Logo
    public static var logoImage: Image {
        #if canImport(AppKit)
        if let url = Bundle.module.url(forResource: "pegasus_emblem", withExtension: "png"),
           let nsImg = NSImage(contentsOf: url) {
            return Image(nsImage: nsImg)
        }
        if let data = Data(base64Encoded: logoBase64),
           let nsImg = NSImage(data: data) {
            return Image(nsImage: nsImg)
        }
        #elseif canImport(UIKit)
        if let url = Bundle.module.url(forResource: "pegasus_emblem", withExtension: "png"),
           let data = try? Data(contentsOf: url),
           let uiImg = UIImage(data: data) {
            return Image(uiImage: uiImg)
        }
        if let data = Data(base64Encoded: logoBase64),
           let uiImg = UIImage(data: data) {
            return Image(uiImage: uiImg)
        }
        #endif
        return Image(systemName: "shippingbox.fill")
    }

    // MARK: - Micro-chamfered CAD card modifier
    public struct CADCardModifier: ViewModifier {
        let isSelected: Bool

        public func body(content: Content) -> some View {
            content
                .background(cardDark)
                .overlay(
                    Rectangle()
                        .stroke(isSelected ? electricCyan : borderStroke, lineWidth: 1)
                )
        }
    }

    private static let logoBase64 = "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"
}

public extension View {
    func cadCard(isSelected: Bool = false) -> some View {
        modifier(SwissCADTheme.CADCardModifier(isSelected: isSelected))
    }
}
