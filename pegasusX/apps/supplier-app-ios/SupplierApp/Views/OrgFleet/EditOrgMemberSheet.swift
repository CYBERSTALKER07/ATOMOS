import SwiftUI

struct EditOrgMemberSheet: View {
    let member: SupplierOrgMember
    let onDone: () -> Void
    @Environment(\.dismiss) private var dismiss
    @State private var viewModel: EditOrgMemberViewModel

    init(member: SupplierOrgMember, onDone: @escaping () -> Void) {
        self.member = member
        self.onDone = onDone
        _viewModel = State(initialValue: EditOrgMemberViewModel(member: member))
    }

    var body: some View {
        @Bindable var vm = viewModel
        NavigationStack {
            Form {
                TextField("retailer_desktop.pos.text.name", text: $vm.name)
                Picker("mobile_supplier.chrome.role", selection: $vm.role) {
                    Text("supplier_portal.residual.text.warehouse_admin").tag("WAREHOUSE_ADMIN")
                    Text("supplier_portal.residual.text.factory_admin").tag("FACTORY_ADMIN")
                    Text("supplier_portal.residual.text.payload_staff").tag("PAYLOAD")
                    Text("supplier_portal.residual.text.supplier_operator").tag("ADMIN")
                }
                Toggle("mobile_supplier.chrome.active", isOn: $vm.isActive)
                if let error = viewModel.error { Text(error).foregroundStyle(SupplierTheme.destructive) }
            }
            .scrollDismissesKeyboard(.interactively)
            .navigationTitle("mobile_supplier.ui.edit_member")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("common.action.cancel") { dismiss() } }
                ToolbarItem(placement: .confirmationAction) {
                    Button(LocalizedStringKey(viewModel.busy ? "…" : "mobile_supplier.chrome.save")) {
                        Task {
                            await viewModel.save {
                                dismiss()
                                onDone()
                            }
                        }
                    }
                    .disabled(viewModel.busy || viewModel.name.isEmpty)
                }
            }
        }
    }
}
