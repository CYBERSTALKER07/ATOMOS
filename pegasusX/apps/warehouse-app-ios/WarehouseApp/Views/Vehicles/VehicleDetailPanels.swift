import SwiftUI

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
            .accessibilityElement(children: .combine)

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
                .frame(minHeight: 44)
                .onChange(of: selectedReason) { _, _ in
                    PegasusHaptics.selection()
                }

                if selectedReason == VehicleUnavailableReasonOption.other.rawValue {
                    TextField("warehouse_portal.dispatch.text.custom_reason", text: $customNote)
                        .textFieldStyle(.roundedBorder)
                        .frame(minHeight: 44)
                }

                Button("mobile_warehouse.ui.mark_unavailable") {
                    PegasusHaptics.warning()
                    let note = selectedReason == VehicleUnavailableReasonOption.other.rawValue
                        ? customNote.trimmingCharacters(in: .whitespacesAndNewlines)
                        : nil
                    onMarkUnavailable(selectedReason, note?.isEmpty == false ? note : nil)
                }
                .buttonStyle(.borderedProminent)
                .frame(minHeight: 44)
                .contentShape(Rectangle())
                .accessibilityLabel("Mark truck unavailable")
                .disabled(
                    mutating
                        || (selectedReason == VehicleUnavailableReasonOption.other.rawValue
                            && customNote.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                )
            } else {
                Button("mobile_warehouse.ui.restore_truck") {
                    PegasusHaptics.light()
                    onRestore()
                }
                .buttonStyle(.bordered)
                .frame(minHeight: 44)
                .contentShape(Rectangle())
                .accessibilityLabel("Restore truck to active fleet")
                .disabled(mutating)
            }
        }
        .padding(LabTheme.spacingMD)
        .background(.quaternary.opacity(0.35), in: RoundedRectangle(cornerRadius: 12, style: .continuous))
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
            .accessibilityElement(children: .combine)

            if vehicle.isActive {
                Picker("Reason", selection: $selectedReason) {
                    ForEach(VehicleUnavailableReasonOption.allCases) { reason in
                        Text(reason.title).tag(reason.rawValue)
                    }
                }
                .pickerStyle(.menu)
                .frame(minHeight: 44)
                .onChange(of: selectedReason) { _, _ in
                    PegasusHaptics.selection()
                }

                if selectedReason == VehicleUnavailableReasonOption.other.rawValue {
                    TextField("warehouse_portal.dispatch.text.custom_reason", text: $customNote)
                        .textFieldStyle(.roundedBorder)
                        .frame(minHeight: 44)
                }

                Button("mobile_warehouse.ui.mark_unavailable") {
                    PegasusHaptics.warning()
                    onMarkUnavailable()
                }
                .buttonStyle(.bordered)
                .frame(minHeight: 44)
                .contentShape(Rectangle())
                .accessibilityLabel("Mark \(vehicle.label.isEmpty ? vehicle.licensePlate : vehicle.label) unavailable")
                .disabled(
                    mutating
                        || (selectedReason == VehicleUnavailableReasonOption.other.rawValue
                            && customNote.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                )
            } else {
                Button("mobile_warehouse.ui.restore") {
                    PegasusHaptics.light()
                    onRestore()
                }
                .buttonStyle(.bordered)
                .frame(minHeight: 44)
                .contentShape(Rectangle())
                .accessibilityLabel("Restore \(vehicle.label.isEmpty ? vehicle.licensePlate : vehicle.label)")
                .disabled(mutating)
            }
        }
        .padding(LabTheme.spacingMD)
        .background(.quaternary.opacity(0.25), in: RoundedRectangle(cornerRadius: 12, style: .continuous))
    }
}
