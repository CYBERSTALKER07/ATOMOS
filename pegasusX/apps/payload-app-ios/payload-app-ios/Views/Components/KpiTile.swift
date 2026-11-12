import SwiftUI

/// Tactical KPI tile — mirrors Android `ManifestKpiGrid` / warehouse `KpiTile` discipline.
struct KpiTile: View {
    let label: String
    let value: String
    var footer: AnyView? = nil

    init(label: String, value: String) {
        self.label = label
        self.value = value
    }

    init<Footer: View>(label: String, value: String, @ViewBuilder footer: () -> Footer) {
        self.label = label
        self.value = value
        self.footer = AnyView(footer())
    }

    var body: some View {
        VStack(alignment: .leading, spacing: TermTheme.s8) {
            Text(label.uppercased())
                .font(.system(.caption2, design: .monospaced, weight: .black))
                .foregroundStyle(TermTheme.tertiary)
            Text(value)
                .font(.system(value.count > 14 ? .subheadline : .callout, design: .monospaced, weight: .bold))
                .foregroundStyle(TermTheme.accent)
                .minimumScaleFactor(0.8)
                .lineLimit(2)
            footer
        }
        .padding(TermTheme.s20)
        .frame(maxWidth: .infinity, alignment: .leading)
        .tacticalCard()
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(label): \(value)")
    }
}
