import SwiftUI

struct DriversList: View {
    let drivers: [Driver]
    let vehicles: [Vehicle]
    let updatingDriverId: String?
    let onAssign: (String, String?) -> Void

    var body: some View {
        ResponsiveGridContentWrapper {
            ForEach(drivers) { driver in
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
                                onAssign(driver.driverId, nil)
                            }
                            ForEach(assignableVehicles(for: driver)) { vehicle in
                                Button(vehicleLabel(for: vehicle)) {
                                    onAssign(driver.driverId, vehicle.vehicleId)
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
