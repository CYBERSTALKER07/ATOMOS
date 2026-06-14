import SwiftUI

private struct WarehouseDraft: Identifiable {
    let id: String
    var warehouseId: String?
    var name: String
    var lat: String
    var lng: String
    var coverageRadiusKm: String
}

private struct FactoryDraft: Identifiable {
    let id: String
    var factoryId: String?
    var name: String
    var lat: String
    var lng: String
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
                    Button("Configure topology") { editing = true }
                }
            } else {
                readOnlyList
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("Factories & warehouses")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                if editing {
                    Button(saving ? "Saving…" : "Save") {
                        Task { await save() }
                    }
                    .disabled(saving)
                } else {
                    Button("Edit") { beginEditing() }
                }
            }
            if editing {
                ToolbarItem(placement: .topBarLeading) {
                    Button("Cancel") {
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
        List {
            Section("Warehouses (\(warehouses.count))") {
                ForEach(warehouses) { node in
                    NodeRow(
                        name: node.name,
                        lat: node.lat,
                        lng: node.lng,
                        meta: String(format: "%.0f km coverage", node.coverageRadiusKm)
                    )
                }
            }
            Section("Factories (\(factories.count))") {
                ForEach(factories) { node in
                    NodeRow(
                        name: node.name,
                        lat: node.lat,
                        lng: node.lng,
                        meta: node.isActive ? "Active" : "Inactive"
                    )
                }
            }
        }
        .listStyle(.insetGrouped)
    }

    private var editForm: some View {
        Form {
            if let error {
                Section {
                    Text(error).foregroundStyle(.red)
                }
            }
            Section("Warehouses") {
                ForEach($warehouseDrafts) { $draft in
                    TextField("Name", text: $draft.name)
                    TextField("Latitude", text: $draft.lat)
                        .keyboardType(.decimalPad)
                    TextField("Longitude", text: $draft.lng)
                        .keyboardType(.decimalPad)
                    TextField("Coverage km", text: $draft.coverageRadiusKm)
                        .keyboardType(.decimalPad)
                }
                Button("Add warehouse") {
                    warehouseDrafts.append(
                        WarehouseDraft(
                            id: "new-wh-\(warehouseDrafts.count)",
                            warehouseId: nil,
                            name: "Warehouse \(warehouseDrafts.count + 1)",
                            lat: "41.2995",
                            lng: "69.2401",
                            coverageRadiusKm: "50"
                        )
                    )
                }
            }
            Section("Factories") {
                ForEach($factoryDrafts) { $draft in
                    TextField("Name", text: $draft.name)
                    TextField("Latitude", text: $draft.lat)
                        .keyboardType(.decimalPad)
                    TextField("Longitude", text: $draft.lng)
                        .keyboardType(.decimalPad)
                }
                Button("Add factory") {
                    factoryDrafts.append(
                        FactoryDraft(
                            id: "new-fc-\(factoryDrafts.count)",
                            factoryId: nil,
                            name: "Factory \(factoryDrafts.count + 1)",
                            lat: "41.3111",
                            lng: "69.2797"
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
                lat: String(node.lat),
                lng: String(node.lng),
                coverageRadiusKm: String(node.coverageRadiusKm)
            )
        }
        factoryDrafts = factories.enumerated().map { index, node in
            FactoryDraft(
                id: node.factoryId.isEmpty ? "fc-\(index)" : node.factoryId,
                factoryId: node.factoryId.isEmpty ? nil : node.factoryId,
                name: node.name,
                lat: String(node.lat),
                lng: String(node.lng)
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
                    guard let lat = Double(draft.lat), let lng = Double(draft.lng) else {
                        throw URLError(.badURL)
                    }
                    return SupplierTopologyWarehouseInput(
                        warehouseId: draft.warehouseId,
                        name: draft.name.trimmingCharacters(in: .whitespacesAndNewlines),
                        lat: lat,
                        lng: lng,
                        coverageRadiusKm: Double(draft.coverageRadiusKm) ?? 50,
                        isActive: true,
                        isOnShift: true,
                        transferMode: "TRUCK"
                    )
                },
                factories: try factoryDrafts.map { draft in
                    guard let lat = Double(draft.lat), let lng = Double(draft.lng) else {
                        throw URLError(.badURL)
                    }
                    return SupplierTopologyFactoryInput(
                        factoryId: draft.factoryId,
                        name: draft.name.trimmingCharacters(in: .whitespacesAndNewlines),
                        lat: lat,
                        lng: lng,
                        isActive: true
                    )
                }
            )
            let resp = try await SupplierOperationsService.updateTopology(request)
            warehouses = resp.warehouses
            factories = resp.factories
            editing = false
        } catch {
            self.error = error.localizedDescription
        }
    }
}

private struct NodeRow: View {
    let name: String
    let lat: Double
    let lng: Double
    let meta: String

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(name.isEmpty ? "Unnamed node" : name).font(.body)
            Text(String(format: "%.4f, %.4f", lat, lng))
                .font(.caption)
                .foregroundStyle(.secondary)
            Text(meta)
                .font(.caption)
                .foregroundStyle(.secondary)
        }
    }
}
