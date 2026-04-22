import SwiftUI

// MARK: - Lab Button Style

struct LabButtonStyle: ButtonStyle {
    enum Variant {
        case primary
        case secondary
        case destructive
        case outline
        case ghost
    }

    let variant: Variant
    let fullWidth: Bool

    init(variant: Variant, fullWidth: Bool = false) {
        self.variant = variant
        self.fullWidth = fullWidth
    }

    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(.system(.subheadline, design: .rounded, weight: .bold))
            .padding(.horizontal, AppTheme.spacingXL)
            .padding(.vertical, 14)
            .frame(maxWidth: fullWidth ? .infinity : nil)
            .foregroundStyle(foregroundColor)
            .background {
                if variant == .primary {
                    AppTheme.accentGradient
                } else {
                    backgroundColor.asLinearGradient
                }
            }
            .clipShape(.rect(cornerRadius: AppTheme.radiusButton))
            .overlay {
                if variant == .outline {
                    RoundedRectangle(cornerRadius: AppTheme.radiusButton)
                        .strokeBorder(AppTheme.accent.opacity(0.5), lineWidth: 1.5)
                }
            }
            .shadow(
                color: variant == .primary ? AppTheme.accent.opacity(0.25) : .clear,
                radius: 8, x: 0, y: 4
            )
            .scaleEffect(configuration.isPressed ? 0.95 : 1.0)
            .opacity(configuration.isPressed ? 0.9 : 1.0)
            .animation(.spring(response: 0.15, dampingFraction: 0.86), value: configuration.isPressed)
            .sensoryFeedback(.impact(weight: .light), trigger: configuration.isPressed)
    }

    private var foregroundColor: Color {
        switch variant {
        case .primary: .white
        case .secondary: AppTheme.accent
        case .destructive: .white
        case .outline: AppTheme.accent
        case .ghost: AppTheme.accent
        }
    }

    private var backgroundColor: Color {
        switch variant {
        case .primary: AppTheme.accent
        case .secondary: AppTheme.accentSoft.opacity(0.6)
        case .destructive: AppTheme.destructive
        case .outline: .clear
        case .ghost: .clear
        }
    }
}

// MARK: - Color to LinearGradient helper

private extension Color {
    var asLinearGradient: LinearGradient {
        LinearGradient(colors: [self], startPoint: .leading, endPoint: .trailing)
    }
}

// MARK: - Convenience

struct LabButton: View {
    let title: String
    let variant: LabButtonStyle.Variant
    let icon: String?
    let fullWidth: Bool
    let action: () -> Void

    init(
        _ title: String,
        variant: LabButtonStyle.Variant = .primary,
        icon: String? = nil,
        fullWidth: Bool = false,
        action: @escaping () -> Void
    ) {
        self.title = title
        self.variant = variant
        self.icon = icon
        self.fullWidth = fullWidth
        self.action = action
    }

    var body: some View {
        Button(action: action) {
            HStack(spacing: AppTheme.spacingSM) {
                if let icon {
                    Image(systemName: icon)
                        .font(.system(size: 14, weight: .semibold))
                }
                Text(title)
            }
        }
        .buttonStyle(LabButtonStyle(variant: variant, fullWidth: fullWidth))
    }
}

#Preview {
    VStack(spacing: 16) {
        LabButton("Place Order", icon: "cart", fullWidth: true) {}
        LabButton("Reorder", variant: .secondary, icon: "arrow.clockwise") {}
        LabButton("Cancel", variant: .destructive) {}
        LabButton("View Details", variant: .outline) {}
        LabButton("Learn More", variant: .ghost) {}
    }
    .padding()
}
