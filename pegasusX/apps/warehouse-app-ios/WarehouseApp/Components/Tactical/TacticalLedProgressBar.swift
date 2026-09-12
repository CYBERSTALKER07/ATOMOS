import SwiftUI

struct TacticalLedProgressBar: View {
    let title: String
    let current: Int
    let total: Int
    var tint: Color = TacticalTheme.accentCobalt
    var blockCount: Int = 12

    private var activeBlocks: Int {
        guard total > 0 else { return 0 }
        let ratio = Double(current) / Double(total)
        return min(max(Int(round(ratio * Double(blockCount))), 0), blockCount)
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack {
                Text(title)
                    .tacticalMicroLabel()
                Spacer()
                Text("\(current)/\(total)")
                    .font(.system(size: 11, weight: .bold, design: .monospaced))
                    .foregroundStyle(TacticalTheme.textPrimary)
            }

            HStack(spacing: 3) {
                ForEach(0..<blockCount, id: \.self) { index in
                    let isActive = index < activeBlocks
                    RoundedRectangle(cornerRadius: 2, style: .continuous)
                        .fill(isActive ? tint : TacticalTheme.surfaceSunken)
                        .frame(height: 8)
                        .overlay(
                            RoundedRectangle(cornerRadius: 2, style: .continuous)
                                .stroke(isActive ? tint.opacity(0.8) : TacticalTheme.border, lineWidth: 0.5)
                        )
                }
            }
        }
    }
}
