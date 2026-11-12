import SwiftUI

struct WarehouseJSONFeedView: View {
    let title: String
    let path: String
    var query: [String: String] = [:]
    @State private var vm = WarehouseJSONFeedViewModel()

    var body: some View {
        Group {
            if vm.loading && vm.bodyText.isEmpty {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let errorText = vm.errorText, vm.bodyText.isEmpty {
                ContentUnavailableView {
                    Label("mobile_warehouse.ui.error", systemImage: "exclamationmark.triangle")
                } description: {
                    Text(errorText)
                } actions: {
                    Button("common.action.retry") {
                        PegasusHaptics.light()
                        Task { await vm.load(path: path, query: query) }
                    }
                    .frame(minWidth: 44, minHeight: 44)
                    .contentShape(Rectangle())
                }
            } else {
                ScrollView {
                    Text(vm.bodyText)
                        .font(.system(.caption, design: .monospaced))
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .padding()
                }
                .scrollDismissesKeyboard(.interactively)
            }
        }
        .navigationTitle(title)
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    PegasusHaptics.light()
                    Task { await vm.load(path: path, query: query) }
                } label: {
                    Label("portal.page.orders.action.refresh", systemImage: "arrow.clockwise")
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                }
                .accessibilityLabel("Refresh feed")
            }
        }
        .task(id: path) {
            await vm.load(path: path, query: query)
        }
        .refreshable {
            await vm.load(path: path, query: query)
        }
    }
}
