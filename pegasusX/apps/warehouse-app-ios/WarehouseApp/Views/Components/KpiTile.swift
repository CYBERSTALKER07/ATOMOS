import SwiftUI

struct KpiTile: View {
    let title: String
    let value: String
    let systemImage: String
    let tint: Color
    var chip: (text: String, tint: Color)? = nil

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
            HStack {
                Image(systemName: systemImage)
                    .foregroundStyle(tint)
                Spacer()
                if let chip {
                    Text(chip.text)
                        .font(.caption2.bold())
                        .padding(.horizontal, LabTheme.spacingSM)
                        .padding(.vertical, LabTheme.spacingXS)
                        .foregroundStyle(chip.tint)
                        .background(chip.tint.opacity(0.14), in: Capsule())
                }
            }
            Text(value)
                .font(.title2.bold())
                .minimumScaleFactor(0.8)
                .lineLimit(1)
            Text(title)
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .labCard()
    }
}
