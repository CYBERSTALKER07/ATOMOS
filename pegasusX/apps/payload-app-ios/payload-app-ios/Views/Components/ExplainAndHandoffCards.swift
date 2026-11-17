import SwiftUI

struct ExplainStatusBanner: View {
    let explain: StatusExplain?
    var fallbackTitle: String?
    var fallbackDetail: String?

    var body: some View {
        let title = explain?.title ?? fallbackTitle
        let summary = explain?.summary ?? fallbackDetail
        if let title, !title.isEmpty {
            VStack(alignment: .leading, spacing: 6) {
                Text(title).font(.headline)
                if let summary, !summary.isEmpty {
                    Text(summary).font(.subheadline)
                }
                if let steps = explain?.nextSteps {
                    ForEach(steps, id: \.self) { step in
                        Text(L10n.format("mobile_payload.ui.step", "\(step)")).font(.caption)
                    }
                }
            }
            .foregroundStyle(.white)
            .padding(12)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(Color.red.opacity(0.9), in: RoundedRectangle(cornerRadius: 12, style: .continuous))
            .accessibilityElement(children: .combine)
            .accessibilityLabel("\(title). \(summary ?? "")")
        }
    }
}

struct HandoffInboxCard: View {
    let metadata: HandoffCardMetadata
    var onAction: ((String) -> Void)? = nil

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(metadata.title).font(.subheadline.bold())
            if let subtitle = metadata.subtitle, !subtitle.isEmpty {
                Text(subtitle).font(.caption).foregroundStyle(.secondary)
            }
            if let fields = metadata.fields {
                ForEach(fields.sorted(by: { $0.key < $1.key }), id: \.key) { key, value in
                    Text(L10n.format("mobile_payload.ui.replacingoccurrences_value", "\(key.replacingOccurrences(of: "_", with: " "))", "\(value)"))
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }
            if let link = metadata.primaryLink, !link.isEmpty {
                Button {
                    PegasusHaptics.selection()
                    onAction?(link)
                } label: {
                    Text(metadata.primaryCta ?? "Open")
                        .font(.caption.bold())
                        .foregroundStyle(Color.accentColor)
                        .frame(minWidth: 44, minHeight: 44, alignment: .leading)
                }
                .buttonStyle(.plain)
                .accessibilityLabel(metadata.primaryCta ?? "Open handoff")
            }
        }
        .padding(10)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(TermTheme.card, in: RoundedRectangle(cornerRadius: 10, style: .continuous))
    }
}
