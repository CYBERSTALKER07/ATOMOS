import SwiftUI

public struct PegasusLoadingView: View {
  public let title: String
  public var message: String = "Fetching the latest data."

  @Environment(\.accessibilityReduceMotion) private var reduceMotion
  @State private var animating = false

  public init(title: String, message: String = "Fetching the latest data.") {
    self.title = title
    self.message = message
  }

  public var body: some View {
    VStack(spacing: PegasusMonochromeTheme.spacingLG) {
      ZStack {
        Circle()
          .fill(PegasusMonochromeTheme.tertiaryBackground)
          .frame(width: 72, height: 72)
          .scaleEffect(reduceMotion ? 1.0 : (animating ? 1.04 : 0.96))
        ProgressView()
          .controlSize(.regular)
      }

      VStack(spacing: PegasusMonochromeTheme.spacingSM) {
        Text(title)
          .font(.title3.bold())
        Text(message)
          .font(.body)
          .foregroundStyle(.secondary)
          .multilineTextAlignment(.center)
      }
    }
    .frame(maxWidth: .infinity, minHeight: 200)
    .padding(PegasusMonochromeTheme.spacingXL)
    .accessibilityElement(children: .combine)
    .accessibilityLabel("\(title). \(message)")
    .onAppear {
      guard !reduceMotion else { return }
      withAnimation(PegasusAnim.smooth.repeatForever(autoreverses: true)) {
        animating = true
      }
    }
  }
}

public struct PegasusErrorView: View {
  public let message: String
  public let retry: () -> Void

  public init(message: String, retry: @escaping () -> Void) {
    self.message = message
    self.retry = retry
  }

  public var body: some View {
    ContentUnavailableView {
      Label("Unable to load", systemImage: "exclamationmark.triangle")
    } description: {
      Text(message)
    } actions: {
      Button("Retry") {
        PegasusHaptics.selection()
        retry()
      }
      .buttonStyle(.borderedProminent)
      .frame(minWidth: 44, minHeight: 44)
    }
    .frame(maxWidth: .infinity, minHeight: 200)
  }
}
