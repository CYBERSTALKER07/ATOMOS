import SwiftUI

struct VehiclesView: View {
    @State private var vehicles: [Vehicle] = []
    @State private var loading = true
    @State private var error: String?
    @State private var showCreate = false
    @State private var mutatingVehicleId: String?
    @State private var reasonVehicle: Vehicle?

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
                } else if vehicles.isEmpty {
                    ContentUnavailableView("No Vehicles", systemImage: "truck.box", description: Text("Add a vehicle to get started"))
                } else {
                    List(vehicles) { vehicle in
                        HStack {
                            VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                Text(vehicle.label.isEmpty ? vehicle.licensePlate : vehicle.label)
                                    .font(.headline)
                                Text("\(vehicle.vehicleClass) · \(vehicle.capacityVu) VU")
                                    .font(.subheadline)
                                    .foregroundStyle(.secondary)
                                Text(vehicle.assignedDriverName ?? "Unassigned")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                                if !vehicle.isActive, let unavailableReason = vehicle.unavailableReason, !unavailableReason.isEmpty {
                                    Text(vehicleUnavailableReasonLabel(unavailableReason))
                                        .font(.caption)
                                        .foregroundStyle(.orange)
                                }
                            }
                            Spacer()
                            if mutatingVehicleId == vehicle.vehicleId {
                                ProgressView()
                                    .controlSize(.small)
                            } else {
                                Text(vehicle.isActive ? (vehicle.status.isEmpty ? "AVAILABLE" : vehicle.status) : "UNAVAILABLE")
                                    .font(.caption.bold())
                                    .padding(.horizontal, LabTheme.spacingSM)
                                    .padding(.vertical, LabTheme.spacingXS)
                                    .background(.quaternary, in: Capsule())
                            }
                        }
                        .swipeActions(edge: .trailing, allowsFullSwipe: false) {
                            if vehicle.isActive {
                                Button("Unavailable") {
                                    reasonVehicle = vehicle
                                }
                                .tint(.orange)
                            } else {
                                Button("Restore") {
                                    toggleAvailability(for: vehicle, isActive: true)
                                }
                                .tint(.green)
                            }
                        }
                    }
                    .listStyle(.insetGrouped)
                }
            }
            .background(LabTheme.background)
            .navigationTitle("Vehicles")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Refresh", systemImage: "arrow.clockwise") { load() }
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Add Vehicle", systemImage: "plus") { showCreate = true }
                }
            }
            .task { load() }
            .refreshable { load() }
            .sheet(isPresented: $showCreate) {
                CreateVehicleSheet { load() }
            }
            .confirmationDialog(
                "Set Vehicle Unavailable",
                isPresented: Binding(
                    get: { reasonVehicle != nil },
                    set: { isPresented in
                        if !isPresented {
                            reasonVehicle = nil
                        }
                    }
                ),
                presenting: reasonVehicle
            ) { vehicle in
                ForEach(VehicleUnavailableReasonOption.allCases) { reason in
                    Button(reason.title) {
                        toggleAvailability(for: vehicle, isActive: false, unavailableReason: reason.rawValue)
                    }
                }
                Button("Cancel", role: .cancel) {
                    reasonVehicle = nil
                }
            } message: { vehicle in
                Text("Choose why \(vehicle.label.isEmpty ? vehicle.licensePlate : vehicle.label) is unavailable.")
            }
        }
    }

    private func load() {
        loading = true
        error = nil
        Task {
            do {
                let resp = try await WarehouseService.vehicles()
                vehicles = resp.vehicles
            } catch {
                self.error = error.localizedDescription
            }
            loading = false
        }
    }

    private func toggleAvailability(for vehicle: Vehicle, isActive: Bool, unavailableReason: String? = nil) {
        mutatingVehicleId = vehicle.vehicleId
        error = nil
        reasonVehicle = nil
        Task {
            do {
                _ = try await WarehouseService.updateVehicleAvailability(vehicleId: vehicle.vehicleId, isActive: isActive, unavailableReason: unavailableReason)
                load()
            } catch {
                self.error = error.localizedDescription
            }
            mutatingVehicleId = nil
        }
    }
}

private struct CreateVehicleSheet: View {
    let onCreated: () -> Void
    @Environment(\.dismiss) private var dismiss
    @State private var label = ""
    @State private var plate = ""
    @State private var selectedClass = "CLASS_A"
    @State private var submitting = false
    @State private var error: String?

    private let vehicleClasses = [
        ("CLASS_A", "50 VU"),
        ("CLASS_B", "150 VU"),
        ("CLASS_C", "400 VU"),
    ]

    var body: some View {
        NavigationStack {
            Form {
                TextField("Label", text: $label)
                TextField("License Plate", text: $plate)
                Section("Vehicle Class") {
                    Picker("Class", selection: $selectedClass) {
                        ForEach(vehicleClasses, id: \.0) { cls, cap in
                            Text("\(cls) (\(cap))").tag(cls)
                        }
                    }
                    .pickerStyle(.segmented)
                }
                if let error {
                    Text(error).foregroundStyle(.red).font(.caption)
                }
            }
            .navigationTitle("Add Vehicle")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Create") { create() }
                        .disabled(submitting || label.isEmpty || plate.isEmpty)
                }
            }
        }
    }

    private func create() {
        submitting = true
        error = nil
        Task {
            do {
                _ = try await WarehouseService.createVehicle(label: label, licensePlate: plate, vehicleClass: selectedClass)
                dismiss()
                onCreated()
            } catch {
                self.error = error.localizedDescription
            }
            submitting = false
        }
    }
}
