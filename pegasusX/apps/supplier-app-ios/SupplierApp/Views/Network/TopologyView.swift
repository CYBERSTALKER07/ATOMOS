import SwiftUI

private struct WarehouseDraft: Identifiable {
    let id: String
    var warehouseId: String?
    var name: String
    var location: AddressLocationValue
    var coverageRadiusKm: String
}

private struct FactoryDraft: Identifiable {
    let id: String
    var factoryId: String?
    var name: String
    var location: AddressLocationValue
    var isActive: Bool = true
}

struct TopologyView: View {
    @State private var loading = true
    @State private var saving = false
    @State private var editing = false
    @State private var error: String?
    @State private var warehouses: [SupplierTopologyWarehouse] = []
    @State private var factories: [SupplierTopologyFactory] = []
    @State private var warehouseDrafts: [WarehouseDraft] = []
    @State private var factoryDrafts: [FactoryDraft] = []

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading topology…")
            } else if let error, warehouses.isEmpty && factories.isEmpty && !editing {
                SupplierErrorView(message: error) { Task { await load() } }
            } else if editing {
                editForm
            } else if warehouses.isEmpty && factories.isEmpty {
                VStack(spacing: SupplierTheme.spacingLG) {
                    SupplierEmptyView(title: "No nodes", message: "No warehouses or factories configured.")
                    Button("mobile_supplier.ui.configure_topology") { editing = true }
                }
            } else {
                readOnlyList
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("supplier_portal.topology.text.factories_and_warehouses")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                if editing {
                    Button(saving ? "Saving…" : "Save") {
                        Task { await save() }
                    }
                    .disabled(saving)
                } else {
                    Button("supplier_portal.demand.signals.text.edit") { beginEditing() }
                }
            }
            if editing {
                ToolbarItem(placement: .topBarLeading) {
                    Button("common.action.cancel") {
                        applyDraftsFromLoaded()
                        editing = false
                        error = nil
                    }
                }
            }
        }
        .task { await load() }
    }

    private var readOnlyList: some View {
        ResponsiveGridContentWrapper {
            Section(L10n.format("mobile_supplier.ui.warehouses_count", "\(warehouses.count)")) {
                ForEach(warehouses) { node in
                    NodeRow(
                        name: node.name,
                        address: node.address,
                        meta: String(format: "%.0f km coverage", node.coverageRadiusKm)
                    )
                }
            }
            Section(L10n.format("mobile_supplier.ui.factories_count", "\(factories.count)")) {
                ForEach(factories) { node in
                    NodeRow(
                        name: node.name,
                        address: node.address,
                        meta: node.isActive ? "Active" : "Inactive"
                    )
                }
            }
        }
    }

    private var editForm: some View {
        Form {
            if let error {
                Section { Text(error).foregroundStyle(.red) }
            }
            Section("Warehouses") {
                ForEach($warehouseDrafts) { $draft in
                    TextField("retailer_desktop.pos.text.name", text: $draft.name)
                    AddressLocationField(value: $draft.location, label: "Warehouse address")
                    TextField("supplier_portal.warehouses.components.warehouse_form.text.coverage_km", text: $draft.coverageRadiusKm)
                        .keyboardType(.decimalPad)
                }
                Button("supplier_portal.warehouses.components.warehouse_form.text.add_warehouse") {
                    warehouseDrafts.append(
                        WarehouseDraft(
                            id: "new-wh-\(warehouseDrafts.count)",
                            warehouseId: nil,
                            name: "Warehouse \(warehouseDrafts.count + 1)",
                            location: AddressLocationValue(lat: TopologyMutation.defaultLat, lng: TopologyMutation.defaultLng),
                            coverageRadiusKm: "50"
                        )
                    )
                }
            }
            Section("Factories") {
                ForEach($factoryDrafts) { $draft in
                    TextField("retailer_desktop.pos.text.name", text: $draft.name)
                    AddressLocationField(value: $draft.location, label: "Factory address")
                }
                Button("supplier_portal.factories.components.factory_form.text.add_factory") {
                    factoryDrafts.append(
                        FactoryDraft(
                            id: "new-fc-\(factoryDrafts.count)",
                            factoryId: nil,
                            name: "Factory \(factoryDrafts.count + 1)",
                            location: AddressLocationValue(lat: 41.3111, lng: 69.2797)
                        )
                    )
                }
            }
        }
    }

    private func beginEditing() {
        applyDraftsFromLoaded()
        editing = true
    }

    private func applyDraftsFromLoaded() {
        warehouseDrafts = warehouses.enumerated().map { index, node in
            WarehouseDraft(
                id: node.warehouseId.isEmpty ? "wh-\(index)" : node.warehouseId,
                warehouseId: node.warehouseId.isEmpty ? nil : node.warehouseId,
                name: node.name,
                location: AddressLocationValue(
                    address: node.address,
                    lat: node.lat,
                    lng: node.lng,
                    placeId: node.placeId
                ),
                coverageRadiusKm: String(node.coverageRadiusKm)
            )
        }
        factoryDrafts = factories.enumerated().map { index, node in
            FactoryDraft(
                id: node.factoryId.isEmpty ? "fc-\(index)" : node.factoryId,
                factoryId: node.factoryId.isEmpty ? nil : node.factoryId,
                name: node.name,
                location: AddressLocationValue(
                    address: node.address,
                    lat: node.lat,
                    lng: node.lng,
                    placeId: node.placeId
                )
            )
        }
    }

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let resp = try await SupplierOperationsService.topology()
            warehouses = resp.warehouses
            factories = resp.factories
            if editing {
                applyDraftsFromLoaded()
            }
        } catch {
            self.error = error.localizedDescription
        }
    }

    private func save() async {
        saving = true
        error = nil
        defer { saving = false }
        do {
            guard !warehouseDrafts.isEmpty else {
                error = "At least one warehouse is required"
                return
            }
            let request = SupplierTopologyUpdateRequest(
                warehouses: try warehouseDrafts.map { draft in
                    guard !draft.location.address.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty,
                          draft.location.lat != 0 || draft.location.lng != 0 else {
                        throw URLError(.badURL)
                    }
                    return SupplierTopologyWarehouseInput(
                        warehouseId: draft.warehouseId,
                        name: draft.name.trimmingCharacters(in: .whitespacesAndNewlines),
                        address: draft.location.address,
                        placeId: draft.location.placeId,
                        lat: draft.location.lat,
                        lng: draft.location.lng,
                        coverageRadiusKm: Double(draft.coverageRadiusKm) ?? 50,
                        isActive: true,
                        isOnShift: true,
                        transferMode: "TRUCK"
                    )
                },
                factories: try factoryDrafts.map { draft in
                    guard !draft.location.address.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty,
                          draft.location.lat != 0 || draft.location.lng != 0 else {
                        throw URLError(.badURL)
                    }
                    return SupplierTopologyFactoryInput(
                        factoryId: draft.factoryId,
                        name: draft.name.trimmingCharacters(in: .whitespacesAndNewlines),
                        address: draft.location.address,
                        placeId: draft.location.placeId,
                        lat: draft.location.lat,
                        lng: draft.location.lng,
                        isActive: draft.isActive
                    )
                }
            )
            let resp = try await SupplierOperationsService.updateTopology(request)
            warehouses = resp.warehouses
            factories = resp.factories
            editing = false
        } catch {
            self.error = "Each node needs a valid address."
        }
    }
}

private struct NodeRow: View {
    let name: String
    let address: String
    let meta: String

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(name.isEmpty ? "Unnamed node" : name).font(.body)
            Text(address.isEmpty ? "Coordinates on file" : address)
                .font(.caption)
                .foregroundStyle(.secondary)
            Text(meta)
                .font(.caption)
                .foregroundStyle(.secondary)
        }
    }
}