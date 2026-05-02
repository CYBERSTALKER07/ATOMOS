import SwiftUI

// MARK: - Design Tokens

enum AppTheme {
    // MARK: Colors (Native Tactical)
    static let background = Color(UIColor.systemGroupedBackground)
    static let cardBackground = Color(UIColor.systemBackground)
    static let secondaryBackground = Color(UIColor.secondarySystemGroupedBackground)
    static let separator = Color(UIColor.separator)
    static let accent = Color(UIColor.label)                              // Pure black (light) / white (dark)
    static let accentSoft = Color(UIColor.systemGray6)                    // Very light gray
    static let accentDark = Color(UIColor.label)                          // Same as accent
    static let destructive = Color(UIColor.systemRed)
    static let destructiveSoft = Color(UIColor.systemRed).opacity(0.1)
    static let success = Color(UIColor.systemGreen)
    static let successSoft = Color(UIColor.systemGreen).opacity(0.1)
    static let warning = Color(UIColor.systemOrange)
    static let warningSoft = Color(UIColor.systemOrange).opacity(0.1)
    static let info = Color(UIColor.systemBlue)
    static let infoSoft = Color(UIColor.systemBlue).opacity(0.1)
    static let textPrimary = Color(UIColor.label)
    static let textSecondary = Color(UIColor.secondaryLabel)
    static let textTertiary = Color(UIColor.tertiaryLabel)
    static let surfaceElevated = Color(UIColor.systemGray6)

    // MARK: Gradients (B&W)
    static let accentGradient = LinearGradient(
        colors: [Color(.label), Color(.label).opacity(0.8)],
        startPoint: .topLeading,
        endPoint: .bottomTrailing
    )
    static let heroGradient = LinearGradient(
        colors: [Color(.label), Color(.systemGray), Color(.systemGray2)],
        startPoint: .topLeading,
        endPoint: .bottomTrailing
    )
    static let warmGradient = LinearGradient(
        colors: [Color(.systemGray), Color(.systemGray3)],
        startPoint: .topLeading,
        endPoint: .bottomTrailing
    )
    static let coolGradient = LinearGradient(
        colors: [Color(.systemGray2), Color(.systemGray4)],
        startPoint: .topLeading,
        endPoint: .bottomTrailing
    )
    static let meshBackground = LinearGradient(
        colors: [Color(.systemGray6), Color(.systemBackground), Color(.systemGroupedBackground)],
        startPoint: .top,
        endPoint: .bottom
    )

    // MARK: Corner Radii (Native Fluid)
    static let radiusXS: Double = 4
    static let radiusSM: Double = 8
    static let radiusMD: Double = 12
    static let radiusLG: Double = 20
    static let radiusXL: Double = 24
    static let radiusXXL: Double = 32
    static let radiusCard: Double = 24
    static let radiusButton: Double = 16
    static let radiusPill: Double = 100

    // MARK: Spacing
    static let spacingXS: Double = 4
    static let spacingSM: Double = 8
    static let spacingMD: Double = 12
    static let spacingLG: Double = 16
    static let spacingXL: Double = 24
    static let spacingXXL: Double = 32
    static let spacingHuge: Double = 48

    // MARK: Shadows (Tactical - Deprioritized)
    static let shadowRadius: Double = 4
    static let shadowColor = Color.black.opacity(0.04)
    static let shadowOffsetY: Double = 2
    static let shadowRadiusLG: Double = 8
    static let shadowColorLG = Color.black.opacity(0.08)

    // MARK: Separator
    static let separatorHeight: Double = 0.33

    // MARK: Icon sizes
    static let iconSM: Double = 16
    static let iconMD: Double = 20
    static let iconLG: Double = 24
    static let iconXL: Double = 32
    static let iconHuge: Double = 48
}

// MARK: - Shimmer Effect

struct ShimmerModifier: ViewModifier {
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @State private var phase: Double = 0

    func body(content: Content) -> some View {
        content
            .overlay {
                if !reduceMotion {
                    LinearGradient(
                        colors: [.clear, .white.opacity(0.4), .clear],
                        startPoint: .init(x: phase - 0.5, y: 0.5),
                        endPoint: .init(x: phase + 0.5, y: 0.5)
                    )
                    .blendMode(.overlay)
                }
            }
            .onAppear {
                guard !reduceMotion else { return }
                withAnimation(.linear(duration: 1.5).repeatForever(autoreverses: false)) {
                    phase = 1.5
                }
            }
    }
}

// MARK: - Skeleton Loading Modifier

struct SkeletonModifier: ViewModifier {
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @State private var opacity: Double = 0.3

    func body(content: Content) -> some View {
        content
            .redacted(reason: .placeholder)
            .opacity(reduceMotion ? 0.5 : opacity)
            .onAppear {
                guard !reduceMotion else { return }
                withAnimation(.easeInOut(duration: 0.8).repeatForever(autoreverses: true)) {
                    opacity = 0.7
                }
            }
    }
}

// MARK: - Glass Card Style

struct GlassCardModifier: ViewModifier {
    var cornerRadius: Double = AppTheme.radiusCard

    func body(content: Content) -> some View {
        content
            .background {
                RoundedRectangle(cornerRadius: cornerRadius)
                    .fill(.ultraThinMaterial)
                    .shadow(color: AppTheme.shadowColor, radius: AppTheme.shadowRadius, x: 0, y: AppTheme.shadowOffsetY)
            }
    }
}

// MARK: - Elevated Card Style

struct ElevatedCardModifier: ViewModifier {
    var cornerRadius: Double = AppTheme.radiusCard

    func body(content: Content) -> some View {
        content
            .background {
                RoundedRectangle(cornerRadius: cornerRadius)
                    .fill(AppTheme.cardBackground)
                    .shadow(color: AppTheme.shadowColor, radius: AppTheme.shadowRadius, x: 0, y: AppTheme.shadowOffsetY)
            }
    }
}

// MARK: - Press Scale Effect

struct PressableModifier: ViewModifier {
    @State private var isPressed = false

    func body(content: Content) -> some View {
        content
            .scaleEffect(isPressed ? 0.97 : 1.0)
            .animation(.spring(response: 0.2, dampingFraction: 0.8), value: isPressed)
            .sensoryFeedback(.impact(weight: .light), trigger: isPressed)
            .onLongPressGesture(minimumDuration: .infinity, pressing: { pressing in
                isPressed = pressing
            }, perform: {})
    }
}

// MARK: - Floating Action Style

struct FloatingModifier: ViewModifier {
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @State private var yOffset: Double = 0

    func body(content: Content) -> some View {
        content
            .offset(y: yOffset)
            .onAppear {
                guard !reduceMotion else { return }
                withAnimation(.easeInOut(duration: 2.0).repeatForever(autoreverses: true)) {
                    yOffset = -4
                }
            }
    }
}

// MARK: - Slide In From Bottom

struct SlideInModifier: ViewModifier {
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @State private var appeared = false
    let delay: Double

    func body(content: Content) -> some View {
        content
            .offset(y: appeared ? 0 : (reduceMotion ? 0 : 30))
            .opacity(appeared ? 1 : 0)
            .onAppear {
                if reduceMotion {
                    appeared = true
                } else {
                    withAnimation(.spring(response: 0.5, dampingFraction: 0.8).delay(delay)) {
                        appeared = true
                    }
                }
            }
    }
}

// MARK: - View Extensions

extension View {
    func cardStyle() -> some View {
        modifier(ElevatedCardModifier())
    }

    func glassCard(cornerRadius: Double = AppTheme.radiusCard) -> some View {
        modifier(GlassCardModifier(cornerRadius: cornerRadius))
    }

    func elevatedCard(cornerRadius: Double = AppTheme.radiusCard) -> some View {
        modifier(ElevatedCardModifier(cornerRadius: cornerRadius))
    }

    func shimmer() -> some View {
        modifier(ShimmerModifier())
    }

    func skeleton() -> some View {
        modifier(SkeletonModifier())
    }

    func pressable() -> some View {
        modifier(PressableModifier())
    }

    func floating() -> some View {
        modifier(FloatingModifier())
    }

    func slideIn(delay: Double = 0) -> some View {
        modifier(SlideInModifier(delay: delay))
    }

    func staggeredSlideIn(index: Int, baseDelay: Double = 0.05) -> some View {
        modifier(SlideInModifier(delay: Double(index) * baseDelay))
    }
}
