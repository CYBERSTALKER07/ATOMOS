import SwiftUI

struct ManifestExceptionsView: View {
    @State private var rows: [SupplierManifestExceptionRow] = []
    @State private var loading = true
    @State private var error: String?
    @State private var escalatedOnly = false

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading gate exceptions…")
            } else if let error {
                SupplierErrorView(message: error) { Task { await load() } }
            } else if rows.isEmpty {
                SupplierEmptyView(title: "No exceptions", message: "No manifest gate exceptions in the current window.")
            } else {
                ResponsiveGridContentWrapper {
                    ForEach(rows) { row in
                    NavigationLink {
                        ManifestDetailView(manifestId: row.manifestId)
                    } label: {
                        VStack(alignment: .leading, spacing: 4) {
                            Text(row.reason).font(.headline)
                            Text("Manifest \(row.manifestId.prefix(8))…").font(.caption)
                            Text("Order \(row.orderId.prefix(8))… · attempts \(row.attemptCount)").font(.caption)
                            if row.escalated {
                                Text("Escalated").font(.caption).foregroundStyle(.red)
                            }
                        }
                    }
                }
            }
        }
        .navigationTitle("Gate exceptions")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Toggle("Escalated", isOn: $escalatedOnly)
                    .labelsHidden()
            }
        }
        .onChange(of: escalatedOnly) { _, _ in Task { await load() } }
        .task { await load() }
    }

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            rows = try await SupplierOperationsService.manifestExceptions(escalatedOnly: escalatedOnly)
        } catch {
            self.error = error.localizedDescription
        }
    }
}
