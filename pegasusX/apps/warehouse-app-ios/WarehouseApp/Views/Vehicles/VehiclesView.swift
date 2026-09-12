import SwiftUI

struct VehiclesView: View {
    @Environment(WarehouseRealtimeHub.self) private var realtimeHub
    @State private var vehicles: [Vehicle] = []
    @State private var loading = true
    @State private var error: String?
    @State private var showCreate = false

    var body: some View {
        NavigationStack {
            Group {
                if loading && vehicles.isEmpty {
                    ProgressView()
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else if let error, vehicles.isEmpty {
                    ContentUnavailableView {
                        Label("mobile_warehouse.ui.error", systemImage: "exclamationmark.triangle")
                    } description: {
                        Text(error)
                    } actions: {
                        Button("common.action.retry") { load() }
                    }
                } else if vehicles.isEmpty {
                    ContentUnavailableView("No Trucks", systemImage: "truck.box", description: Text("mobile_warehouse.ui.add_a_truck_to_get_started"))
                } else {
                    VehiclesList(vehicles: vehicles)
                }
            }
            .background(LabTheme.background)
            .navigationTitle("portal.nav.trucks")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") { load() }
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Button("mobile_warehouse.ui.add_truck", systemImage: "plus") { showCreate = true }
                }
            }
            .task { load() }
            .refreshable { await load(silent: false) }
            .silentRealtimeRefresh(refreshEpoch: realtimeHub.refreshEpoch, reconnectEpoch: realtimeHub.reconnectEpoch) { silent in
                load(silent: silent)
            }
            .sheet(isPresented: $showCreate) {
                CreateVehicleSheet { load() }
            }
        }
    }

    private func load(silent: Bool = false) {
        if !silent { loading = true }
        error = nil
        Task {
            do {
                let resp = try await WarehouseService.vehicles()
                vehicles = resp.vehicles
            } catch {
                if !silent { self.error = error.localizedDescription }
            }
            if !silent { loading = false }
        }
    }
}

private struct CreateVehicleSheet: View {
    let onCreated: () -> Void
    @Environment(\.dismiss) private var dismiss
    @Environment(WarehouseRealtimeHub.self) private var realtimeHub
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
                TextField("mobile_warehouse.ui.label", text: $label)
                TextField("mobile_warehouse.ui.license_plate", text: $plate)
                Section("Vehicle Class") {
                    Picker("Class", selection: $selectedClass) {
                        ForEach(vehicleClasses, id: \.0) { cls, cap in
                            Text(L10n.format("mobile_warehouse.ui.cls_cap", "\(cls)", "\(cap)")).tag(cls)
                        }
                    }
                    .pickerStyle(.segmented)
                }
                if let error {
                    Text(error).foregroundStyle(.red).font(.caption)
                }
            }
            .navigationTitle("mobile_warehouse.ui.add_truck")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("common.action.cancel") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("mobile_warehouse.ui.create") { create() }
                        .disabled(submitting || label.isEmpty || plate.isEmpty)
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
