import SwiftUI

struct MapButton: View {
    var pendingCount: Int
    let onOpenMap: () -> Void

    var body: some View {
        Button {
            Haptics.medium()
            onOpenMap()
        } label: {
            HStack(spacing: 14) {
                ZStack {
                    RoundedRectangle(cornerRadius: 14, style: .continuous)
                        .fill(LabTheme.fg)
                        .frame(width: 48, height: 48)

                    Image(systemName: "map.fill")
                        .font(.system(.title3, weight: .semibold))
                        .foregroundStyle(LabTheme.buttonFg)
                }

                VStack(alignment: .leading, spacing: 3) {
                    Text("mobile_driver.ui.open_map")
                        .font(.system(.callout, weight: .bold))
                        .foregroundStyle(LabTheme.fg)

                    Text(L10n.format("mobile_driver.ui.pendingcount_deliveries_waiting_3", "\(pendingCount)"))
                        .font(.system(.caption, weight: .medium))
                        .foregroundStyle(LabTheme.fgSecondary)
                }

                Spacer()

                Image(systemName: "arrow.right")
                    .font(.system(.footnote, weight: .bold))
                    .foregroundStyle(LabTheme.fgTertiary)
            }
            .padding(LabTheme.s16)
            .labCard()
        }
        .buttonStyle(.pressable)
    }
}
