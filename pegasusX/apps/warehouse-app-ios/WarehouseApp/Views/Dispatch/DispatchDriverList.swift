import SwiftUI

/// Reusable drivers segment for the Dispatch screen.
///
/// Renders available and unavailable driver cards with status badges,
/// VU capacity, and unavailability reasons.
struct DispatchDriverList: View {
    let availableDrivers: [AvailableDriver]
    let unavailableDrivers: [AvailableDriver]

    var body: some View {
        if availableDrivers.isEmpty && unavailableDrivers.isEmpty {
            ContentUnavailableView("No Drivers", systemImage: "person.badge.key", description: Text("No available drivers"))
        } else {
            ResponsiveGridContentWrapper {
                if !availableDrivers.isEmpty {
                    Section("Available") {
                        ForEach(availableDrivers) { driver in
                            driverRow(driver, unavailableDetail: false)
                        }
                    }
                }
                if !unavailableDrivers.isEmpty {
                    Section("Vehicle Unavailable") {
                        ForEach(unavailableDrivers) { driver in
                            driverRow(driver, unavailableDetail: true)
                        }
                    }
                }
            }
        }
    }

    @ViewBuilder
    private func driverRow(_ driver: AvailableDriver, unavailableDetail: Bool) -> some View {
        HStack(alignment: unavailableDetail ? .top : .center) {
            VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                Text(driver.name)
                    .font(.headline)
                Text(driverSubtitle(driver, unavailableDetail: unavailableDetail))
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
                if unavailableDetail, let reason = driver.unavailableReason, !reason.isEmpty {
                    Text(vehicleUnavailableReasonLabel(reason))
                        .font(.caption)
                        .foregroundStyle(.orange)
                }
            }
            Spacer()
            VStack(alignment: .trailing, spacing: LabTheme.spacingXS) {
                WarehouseStatusBadge(text: driver.truckStatus.isEmpty ? "IDLE" : driver.truckStatus)
                if driver.maxVolumeVu > 0 {
                    Text("\(driver.maxVolumeVu, specifier: "%.0f") VU")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }
            }
        }
    }

    private func driverSubtitle(_ driver: AvailableDriver, unavailableDetail: Bool) -> String {
        let vehicle = driver.vehicleLabel ?? ""
        let phone = driver.phone ?? ""
        let label = vehicle.isEmpty ? (phone.isEmpty ? (unavailableDetail ? "Assigned vehicle unavailable" : "No vehicle") : phone) : vehicle
        if let free = driver.freeVolumeVu, free > 0 {
            return "\(label) · \(String(format: "%.1f", free)) VU free"
        }
        return label
    }
}
