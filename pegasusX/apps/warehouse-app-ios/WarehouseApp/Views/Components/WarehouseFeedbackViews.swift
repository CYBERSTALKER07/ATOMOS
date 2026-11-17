import SwiftUI

struct WarehouseErrorView: View {
    let message: String
    let retry: () -> Void

    var body: some View {
        ContentUnavailableView {
            Label("warehouse_portal.residual.text.unable_to_load", systemImage: "exclamationmark.triangle")
        } description: {
            Text(message)
        } actions: {
            Button {
                PegasusHaptics.light()
                retry()
            } label: {
                Text("common.action.retry")
                    .pegasusMinimumHitTarget()
            }
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
