import SwiftUI

struct KpiTile: View {
    let label: String
    let value: String
    var icon: String? = nil
    var tint: Color = LabTheme.fg

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.s8) {
            if let icon {
                Image(systemName: icon)
                    .font(.system(.caption, weight: .semibold))
                    .foregroundStyle(LabTheme.fgTertiary)
            }
            Text(label.uppercased())
                .font(.system(.caption2, design: .monospaced, weight: .black))
                .foregroundStyle(LabTheme.fgTertiary)
            Text(value)
                .font(.system(value.count > 10 ? .subheadline : .headline, design: .monospaced, weight: .bold))
                .foregroundStyle(tint)
                .minimumScaleFactor(0.7)
                .lineLimit(1)
        }
        .padding(LabTheme.s16)
        .frame(maxWidth: .infinity, minHeight: 44, alignment: .leading)
        .labCard()
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(label): \(value)")
    }
}
