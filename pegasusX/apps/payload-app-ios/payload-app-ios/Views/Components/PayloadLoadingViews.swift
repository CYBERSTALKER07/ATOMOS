import SwiftUI

struct PayloadLoadingView: View {
    let title: String
    var message: String = "Fetching the latest manifest data."

    @State private var animating = false

    var body: some View {
        VStack(spacing: TermTheme.s16) {
            ZStack {
                Circle()
                    .fill(TermTheme.accent.opacity(0.08))
                    .frame(width: 72, height: 72)
                    .scaleEffect(animating ? 1.04 : 0.96)
                ProgressView()
                    .controlSize(.regular)
                    .tint(TermTheme.accent)
            }

            VStack(spacing: TermTheme.s8) {
                Text(title.uppercased())
                    .font(.system(size: 16, weight: .black, design: .monospaced))
                    .foregroundStyle(TermTheme.accent)
                Text(message)
                    .font(.system(size: 13, weight: .medium, design: .monospaced))
                    .foregroundStyle(TermTheme.secondary)
                    .multilineTextAlignment(.center)
            }
        }
        .frame(maxWidth: .infinity, minHeight: 200)
        .padding(TermTheme.s24)
        .onAppear {
            withAnimation(TermAnim.fluid.repeatForever(autoreverses: true)) {
                animating = true
            }
        }
    }
}

struct PayloadErrorView: View {
    let message: String
    let retry: () -> Void

    var body: some View {
        VStack(spacing: TermTheme.s16) {
            PayloadStateView(
                variant: .warning,
                title: "LOAD_FAILED",
                message: message,
                compact: true,
                tone: .warning
            )
            Button("RETRY", action: retry)
                .font(.system(size: 14, weight: .black, design: .monospaced))
                .buttonStyle(.borderedProminent)
        }
        .frame(maxWidth: .infinity, minHeight: 200)
        .padding(TermTheme.s24)
    }
}

struct PayloadEmptyView: View {
    let title: String
    let message: String
    var variant: PayloadStateVariant = .manifest

    var body: some View {
        PayloadStateView(
            variant: variant,
            title: title,
            message: message,
            compact: false
        )
        .frame(maxWidth: .infinity, minHeight: 200)
        .padding(TermTheme.s24)
    }
}
