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
                WarehouseList(warehouses: warehouses)
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("portal.nav.warehouses")
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

struct CRMView: View {
    @State private var retailers: [SupplierCRMRetailer] = []
    @State private var loading = true
    @State private var error: String?
    @State private var selected: SupplierCRMRetailerDetail?

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading CRM…")
            } else if let error, retailers.isEmpty {
                SupplierErrorView(message: error) { Task { await load() } }
            } else if retailers.isEmpty {
                Text("No retailers")
                    .foregroundStyle(.secondary)
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else {
                List {
                    ForEach(retailers) { row in
                        Button {
                            Task { await loadDetail(row.retailerId) }
                        } label: {
                            VStack(alignment: .leading, spacing: 4) {
                                Text(row.retailerName.isEmpty ? row.retailerId : row.retailerName)
                                    .font(.headline)
                                Text("\(row.status) · lifetime \(row.lifetime) · \(row.orderCount) orders")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                                if !row.phone.isEmpty {
                                    Text(row.phone)
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                }
                                if !row.email.isEmpty {
                                    Text(row.email)
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                }
                            }
                        }
                    }
                    if let selected {
                        Section("Detail") {
                            Text(selected.retailerName)
                            if !selected.email.isEmpty {
                                Text(selected.email)
                                    .font(.caption)
                            }
                            Text("Lifetime \(selected.lifetime) (minor, not divided)")
                                .font(.caption)
                            ForEach(selected.orders) { order in
                                Text("\(order.orderId.prefix(8)) · \(order.state) · \(order.amount)")
                                    .font(.caption)
                            }
                        }
                    }
                }
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("portal.nav.crm")
        .task { await load() }
        .refreshable { await load(silent: true) }
    }

    @MainActor
    private func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        defer { loading = false }
        do {
            retailers = try await SupplierOperationsService.crmRetailers()
        } catch let err as APIError {
            if case .httpError(503) = err {
                if !silent { error = "CRM unavailable" }
            } else if !silent {
                error = err.localizedDescription
            }
            retailers = []
        } catch {
            if !silent { self.error = error.localizedDescription }
            retailers = []
        }
    }

    @MainActor
    private func loadDetail(_ id: String) async {
        do {
            selected = try await SupplierOperationsService.crmRetailer(id)
        } catch {
            selected = nil
        }
    }
}
