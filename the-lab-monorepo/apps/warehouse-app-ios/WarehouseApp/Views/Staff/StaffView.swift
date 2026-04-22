import SwiftUI

struct StaffView: View {
    @State private var staff: [StaffMember] = []
    @State private var loading = true
    @State private var error: String?
    @State private var showCreate = false
    @State private var createdPin: String?

    var body: some View {
        NavigationStack {
            Group {
                if loading {
                    ProgressView()
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else if let error {
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
                    List(staff) { member in
                        HStack {
                            VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                Text(member.name)
                                    .font(.headline)
                                Text("\(member.role) · \(member.phone)")
                                    .font(.subheadline)
                                    .foregroundStyle(.secondary)
                            }
                            Spacer()
                            Text(member.isActive ? "Active" : "Inactive")
                                .font(.caption.bold())
                                .padding(.horizontal, LabTheme.spacingSM)
                                .padding(.vertical, LabTheme.spacingXS)
                                .foregroundStyle(member.isActive ? .primary : .white)
                                .background(member.isActive ? Color.clear : .red, in: Capsule())
                                .overlay {
                                    if member.isActive {
                                        Capsule().strokeBorder(.quaternary)
                                    }
                                }
                        }
                    }
                    .listStyle(.insetGrouped)
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
            .refreshable { load() }
            .sheet(isPresented: $showCreate) {
                CreateStaffSheet { pin in
                    createdPin = pin
                    load()
                }
            }
            .alert("Staff Created", isPresented: .init(
                get: { createdPin != nil },
                set: { if !$0 { createdPin = nil } }
            )) {
                Button("Done") { createdPin = nil }
            } message: {
                Text("One-time PIN: \(createdPin ?? "")\nSave it now.")
            }
        }
    }

    private func load() {
        loading = true
        error = nil
        Task {
            do {
                let resp = try await WarehouseService.staff()
                staff = resp.staff
            } catch {
                self.error = error.localizedDescription
            }
            loading = false
        }
    }
}

private struct CreateStaffSheet: View {
    let onCreated: (String) -> Void
    @Environment(\.dismiss) private var dismiss
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
