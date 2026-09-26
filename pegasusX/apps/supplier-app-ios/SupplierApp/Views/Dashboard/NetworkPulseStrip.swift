import SwiftUI

struct NetworkPulseStrip: View {
    let events: [SupplierPulseEvent]
    let loading: Bool
    var error: String? = nil

    var body: some View {
        if loading && events.isEmpty && (error ?? "").isEmpty {
            Text("mobile_supplier.ui.loading_network_pulse")
                .font(.caption)
                .foregroundStyle(.secondary)
        } else if let error, !error.isEmpty {
            Text(error)
                .font(.caption)
                .foregroundStyle(.red)
        } else if !events.isEmpty {
            VStack(alignment: .leading, spacing: SupplierTheme.spacingSM) {
                Text("factory_portal.app.text.network_pulse")
                    .font(.headline)
                ScrollView(.horizontal, showsIndicators: false) {
                    HStack(spacing: SupplierTheme.spacingMD) {
                        ForEach(events.prefix(12)) { event in
                            VStack(alignment: .leading, spacing: 4) {
                                Text(event.title)
                                    .font(.subheadline.bold())
                                    .lineLimit(2)
                                if let description = event.description, !description.isEmpty {
                                    Text(description)
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                        .lineLimit(2)
                                }
                            }
                            .frame(minWidth: 180, maxWidth: 240, alignment: .leading)
                            .padding()
                            .background(SupplierTheme.card)
                            .clipShape(RoundedRectangle(cornerRadius: SupplierTheme.radiusMD))
                        }
                    }
                }
            }
        }
    }
}
