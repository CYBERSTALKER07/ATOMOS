import SwiftUI

struct WarehouseLoadingView: View {
    let title: String
    var message: String = "Fetching the latest warehouse data."

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
