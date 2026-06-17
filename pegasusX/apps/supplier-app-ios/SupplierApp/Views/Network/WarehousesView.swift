import SwiftUI

struct WarehousesView: View {
    @State private var warehouses: [SupplierTopologyWarehouse] = []
    @State private var loading = true
    @State private var error: String?

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading warehouses…")
            } else if let error, warehouses.isEmpty {
                SupplierErrorView(message: error) { Task { await load() } }
            } else if warehouses.isEmpty {
                SupplierEmptyView(title: "No warehouses", message: "Configure warehouses in topology.")
            } else {
                List(warehouses) { warehouse in
                    VStack(alignment: .leading, spacing: SupplierTheme.spacingXS) {
                        Text(warehouse.name).font(.headline)
                        Text(String(format: "%.4f, %.4f · %.0f km coverage", warehouse.lat, warehouse.lng, warehouse.coverageRadiusKm))
                            .font(.caption.monospaced())
                            .foregroundStyle(.secondary)
                        SupplierStatusBadge(text: warehouse.isOnShift ? "ON_SHIFT" : "OFF_SHIFT")
                    }
                }
                .listStyle(.insetGrouped)
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("Warehouses")
        .task { await load() }
        .refreshable { await load(silent: true) }
    }

    @MainActor
    private func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        defer { loading = false }
        do {
            let topology = try await SupplierOperationsService.topology()
            warehouses = topology.warehouses
        } catch {
            if !silent { self.error = error.localizedDescription }
        }
    }
}
