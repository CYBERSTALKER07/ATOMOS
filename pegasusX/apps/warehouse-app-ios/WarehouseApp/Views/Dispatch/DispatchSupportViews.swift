import SwiftUI

struct DispatchStatusBanner: View {
    let systemImage: String
    let title: String
    let tint: Color

    var body: some View {
        Label(title, systemImage: systemImage)
            .font(.caption.weight(.semibold))
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(.horizontal, LabTheme.spacingSM)
            .padding(.vertical, LabTheme.spacingXS)
            .foregroundStyle(tint)
            .background(tint.opacity(0.12), in: RoundedRectangle(cornerRadius: 12, style: .continuous))
            .accessibilityElement(children: .combine)
            .accessibilityLabel(title)
    }
}

struct HandoffTimelineSection: View {
    let events: [WarehousePulseEvent]
    let loading: Bool
    var error: String? = nil

    private var subtitle: String {
        if let error, !error.isEmpty { return error }
        if loading && events.isEmpty { return "Loading handoff chain…" }
        if events.isEmpty { return "No preorder → dispatch → seal events in the recent pulse window." }
        return "\(events.count) handoff event(s) in recent pulse."
    }

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
            Text("warehouse_portal.dispatch.text.handoff_timeline")
                .font(.headline)
            Text(subtitle)
                .font(.caption)
                .foregroundStyle((error ?? "").isEmpty ? PegasusMonochromeTheme.secondaryLabel : PegasusMonochromeTheme.destructive)
            if (error ?? "").isEmpty {
                ForEach(events.prefix(8)) { event in
                    VStack(alignment: .leading, spacing: 4) {
                        Text(event.title)
                            .font(.subheadline.bold())
                            .lineLimit(2)
                        if let description = event.description, !description.isEmpty {
                            Text(description)
                                .font(.caption)
                                .foregroundStyle(.secondary)
                                .lineLimit(3)
                        }
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding()
                    .background(LabTheme.card)
                    .clipShape(RoundedRectangle(cornerRadius: LabTheme.radiusMD))
                    .accessibilityElement(children: .combine)
                }
            }
        }
    }
}
