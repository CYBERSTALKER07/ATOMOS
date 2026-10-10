import SwiftUI

struct CreateOrgMemberSheet: View {
    let topology: SupplierTopologyResponse
    let onDone: () -> Void
    @Environment(\.dismiss) private var dismiss
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    @State private var name = ""
    @State private var email = ""
    @State private var phone = ""
    @State private var password = ""
    @State private var role = "WAREHOUSE_ADMIN"
    @State private var nodeType = "WAREHOUSE"
    @State private var nodeId = ""
    @State private var busy = false
    @State private var error: String?

    var body: some View {
        NavigationStack {
            Form {
                TextField("retailer_desktop.pos.text.name", text: $name)
                TextField("supplier_portal.auth.login.email_label", text: $email)
                TextField("common.field.phone", text: $phone)
                SecureField("Password", text: $password)
                Picker("Role", selection: $role) {
                    Text("supplier_portal.residual.text.warehouse_admin").tag("WAREHOUSE_ADMIN")
                    Text("supplier_portal.residual.text.factory_admin").tag("FACTORY_ADMIN")
                    Text("supplier_portal.residual.text.payload_staff").tag("PAYLOAD")
                    Text("supplier_portal.residual.text.supplier_operator").tag("ADMIN")
                }
                .onChange(of: role) { _, _ in nodeId = "" }
                if role == "PAYLOAD" {
                    Picker("Node type", selection: $nodeType) {
                        Text("factory_portal.insights.text.warehouse").tag("WAREHOUSE")
                        Text("factory_portal.setup.factory.text.factory").tag("FACTORY")
                    }
                    .onChange(of: nodeType) { _, _ in nodeId = "" }
                }
                if role != "ADMIN" {
                    Picker("Node", selection: $nodeId) {
                        Text("supplier_portal.org_fleet.components.org_member_form.text.select_node").tag("")
                        ForEach(nodeOptions, id: \.0) { id, label in Text(label).tag(id) }
                    }
                }
                if let error { Text(error).foregroundStyle(.red) }
            }
            .navigationTitle("mobile_supplier.ui.create_org_member")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("common.action.cancel") { dismiss() } }
                ToolbarItem(placement: .confirmationAction) {
                    Button(busy ? "…" : "Create") { Task { await create() } }
                        .disabled(busy || name.isEmpty || phone.isEmpty || password.isEmpty || (role != "ADMIN" && nodeId.isEmpty))
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
        switch role {
        case "FACTORY_ADMIN":
            return topology.factories.map { ($0.factoryId, $0.name) }
        case "ADMIN":
            return []
        default:
            if nodeType == "FACTORY" {
                return topology.factories.map { ($0.factoryId, $0.name) }
            }
            return topology.warehouses.map { ($0.warehouseId, $0.name) }
        }
    }

    private func create() async {
        busy = true
        error = nil
        defer { busy = false }
        let warehouseId: String? = {
            if role == "WAREHOUSE_ADMIN" { return nodeId }
            if role == "PAYLOAD" && nodeType == "WAREHOUSE" { return nodeId }
            return nil
        }()
        let factoryId: String? = {
            if role == "FACTORY_ADMIN" { return nodeId }
            if role == "PAYLOAD" && nodeType == "FACTORY" { return nodeId }
            return nil
        }()
        do {
            let request = SupplierOrgMemberCreateRequest(
                name: name.trimmingCharacters(in: .whitespaces),
                email: email.isEmpty ? nil : email,
                phone: phone.trimmingCharacters(in: .whitespaces),
                password: password,
                supplierRole: role,
                assignedWarehouseId: warehouseId,
                assignedFactoryId: factoryId
            )
            _ = try await SupplierOperationsService.createOrgMember(
                request,
                idempotencyKey: SupplierIdempotencyKeys.orgMemberCreate(scopeId: SupplierIdempotencyKeys.supplierScopeId(), phone: request.phone)
            )
            dismiss()
            onDone()
        } catch {
            self.error = error.localizedDescription
        }
    }
}
