import SwiftUI

struct CreateOrgMemberSheet: View {
    let topology: SupplierTopologyResponse
    let onDone: () -> Void
    @Environment(\.dismiss) private var dismiss
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    @State private var viewModel = CreateOrgMemberViewModel()

    var body: some View {
        @Bindable var vm = viewModel
        NavigationStack {
            Form {
                TextField("retailer_desktop.pos.text.name", text: $vm.name)
                TextField("supplier_portal.auth.login.email_label", text: $vm.email)
                TextField("common.field.phone", text: $vm.phone)
                SecureField("mobile_supplier.chrome.password", text: $vm.password)
                Picker("mobile_supplier.chrome.role", selection: $vm.role) {
                    Text("supplier_portal.residual.text.warehouse_admin").tag("WAREHOUSE_ADMIN")
                    Text("supplier_portal.residual.text.factory_admin").tag("FACTORY_ADMIN")
                    Text("supplier_portal.residual.text.payload_staff").tag("PAYLOAD")
                    Text("supplier_portal.residual.text.supplier_operator").tag("ADMIN")
                }
                .onChange(of: viewModel.role) { _, _ in viewModel.nodeId = "" }
                if viewModel.role == "PAYLOAD" {
                    Picker("mobile_supplier.chrome.node_type", selection: $vm.nodeType) {
                        Text("factory_portal.insights.text.warehouse").tag("WAREHOUSE")
                        Text("factory_portal.setup.factory.text.factory").tag("FACTORY")
                    }
                    .onChange(of: viewModel.nodeType) { _, _ in viewModel.nodeId = "" }
                }
                if viewModel.role != "ADMIN" {
                    Picker("mobile_supplier.chrome.node", selection: $vm.nodeId) {
                        Text("supplier_portal.org_fleet.components.org_member_form.text.select_node").tag("")
                        ForEach(viewModel.nodeOptions(topology: topology), id: \.0) { id, label in Text(label).tag(id) }
                    }
                }
                if let error = viewModel.error { Text(error).foregroundStyle(SupplierTheme.destructive) }
            }
            .scrollDismissesKeyboard(.interactively)
            .navigationTitle("mobile_supplier.ui.create_org_member")
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
                    .disabled(viewModel.busy || viewModel.name.isEmpty || viewModel.phone.isEmpty || viewModel.password.isEmpty || (viewModel.role != "ADMIN" && viewModel.nodeId.isEmpty))
                }
            }
            .onChange(of: realtimeHub.reconnectEpoch) { _, _ in
                viewModel.handleReconnect()
            }
        }
    }
}
