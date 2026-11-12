import SwiftUI

struct PayloadLoadingView: View {
    let title: String
    var message: String = "Fetching the latest manifest data."

    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @State private var animating = false

    var body: some View {
        VStack(spacing: TermTheme.s16) {
            ZStack {
                Circle()
                    .fill(TermTheme.accent.opacity(0.08))
                    .frame(width: 72, height: 72)
                    .scaleEffect(reduceMotion ? 1.0 : (animating ? 1.04 : 0.96))
                ProgressView()
                    .controlSize(.regular)
                    .tint(TermTheme.accent)
            }

            VStack(spacing: TermTheme.s8) {
                Text(title.uppercased())
                    .font(.system(.callout, design: .monospaced, weight: .black))
                    .foregroundStyle(TermTheme.accent)
                Text(message)
                    .font(.system(.footnote, design: .monospaced, weight: .medium))
                    .foregroundStyle(TermTheme.secondary)
                    .multilineTextAlignment(.center)
            }
        }
        .frame(maxWidth: .infinity, minHeight: 200)
        .padding(TermTheme.s24)
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(title). \(message)")
        .onAppear {
            guard !reduceMotion else { return }
            withAnimation(TermAnim.fluid.repeatForever(autoreverses: true)) {
                animating = true
            }
        }
    }
}
