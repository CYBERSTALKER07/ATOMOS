import SwiftUI

public struct PegasusEmptyView: View {
  public let title: String
  public let message: String

  public init(title: String, message: String) {
    self.title = title
    self.message = message
  }

  public var body: some View {
    ContentUnavailableView(
      LocalizedStringKey(title),
      systemImage: "tray",
      description: Text(LocalizedStringKey(message))
    )
    .frame(maxWidth: .infinity, minHeight: 200)
  }
}
