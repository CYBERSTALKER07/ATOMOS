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
                                Text(L10n.format("mobile_supplier.ui.manifest_prefix", "\(row.manifestId.prefix(8))")).font(.caption)
                                Text(L10n.format("mobile_supplier.ui.order_prefix_attempts_attemptcount", "\(row.orderId.prefix(8))", "\(row.attemptCount)")).font(.caption)
                                if row.escalated {
                                    Text("factory_portal.residual.text.escalated").font(.caption).foregroundStyle(.red)
                                }
                            }
                        }
                    }
                }
            }
        }
        .navigationTitle("factory_portal.manifest_exceptions.text.gate_exceptions")
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
