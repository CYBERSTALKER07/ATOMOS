import SwiftUI

struct FactoryHandoffTimelineSection: View {
    let events: [FactoryPulseEvent]
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
                .foregroundStyle((error ?? "").isEmpty ? Color.secondary : Color.red)
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
                }
            }
        }
    }
}
