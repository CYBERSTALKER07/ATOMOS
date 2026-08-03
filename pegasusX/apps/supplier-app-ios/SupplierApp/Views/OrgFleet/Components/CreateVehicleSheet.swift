import SwiftUI

struct CreateVehicleSheet: View {
    let topology: SupplierTopologyResponse
    let onDone: () -> Void
    @Environment(\.dismiss) private var dismiss
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    @State private var label = ""
    @State private var plate = ""
    @State private var nodeType = "WAREHOUSE"
    @State private var nodeId = ""
    @State private var busy = false
    @State private var error: String?

    var body: some View {
        NavigationStack {
            Form {
                TextField("Label", text: $label)
                TextField("License plate", text: $plate)
                    .textInputAutocapitalization(.characters)
                Picker("Node type", selection: $nodeType) {
                    Text("Warehouse").tag("WAREHOUSE")
                    Text("Factory").tag("FACTORY")
                }
                .onChange(of: nodeType) { _, _ in nodeId = "" }
                Picker("Home node", selection: $nodeId) {
                    Text("Select node").tag("")
                    ForEach(nodeOptions, id: \.0) { id, name in Text(name).tag(id) }
                }
                if let error { Text(error).foregroundStyle(.red) }
            }
            .navigationTitle("Create vehicle")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } }
                ToolbarItem(placement: .confirmationAction) {
                    Button(busy ? "…" : "Create") { Task { await create() } }
                        .disabled(busy || plate.isEmpty || nodeId.isEmpty)
                }
            }
            .onChange(of: realtimeHub.reconnectEpoch) { _, _ in
                if busy {
                    busy = false
                    error = "Connection restored — verify status before retrying."
                }
            }
        }
    }

    private var nodeOptions: [(String, String)] {
        if nodeType == "FACTORY" {
            return topology.factories.map { ($0.factoryId, $0.name) }
        }
        return topology.warehouses.map { ($0.warehouseId, $0.name) }
    }

    private func create() async {
        busy = true
        error = nil
        defer { busy = false }
        do {
            let request = FleetVehicleCreateRequest(
                label: label.isEmpty ? nil : label,
                licensePlate: plate.uppercased(),
                homeNodeType: nodeType,
                homeNodeId: nodeId,
                isActive: nil
            )
            _ = try await SupplierOperationsService.createFleetVehicle(
                request,
                idempotencyKey: SupplierIdempotencyKeys.fleetVehicleCreate(scopeId: SupplierIdempotencyKeys.supplierScopeId(), licensePlate: request.licensePlate)
            )
            dismiss()
            onDone()
        } catch {
            self.error = error.localizedDescription
        }
    }
}
