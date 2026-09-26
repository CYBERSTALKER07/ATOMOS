import SwiftUI

struct KpiTile: View {
    let title: String
    let value: String
    let systemImage: String
    let tint: Color
    var chip: (text: String, tint: Color)? = nil

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(alignment: .center) {
                Text(title)
                    .tacticalMicroLabel()
                    .lineLimit(1)

                Spacer()

                if let chip {
                    Text(chip.text)
                        .font(.system(size: 9, weight: .bold))
                        .padding(.horizontal, 6)
                        .padding(.vertical, 2)
                        .foregroundStyle(chip.tint)
                        .background(chip.tint.opacity(0.15))
                        .clipShape(Capsule())
                        .overlay(
                            Capsule().stroke(chip.tint.opacity(0.4), lineWidth: 1)
                        )
                }

                Image(systemName: systemImage)
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundStyle(tint)
                    .frame(width: 28, height: 28)
                    .background(TacticalTheme.surfaceSunken)
                    .clipShape(RoundedRectangle(cornerRadius: 6, style: .continuous))
                    .overlay(
                        RoundedRectangle(cornerRadius: 6, style: .continuous)
                            .stroke(TacticalTheme.border, lineWidth: 0.5)
                    )
            }

            Text(value)
                .tacticalMonospaceValue(size: 24)
                .lineLimit(1)
                .minimumScaleFactor(0.75)
                .contentTransition(.numericText(value: 0))
                .animation(.snappy, value: value)
        }
        .tacticalCard(padding: 14)
    }
}
