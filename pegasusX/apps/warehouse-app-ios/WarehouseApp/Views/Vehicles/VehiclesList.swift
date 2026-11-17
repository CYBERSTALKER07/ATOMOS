import SwiftUI

struct VehiclesList: View {
    let vehicles: [Vehicle]

    var body: some View {
        ResponsiveGridContentWrapper {
            ForEach(vehicles) { vehicle in
                NavigationLink {
                    VehicleDetailView(vehicleId: vehicle.vehicleId)
                } label: {
                    HStack {
                        VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                            Text(vehicle.label.isEmpty ? vehicle.licensePlate : vehicle.label)
                                .font(.headline)
                            Text(L10n.format("mobile_warehouse.ui.vehicleclass_capacityvu_vu", "\(vehicle.vehicleClass)", "\(vehicle.capacityVu)"))
                                .font(.subheadline)
                                .foregroundStyle(.secondary)
                            Text(vehicle.assignedDriverName ?? "Unassigned")
                                .font(.caption)
                                .foregroundStyle(.secondary)
                            if !vehicle.isActive {
                                Text(formatUnavailableReason(vehicle.unavailableReason, note: vehicle.unavailableNote))
                                    .font(.caption)
                                    .foregroundStyle(.orange)
                            }
                        }
                        Spacer()
                        Text(vehicle.isActive ? (vehicle.status.isEmpty ? "AVAILABLE" : vehicle.status) : "UNAVAILABLE")
                            .font(.caption.bold())
                            .padding(.horizontal, LabTheme.spacingSM)
                            .padding(.vertical, LabTheme.spacingXS)
                            .background(.quaternary, in: Capsule())
                    }
                    .frame(minHeight: 44)
                    .contentShape(Rectangle())
                    .accessibilityElement(children: .combine)
                    .accessibilityLabel("\(vehicle.label.isEmpty ? vehicle.licensePlate : vehicle.label), class \(vehicle.vehicleClass), \(vehicle.capacityVu) VU, \(vehicle.assignedDriverName ?? "Unassigned"), \(vehicle.isActive ? (vehicle.status.isEmpty ? "AVAILABLE" : vehicle.status) : "UNAVAILABLE")")
                }
            }
        }
    }
}
