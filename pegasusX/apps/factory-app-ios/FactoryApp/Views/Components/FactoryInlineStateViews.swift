import SwiftUI

/// Inline loading pane for scroll contexts (lighter than full-screen `FactoryLoadingState`).
struct FactoryLoadingView: View {
    let title: String
    var message: String = "Fetching the latest factory data."

    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @State private var animating = false

    var body: some View {
        VStack(spacing: LabTheme.spacingLG) {
            ZStack {
                Circle()
                    .fill(LabTheme.tertiaryBackground)
                    .frame(width: 72, height: 72)
                    .scaleEffect(reduceMotion ? 1.0 : (animating ? 1.04 : 0.96))
                ProgressView()
                    .controlSize(.regular)
            }

            VStack(spacing: LabTheme.spacingSM) {
                Text(title)
                    .font(.title3.bold())
                Text(message)
                    .font(.body)
                    .foregroundStyle(.secondary)
                    .multilineTextAlignment(.center)
            }
        }
        .frame(maxWidth: .infinity, minHeight: 200)
        .padding(LabTheme.spacingXL)
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(title). \(message)")
        .onAppear {
            guard !reduceMotion else { return }
            withAnimation(Anim.smooth.repeatForever(autoreverses: true)) {
                animating = true
            }
        }
    }
}

struct FactoryErrorView: View {
    let message: String
    let retry: () -> Void

    var body: some View {
        ContentUnavailableView {
            Label("factory_portal.residual.text.unable_to_load", systemImage: "exclamationmark.triangle")
        } description: {
            Text(message)
        } actions: {
            Button("common.action.retry") {
                PegasusHaptics.selection()
                retry()
            }
            .buttonStyle(.borderedProminent)
            .frame(minWidth: 44, minHeight: 44)
        }
        .frame(maxWidth: .infinity, minHeight: 200)
    }
}
