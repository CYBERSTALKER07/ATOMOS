import SwiftUI

struct ManifestsView: View {
    @State private var rows: [SupplierManifestRow] = []
    @State private var loading = true
    @State private var error: String?

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading manifests…")
            } else if let error {
                SupplierErrorView(message: error) { Task { await load() } }
            } else if rows.isEmpty {
                SupplierEmptyView(title: "No manifests", message: "Loading manifests will appear here.")
            } else {
                List(rows) { row in
                    NavigationLink {
                        ManifestDetailView(manifestId: row.manifestId)
                    } label: {
                        VStack(alignment: .leading, spacing: 4) {
                            Text(row.manifestId).font(.headline)
                            Text("\(row.status) · \(row.state)").font(.subheadline)
                            Text("\(row.ordersCount) orders · \(row.driverName.isEmpty ? (row.driverId ?? "—") : row.driverName)")
                                .font(.caption)
                            if let plate = row.vehiclePlate { Text("Vehicle \(plate)").font(.caption) }
                        }
                    }
                }
                .listStyle(.insetGrouped)
            }
        }
        .navigationTitle("Manifests")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                NavigationLink { ManifestExceptionsView() } label: {
                    Text("Gate")
                }
            }
        }
        .task { await load() }
    }

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            rows = try await SupplierOperationsService.manifests()
        } catch {
            self.error = error.localizedDescription
        }
    }
}
