import SwiftUI

struct TacticalGaugeCard: View {
    let title: String
    let percentage: Double // 0.0 to 1.0
    var subtitle: String? = nil
    var tint: Color = TacticalTheme.accentCobalt

    private var percentageClamped: Double {
        min(max(percentage, 0.0), 1.0)
    }

    var body: some View {
        VStack(spacing: 8) {
            HStack {
                Text(title)
                    .tacticalMicroLabel()
                Spacer()
            }

            ZStack {
                // Background Track Arc
                Circle()
                    .trim(from: 0.15, to: 0.85)
                    .stroke(
                        TacticalTheme.surfaceSunken,
                        style: StrokeStyle(lineWidth: 10, lineCap: .round)
                    )
                    .rotationEffect(.degrees(90))

                // Progress Arc
                Circle()
                    .trim(from: 0.15, to: 0.15 + (percentageClamped * 0.70))
                    .stroke(
                        tint,
                        style: StrokeStyle(lineWidth: 10, lineCap: .round)
                    )
                    .rotationEffect(.degrees(90))
                    .animation(.spring(response: 0.6, dampingFraction: 0.8), value: percentageClamped)

                // Central Readout
                VStack(spacing: 2) {
                    Text(String(format: "%.0f%%", percentageClamped * 100))
                        .tacticalMonospaceValue(size: 22)

                    if let subtitle {
                        Text(subtitle)
                            .font(.system(size: 9, weight: .bold))
                            .textCase(.uppercase)
                            .foregroundStyle(TacticalTheme.textTertiary)
                    }
                }
                .offset(y: 4)
            }
            .frame(height: 100)
            .padding(.vertical, 4)
        }
        .tacticalCard(padding: 14)
    }
}
