import SwiftUI

typealias LabTheme = PegasusMonochromeTheme

// MARK: - Animation Presets
enum Anim {
    static let snappy = Animation.snappy(duration: 0.3)
    static let smooth = Animation.smooth(duration: 0.35)
    static let spring = Animation.spring(response: 0.4, dampingFraction: 0.85)
    static let quick = Animation.easeOut(duration: 0.15)
}

// MARK: - Lab Card Modifier
struct LabCardModifier: ViewModifier {
    func body(content: Content) -> some View {
        content
            .padding(LabTheme.spacingLG)
            .background(LabTheme.card)
            .clipShape(RoundedRectangle(cornerRadius: LabTheme.radiusMD, style: .continuous))
            .overlay(
                RoundedRectangle(cornerRadius: LabTheme.radiusMD, style: .continuous)
                    .stroke(LabTheme.cardBorder, lineWidth: 1)
            )
            .shadow(color: Color.black.opacity(0.03), radius: 6, x: 0, y: 2)
    }
}

extension View {
    func labCard() -> some View {
        modifier(LabCardModifier())
    }

    func labReadableWidth() -> some View {
        frame(maxWidth: LabTheme.readableMaxWidth)
            .frame(maxWidth: .infinity)
    }
}

struct PressableButtonStyle: ButtonStyle {
    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .scaleEffect(configuration.isPressed ? 0.97 : 1.0)
            .opacity(configuration.isPressed ? 0.85 : 1.0)
            .animation(Anim.quick, value: configuration.isPressed)
    }
}

extension ButtonStyle where Self == PressableButtonStyle {
    static var pressable: PressableButtonStyle { PressableButtonStyle() }
}
