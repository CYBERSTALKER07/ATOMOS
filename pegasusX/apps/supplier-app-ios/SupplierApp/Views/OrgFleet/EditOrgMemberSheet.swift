import SwiftUI

struct EditOrgMemberSheet: View {
    let member: SupplierOrgMember
    let onDone: () -> Void
    @Environment(\.dismiss) private var dismiss
    @State private var name: String
    @State private var role: String
    @State private var isActive: Bool
    @State private var busy = false
    @State private var error: String?

    init(member: SupplierOrgMember, onDone: @escaping () -> Void) {
        self.member = member
        self.onDone = onDone
        _name = State(initialValue: member.name)
        _role = State(initialValue: member.supplierRole)
        _isActive = State(initialValue: member.isActive)
    }

    var body: some View {
        NavigationStack {
            Form {
                TextField("retailer_desktop.pos.text.name", text: $name)
                Picker("Role", selection: $role) {
                    Text("supplier_portal.residual.text.warehouse_admin").tag("WAREHOUSE_ADMIN")
                    Text("supplier_portal.residual.text.factory_admin").tag("FACTORY_ADMIN")
                    Text("supplier_portal.residual.text.payload_staff").tag("PAYLOAD")
                    Text("supplier_portal.residual.text.supplier_operator").tag("ADMIN")
                }
                Toggle("Active", isOn: $isActive)
                if let error { Text(error).foregroundStyle(.red) }
            }
            .navigationTitle("mobile_supplier.ui.edit_member")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("common.action.cancel") { dismiss() } }
                ToolbarItem(placement: .confirmationAction) {
                    Button(busy ? "…" : "Save") { Task { await save() } }
                        .disabled(busy || name.isEmpty)
                }
            }
        }
    }

    private func save() async {
        busy = true
        error = nil
        defer { busy = false }
        do {
            let revision = "\(name):\(role):\(isActive)"
            _ = try await SupplierOperationsService.updateOrgMember(
                member.userId,
                request: SupplierOrgMemberUpdateRequest(
                    name: name.trimmingCharacters(in: .whitespaces),
                    supplierRole: role,
                    assignedWarehouseId: member.assignedWarehouseId,
                    assignedFactoryId: member.assignedFactoryId,
                    isActive: isActive
                ),
                idempotencyKey: SupplierIdempotencyKeys.orgMemberUpdate(scopeId: SupplierIdempotencyKeys.supplierScopeId(), userId: member.userId, revision: revision)
            )
            dismiss()
            onDone()
        } catch {
            self.error = error.localizedDescription
        }
    }
}
