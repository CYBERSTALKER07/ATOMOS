import SwiftUI

struct DriversView: View {
    @State private var drivers: [Driver] = []
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
                } else if drivers.isEmpty {
                    ContentUnavailableView("No Drivers", systemImage: "person.badge.key", description: Text("Add a driver to get started"))
                } else {
                    List(drivers) { driver in
                        HStack {
                            VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                Text(driver.name)
                                    .font(.headline)
                                Text(driver.phone)
                                    .font(.subheadline)
                                    .foregroundStyle(.secondary)
                            }
                            Spacer()
                            Text(driver.truckStatus.isEmpty ? "IDLE" : driver.truckStatus)
                                .font(.caption.bold())
                                .padding(.horizontal, LabTheme.spacingSM)
                                .padding(.vertical, LabTheme.spacingXS)
                                .background(.quaternary, in: Capsule())
                        }
                    }
                    .listStyle(.insetGrouped)
                }
            }
            .background(LabTheme.background)
            .navigationTitle("Drivers")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Refresh", systemImage: "arrow.clockwise") { load() }
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Add Driver", systemImage: "plus") { showCreate = true }
                }
            }
            .task { load() }
            .refreshable { load() }
            .sheet(isPresented: $showCreate) {
                CreateDriverSheet { pin in
                    createdPin = pin
                    load()
                }
            }
            .alert("Driver Created", isPresented: .init(
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
                let resp = try await WarehouseService.drivers()
                drivers = resp.drivers
            } catch {
                self.error = error.localizedDescription
            }
            loading = false
        }
    }
}

private struct CreateDriverSheet: View {
    let onCreated: (String) -> Void
    @Environment(\.dismiss) private var dismiss
    @State private var name = ""
    @State private var phone = ""
    @State private var submitting = false
    @State private var error: String?

    var body: some View {
        NavigationStack {
            Form {
                TextField("Name", text: $name)
                TextField("Phone", text: $phone)
                    .textContentType(.telephoneNumber)
                    .keyboardType(.phonePad)
                if let error {
                    Text(error).foregroundStyle(.red).font(.caption)
                }
            }
            .navigationTitle("Add Driver")
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
                let resp = try await WarehouseService.createDriver(name: name, phone: phone)
                dismiss()
                onCreated(resp.pin)
            } catch {
                self.error = error.localizedDescription
            }
            submitting = false
        }
    }
}
