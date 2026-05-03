import SwiftUI

struct DriversView: View {
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
                                Text(assignedVehicleLabel(for: driver))
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                                if let reason = assignedVehicleReason(for: driver), !reason.isEmpty {
                                    Text(reason)
                                        .font(.caption)
                                        .foregroundStyle(.orange)
                                }
                            }
                            Spacer()
                            VStack(alignment: .trailing, spacing: LabTheme.spacingXS) {
                                Text(driver.truckStatus.isEmpty ? "IDLE" : driver.truckStatus)
                                    .font(.caption.bold())
                                    .padding(.horizontal, LabTheme.spacingSM)
                                    .padding(.vertical, LabTheme.spacingXS)
                                    .background(.quaternary, in: Capsule())
                                Menu {
                                    Button("Unassign") {
                                        assign(driverId: driver.driverId, vehicleId: nil)
                                    }
                                    ForEach(assignableVehicles(for: driver)) { vehicle in
                                        Button(vehicleLabel(for: vehicle)) {
                                            assign(driverId: driver.driverId, vehicleId: vehicle.vehicleId)
                                        }
                                    }
                                } label: {
                                    if updatingDriverId == driver.driverId {
                                        ProgressView()
                                            .controlSize(.small)
                                    } else {
                                        Label(driver.vehicleId == nil ? "Assign" : "Reassign", systemImage: "truck.box")
                                            .font(.caption)
                                    }
                                }
                                .disabled(updatingDriverId == driver.driverId)
                            }
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
                async let driverResponse = WarehouseService.drivers()
                async let vehicleResponse = WarehouseService.vehicles()
                let (driverResp, vehicleResp) = try await (driverResponse, vehicleResponse)
                drivers = driverResp.drivers
                vehicles = vehicleResp.vehicles
            } catch {
                self.error = error.localizedDescription
            }
            loading = false
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

    private func assignedVehicleLabel(for driver: Driver) -> String {
        guard let vehicleId = driver.vehicleId,
              let vehicle = vehicles.first(where: { $0.vehicleId == vehicleId }) else {
            return "Unassigned"
        }
        return vehicleLabel(for: vehicle)
    }

    private func assignedVehicleReason(for driver: Driver) -> String? {
        guard driver.vehicleId != nil else {
            return nil
        }

        if driver.vehicleIsActive == false {
            if let reason = driver.vehicleUnavailableReason, !reason.isEmpty {
                return "Vehicle unavailable: \(vehicleUnavailableReasonLabel(reason))"
            }
            return "Vehicle unavailable"
        }

        guard let vehicleId = driver.vehicleId,
              let vehicle = vehicles.first(where: { $0.vehicleId == vehicleId }),
              !vehicle.isActive else {
            return nil
        }

        if let reason = vehicle.unavailableReason, !reason.isEmpty {
            return "Vehicle unavailable: \(vehicleUnavailableReasonLabel(reason))"
        }
        return "Vehicle unavailable"
    }

    private func assignableVehicles(for driver: Driver) -> [Vehicle] {
        vehicles.filter { $0.isActive || $0.vehicleId == driver.vehicleId }
    }

    private func vehicleLabel(for vehicle: Vehicle) -> String {
        let title = vehicle.label.isEmpty ? vehicle.licensePlate : vehicle.label
        return [title, vehicle.vehicleClass].filter { !$0.isEmpty }.joined(separator: " · ")
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
