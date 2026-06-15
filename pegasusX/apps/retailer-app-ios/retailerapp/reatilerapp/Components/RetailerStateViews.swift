import SwiftUI

struct RetailerLoadingView: View {
    let title: String
    var message: String = "Fetching the latest retailer data."

    @State private var animating = false

    var body: some View {
        VStack(spacing: AppTheme.spacingLG) {
            ZStack {
                Circle()
                    .fill(AppTheme.surfaceElevated)
                    .frame(width: 72, height: 72)
                    .scaleEffect(animating ? 1.04 : 0.96)
                ProgressView()
                    .controlSize(.regular)
            }

            VStack(spacing: AppTheme.spacingSM) {
                Text(title)
                    .font(.system(.title3, design: .rounded, weight: .bold))
                Text(message)
                    .font(.system(.body, design: .rounded))
                    .foregroundStyle(AppTheme.textSecondary)
                    .multilineTextAlignment(.center)
            }
        }
        .frame(maxWidth: .infinity, minHeight: 200)
        .padding(AppTheme.spacingXL)
        .onAppear {
            withAnimation(AnimationConstants.fluid.repeatForever(autoreverses: true)) {
                animating = true
            }
        }
    }
}

struct RetailerErrorView: View {
    let message: String
    let retry: () -> Void

    var body: some View {
        ContentUnavailableView {
            Label("Unable to load", systemImage: "exclamationmark.triangle")
        } description: {
            Text(message)
        } actions: {
            Button("Retry", action: retry)
                .buttonStyle(.borderedProminent)
        }
        .frame(maxWidth: .infinity, minHeight: 200)
    }
}

struct RetailerEmptyView: View {
    let title: String
    let message: String
    var systemImage: String = "tray"

    var body: some View {
        ContentUnavailableView(title, systemImage: systemImage, description: Text(message))
            .frame(maxWidth: .infinity, minHeight: 200)
    }
}
