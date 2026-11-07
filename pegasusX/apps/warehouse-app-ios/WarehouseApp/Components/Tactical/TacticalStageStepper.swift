import SwiftUI

struct TacticalStageStepper: View {
    let stages: [String]
    let currentStageIndex: Int

    var body: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 6) {
                ForEach(0..<stages.count, id: \.self) { index in
                    let isCurrent = index == currentStageIndex
                    let isCompleted = index < currentStageIndex

                    HStack(spacing: 5) {
                        if isCompleted {
                            Image(systemName: "checkmark")
                                .font(.system(size: 9, weight: .bold))
                                .foregroundStyle(TacticalTheme.statusSuccess)
                        } else {
                            Text("\(index + 1)")
                                .font(.system(size: 10, weight: .bold, design: .monospaced))
                                .foregroundStyle(isCurrent ? Color.white : TacticalTheme.textTertiary)
                        }

                        Text(stages[index])
                            .font(.system(size: 11, weight: isCurrent ? .bold : .medium))
                            .foregroundStyle(isCurrent ? Color.white : (isCompleted ? TacticalTheme.textSecondary : TacticalTheme.textTertiary))
                    }
                    .padding(.horizontal, 10)
                    .padding(.vertical, 6)
                    .background(
                        isCurrent
                            ? TacticalTheme.accentCobalt
                            : (isCompleted ? TacticalTheme.surfaceSubtle : TacticalTheme.surfaceSunken)
                    )
                    .clipShape(RoundedRectangle(cornerRadius: 6, style: .continuous))
                    .overlay(
                        RoundedRectangle(cornerRadius: 6, style: .continuous)
                            .stroke(isCurrent ? TacticalTheme.accentCobalt : TacticalTheme.border, lineWidth: 1)
                    )

                    if index < stages.count - 1 {
                        Image(systemName: "chevron.right")
                            .font(.system(size: 8, weight: .semibold))
                            .foregroundStyle(TacticalTheme.textTertiary)
                    }
                }
            }
            .padding(.vertical, 2)
        }
    }
}
