import SwiftUI

enum PulseHonesty {
    static let failed = "pulse_failed"

    struct Result<T> {
        let events: [T]
        let error: String?
    }

    static func apply<T>(ok: Bool, incoming: [T]?, previous: [T]) -> Result<T> {
        if ok, let incoming {
            return Result(events: incoming, error: nil)
        }
        return Result(events: previous, error: failed)
    }
}

struct PulseStrip: View {
    let events: [PulseEvent]
    let loading: Bool
    var error: String? = nil

    var body: some View {
        if loading && events.isEmpty && (error ?? "").isEmpty {
            Text("mobile_driver.ui.loading_network_pulse")
                .font(.system(size: 12, weight: .medium))
                .foregroundStyle(LabTheme.fgTertiary)
                .padding(.vertical, LabTheme.s8)
        } else if let error, !error.isEmpty {
            Text(error)
                .font(.system(size: 12, weight: .medium))
                .foregroundStyle(LabTheme.destructive)
                .padding(.vertical, LabTheme.s8)
        } else if !events.isEmpty {
            VStack(alignment: .leading, spacing: LabTheme.s8) {
                Text("factory_portal.app.text.network_pulse")
                    .font(.system(size: 11, weight: .bold, design: .monospaced))
                    .foregroundStyle(LabTheme.fgTertiary)
                    .tracking(1.1)

                ScrollView(.horizontal, showsIndicators: false) {
                    HStack(spacing: LabTheme.s12) {
                        ForEach(events.prefix(12)) { event in
                            VStack(alignment: .leading, spacing: 4) {
                                Text(event.title)
                                    .font(.system(size: 13, weight: .semibold))
                                    .foregroundStyle(LabTheme.fg)
                                    .lineLimit(2)
                                if let body = event.description, !body.isEmpty {
                                    Text(body)
                                        .font(.system(size: 11, weight: .medium))
                                        .foregroundStyle(LabTheme.fgSecondary)
                                        .lineLimit(2)
                                }
                                Text(formatTimestamp(event.occurredAt))
                                    .font(.system(size: 10, weight: .medium, design: .monospaced))
                                    .foregroundStyle(LabTheme.fgTertiary)
                            }
                            .padding(.horizontal, 12)
                            .padding(.vertical, 10)
                            .frame(minWidth: 180, maxWidth: 240, alignment: .leading)
                            .background(LabTheme.fg.opacity(0.06), in: RoundedRectangle(cornerRadius: 12))
                        }
                    }
                }
            }
        }
    }

    private func formatTimestamp(_ iso: String) -> String {
        guard let date = ISO8601DateFormatter().date(from: iso) else { return "" }
        let diff = Date().timeIntervalSince(date)
        let mins = Int(diff / 60)
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
                Text(title)
                    .font(.system(size: 13, weight: .bold))
                if let summary, !summary.isEmpty {
                    Text(summary)
                        .font(.system(size: 12, weight: .medium))
                }
                if let steps = explain?.nextSteps, !steps.isEmpty {
                    ForEach(steps, id: \.self) { step in
                        Text(L10n.format("mobile_driver.ui.step", "\(step)"))
                            .font(.system(size: 11, weight: .medium))
                    }
                }
            }
            .foregroundStyle(.white)
            .padding(LabTheme.s16)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(LabTheme.destructive, in: RoundedRectangle(cornerRadius: LabTheme.buttonRadius))
        }
    }
}

struct HandoffInboxCard: View {
    let metadata: HandoffCardMetadata
    var onAction: ((String) -> Void)? = nil

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(metadata.title)
                .font(.system(size: 13, weight: .semibold))
            if let subtitle = metadata.subtitle, !subtitle.isEmpty {
                Text(subtitle)
                    .font(.system(size: 11, weight: .medium))
                    .foregroundStyle(LabTheme.fgSecondary)
            }
            if let fields = metadata.fields {
                ForEach(fields.sorted(by: { $0.key < $1.key }), id: \.key) { key, value in
                    Text(L10n.format("mobile_driver.ui.replacingoccurrences_value", "\(key.replacingOccurrences(of: "_", with: " "))", "\(value)"))
                        .font(.system(size: 11, weight: .medium))
                        .foregroundStyle(LabTheme.fgSecondary)
                }
            }
            if let link = metadata.primaryLink, !link.isEmpty {
                Button {
                    onAction?(link)
                } label: {
                    Text(metadata.primaryCta ?? "Open")
                        .font(.system(size: 11, weight: .bold))
                        .foregroundStyle(LabTheme.transit)
                }
                .buttonStyle(.plain)
            }
        }
        .padding(10)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(LabTheme.fg.opacity(0.06), in: RoundedRectangle(cornerRadius: 10))
    }
}
