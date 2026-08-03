import SwiftUI

struct QuickActionsSection: View {
    @Binding var showRescueSheet: Bool

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            DriverSectionHeader(title: "Quick Actions")
                .padding(.horizontal, LabTheme.s4)

            HStack(spacing: 12) {
                actionTile(icon: "qrcode.viewfinder", label: "Scan QR")
                actionTile(icon: "shield.checkered", label: "Offline\nVerify")
                actionTileButton(icon: "exclamationmark.triangle.fill", label: "Rescue", tint: LabTheme.warning) {
                    showRescueSheet = true
                }
            }
        }
    }

    private func actionTile(icon: String, label: String) -> some View {
        VStack(spacing: 8) {
            Image(systemName: icon)
                .font(.system(size: 18, weight: .medium))
                .foregroundStyle(LabTheme.fg)

            Text(label)
                .font(.system(size: 10, weight: .semibold))
                .foregroundStyle(LabTheme.fgSecondary)
                .multilineTextAlignment(.center)
                .lineLimit(2)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 16)
        .labCard()
    }

    private func actionTileButton(icon: String, label: String, tint: Color = LabTheme.fg, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            VStack(spacing: 8) {
                Image(systemName: icon)
                    .font(.system(size: 18, weight: .medium))
                    .foregroundStyle(tint)

                Text(label)
                    .font(.system(size: 10, weight: .semibold))
                    .foregroundStyle(LabTheme.fgSecondary)
                    .multilineTextAlignment(.center)
                    .lineLimit(2)
            }
            .frame(maxWidth: .infinity)
            .padding(.vertical, 16)
            .labCard()
        }
        .buttonStyle(.pressable)
    }
}
