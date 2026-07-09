import SwiftUI

struct ExceptionsView: View {
    @State private var rows: [SupplierExceptionRow] = []
    @State private var loading = true
    @State private var error: String?

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading exceptions…")
            } else if let error {
                SupplierErrorView(message: error) { Task { await load() } }
            } else if rows.isEmpty {
                SupplierEmptyView(title: "No exceptions", message: "Operational exceptions will appear here.")
            } else {
                ResponsiveGridContentWrapper {
                    ForEach(rows) { row in
                    VStack(alignment: .leading, spacing: 4) {
                        Text(row.orderId).font(.headline)
                        Text("\(row.kind) · \(row.status)").font(.subheadline)
                        if let note = row.note, !note.isEmpty { Text(note).font(.caption) }
                        if let manifestId = row.manifestId { Text("Manifest \(manifestId)").font(.caption) }
                    }
                }
            }
        }
        .navigationTitle("Exceptions")
        .task { await load() }
    }

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            rows = try await SupplierOperationsService.exceptions()
        } catch {
            self.error = error.localizedDescription
        }
    }
}
