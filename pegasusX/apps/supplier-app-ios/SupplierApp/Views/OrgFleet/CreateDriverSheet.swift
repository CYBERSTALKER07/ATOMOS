import SwiftUI

struct CreateDriverSheet: View {
    let topology: SupplierTopologyResponse
    let vehicles: [FleetVehicle]
    let onDone: () -> Void
    @Environment(\.dismiss) private var dismiss
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    @State private var name = ""
    @State private var phone = ""
    @State private var pin = ""
    @State private var nodeType = "WAREHOUSE"
    @State private var nodeId = ""
    @State private var vehicleId = ""
    @State private var busy = false
    @State private var error: String?

    var body: some View {
        NavigationStack {
            Form {
                TextField("retailer_desktop.pos.text.name", text: $name)
                TextField("common.field.phone", text: $phone)
                SecureField("PIN", text: $pin)
                Picker("Node type", selection: $nodeType) {
                    Text("factory_portal.insights.text.warehouse").tag("WAREHOUSE")
                    Text("factory_portal.setup.factory.text.factory").tag("FACTORY")
                }
                .onChange(of: nodeType) { _, _ in nodeId = ""; vehicleId = "" }
                Picker("Home node", selection: $nodeId) {
                    Text("supplier_portal.org_fleet.components.org_member_form.text.select_node").tag("")
                    ForEach(nodeOptions, id: \.0) { id, label in
                        Text(label).tag(id)
                    }
                }
                if !filteredVehicles.isEmpty {
                    Picker("Vehicle", selection: $vehicleId) {
                        Text("mobile_supplier.ui.assign_later").tag("")
                        ForEach(filteredVehicles) { vehicle in
                            Text(vehicle.licensePlate).tag(vehicle.vehicleId)
                        }
                    }
                }
                if let error { Text(error).foregroundStyle(.red) }
            }
            .navigationTitle("mobile_supplier.ui.create_driver")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("common.action.cancel") { dismiss() } }
                ToolbarItem(placement: .confirmationAction) {
                    Button(busy ? "…" : "Create") { Task { await create() } }
                        .disabled(busy || name.isEmpty || phone.isEmpty || pin.isEmpty || nodeId.isEmpty)
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

    private var filteredVehicles: [FleetVehicle] {
        vehicles.filter { $0.homeNodeType == nodeType && $0.homeNodeId == nodeId }
    }

    private func create() async {
        busy = true
        error = nil
        defer { busy = false }
        do {
            let request = FleetDriverCreateRequest(
                name: name.trimmingCharacters(in: .whitespaces),
                phone: phone.trimmingCharacters(in: .whitespaces),
                pin: pin,
                homeNodeType: nodeType,
                homeNodeId: nodeId,
                vehicleId: vehicleId.isEmpty ? nil : vehicleId,
                isActive: nil
            )
            _ = try await SupplierOperationsService.createFleetDriver(
                request,
                idempotencyKey: SupplierIdempotencyKeys.fleetDriverCreate(scopeId: SupplierIdempotencyKeys.supplierScopeId(), phone: request.phone)
            )
            dismiss()
            onDone()
        } catch {
            self.error = error.localizedDescription
        }
    }
}
