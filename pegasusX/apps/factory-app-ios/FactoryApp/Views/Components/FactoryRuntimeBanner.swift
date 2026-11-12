import SwiftUI

struct FactoryRuntimeBanner: View {
    let tone: FactoryRuntimeTone
    let message: String

    var body: some View {
        let palette = paletteFor(tone)

        HStack(spacing: LabTheme.spacingSM) {
            Image(systemName: palette.icon)
                .font(.footnote.weight(.semibold))
            Text(message)
                .font(.footnote.weight(.medium))
        }
        .foregroundStyle(palette.tint)
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.horizontal, LabTheme.spacingMD)
        .padding(.vertical, LabTheme.spacingSM)
        .background(palette.fill, in: RoundedRectangle(cornerRadius: LabTheme.radiusMD, style: .continuous))
        .accessibilityElement(children: .combine)
        .accessibilityLabel(message)
    }

    private func paletteFor(_ tone: FactoryRuntimeTone) -> (icon: String, fill: Color, tint: Color) {
        switch tone {
        case .live:
            return ("arrow.triangle.2.circlepath", LabTheme.tertiaryBackground, LabTheme.secondaryLabel)
        case .refreshing:
            return ("arrow.triangle.2.circlepath", LabTheme.fill, LabTheme.label)
        case .warning:
            return ("exclamationmark.triangle", LabTheme.warning.opacity(0.16), LabTheme.warning)
        case .offline:
            return ("wifi.slash", LabTheme.tertiaryBackground, LabTheme.secondaryLabel)
        }
    }
}
