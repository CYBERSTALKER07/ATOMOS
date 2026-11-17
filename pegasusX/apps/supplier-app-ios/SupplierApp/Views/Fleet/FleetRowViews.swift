import SwiftUI

struct DriverRow: View {
    let driver: FleetDriver

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            HStack {
                Text(driver.name)
                    .font(.headline)
                Spacer()
                SupplierStatusBadge(
                    text: driver.isActive ? "ACTIVE" : "OFFLINE",
                    tint: driver.isActive ? SupplierTheme.live : SupplierTheme.secondaryLabel
                )
            }
            Text(driver.phone)
                .font(.subheadline)
                .foregroundStyle(.secondary)
            Text(L10n.format("mobile_supplier.ui.homenodetype_homenodeid", "\(driver.homeNodeType)", "\(driver.homeNodeId)"))
                .font(.caption2)
                .foregroundStyle(.tertiary)
        }
        .padding(.vertical, 4)
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(driver.name), \(driver.isActive ? "active" : "offline"), \(driver.phone)")
    }
}

struct VehicleRow: View {
    let vehicle: FleetVehicle

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(vehicle.licensePlate)
                .font(.headline)
            if let label = vehicle.label, !label.isEmpty {
                Text(label)
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
            }
            Text(L10n.format("mobile_supplier.ui.homenodetype_homenodeid", "\(vehicle.homeNodeType)", "\(vehicle.homeNodeId)"))
                .font(.caption2)
                .foregroundStyle(.tertiary)
        }
        .padding(.vertical, 4)
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(vehicle.licensePlate), \(vehicle.label ?? "")")
    }
}
