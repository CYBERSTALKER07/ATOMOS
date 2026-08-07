import SwiftUI

struct WarehouseLoadingView: View {
    let title: String
    var message: String = "Fetching the latest warehouse data."

    @State private var animating = false

    var body: some View {
        VStack(spacing: LabTheme.spacingLG) {
            ZStack {
                Circle()
                    .fill(LabTheme.tertiaryBackground)
                    .frame(width: 72, height: 72)
                    .scaleEffect(animating ? 1.04 : 0.96)
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
        .onAppear {
            withAnimation(Anim.smooth.repeatForever(autoreverses: true)) {
                animating = true
            }
        }
    }
}

struct WarehouseErrorView: View {
    let message: String
    let retry: () -> Void

    var body: some View {
        ContentUnavailableView {
            Label("factory_portal.residual.text.unable_to_load", systemImage: "exclamationmark.triangle")
        } description: {
            Text(message)
        } actions: {
            Button("common.action.retry", action: retry)
                .buttonStyle(.borderedProminent)
        }
        .frame(maxWidth: .infinity, minHeight: 200)
    }
}

struct WarehouseEmptyView: View {
    let title: String
    let message: String

    var body: some View {
        ContentUnavailableView(title, systemImage: "tray", description: Text(message))
            .frame(maxWidth: .infinity, minHeight: 200)
    }
}
