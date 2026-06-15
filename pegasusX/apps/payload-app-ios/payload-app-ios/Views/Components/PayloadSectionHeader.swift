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
                    .font(.system(size: 12, weight: .black, design: .monospaced))
                    .foregroundStyle(TermTheme.secondary)
                if let subtitle, !subtitle.isEmpty {
                    Text(subtitle)
                        .font(.system(size: 11, weight: .medium, design: .monospaced))
                        .foregroundStyle(TermTheme.tertiary)
                }
            }
            Spacer()
            if let trailing, !trailing.isEmpty {
                Text(trailing.uppercased())
                    .font(.system(size: 12, weight: .black, design: .monospaced))
                    .foregroundStyle(trailingTint)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
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
        VStack(alignment: .leading, spacing: TermTheme.s8) {
            if large {
                Text("PROTOCOL_STATE")
                    .font(.system(size: 10, weight: .black, design: .monospaced))
                    .foregroundStyle(TermTheme.tertiary)
                    .tracking(1.4)
            }
            Text(text.replacingOccurrences(of: "_", with: " ").uppercased())
                .font(.system(size: large ? 24 : 12, weight: .black, design: .monospaced))
                .foregroundStyle(resolvedTint)
                .tracking(large ? 2.0 : 0)
        }
        .padding(large ? TermTheme.s24 : TermTheme.s12)
        .frame(minWidth: large ? 200 : nil, alignment: .leading)
        .background(resolvedTint.opacity(large ? 0.06 : 0.12), in: RoundedRectangle(cornerRadius: large ? TermTheme.radiusMD : TermTheme.radiusSM, style: .continuous))
        .overlay {
            if large {
                RoundedRectangle(cornerRadius: TermTheme.radiusMD, style: .continuous)
                    .stroke(TermTheme.separator.opacity(0.12), lineWidth: 1)
            }
        }
    }
}
