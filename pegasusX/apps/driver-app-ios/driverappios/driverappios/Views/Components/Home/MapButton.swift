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
                        .font(.system(size: 18, weight: .semibold))
                        .foregroundStyle(LabTheme.buttonFg)
                }

                VStack(alignment: .leading, spacing: 3) {
                    Text("Open Map")
                        .font(.system(size: 16, weight: .bold))
                        .foregroundStyle(LabTheme.fg)

                    Text("\(pendingCount) deliveries waiting")
                        .font(.system(size: 12, weight: .medium))
                        .foregroundStyle(LabTheme.fgSecondary)
                }

                Spacer()

                Image(systemName: "arrow.right")
                    .font(.system(size: 13, weight: .bold))
                    .foregroundStyle(LabTheme.fgTertiary)
            }
            .padding(LabTheme.s16)
            .labCard()
        }
        .buttonStyle(.pressable)
    }
}
