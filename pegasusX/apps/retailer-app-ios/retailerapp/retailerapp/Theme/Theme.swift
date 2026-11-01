import SwiftUI

// MARK: - Design Tokens

enum AppTheme {
    // MARK: Colors (Warm Tactile Minimalist - Sampled from Reference Image)
    #if canImport(UIKit)
    static let background = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 14/255.0, green: 14/255.0, blue: 16/255.0, alpha: 1.0) : UIColor(red: 237/255.0, green: 237/255.0, blue: 240/255.0, alpha: 1.0) })
    static let cardBackground = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 24/255.0, green: 24/255.0, blue: 27/255.0, alpha: 1.0) : UIColor.white })
    static let secondaryBackground = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 32/255.0, green: 32/255.0, blue: 36/255.0, alpha: 1.0) : UIColor(red: 246/255.0, green: 246/255.0, blue: 248/255.0, alpha: 1.0) })
    static let separator = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 39/255.0, green: 39/255.0, blue: 42/255.0, alpha: 1.0) : UIColor(red: 228/255.0, green: 228/255.0, blue: 231/255.0, alpha: 1.0) })
    static let cardBorder = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor.white.withAlphaComponent(0.08) : UIColor.black.withAlphaComponent(0.04) })
    static let accent = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor.white : UIColor(red: 17/255.0, green: 17/255.0, blue: 19/255.0, alpha: 1.0) })
    static let accentSoft = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 39/255.0, green: 39/255.0, blue: 42/255.0, alpha: 1.0) : UIColor(red: 226/255.0, green: 226/255.0, blue: 230/255.0, alpha: 1.0) })
    static let iconTile = accentSoft
    static let accentDark = accent
    static let destructive = Color(uiColor: .systemRed)
    static let destructiveSoft = Color(uiColor: .systemRed).opacity(0.1)
    // Alias kept for semantic clarity in some views (e.g. AccountProfileView uses `AppTheme.error`).
    static let error = destructive
    static let success = Color(uiColor: .systemGreen)
    static let successSoft = Color(uiColor: .systemGreen).opacity(0.1)
    static let warning = Color(uiColor: .systemOrange)
    static let warningSoft = Color(uiColor: .systemOrange).opacity(0.1)
    static let info = Color(uiColor: .systemBlue)
    static let infoSoft = Color(uiColor: .systemBlue).opacity(0.1)
    static let textPrimary = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 244/255.0, green: 244/255.0, blue: 245/255.0, alpha: 1.0) : UIColor(red: 17/255.0, green: 17/255.0, blue: 19/255.0, alpha: 1.0) })
    static let textSecondary = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 161/255.0, green: 161/255.0, blue: 170/255.0, alpha: 1.0) : UIColor(red: 113/255.0, green: 113/255.0, blue: 122/255.0, alpha: 1.0) })
    static let textTertiary = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 113/255.0, green: 113/255.0, blue: 122/255.0, alpha: 1.0) : UIColor(red: 161/255.0, green: 161/255.0, blue: 170/255.0, alpha: 1.0) })
    static let surfaceElevated = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 31/255.0, green: 31/255.0, blue: 35/255.0, alpha: 1.0) : UIColor.white })
    #else
    static let background = Color.gray.opacity(0.15)
    static let cardBackground = Color.white
    static let secondaryBackground = Color.gray.opacity(0.1)
    static let separator = Color.gray.opacity(0.2)
    static let cardBorder = Color.black.opacity(0.04)
    static let accent = Color.black
    static let accentSoft = Color.gray.opacity(0.15)
    static let iconTile = accentSoft
    static let accentDark = accent
    static let destructive = Color.red
    static let destructiveSoft = Color.red.opacity(0.1)
    static let error = destructive
    static let success = Color.green
    static let successSoft = Color.green.opacity(0.1)
    static let warning = Color.orange
    static let warningSoft = Color.orange.opacity(0.1)
    static let info = Color.blue
    static let infoSoft = Color.blue.opacity(0.1)
    static let textPrimary = Color.black
    static let textSecondary = Color.gray
    static let textTertiary = Color.gray.opacity(0.6)
    static let surfaceElevated = Color.white
    #endif

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

    // MARK: Layout
    static let readableMaxWidth: CGFloat = 960
    /// Minimum touch target size adhering to Apple HIG (44x44 pt).
    static let minTouchTarget: CGFloat = 44

    /// Maps order/ops status strings to semantic badge tints (aligned with supplier/warehouse native).
    static func statusTint(for status: String) -> Color {
        switch status.uppercased() {
        case "COMPLETED", "DONE", "ACTIVE", "APPROVED", "ARRIVED", "RECEIVED", "PAID", "LIVE", "SUCCESS":
            return success
        case "PENDING", "AWAITING_PAYMENT", "LOADING", "IN_TRANSIT", "DISPATCHED", "ARRIVING",
             "OPEN", "SUBMITTED", "WAITING", "AWAITING DISPATCH", "FISCALIZING", "PENDING_CASH_COLLECTION":
            return warning
        case "CANCELLED", "REJECTED", "FAILED", "EXCEPTION", "CRITICAL", "FISCAL_FAILED":
            return destructive
        default:
            return textSecondary
        }
    }
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
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @State private var isPressed = false

    func body(content: Content) -> some View {
        content
            .scaleEffect((isPressed && !reduceMotion) ? 0.97 : 1.0)
            .animation(reduceMotion ? nil : .spring(response: 0.2, dampingFraction: 0.8), value: isPressed)
            .simultaneousGesture(
                DragGesture(minimumDistance: 0)
                    .onChanged { _ in
                        if !isPressed {
                            isPressed = true
                        }
                    }
                    .onEnded { _ in
                        isPressed = false
                    }
            )
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

    func retailerCard() -> some View {
        padding(AppTheme.spacingMD)
            .background {
                RoundedRectangle(cornerRadius: AppTheme.radiusCard, style: .continuous)
                    .fill(AppTheme.cardBackground)
                    .overlay {
                        RoundedRectangle(cornerRadius: AppTheme.radiusCard, style: .continuous)
                            .stroke(AppTheme.cardBorder, lineWidth: 1)
                    }
            }
            .shadow(color: AppTheme.shadowColor, radius: 4, x: 0, y: 2)
    }

    func retailerReadableWidth() -> some View {
        frame(maxWidth: AppTheme.readableMaxWidth)
            .frame(maxWidth: .infinity)
    }

    /// Enforces the Apple HIG minimum interactive hit target dimension (44x44 pt).
    func pegasusMinimumHitTarget(minWidth: CGFloat = AppTheme.minTouchTarget, minHeight: CGFloat = AppTheme.minTouchTarget) -> some View {
        frame(minWidth: minWidth, minHeight: minHeight)
            .contentShape(Rectangle())
    }
}
