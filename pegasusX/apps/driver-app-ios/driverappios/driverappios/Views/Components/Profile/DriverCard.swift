import SwiftUI

struct DriverCard: View {
    var vm: FleetViewModel
    
    var body: some View {
        VStack(spacing: 16) {
            HStack(spacing: 14) {
                // Avatar
                ZStack {
                    Circle()
                        .fill(LabTheme.fg)
                        .frame(width: 52, height: 52)

                    Text(String(vm.driverName.prefix(1)))
                        .font(.system(size: 22, weight: .bold))
                        .foregroundStyle(LabTheme.buttonFg)
                }

                VStack(alignment: .leading, spacing: 4) {
                    Text(vm.driverName)
                        .font(.system(size: 17, weight: .bold))
                        .foregroundStyle(LabTheme.fg)

                    Text(vm.driverId)
                        .font(.system(size: 12, weight: .semibold, design: .monospaced))
                        .foregroundStyle(LabTheme.fgSecondary)
                }

                Spacer()

                StatusPill(
                    label: vm.hasActiveRoute ? "ON DUTY" : "IDLE",
                    color: vm.hasActiveRoute ? LabTheme.success : LabTheme.fgSecondary
                )
            }

            // Info grid
            HStack(spacing: 12) {
                InfoTile(label: "Truck", value: vm.truckId, icon: "truck.box.fill")
                InfoTile(label: "Plate", value: vm.licensePlate, icon: "car.fill")
                InfoTile(label: "Capacity", value: "\(Int(vm.maxVolumeVU)) VU", icon: "shippingbox.fill")
                InfoTile(label: "Done", value: "\(vm.completedIds.count)", icon: "checkmark.circle.fill")
            }
        }
        .padding(LabTheme.s20)
        .labCard()
    }
}

struct InfoTile: View {
    let label: String
    let value: String
    let icon: String
    
    var body: some View {
        VStack(spacing: 6) {
            Image(systemName: icon)
                .font(.system(size: 14))
                .foregroundStyle(LabTheme.fgSecondary)

            Text(value)
                .font(.system(size: 14, weight: .bold, design: .monospaced))
                .foregroundStyle(LabTheme.fg)

            Text(label)
                .font(.system(size: 10, weight: .medium))
                .foregroundStyle(LabTheme.fgTertiary)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 12)
        .background(LabTheme.fg.opacity(0.03), in: .rect(cornerRadius: 12))
    }
}
