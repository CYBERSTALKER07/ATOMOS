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
            .background(LabTheme.secondaryBackground, in: RoundedRectangle(cornerRadius: LabTheme.radiusMD))
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

// MARK: - Staggered Appear
struct StaggeredAppearModifier: ViewModifier {
    let index: Int
    @State private var appeared = false

    func body(content: Content) -> some View {
        content
            .opacity(appeared ? 1 : 0)
            .offset(y: appeared ? 0 : 12)
            .onAppear {
                withAnimation(Anim.smooth.delay(Double(index) * 0.05)) {
                    appeared = true
                }
            }
    }
}

extension View {
    func staggeredAppear(index: Int) -> some View {
        modifier(StaggeredAppearModifier(index: index))
    }
}

// MARK: - Pressable Button Style
struct PressableButtonStyle: ButtonStyle {
    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .scaleEffect(configuration.isPressed ? 0.97 : 1.0)
            .opacity(configuration.isPressed ? 0.85 : 1.0)
            .animation(Anim.quick, value: configuration.isPressed)
    }
}
