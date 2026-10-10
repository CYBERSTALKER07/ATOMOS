import SwiftUI

struct QuantityStepper: View {
    @Binding var quantity: Int
    var minimum: Int = 1
    var maximum: Int = 99
    var compact: Bool = false

    var body: some View {
        HStack(spacing: compact ? 4 : AppTheme.spacingSM) {
            stepperButton(icon: "minus", enabled: quantity > minimum) {
                if quantity > minimum {
                    Haptics.light()
                    withAnimation(AnimationConstants.express) {
                        quantity -= 1
                    }
                }
            }

            Text("\(quantity)")
                .font(.system(compact ? .caption : .subheadline, design: .rounded, weight: .bold))
                .monospacedDigit()
                .foregroundStyle(AppTheme.textPrimary)
                .frame(minWidth: compact ? 22 : 28)
                .contentTransition(.numericText())
                .animation(.snappy, value: quantity)

            stepperButton(icon: "plus", enabled: quantity < maximum) {
                if quantity < maximum {
                    Haptics.light()
                    withAnimation(AnimationConstants.express) {
                        quantity += 1
                    }
                }
            }
        }
        .padding(compact ? 2 : 4)
        .background(AppTheme.surfaceElevated.opacity(0.5))
        .clipShape(.capsule)
    }

    private func stepperButton(icon: String, enabled: Bool, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            Image(systemName: icon)
                .font(.system(size: compact ? 10 : 12, weight: .bold))
                .foregroundStyle(enabled ? AppTheme.accent : AppTheme.textTertiary.opacity(0.4))
                .frame(width: compact ? 26 : 32, height: compact ? 26 : 32)
                .background(enabled ? AppTheme.accentSoft.opacity(0.5) : AppTheme.separator.opacity(0.2))
                .clipShape(.circle)
        }
        .accessibilityLabel(icon == "minus" ? "Decrease quantity" : "Increase quantity")
        .disabled(!enabled)
    }
}

#Preview {
    @Previewable @State var qty = 3
    VStack(spacing: 20) {
        QuantityStepper(quantity: $qty)
        QuantityStepper(quantity: $qty, compact: true)
    }
}
