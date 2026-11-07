import SwiftUI

struct RemainingStopsStepper: View {
    let stops: [RemainingStop]
    var onSelect: ((String) -> Void)?

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("REMAINING STOPS")
                .font(.system(size: 11, weight: .heavy, design: .monospaced))
                .foregroundStyle(LabTheme.fgTertiary)
            if stops.isEmpty {
                Text("No remaining stops")
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(LabTheme.fgTertiary)
            } else {
                ForEach(Array(stops.enumerated()), id: \.element.id) { index, stop in
                    Button {
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
                                    .font(.system(size: 14, weight: .semibold))
                                    .foregroundStyle(LabTheme.fg)
                                    .multilineTextAlignment(.leading)
                                Text(stop.state.label)
                                    .font(.system(size: 11, weight: .bold, design: .monospaced))
                                    .foregroundStyle(stop.firstClass ? LabTheme.destructive : LabTheme.fgSecondary)
                            }
                            Spacer(minLength: 0)
                        }
                    }
                    .buttonStyle(.plain)
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

struct FieldMoneyStrip: View {
    let counts: MoneyHealthCounts

    var body: some View {
        HStack(spacing: 8) {
            strip("CASH", counts.pendingCash)
            strip("FISCAL", counts.openFiscal)
            strip("CREDIT", counts.creditLeave)
        }
    }

    private func strip(_ label: String, _ count: Int) -> some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(label)
                .font(.system(size: 10, weight: .heavy, design: .monospaced))
                .foregroundStyle(LabTheme.fgTertiary)
            Text(count == 0 ? "empty" : "\(count)")
                .font(.system(size: 13, weight: .semibold))
                .foregroundStyle(count == 0 ? LabTheme.fgTertiary : LabTheme.fg)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(10)
        .background(LabTheme.card, in: RoundedRectangle(cornerRadius: LabTheme.buttonRadius, style: .continuous))
    }
}

struct ManifestBulletMeter: View {
    let state: String?
    let usedVU: Double?
    let maxVU: Double

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("MANIFEST")
                .font(.system(size: 11, weight: .heavy, design: .monospaced))
                .foregroundStyle(LabTheme.fgTertiary)
            Text((state?.isEmpty == false ? state! : "unavailable"))
                .font(.system(size: 13, weight: .bold, design: .monospaced))
                .foregroundStyle(LabTheme.fg)
            if let used = usedVU, maxVU > 0 {
                ProgressView(value: min(max(used / maxVU, 0), 1))
                Text(String(format: "%.0f / %.0f VU", used, maxVU))
                    .font(.system(size: 11, weight: .medium, design: .monospaced))
                    .foregroundStyle(LabTheme.fgSecondary)
            } else {
                Text("VU unavailable")
                    .font(.system(size: 11, weight: .medium))
                    .foregroundStyle(LabTheme.fgTertiary)
            }
        }
        .padding(LabTheme.s16)
        .labCard()
    }
}
