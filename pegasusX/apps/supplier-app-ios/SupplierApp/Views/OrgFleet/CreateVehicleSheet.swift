import SwiftUI

struct CreateVehicleSheet: View {
    let topology: SupplierTopologyResponse
    let onDone: () -> Void
    @Environment(\.dismiss) private var dismiss
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    @State private var viewModel = CreateVehicleViewModel()

    var body: some View {
        @Bindable var vm = viewModel
        NavigationStack {
            Form {
                TextField("mobile_supplier.ui.label", text: $vm.label)
                TextField("warehouse_portal.vehicles._vehicle_id_.text.license_plate", text: $vm.plate)
                    .textInputAutocapitalization(.characters)
                Picker("mobile_supplier.chrome.node_type", selection: $vm.nodeType) {
                    Text("factory_portal.insights.text.warehouse").tag("WAREHOUSE")
                    Text("factory_portal.setup.factory.text.factory").tag("FACTORY")
                }
                .onChange(of: viewModel.nodeType) { _, _ in viewModel.nodeId = "" }
                Picker("mobile_supplier.chrome.home_node", selection: $vm.nodeId) {
                    Text("supplier_portal.org_fleet.components.org_member_form.text.select_node").tag("")
                    ForEach(viewModel.nodeOptions(topology: topology), id: \.0) { id, name in Text(name).tag(id) }
                }
                if let error = viewModel.error { Text(error).foregroundStyle(SupplierTheme.destructive) }
            }
            .scrollDismissesKeyboard(.interactively)
            .navigationTitle("mobile_supplier.ui.create_vehicle")
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
                    .disabled(viewModel.busy || viewModel.plate.isEmpty || viewModel.nodeId.isEmpty)
                }
            }
            .onChange(of: realtimeHub.reconnectEpoch) { _, _ in
                viewModel.handleReconnect()
            }
        }
    }
}
