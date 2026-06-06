import SwiftUI

// MARK: - Payload Theme (Native Tactical)
// Unified design tokens for the iPad Payload Terminal.
// Follows the B&W Tactical Aesthetic with 32px standard iPad radii.

enum TermTheme {
    // MARK: Colors
    static let bg = Color(UIColor.systemGroupedBackground)
    static let card = Color(UIColor.systemBackground)
    static let accent = Color(UIColor.label)
    static let secondary = Color(UIColor.secondaryLabel)
    static let tertiary = Color(UIColor.tertiaryLabel)
    static let separator = Color(UIColor.separator)
    
    // Status
    static let live = Color(UIColor.systemGreen)
    static let alert = Color(UIColor.systemRed)
    static let warn = Color(UIColor.systemOrange)
    static let progress = Color(UIColor.systemBlue)
    
    // MARK: Corner Radii (iPad Optimization)
    static let radiusLG: Double = 32
    static let radiusMD: Double = 24
    static let radiusSM: Double = 16
    static let radiusXS: Double = 8
    
    // MARK: Spacing
    static let s4: Double = 4
    static let s8: Double = 8
    static let s12: Double = 12
    static let s16: Double = 16
    static let s20: Double = 20
    static let s24: Double = 24
    static let s32: Double = 32
}

// MARK: - Global Tactical Modifiers

extension View {
    func tacticalCard(radius: Double = TermTheme.radiusMD) -> some View {
        self
            .background {
                RoundedRectangle(cornerRadius: radius, style: .continuous)
                    .fill(TermTheme.card)
                    .overlay {
                        RoundedRectangle(cornerRadius: radius, style: .continuous)
                            .stroke(TermTheme.separator.opacity(0.12), lineWidth: 1)
                    }
            }
    }
    
    func tacticalButton(radius: Double = TermTheme.radiusSM) -> some View {
        self
            .background {
                RoundedRectangle(cornerRadius: radius, style: .continuous)
                    .fill(TermTheme.accent)
            }
    }
}

// MARK: - Common Animation

enum TermAnim {
    static let snappy = Animation.spring(response: 0.35, dampingFraction: 0.85, blendDuration: 0)
    static let fluid = Animation.spring(response: 0.5, dampingFraction: 0.9, blendDuration: 0)
}

// MARK: - Pressable Button Style

private struct PressableModifier: ViewModifier {
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @State private var isPressed = false

    func body(content: Content) -> some View {
        content
            .scaleEffect(isPressed && !reduceMotion ? 0.96 : 1)
            .opacity(isPressed ? 0.82 : 1)
            .animation(TermAnim.snappy, value: isPressed)
            .simultaneousGesture(
                DragGesture(minimumDistance: 0)
                    .onChanged { _ in isPressed = true }
                    .onEnded { _ in isPressed = false }
            )
    }
}

extension View {
    /// Adds a tactile scale + fade press animation to any interactive element.
    func pressable() -> some View { modifier(PressableModifier()) }
}

// MARK: - Tactical Button Style (for Button-based interactive elements)

struct TacticalButtonStyle: ButtonStyle {
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .scaleEffect(configuration.isPressed && !reduceMotion ? 0.96 : 1)
            .opacity(configuration.isPressed ? 0.82 : 1)
            .animation(TermAnim.snappy, value: configuration.isPressed)
    }
}

extension ButtonStyle where Self == TacticalButtonStyle {
    static var tactical: TacticalButtonStyle { TacticalButtonStyle() }
}
