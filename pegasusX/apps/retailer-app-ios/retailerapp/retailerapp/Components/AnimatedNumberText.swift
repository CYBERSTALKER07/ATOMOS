import SwiftUI

struct AnimatedNumberText: View {
    let value: Int
    var font: Font = .headline
    var color: Color = AppTheme.textPrimary

    var body: some View {
        Text("\(value)")
            .font(font.monospacedDigit())
            .foregroundStyle(color)
            .contentTransition(.numericText())
            .animation(.snappy, value: value)
    }
}

struct AnimatedCurrencyText: View {
    let value: Double
    var font: Font = .headline
    var color: Color = AppTheme.accent

    var body: some View {
        Text("\(Int(value).formatted())")
            .font(font.monospacedDigit())
            .foregroundStyle(color)
            .contentTransition(.numericText())
            .animation(.snappy, value: value)
    }
}

#Preview {
    VStack(spacing: 20) {
        AnimatedNumberText(value: 42)
        AnimatedCurrencyText(value: 123.45)
    }
}
