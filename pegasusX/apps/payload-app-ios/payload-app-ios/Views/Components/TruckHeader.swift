import SwiftUI

struct TruckHeader: View {
    let truck: Truck

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(truck.label?.uppercased() ?? truck.licensePlate?.uppercased() ?? "VEHICLE-\(truck.id.prefix(8).uppercased())")
                .font(.system(.title, design: .monospaced, weight: .black))
                .foregroundStyle(TermTheme.accent)
                .tracking(1.4)

            HStack(spacing: 8) {
                if let p = truck.licensePlate, !p.isEmpty {
                    Text(p.uppercased())
                        .font(.system(.subheadline, design: .monospaced, weight: .bold))
                        .foregroundStyle(TermTheme.secondary)
                }

                if let c = truck.vehicleClass, !c.isEmpty {
                    Text("—")
                        .foregroundStyle(TermTheme.tertiary)
                    Text(c.uppercased())
                        .font(.system(.subheadline, design: .monospaced, weight: .bold))
                        .foregroundStyle(TermTheme.secondary)
                }

                Spacer()

                HStack(spacing: 6) {
                    Circle()
                        .fill(TermTheme.live)
                        .frame(width: 8, height: 8)
                    Text("ACTIVE_NODE")
                        .font(.system(.caption2, design: .monospaced, weight: .black))
                        .foregroundStyle(TermTheme.live)
                }
                .padding(.horizontal, 10)
                .padding(.vertical, 4)
                .background(TermTheme.live.opacity(0.12), in: Capsule())
            }
        }
        .accessibilityElement(children: .combine)
        .accessibilityLabel("Assigned truck: \(truck.label ?? truck.licensePlate ?? truck.id), class \(truck.vehicleClass ?? "standard")")
        .padding(TermTheme.s24)
        .tacticalCard(radius: TermTheme.radiusMD)
    }
}
