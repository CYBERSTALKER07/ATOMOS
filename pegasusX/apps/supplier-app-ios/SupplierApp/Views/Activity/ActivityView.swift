import SwiftUI

struct ActivityView: View {
    @State private var rows: [SupplierActivityEvent] = []
    @State private var loading = true
    @State private var error: String?

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading activity…")
            } else if let error {
                SupplierErrorView(message: error) { Task { await load() } }
            } else if rows.isEmpty {
                SupplierEmptyView(title: "No recent activity", message: "Operational events will stream here.")
            } else {
                List(rows) { row in
                    VStack(alignment: .leading, spacing: 4) {
                        Text(row.type).font(.headline)
                        Text(row.description).font(.subheadline)
                        Text(row.timestamp).font(.caption)
                    }
                }
                .listStyle(.insetGrouped)
            }
        }
        .navigationTitle("Activity")
        .task { await load() }
    }

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            rows = try await SupplierOperationsService.activity()
        } catch {
            self.error = error.localizedDescription
        }
    }
}
