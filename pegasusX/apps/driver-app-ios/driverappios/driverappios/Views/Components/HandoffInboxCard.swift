import SwiftUI

struct HandoffInboxCard: View {
    let metadata: HandoffCardMetadata
    var onAction: ((String) -> Void)? = nil

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(metadata.title)
                .font(.system(.footnote, weight: .semibold))
            if let subtitle = metadata.subtitle, !subtitle.isEmpty {
                Text(subtitle)
                    .font(.system(.caption2, weight: .medium))
                    .foregroundStyle(LabTheme.fgSecondary)
            }
            if let fields = metadata.fields {
                ForEach(fields.sorted(by: { $0.key < $1.key }), id: \.key) { key, value in
                    Text(L10n.format("mobile_driver.ui.replacingoccurrences_value", "\(key.replacingOccurrences(of: "_", with: " "))", "\(value)"))
                        .font(.system(.caption2, weight: .medium))
                        .foregroundStyle(LabTheme.fgSecondary)
                }
            }
            if let link = metadata.primaryLink, !link.isEmpty {
                Button {
                    Haptics.light()
                    onAction?(link)
                } label: {
                    Text(metadata.primaryCta ?? "Open")
                        .font(.system(.caption, weight: .bold))
                        .foregroundStyle(LabTheme.transit)
                        .frame(minHeight: 44, alignment: .leading)
                        .contentShape(Rectangle())
                }
                .buttonStyle(.plain)
                .accessibilityLabel(metadata.primaryCta ?? "Open")
            }
        }
        .padding(10)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(LabTheme.fg.opacity(0.06), in: RoundedRectangle(cornerRadius: 10, style: .continuous))
    }
}
