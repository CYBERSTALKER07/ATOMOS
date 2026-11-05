import SwiftUI

struct QuickActionsSection: View {
    @Binding var showRescueSheet: Bool
    var onScanQR: (() -> Void)? = nil
    var onOfflineVerify: (() -> Void)? = nil

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            DriverSectionHeader(title: "Quick Actions")
                .padding(.horizontal, LabTheme.s4)

            HStack(spacing: 12) {
                actionTileButton(icon: "qrcode.viewfinder", label: "Scan QR") {
                    Haptics.light()
                    onScanQR?()
                }
                actionTileButton(icon: "shield.checkered", label: "Offline\nVerify") {
                    Haptics.light()
                    onOfflineVerify?()
                }
                actionTileButton(icon: "exclamationmark.triangle.fill", label: "Rescue", tint: LabTheme.warning) {
                    Haptics.light()
                    showRescueSheet = true
                }
            }
        }
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
            .frame(maxWidth: .infinity, minHeight: 72)
            .padding(.vertical, 16)
            .labCard()
        }
        .buttonStyle(.pressable)
    }
}
