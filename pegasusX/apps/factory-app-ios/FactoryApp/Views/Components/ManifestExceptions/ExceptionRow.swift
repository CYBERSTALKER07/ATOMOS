import SwiftUI

struct ExceptionRow: View {
    let exception: ManifestException

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
            HStack {
                Text(exception.reason)
                    .font(.headline)
                Spacer()
                if exception.escalated {
                    Text("Escalated")
                        .font(.caption.bold())
                        .padding(.horizontal, 8)
                        .padding(.vertical, 4)
                        .background(LabTheme.destructive.opacity(0.15), in: Capsule())
                }
            }
            Text("Transfer \(shortId(exception.transferId)) · Manifest \(shortId(exception.manifestId))")
                .font(.footnote.monospaced())
                .foregroundStyle(.secondary)
            Text(attemptLabel)
                .font(.subheadline)
                .foregroundStyle(exception.attemptCount >= 3 ? LabTheme.destructive : .secondary)
            if !exception.createdAt.isEmpty {
                Text(formattedDate(exception.createdAt))
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
        .padding(.vertical, LabTheme.spacingXS)
    }

    private var attemptLabel: String {
        var label = "Attempts: \(exception.attemptCount)"
        if exception.attemptCount >= 3 {
            label += " — DLQ"
        }
        return label
    }

    private func shortId(_ value: String) -> String {
        value.count > 12 ? String(value.prefix(8)) + "…" : value
    }

    private func formattedDate(_ raw: String) -> String {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        if let date = formatter.date(from: raw) ?? ISO8601DateFormatter().date(from: raw) {
            return date.formatted(date: .abbreviated, time: .shortened)
        }
        return raw
    }
}
