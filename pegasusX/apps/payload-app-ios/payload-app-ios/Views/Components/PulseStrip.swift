import SwiftUI

struct PulseStrip: View {
    let events: [PulseEvent]
    let loading: Bool

    var body: some View {
        if loading && events.isEmpty {
            Text("mobile_payload.ui.loading_network_pulse")
                .font(.footnote)
                .foregroundStyle(TermTheme.tertiary)
                .padding(.vertical, 8)
        } else if !events.isEmpty {
            VStack(alignment: .leading, spacing: 8) {
                Text("mobile_payload.ui.network_pulse")
                    .font(.system(size: 10, weight: .bold, design: .monospaced))
                    .foregroundStyle(TermTheme.tertiary)
                ScrollView(.horizontal, showsIndicators: false) {
                    HStack(spacing: 12) {
                        ForEach(events.prefix(12)) { event in
                            VStack(alignment: .leading, spacing: 4) {
                                Text(event.title)
                                    .font(.system(size: 13, weight: .semibold))
                                    .foregroundStyle(TermTheme.accent)
                                    .lineLimit(2)
                                if let body = event.description, !body.isEmpty {
                                    Text(body)
                                        .font(.caption)
                                        .foregroundStyle(TermTheme.secondary)
                                        .lineLimit(2)
                                }
                                Text(formatTimestamp(event.occurredAt))
                                    .font(.caption2.monospaced())
                                    .foregroundStyle(TermTheme.tertiary)
                            }
                            .padding(12)
                            .frame(minWidth: 180, maxWidth: 240, alignment: .leading)
                            .background(TermTheme.card, in: RoundedRectangle(cornerRadius: 12))
                        }
                    }
                }
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 8)
        }
    }

    private func formatTimestamp(_ iso: String) -> String {
        guard let date = ISO8601DateFormatter().date(from: iso) else { return "" }
        let mins = Int(Date().timeIntervalSince(date) / 60)
        if mins < 1 { return "now" }
        if mins < 60 { return "\(mins)m" }
        return "\(mins / 60)h"
    }
}

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
            .background(Color.red.opacity(0.9), in: RoundedRectangle(cornerRadius: 12))
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
                    onAction?(link)
                } label: {
                    Text(metadata.primaryCta ?? "Open")
                        .font(.caption.bold())
                        .foregroundStyle(Color.accentColor)
                }
                .buttonStyle(.plain)
            }
        }
        .padding(10)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(TermTheme.card, in: RoundedRectangle(cornerRadius: 10))
    }
}
