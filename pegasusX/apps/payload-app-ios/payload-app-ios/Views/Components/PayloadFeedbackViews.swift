import SwiftUI

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
            Button {
                PegasusHaptics.light()
                retry()
            } label: {
                Text("RETRY")
                    .font(.system(.subheadline, design: .monospaced, weight: .black))
                    .frame(minWidth: 44, minHeight: 44)
                    .padding(.horizontal, 12)
            }
            .buttonStyle(.borderedProminent)
            .accessibilityLabel("Retry loading")
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
