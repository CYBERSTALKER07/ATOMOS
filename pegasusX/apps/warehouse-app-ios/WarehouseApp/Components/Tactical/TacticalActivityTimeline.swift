import SwiftUI

struct TacticalTimelineItem: Identifiable {
    let id: String
    let timestamp: String
    let title: String
    let description: String
    let actor: String
    var icon: String = "circle.fill"
    var tint: Color = TacticalTheme.accentCobalt
}

struct TacticalActivityTimeline: View {
    let items: [TacticalTimelineItem]

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            ForEach(Array(items.enumerated()), id: \.element.id) { index, item in
                HStack(alignment: .top, spacing: 12) {
                    // Timeline indicator column
                    VStack(spacing: 0) {
                        Circle()
                            .fill(item.tint)
                            .frame(width: 10, height: 10)
                            .overlay(
                                Circle()
                                    .stroke(item.tint.opacity(0.3), lineWidth: 4)
                            )
                            .padding(.top, 4)

                        if index < items.count - 1 {
                            Rectangle()
                                .fill(TacticalTheme.border)
                                .frame(width: 1.5)
                                .frame(maxHeight: .infinity)
                                .padding(.vertical, 2)
                        }
                    }
                    .frame(width: 16)

                    // Event Content
                    VStack(alignment: .leading, spacing: 4) {
                        HStack {
                            Text(item.title)
                                .font(.system(size: 13, weight: .bold))
                                .foregroundStyle(TacticalTheme.textPrimary)

                            Spacer()

                            Text(item.timestamp)
                                .font(.system(size: 11, weight: .semibold, design: .monospaced))
                                .foregroundStyle(TacticalTheme.textTertiary)
                        }

                        Text(item.description)
                            .font(.system(size: 12, weight: .regular))
                            .foregroundStyle(TacticalTheme.textSecondary)

                        HStack(spacing: 6) {
                            Text("ACTOR:")
                                .tacticalMicroLabel()
                            Text(item.actor)
                                .font(.system(size: 11, weight: .semibold, design: .monospaced))
                                .foregroundStyle(TacticalTheme.accentCobalt)
                                .padding(.horizontal, 6)
                                .padding(.vertical, 2)
                                .background(TacticalTheme.accentCobalt.opacity(0.12))
                                .clipShape(RoundedRectangle(cornerRadius: 4, style: .continuous))
                        }
                        .padding(.top, 2)
                    }
                    .padding(.bottom, 16)
                }
            }
        }
    }
}
