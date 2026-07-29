import SwiftUI

struct QuickActions: View {
    let onOfflineVerifier: () -> Void
    let onEndSession: () -> Void
    
    var body: some View {
        VStack(spacing: 10) {
            ActionRow(icon: "shield.checkered", title: "Offline Verifier", subtitle: "Hash manifest protocol") {
                onOfflineVerifier()
            }
            ActionRow(icon: "arrow.triangle.2.circlepath", title: "Sync Queue", subtitle: "Upload pending deliveries") {
                Haptics.light()
            }
            ActionRow(icon: "gearshape.fill", title: "Settings", subtitle: "App configuration") {
                Haptics.light()
            }
            ActionRow(icon: "rectangle.portrait.and.arrow.right", title: "End Session", subtitle: "Go offline and sign out", destructive: true) {
                Haptics.medium()
                onEndSession()
            }
        }
    }
}

struct ActionRow: View {
    let icon: String
    let title: String
    let subtitle: String
    var destructive: Bool = false
    let action: () -> Void
    
    var body: some View {
        Button(action: action) {
            HStack(spacing: 14) {
                Image(systemName: icon)
                    .font(.system(size: 15, weight: .semibold))
                    .foregroundStyle(destructive ? LabTheme.destructive : LabTheme.fg)
                    .frame(width: 36, height: 36)
                    .background((destructive ? LabTheme.destructive : LabTheme.fg).opacity(0.06), in: .rect(cornerRadius: 10))

                VStack(alignment: .leading, spacing: 2) {
                    Text(title)
                        .font(.system(size: 15, weight: .semibold))
                        .foregroundStyle(destructive ? LabTheme.destructive : LabTheme.fg)
                    Text(subtitle)
                        .font(.system(size: 12, weight: .medium))
                        .foregroundStyle(LabTheme.fgSecondary)
                }

                Spacer()

                Image(systemName: "chevron.right")
                    .font(.system(size: 11, weight: .bold))
                    .foregroundStyle(LabTheme.fgTertiary)
            }
            .padding(LabTheme.s16)
            .labCard()
        }
        .buttonStyle(.pressable)
    }
}
