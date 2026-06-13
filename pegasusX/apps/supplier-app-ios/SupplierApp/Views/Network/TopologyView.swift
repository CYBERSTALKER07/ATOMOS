import SwiftUI

struct TopologyView: View {
    @State private var loading = true
    @State private var error: String?
    @State private var warehouses: [SupplierTopologyWarehouse] = []
    @State private var factories: [SupplierTopologyFactory] = []

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading topology…")
            } else if let error {
                SupplierErrorView(message: error) { Task { await load() } }
            } else if warehouses.isEmpty && factories.isEmpty {
                SupplierEmptyView(title: "No nodes", message: "No warehouses or factories configured.")
            } else {
                List {
                    Section("Warehouses (\(warehouses.count))") {
                        if warehouses.isEmpty {
                            Text("None").foregroundStyle(.secondary)
                        }
                        ForEach(warehouses) { node in
                            NodeRow(name: node.name, lat: node.lat, lng: node.lng)
                        }
                    }
                    Section("Factories (\(factories.count))") {
                        if factories.isEmpty {
                            Text("None").foregroundStyle(.secondary)
                        }
                        ForEach(factories) { node in
                            NodeRow(name: node.name, lat: node.lat, lng: node.lng)
                        }
                    }
                }
                .listStyle(.insetGrouped)
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("Factories & warehouses")
        .task { await load() }
    }

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let resp = try await SupplierOperationsService.topology()
            warehouses = resp.warehouses
            factories = resp.factories
        } catch {
            self.error = error.localizedDescription
        }
    }
}

private struct NodeRow: View {
    let name: String
    let lat: Double
    let lng: Double

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(name.isEmpty ? "Unnamed node" : name).font(.body)
            Text(String(format: "%.4f, %.4f", lat, lng))
                .font(.caption)
                .foregroundStyle(.secondary)
        }
    }
}
