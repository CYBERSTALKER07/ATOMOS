import SwiftUI

struct CreateDriverSheet: View {
    let topology: SupplierTopologyResponse
    let vehicles: [FleetVehicle]
    let onDone: () -> Void
    @Environment(\.dismiss) private var dismiss
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    @State private var viewModel = CreateDriverViewModel()

    var body: some View {
        @Bindable var vm = viewModel
        let filteredVehicles = viewModel.filteredVehicles(from: vehicles)
        NavigationStack {
            Form {
                TextField("retailer_desktop.pos.text.name", text: $vm.name)
                TextField("common.field.phone", text: $vm.phone)
                SecureField("PIN", text: $vm.pin)
                Picker("mobile_supplier.chrome.node_type", selection: $vm.nodeType) {
                    Text("factory_portal.insights.text.warehouse").tag("WAREHOUSE")
                    Text("factory_portal.setup.factory.text.factory").tag("FACTORY")
                }
                .onChange(of: viewModel.nodeType) { _, _ in viewModel.resetNodeSelection() }
                Picker("mobile_supplier.chrome.home_node", selection: $vm.nodeId) {
                    Text("supplier_portal.org_fleet.components.org_member_form.text.select_node").tag("")
                    ForEach(viewModel.nodeOptions(topology: topology), id: \.0) { id, label in
                        Text(label).tag(id)
                    }
                }
                if !filteredVehicles.isEmpty {
                    Picker("mobile_supplier.chrome.vehicle", selection: $vm.vehicleId) {
                        Text("mobile_supplier.ui.assign_later").tag("")
                        ForEach(filteredVehicles) { vehicle in
                            Text(vehicle.licensePlate).tag(vehicle.vehicleId)
                        }
                    }
                }
                if let error = viewModel.error { Text(error).foregroundStyle(SupplierTheme.destructive) }
            }
            .scrollDismissesKeyboard(.interactively)
            .navigationTitle("mobile_supplier.ui.create_driver")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("common.action.cancel") { dismiss() } }
                ToolbarItem(placement: .confirmationAction) {
                    Button(LocalizedStringKey(viewModel.busy ? "mobile_supplier.chrome.creating" : "mobile_supplier.chrome.create")) {
                        Task {
                            await viewModel.create {
                                dismiss()
                                onDone()
                            }
                        }
                    }
                    .disabled(viewModel.busy || viewModel.name.isEmpty || viewModel.phone.isEmpty || viewModel.pin.isEmpty || viewModel.nodeId.isEmpty)
                }
            }
            .onChange(of: realtimeHub.reconnectEpoch) { _, _ in
                viewModel.handleReconnect()
            }
        }
    }
}
