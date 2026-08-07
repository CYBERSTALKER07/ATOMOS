import SwiftUI

struct StaffView: View {
    @Environment(WarehouseRealtimeHub.self) private var realtimeHub
    @State private var staff: [StaffMember] = []
    @State private var loading = true
    @State private var error: String?
    @State private var showCreate = false
    @State private var createdPin: String?

    var body: some View {
        NavigationStack {
            StaffList(staff: staff, loading: loading, error: error, onRetry: { load() })
            .background(LabTheme.background)
            .navigationTitle("portal.nav.staff")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") { load() }
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Button("mobile_warehouse.ui.add_staff", systemImage: "plus") { showCreate = true }
                }
            }
            .task { load() }
            .refreshable { load(silent: false) }
            .silentRealtimeRefresh(refreshEpoch: realtimeHub.refreshEpoch, reconnectEpoch: realtimeHub.reconnectEpoch) { silent in
                load(silent: silent)
            }
            .sheet(isPresented: $showCreate) {
                CreateStaffSheet { pin in
                    createdPin = pin
                    load()
                }
            }
            .alert("Staff Created", isPresented: Binding(
                get: { createdPin != nil },
                set: { if !$0 { createdPin = nil } }
            )) {
                Button("warehouse_portal.kpi_stat_card.text.done") { createdPin = nil }
            } message: {
                Text(L10n.format("mobile_warehouse.ui.one_time_pin_createdpin_nsave_it_now", "\(createdPin ?? "")"))
            }
        }
    }

    private func load(silent: Bool = false) {
        if !silent { loading = true }
        error = nil
        Task {
            do {
                let resp = try await WarehouseService.staff()
                staff = resp.staff
            } catch {
                if !silent { self.error = error.localizedDescription }
            }
            if !silent { loading = false }
        }
    }
}

private struct CreateStaffSheet: View {
    let onCreated: (String) -> Void
    @Environment(\.dismiss) private var dismiss
    @Environment(WarehouseRealtimeHub.self) private var realtimeHub
    @State private var name = ""
    @State private var phone = ""
    @State private var role = "WAREHOUSE_ADMIN"
    @State private var submitting = false
    @State private var error: String?

    var body: some View {
        NavigationStack {
            Form {
                TextField("retailer_desktop.pos.text.name", text: $name)
                TextField("common.field.phone", text: $phone)
                    .textContentType(.telephoneNumber)
                    .keyboardType(.phonePad)
                TextField("warehouse_portal.staff.text.role", text: $role)
                if let error {
                    Text(error).foregroundStyle(.red).font(.caption)
                }
            }
            .navigationTitle("mobile_warehouse.ui.add_staff")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("common.action.cancel") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("mobile_warehouse.ui.create") { create() }
                        .disabled(submitting || name.isEmpty || phone.isEmpty)
                }
            }
            .onChange(of: realtimeHub.reconnectEpoch) { _, _ in
                if submitting {
                    submitting = false
                    error = "Connection restored — verify status before retrying."
                }
            }
        }
    }

    private func create() {
        submitting = true
        error = nil
        Task {
            do {
                let resp = try await WarehouseService.createStaff(name: name, phone: phone, role: role)
                dismiss()
                onCreated(resp.pin)
            } catch {
                self.error = error.localizedDescription
            }
            submitting = false
        }
    }
}
