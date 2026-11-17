import SwiftUI

struct PulseStrip: View {
    let events: [PulseEvent]
    let loading: Bool
    var error: String? = nil

    var body: some View {
        if loading && events.isEmpty && (error ?? "").isEmpty {
            Text("mobile_payload.ui.loading_network_pulse")
                .font(.footnote)
                .foregroundStyle(TermTheme.tertiary)
                .padding(.vertical, 8)
        } else if let error, !error.isEmpty {
            Text(error)
                .font(.footnote)
                .foregroundStyle(.red)
                .padding(.vertical, 8)
                .accessibilityLabel("Network pulse error: \(error)")
        } else if !events.isEmpty {
            VStack(alignment: .leading, spacing: 8) {
                Text("mobile_payload.ui.network_pulse")
                    .font(.system(.caption2, design: .monospaced, weight: .bold))
                    .foregroundStyle(TermTheme.tertiary)
                ScrollView(.horizontal, showsIndicators: false) {
                    HStack(spacing: 12) {
                        ForEach(events.prefix(12)) { event in
                            VStack(alignment: .leading, spacing: 4) {
                                Text(event.title)
                                    .font(.system(.footnote, weight: .semibold))
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
                            .frame(minWidth: 180, maxWidth: 240, minHeight: 44, alignment: .leading)
                            .background(TermTheme.card, in: RoundedRectangle(cornerRadius: 12, style: .continuous))
                            .accessibilityElement(children: .combine)
                            .accessibilityLabel("\(event.title), \(event.description ?? ""), \(formatTimestamp(event.occurredAt))")
                        }
                    }
                }
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 8)
        }
    }

    private static let isoFormatter = ISO8601DateFormatter()

    private func formatTimestamp(_ iso: String) -> String {
        guard let date = Self.isoFormatter.date(from: iso) else { return "" }
        let mins = Int(Date().timeIntervalSince(date) / 60)
        if mins < 1 { return "now" }
        if mins < 60 { return "\(mins)m" }
        return "\(mins / 60)h"
    }
}
