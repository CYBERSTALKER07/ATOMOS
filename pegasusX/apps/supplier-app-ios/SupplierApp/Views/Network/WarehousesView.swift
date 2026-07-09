import SwiftUI

struct WarehousesView: View {
    @State private var warehouses: [SupplierTopologyWarehouse] = []
    @State private var loading = true
    @State private var error: String?
    @State private var showAdd = false

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading warehouses…")
            } else if let error, warehouses.isEmpty {
                SupplierErrorView(message: error) { Task { await load() } }
            } else if warehouses.isEmpty {
                TopologyCenteredEmptyState(
                    title: "No warehouses",
                    message: "Add your first distribution node to start fulfilling orders.",
                    actionLabel: "Add first warehouse"
                ) {
                    showAdd = true
                }
            } else {
                ResponsiveGridContentWrapper {
                    ForEach(warehouses) { warehouse in
                    VStack(alignment: .leading, spacing: SupplierTheme.spacingXS) {
                        Text(warehouse.name).font(.headline)
                        Text(warehouse.address.isEmpty ? "Coordinates on file" : warehouse.address)
                            .font(.caption.monospaced())
                            .foregroundStyle(.secondary)
                        SupplierStatusBadge(text: warehouse.isOnShift ? "ON_SHIFT" : "OFF_SHIFT")
                    }
                }
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("Warehouses")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    showAdd = true
                } label: {
                    Image(systemName: "plus")
                }
            }
        }
        .sheet(isPresented: $showAdd) {
            AddWarehouseSheet {
                Task { await load(silent: true) }
            }
        }
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
