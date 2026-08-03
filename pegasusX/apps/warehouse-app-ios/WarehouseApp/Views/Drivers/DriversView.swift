import SwiftUI

struct DriversView: View {
    @Environment(WarehouseRealtimeHub.self) private var realtimeHub
    @State private var drivers: [Driver] = []
    @State private var vehicles: [Vehicle] = []
    @State private var loading = true
    @State private var error: String?
    @State private var showCreate = false
    @State private var createdPin: String?
    @State private var updatingDriverId: String?

    var body: some View {
        NavigationStack {
            Group {
                if loading && drivers.isEmpty {
                    ProgressView()
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else if let error, drivers.isEmpty {
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
                    DriversList(
                        drivers: drivers,
                        vehicles: vehicles,
                        updatingDriverId: updatingDriverId,
                        onAssign: { driverId, vehicleId in
                            assign(driverId: driverId, vehicleId: vehicleId)
                        }
                    )
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
            .refreshable { load(silent: false) }
            .silentRealtimeRefresh(refreshEpoch: realtimeHub.refreshEpoch, reconnectEpoch: realtimeHub.reconnectEpoch) { silent in
                load(silent: silent)
            }
            .onChange(of: realtimeHub.reconnectEpoch) { _, _ in
                if updatingDriverId != nil {
                    updatingDriverId = nil
                    error = "Connection restored — verify assignment status before retrying."
                }
            }
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

    private func load(silent: Bool = false) {
        if !silent { loading = true }
        error = nil
        Task {
            do {
                async let driverResponse = WarehouseService.drivers()
                async let vehicleResponse = WarehouseService.vehicles()
                let (driverResp, vehicleResp) = try await (driverResponse, vehicleResponse)
                drivers = driverResp.drivers
                vehicles = vehicleResp.vehicles
            } catch {
                if !silent { self.error = error.localizedDescription }
            }
            if !silent { loading = false }
        }
    }

    private func assign(driverId: String, vehicleId: String?) {
        updatingDriverId = driverId
        error = nil
        Task {
            do {
                _ = try await WarehouseService.assignDriver(driverId: driverId, vehicleId: vehicleId)
                load()
            } catch {
                self.error = error.localizedDescription
            }
            updatingDriverId = nil
        }
    }

}

private struct CreateDriverSheet: View {
    let onCreated: (String) -> Void
    @Environment(\.dismiss) private var dismiss
    @Environment(WarehouseRealtimeHub.self) private var realtimeHub
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
