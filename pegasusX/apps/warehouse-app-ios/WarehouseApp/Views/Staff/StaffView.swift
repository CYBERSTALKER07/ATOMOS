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
            Group {
                if loading && staff.isEmpty {
                    ProgressView()
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else if let error, staff.isEmpty {
                    ContentUnavailableView {
                        Label("Error", systemImage: "exclamationmark.triangle")
                    } description: {
                        Text(error)
                    } actions: {
                        Button("Retry") { load() }
                    }
                } else if staff.isEmpty {
                    ContentUnavailableView("No Staff", systemImage: "person.2", description: Text("Add staff members"))
                } else {
                    StaffList(staff: staff)
                }
            }
            .background(LabTheme.background)
            .navigationTitle("Staff")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Refresh", systemImage: "arrow.clockwise") { load() }
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Add Staff", systemImage: "plus") { showCreate = true }
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
                Button("Done") { createdPin = nil }
            } message: {
                Text("One-time PIN: \(createdPin ?? "")\nSave it now.")
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
                TextField("Name", text: $name)
                TextField("Phone", text: $phone)
                    .textContentType(.telephoneNumber)
                    .keyboardType(.phonePad)
                TextField("Role", text: $role)
                if let error {
                    Text(error).foregroundStyle(.red).font(.caption)
                }
            }
            .navigationTitle("Add Staff")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Create") { create() }
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
