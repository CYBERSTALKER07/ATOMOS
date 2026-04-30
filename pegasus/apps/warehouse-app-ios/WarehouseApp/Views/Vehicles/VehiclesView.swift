import SwiftUI

struct VehiclesView: View {
    @State private var vehicles: [Vehicle] = []
    @State private var loading = true
    @State private var error: String?
    @State private var showCreate = false

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
                            }
                            Spacer()
                            Text(vehicle.status.isEmpty ? "AVAILABLE" : vehicle.status)
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
