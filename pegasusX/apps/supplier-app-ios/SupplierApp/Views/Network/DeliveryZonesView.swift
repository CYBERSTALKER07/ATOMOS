import SwiftUI

struct DeliveryZonesView: View {
    @State private var loading = true
    @State private var error: String?
    @State private var warehouses: [SupplierTopologyWarehouse] = []

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading delivery zones…")
            } else if let error {
                SupplierErrorView(message: error) { Task { await load() } }
            } else if warehouses.isEmpty {
                SupplierEmptyView(title: "No coverage", message: "No warehouse coverage configured.")
            } else {
                ResponsiveGridContentWrapper {
                    Section("Warehouse coverage") {
                        ForEach(warehouses) { node in
                            VStack(alignment: .leading, spacing: 4) {
                                Text(node.name.isEmpty ? "Unnamed warehouse" : node.name).font(.body)
                                Text(node.address.isEmpty ? "Coordinates on file" : node.address)
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                        }
                    }
                    Section {
                        Text("H3 perimeter and warehouse coverage are configured via topology.")
                            .font(.footnote)
                            .foregroundStyle(.secondary)
                    }
                }
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("Delivery zones")
        .task { await load() }
    }

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let resp = try await SupplierOperationsService.topology()
            warehouses = resp.warehouses
        } catch {
            self.error = error.localizedDescription
        }
    }
}
