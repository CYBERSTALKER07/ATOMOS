import SwiftUI

struct VehicleInfoCard: View {
    var vm: FleetViewModel

    var body: some View {
        HStack(spacing: 14) {
            Image(systemName: "truck.box.fill")
                .font(.system(size: 22))
                .foregroundStyle(LabTheme.fg)
                .frame(width: 44, height: 44)
                .background(LabTheme.separator)
                .clipShape(.rect(cornerRadius: 12))

            VStack(alignment: .leading, spacing: 4) {
                Text(vm.truckId)
                    .font(.system(size: 15, weight: .bold))
                    .foregroundStyle(LabTheme.fg)
                HStack(spacing: 6) {
                    Text(vm.licensePlate)
                        .font(.system(size: 12, weight: .medium, design: .monospaced))
                        .foregroundStyle(LabTheme.fgTertiary)
                    if vm.vehicleClass != "—" {
                        Text("•")
                            .foregroundStyle(LabTheme.fgTertiary)
                        Text(L10n.format("mobile_driver.ui.vehicleclass_maxvolumevu_vu", "\(vm.vehicleClass)", "\(Int(vm.maxVolumeVU))"))
                            .font(.system(size: 12, weight: .medium, design: .monospaced))
                            .foregroundStyle(LabTheme.fgTertiary)
                    }
                }
            }

            Spacer()

            DriverStatusBadge(text: "ASSIGNED", tint: LabTheme.success)
        }
        .padding(LabTheme.s16)
        .labCard()
    }
}
