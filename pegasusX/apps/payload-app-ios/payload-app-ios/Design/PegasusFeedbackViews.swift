import SwiftUI

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
      Button {
        PegasusHaptics.light()
        retry()
      } label: {
        Text("Retry")
          .frame(minWidth: 44, minHeight: 44)
      }
      .buttonStyle(.borderedProminent)
    }
    .frame(maxWidth: .infinity, minHeight: 200)
  }
}

public struct PegasusEmptyView: View {
  public let title: String
  public let message: String

  public init(title: String, message: String) {
    self.title = title
    self.message = message
  }

  public var body: some View {
    ContentUnavailableView(title, systemImage: "tray", description: Text(message))
      .frame(maxWidth: .infinity, minHeight: 200)
  }
}
