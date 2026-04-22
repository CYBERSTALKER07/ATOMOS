import SwiftUI

enum LabTheme {
    // MARK: - Colors (Adaptive)
    static let background = Color(uiColor: .systemGroupedBackground)
    static let secondaryBackground = Color(uiColor: .secondarySystemGroupedBackground)
    static let tertiaryBackground = Color(uiColor: .tertiarySystemGroupedBackground)
    static let label = Color(uiColor: .label)
    static let secondaryLabel = Color(uiColor: .secondaryLabel)
    static let tertiaryLabel = Color(uiColor: .tertiaryLabel)
    static let separator = Color(uiColor: .separator)
    static let fill = Color(uiColor: .systemFill)

    // Semantic
    static let destructive = Color.red
    static let success = Color.green
    static let warning = Color.orange
    static let live = Color.green

    // MARK: - Spacing
    static let spacingXS: CGFloat = 4
    static let spacingSM: CGFloat = 8
    static let spacingMD: CGFloat = 12
    static let spacingLG: CGFloat = 16
    static let spacingXL: CGFloat = 24
    static let spacingXXL: CGFloat = 32

    // MARK: - Radius
    static let radiusXS: CGFloat = 4
    static let radiusSM: CGFloat = 8
    static let radiusMD: CGFloat = 12
    static let radiusLG: CGFloat = 16
    static let radiusXL: CGFloat = 28
}

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
