import SwiftUI

struct TopologyCenteredEmptyState: View {
  let title: String
  let message: String
  let actionLabel: String
  let action: () -> Void

  var body: some View {
    VStack(spacing: SupplierTheme.spacingLG) {
      Spacer()
      SupplierEmptyView(title: title, message: message)
      Button(LocalizedStringKey(actionLabel), action: action)
        .buttonStyle(.borderedProminent)
        .tint(.primary)
        .frame(minHeight: 44)
      Spacer()
    }
    .frame(maxWidth: .infinity, maxHeight: .infinity)
    .padding()
  }
}
