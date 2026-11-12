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
                                .font(.system(.caption2, weight: .bold))
                                .foregroundStyle(TacticalTheme.statusSuccess)
                        } else {
                            Text("\(index + 1)")
                                .font(.system(.caption2, design: .monospaced, weight: .bold))
                                .foregroundStyle(isCurrent ? Color.white : TacticalTheme.textTertiary)
                        }

                        Text(stages[index])
                            .font(.system(.caption2, weight: isCurrent ? .bold : .medium))
                            .foregroundStyle(isCurrent ? Color.white : (isCompleted ? TacticalTheme.textSecondary : TacticalTheme.textTertiary))
                    }
                    .padding(.horizontal, 10)
                    .padding(.vertical, 6)
                    .frame(minHeight: 44)
                    .contentShape(Rectangle())
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
                    .accessibilityElement(children: .combine)
                    .accessibilityLabel("Step \(index + 1): \(stages[index]), \(isCurrent ? "current stage" : (isCompleted ? "completed" : "pending"))")

                    if index < stages.count - 1 {
                        Image(systemName: "chevron.right")
                            .font(.system(.caption2, weight: .semibold))
                            .foregroundStyle(TacticalTheme.textTertiary)
                            .accessibilityHidden(true)
                    }
                }
            }
            .padding(.vertical, 2)
        }
    }
}
