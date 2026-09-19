import SwiftUI

struct TacticalKpiCard: View {
    let title: String
    let value: String
    let systemImage: String
    var tint: Color = TacticalTheme.accentCobalt
    var delta: String? = nil
    var deltaIsPositive: Bool? = nil
    var subtitle: String? = nil
    var alertChip: String? = nil

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            // Top Row: Micro-label and Icon Badge
            HStack(alignment: .center) {
                Text(title)
                    .tacticalMicroLabel()
                    .lineLimit(1)

                Spacer()

                if let alertChip {
                    Text(alertChip)
                        .font(.system(size: 9, weight: .bold))
                        .padding(.horizontal, 6)
                        .padding(.vertical, 2)
                        .foregroundStyle(TacticalTheme.statusDanger)
                        .background(TacticalTheme.statusDanger.opacity(0.15))
                        .clipShape(Capsule())
                        .overlay(
                            Capsule().stroke(TacticalTheme.statusDanger.opacity(0.4), lineWidth: 1)
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

            // Main Metric: Tabular Monospace
            Text(value)
                .tacticalMonospaceValue(size: 28)
                .lineLimit(1)
                .minimumScaleFactor(0.7)
                .contentTransition(.numericText(value: 0))

            // Bottom Row: Delta & Subtitle
            HStack(spacing: 6) {
                if let delta, let deltaIsPositive {
                    HStack(spacing: 2) {
                        Image(systemName: deltaIsPositive ? "arrow.up.right" : "arrow.down.right")
                            .font(.system(size: 9, weight: .bold))
                        Text(delta)
                            .font(.system(size: 11, weight: .bold, design: .monospaced))
                    }
                    .foregroundStyle(deltaIsPositive ? TacticalTheme.accentLime : TacticalTheme.statusDanger)
                    .padding(.horizontal, 6)
                    .padding(.vertical, 2)
                    .background((deltaIsPositive ? TacticalTheme.accentLime : TacticalTheme.statusDanger).opacity(0.12))
                    .clipShape(RoundedRectangle(cornerRadius: 4, style: .continuous))
                }

                if let subtitle {
                    Text(subtitle)
                        .font(.system(size: 11, weight: .medium))
                        .foregroundStyle(TacticalTheme.textTertiary)
                        .lineLimit(1)
                }
            }
        }
        .tacticalCard(padding: 14)
    }
}
