import SwiftUI

struct ReturnsView: View {
    @State private var returns: [ReturnItem] = []
    @State private var loading = true
    @State private var error: String?

    var body: some View {
        NavigationStack {
            Group {
                if loading {
                    ProgressView()
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else if let error {
                    ContentUnavailableView {
                        Label("Error", systemImage: "exclamationmark.triangle")
                    } description: {
                        Text(error)
                    } actions: {
                        Button("Retry") { load() }
                    }
                } else if returns.isEmpty {
                    ContentUnavailableView("No Returns", systemImage: "arrow.uturn.backward.circle", description: Text("No return requests"))
                } else {
                    List(returns) { item in
                        HStack {
                            VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                Text(item.productName)
                                    .font(.headline)
                                Text("Qty: \(item.quantity) · \(item.reason)")
                                    .font(.subheadline)
                                    .foregroundStyle(.secondary)
                            }
                            Spacer()
                            Text(item.status)
                                .font(.caption.bold())
                                .padding(.horizontal, LabTheme.spacingSM)
                                .padding(.vertical, LabTheme.spacingXS)
                                .background(.quaternary, in: Capsule())
                        }
                    }
                    .listStyle(.insetGrouped)
                }
            }
            .background(LabTheme.background)
            .navigationTitle("Returns")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Refresh", systemImage: "arrow.clockwise") { load() }
                }
            }
            .task { load() }
            .refreshable { load() }
        }
    }

    private func load() {
        loading = true
        error = nil
        Task {
            do {
                let resp = try await WarehouseService.returns()
                returns = resp.returns
            } catch {
                self.error = error.localizedDescription
            }
            loading = false
        }
    }
}
