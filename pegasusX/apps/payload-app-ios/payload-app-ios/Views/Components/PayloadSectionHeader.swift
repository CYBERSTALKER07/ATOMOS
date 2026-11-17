import SwiftUI

struct PayloadSectionHeader: View {
    let title: String
    var subtitle: String? = nil
    var trailing: String? = nil
    var trailingTint: Color = TermTheme.accent

    var body: some View {
        HStack(alignment: .firstTextBaseline) {
            VStack(alignment: .leading, spacing: TermTheme.s4) {
                Text(title.uppercased())
                    .font(.system(.caption, design: .monospaced, weight: .black))
                    .foregroundStyle(TermTheme.secondary)
                if let subtitle, !subtitle.isEmpty {
                    Text(subtitle)
                        .font(.system(.caption, design: .monospaced, weight: .medium))
                        .foregroundStyle(TermTheme.tertiary)
                }
            }
            Spacer()
            if let trailing, !trailing.isEmpty {
                Text(trailing.uppercased())
                    .font(.system(.caption, design: .monospaced, weight: .black))
                    .foregroundStyle(trailingTint)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .accessibilityElement(children: .combine)
    }
}

struct PayloadStatusBadge: View {
    let text: String
    var tint: Color? = nil
    var large = false

    private var resolvedTint: Color {
        tint ?? TermTheme.statusTint(for: text)
    }

    var body: some View {
        let displayText = text.replacingOccurrences(of: "_", with: " ").uppercased()
        Group {
            if large {
                VStack(alignment: .leading, spacing: 6) {
                    Text("PROTOCOL_STATE")
                        .font(.system(.caption2, design: .monospaced, weight: .black))
                        .foregroundStyle(TermTheme.tertiary)
                        .tracking(1.4)
                    Text(displayText)
                        .font(.system(.title2, design: .monospaced, weight: .black))
                        .foregroundStyle(resolvedTint)
                        .tracking(2.0)
                }
                .padding(TermTheme.s24)
                .frame(minWidth: 200, alignment: .leading)
                .tacticalCard()
            } else {
                VStack(alignment: .leading, spacing: TermTheme.s4) {
                    Text(displayText)
                        .font(.system(.caption, design: .monospaced, weight: .black))
                        .foregroundStyle(resolvedTint)
                }
                .padding(TermTheme.s12)
                .background(resolvedTint.opacity(0.12), in: RoundedRectangle(cornerRadius: TermTheme.radiusSM, style: .continuous))
            }
        }
        .accessibilityElement(children: .combine)
        .accessibilityLabel("Protocol state: \(displayText)")
    }
}
