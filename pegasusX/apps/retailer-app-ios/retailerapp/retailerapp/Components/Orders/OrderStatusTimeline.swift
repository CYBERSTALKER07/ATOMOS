import SwiftUI

struct OrderStatusTimeline: View {
    let currentStep: Int

    private let steps = OrderStatus.timelineSteps

    var body: some View {
        HStack(spacing: 0) {
            ForEach(Array(steps.enumerated()), id: \.offset) { index, label in
                let isCompleted = index < currentStep
                let isActive = index == currentStep

                VStack(spacing: 4) {
                    Circle()
                        .fill(dotColor(isCompleted: isCompleted, isActive: isActive))
                        .frame(width: isActive ? 10 : 8, height: isActive ? 10 : 8)

                    Text(label)
                        .font(.system(size: 8, weight: isActive ? .bold : .medium, design: .rounded))
                        .foregroundStyle(labelColor(isCompleted: isCompleted, isActive: isActive))
                        .lineLimit(1)
                }
                .frame(maxWidth: .infinity)
            }
        }
        .padding(.horizontal, 10)
        .padding(.vertical, 10)
        .background(AppTheme.surfaceElevated.opacity(0.5))
        .clipShape(.rect(cornerRadius: AppTheme.radiusSM))
    }

    private func dotColor(isCompleted: Bool, isActive: Bool) -> Color {
        if isCompleted { return AppTheme.success }
        if isActive { return .teal }
        return AppTheme.textTertiary.opacity(0.4)
    }

    private func labelColor(isCompleted: Bool, isActive: Bool) -> Color {
        if isCompleted { return AppTheme.textSecondary }
        if isActive { return .teal }
        return AppTheme.textTertiary.opacity(0.5)
    }
}
