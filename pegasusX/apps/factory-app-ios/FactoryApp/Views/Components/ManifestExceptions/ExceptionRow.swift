import SwiftUI

struct ExceptionRow: View {
    let exception: ManifestException
    var resolving: Bool = false
    var onResolve: (() -> Void)? = nil

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
            HStack {
                Text(exception.reason)
                    .font(.headline)
                Spacer()
                if exception.escalated {
                    Text("factory_portal.residual.text.escalated")
                        .font(.caption.bold())
                        .padding(.horizontal, 8)
                        .padding(.vertical, 4)
                        .background(LabTheme.destructive.opacity(0.15), in: Capsule())
                }
            }
            Text(L10n.format("mobile_factory.ui.transfer_shortid_manifest_shortid_2", "\(shortId(exception.transferId))", "\(shortId(exception.manifestId))"))
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
            if let onResolve {
                Button(resolving ? "Resolving…" : "Resolve", action: onResolve)
                    .buttonStyle(.borderedProminent)
                    .disabled(resolving)
                    .frame(maxWidth: .infinity, alignment: .leading)
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
