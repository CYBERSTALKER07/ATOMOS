import SwiftUI

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
                .font(.system(.caption2, design: .monospaced, weight: .heavy))
                .foregroundStyle(LabTheme.fgTertiary)
            Text(count == 0 ? "empty" : "\(count)")
                .font(.system(.footnote, weight: .semibold))
                .foregroundStyle(count == 0 ? LabTheme.fgTertiary : LabTheme.fg)
        }
        .frame(maxWidth: .infinity, minHeight: 44, alignment: .leading)
        .padding(10)
        .background(LabTheme.card, in: RoundedRectangle(cornerRadius: LabTheme.buttonRadius, style: .continuous))
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(label): \(count == 0 ? "empty" : "\(count)")")
    }
}

struct ManifestBulletMeter: View {
    let state: String?
    let usedVU: Double?
    let maxVU: Double

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("MANIFEST")
                .font(.system(.caption2, design: .monospaced, weight: .heavy))
                .foregroundStyle(LabTheme.fgTertiary)
            Text(state?.isEmpty == false ? (state ?? "unavailable") : "unavailable")
                .font(.system(.footnote, design: .monospaced, weight: .bold))
                .foregroundStyle(LabTheme.fg)
            if let used = usedVU, maxVU > 0 {
                ProgressView(value: min(max(used / maxVU, 0), 1))
                Text(String(format: "%.0f / %.0f VU", used, maxVU))
                    .font(.system(.caption2, design: .monospaced, weight: .medium))
                    .foregroundStyle(LabTheme.fgSecondary)
            } else {
                Text("VU unavailable")
                    .font(.system(.caption2, weight: .medium))
                    .foregroundStyle(LabTheme.fgTertiary)
            }
        }
        .padding(LabTheme.s16)
        .labCard()
        .accessibilityElement(children: .combine)
    }
}
