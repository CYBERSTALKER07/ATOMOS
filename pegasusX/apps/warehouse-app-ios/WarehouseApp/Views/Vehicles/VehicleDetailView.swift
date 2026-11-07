import SwiftUI

struct VehicleDetailView: View {
    let vehicleId: String

    @Environment(WarehouseRealtimeHub.self) private var realtimeHub
    @State private var vehicle: Vehicle?
    @State private var loading = true
    @State private var error: String?
    @State private var mutating = false
    @State private var selectedReason = "MANUAL_HOLD"
    @State private var customNote = ""

    var body: some View {
        Group {
            if loading && vehicle == nil {
                ProgressView("Loading truck…")
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let error, vehicle == nil {
                ContentUnavailableView {
                    Label("mobile_warehouse.ui.error", systemImage: "exclamationmark.triangle")
                } description: {
                    Text(error)
                } actions: {
                    Button("common.action.retry") { load() }
                }
            } else if let vehicle {
                ScrollView {
                    VStack(alignment: .leading, spacing: LabTheme.spacingLG) {
                        VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                            Text(vehicle.label.isEmpty ? vehicle.licensePlate : vehicle.label)
                                .font(.title2.bold())
                            Text(L10n.format("mobile_warehouse.ui.licenseplate_vehicleclass_capacityvu_vu", "\(vehicle.licensePlate)", "\(vehicle.vehicleClass)", "\(vehicle.capacityVu)"))
                                .font(.subheadline)
                                .foregroundStyle(.secondary)
                            Text(vehicle.isActive ? (vehicle.status.isEmpty ? "ACTIVE" : vehicle.status) : "UNAVAILABLE")
                                .font(.caption.bold())
                                .padding(.horizontal, LabTheme.spacingSM)
                                .padding(.vertical, LabTheme.spacingXS)
                                .background(.quaternary, in: Capsule())
                        }

                        WarehouseSectionHeader(title: "Assignment", subtitle: nil)
                        Text(vehicle.assignedDriverName ?? "Unassigned")
                            .font(.body)

                        WarehouseSectionHeader(title: "Dispatch impact", subtitle: nil)
                        Text(
                            vehicle.isActive
                                ? "This truck is eligible for manual and smart dispatch."
                                : "Excluded from dispatch: \(formatUnavailableReason(vehicle.unavailableReason, note: vehicle.unavailableNote))"
                        )
                        .font(.subheadline)
                        .foregroundStyle(.secondary)

                        VehicleAvailabilityPanel(
                            vehicle: vehicle,
                            mutating: mutating,
                            selectedReason: $selectedReason,
                            customNote: $customNote,
                            onMarkUnavailable: { reason, note in
                                updateAvailability(isActive: false, reason: reason, note: note)
                            },
                            onRestore: {
                                updateAvailability(isActive: true)
                            }
                        )
                    }
                    .padding(LabTheme.spacingLG)
                }
            }
        }
        .background(LabTheme.background)
        .navigationTitle(vehicle?.label.isEmpty == false ? vehicle!.label : "Truck")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") { load() }
            }
        }
        .task(id: vehicleId) { load() }
        .silentRealtimeRefresh(refreshEpoch: realtimeHub.refreshEpoch, reconnectEpoch: realtimeHub.reconnectEpoch) { silent in
            load(silent: silent)
        }
    }

    private func load(silent: Bool = false) {
        if !silent && vehicle == nil { loading = true }
        error = nil
        Task {
            do {
                let resp = try await WarehouseService.vehicle(id: vehicleId)
                vehicle = resp.vehicle
                selectedReason = resp.vehicle.unavailableReason?.isEmpty == false
                    ? resp.vehicle.unavailableReason!
                    : "MANUAL_HOLD"
                customNote = resp.vehicle.unavailableNote ?? ""
            } catch {
                if !silent { self.error = error.localizedDescription }
            }
            if !silent { loading = false }
        }
    }

    private func updateAvailability(isActive: Bool, reason: String? = nil, note: String? = nil) {
        mutating = true
        error = nil
        Task {
            do {
                _ = try await WarehouseService.updateVehicleAvailability(
                    vehicleId: vehicleId,
                    isActive: isActive,
                    unavailableReason: reason,
                    unavailableNote: note
                )
                load()
            } catch {
                self.error = error.localizedDescription
            }
            mutating = false
        }
    }
}

struct VehicleAvailabilityPanel: View {
    let vehicle: Vehicle
    let mutating: Bool
    @Binding var selectedReason: String
    @Binding var customNote: String
    let onMarkUnavailable: (String, String?) -> Void
    let onRestore: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingMD) {
            HStack {
                VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                    Text("warehouse_portal.vehicle_availability_panel.text.availability")
                        .font(.headline)
                    Text("mobile_warehouse.ui.dispatch_excludes_unavailable_trucks_immediately")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                Spacer()
                Text(vehicle.isActive ? "Active" : "Unavailable")
                    .font(.caption.bold())
                    .padding(.horizontal, LabTheme.spacingSM)
                    .padding(.vertical, LabTheme.spacingXS)
                    .background(.quaternary, in: Capsule())
            }

            if !vehicle.isActive {
                Text(formatUnavailableReason(vehicle.unavailableReason, note: vehicle.unavailableNote))
                    .font(.caption)
                    .foregroundStyle(.orange)
            }

            if vehicle.isActive {
                Text("warehouse_portal.vehicle_availability_panel.text.unavailable_reason")
                    .font(.caption.bold())
                    .foregroundStyle(.secondary)
                Picker("Reason", selection: $selectedReason) {
                    ForEach(VehicleUnavailableReasonOption.allCases) { reason in
                        Text(reason.title).tag(reason.rawValue)
                    }
                }
                .pickerStyle(.menu)

                if selectedReason == VehicleUnavailableReasonOption.other.rawValue {
                    TextField("warehouse_portal.dispatch.text.custom_reason", text: $customNote)
                        .textFieldStyle(.roundedBorder)
                }

                Button("mobile_warehouse.ui.mark_unavailable") {
                    let note = selectedReason == VehicleUnavailableReasonOption.other.rawValue
                        ? customNote.trimmingCharacters(in: .whitespacesAndNewlines)
                        : nil
                    onMarkUnavailable(selectedReason, note?.isEmpty == false ? note : nil)
                }
                .buttonStyle(.borderedProminent)
                .disabled(
                    mutating
                        || (selectedReason == VehicleUnavailableReasonOption.other.rawValue
                            && customNote.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                )
            } else {
                Button("mobile_warehouse.ui.restore_truck") {
                    onRestore()
                }
                .buttonStyle(.bordered)
                .disabled(mutating)
            }
        }
        .padding(LabTheme.spacingMD)
        .background(.quaternary.opacity(0.35), in: RoundedRectangle(cornerRadius: 12))
    }
}

struct FleetTruckDispatchCard: View {
    let vehicle: Vehicle
    @Binding var selectedReason: String
    @Binding var customNote: String
    let mutating: Bool
    let onMarkUnavailable: () -> Void
    let onRestore: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
            HStack(alignment: .top) {
                VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                    Text(vehicle.label.isEmpty ? vehicle.licensePlate : vehicle.label)
                        .font(.subheadline.bold())
                    Text(L10n.format("mobile_warehouse.ui.licenseplate_vehicleclass", "\(vehicle.licensePlate)", "\(vehicle.vehicleClass)"))
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    if !vehicle.isActive {
                        Text(formatUnavailableReason(vehicle.unavailableReason, note: vehicle.unavailableNote))
                            .font(.caption)
                            .foregroundStyle(.orange)
                    }
                }
                Spacer()
                Text(vehicle.isActive ? "Active" : "Unavailable")
                    .font(.caption2.bold())
                    .padding(.horizontal, LabTheme.spacingSM)
                    .padding(.vertical, LabTheme.spacingXS)
                    .background(.quaternary, in: Capsule())
            }

            if vehicle.isActive {
                Picker("Reason", selection: $selectedReason) {
                    ForEach(VehicleUnavailableReasonOption.allCases) { reason in
                        Text(reason.title).tag(reason.rawValue)
                    }
                }
                .pickerStyle(.menu)

                if selectedReason == VehicleUnavailableReasonOption.other.rawValue {
                    TextField("warehouse_portal.dispatch.text.custom_reason", text: $customNote)
                        .textFieldStyle(.roundedBorder)
                }

                Button("mobile_warehouse.ui.mark_unavailable", action: onMarkUnavailable)
                    .buttonStyle(.bordered)
                    .disabled(
                        mutating
                            || (selectedReason == VehicleUnavailableReasonOption.other.rawValue
                                && customNote.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                    )
            } else {
                Button("mobile_warehouse.ui.restore", action: onRestore)
                    .buttonStyle(.bordered)
                    .disabled(mutating)
            }
        }
        .padding(LabTheme.spacingMD)
        .background(.quaternary.opacity(0.25), in: RoundedRectangle(cornerRadius: 12))
    }
}
