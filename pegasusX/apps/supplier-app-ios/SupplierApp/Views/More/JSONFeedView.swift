import SwiftUI

struct JSONFeedView: View {
    let title: String
    let path: String
    var query: [String: String] = [:]
    @State private var vm = JSONFeedViewModel()

    var body: some View {
        Group {
            if vm.loading && vm.bodyText.isEmpty {
                ProgressView()
            } else if let errorText = vm.errorText {
                Text(errorText).padding()
            } else {
                ScrollView {
                    Text(vm.bodyText)
                        .font(.caption.monospaced())
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .padding()
                }
            }
        }
        .navigationTitle(LocalizedStringKey(title))
        .task(id: path) { await vm.load(path: path, query: query) }
        .refreshable {
            PegasusHaptics.light()
            await vm.load(path: path, query: query, silent: true)
        }
    }
}
