import SwiftUI

struct SupplyRequestRow: View {
    let request: SupplyRequest

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
            Text(L10n.format("mobile_factory.ui.request_prefix", "\(request.id.prefix(8))"))
                .font(.headline)
            Text(L10n.format("mobile_factory.ui.priority_totalvolumevu_vu", "\(request.priority)", "\(Int(request.totalVolumeVU))"))
                .font(.subheadline)
                .foregroundStyle(.secondary)
            HStack {
                FactoryStatusBadge(text: request.state)
                Spacer()
                if let target = request.requestedDeliveryDate {
                    Text(L10n.format("mobile_factory.ui.due_target_2", "\(target)"))
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }
            if !request.notes.isEmpty {
                Text(request.notes)
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .lineLimit(1)
            }
        }
        .padding(.vertical, LabTheme.spacingXS)
    }
}
