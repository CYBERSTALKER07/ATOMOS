import SwiftUI

struct RemainingStopsStepper: View {
    let stops: [RemainingStop]
    var onSelect: ((String) -> Void)?

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("REMAINING STOPS")
                .font(.system(.caption2, design: .monospaced, weight: .heavy))
                .foregroundStyle(LabTheme.fgTertiary)
            if stops.isEmpty {
                Text("No remaining stops")
                    .font(.system(.footnote, weight: .medium))
                    .foregroundStyle(LabTheme.fgTertiary)
            } else {
                ForEach(Array(stops.enumerated()), id: \.element.id) { index, stop in
                    Button {
                        Haptics.selection()
                        onSelect?(stop.id)
                    } label: {
                        HStack(alignment: .top, spacing: 10) {
                            VStack(spacing: 0) {
                                Circle()
                                    .fill(dotColor(stop))
                                    .frame(width: 10, height: 10)
                                if index < stops.count - 1 {
                                    Rectangle()
                                        .fill(LabTheme.separator.opacity(0.35))
                                        .frame(width: 2, height: 22)
                                }
                            }
                            VStack(alignment: .leading, spacing: 2) {
                                Text(stop.title.isEmpty ? stop.id : stop.title)
                                    .font(.system(.subheadline, weight: .semibold))
                                    .foregroundStyle(LabTheme.fg)
                                    .multilineTextAlignment(.leading)
                                Text(stop.state.label)
                                    .font(.system(.caption2, design: .monospaced, weight: .bold))
                                    .foregroundStyle(stop.firstClass ? LabTheme.destructive : LabTheme.fgSecondary)
                            }
                            Spacer(minLength: 0)
                        }
                        .frame(minHeight: 44)
                        .contentShape(Rectangle())
                    }
                    .buttonStyle(.plain)
                    .accessibilityElement(children: .combine)
                    .accessibilityLabel("\(stop.title), \(stop.state.label)")
                }
            }
        }
        .padding(LabTheme.s16)
        .labCard()
    }

    private func dotColor(_ stop: RemainingStop) -> Color {
        if stop.state == .FISCAL_FAILED { return LabTheme.destructive }
        if stop.state == .ARRIVED_SHOP_CLOSED { return LabTheme.warning }
        return LabTheme.fgSecondary
    }
}
