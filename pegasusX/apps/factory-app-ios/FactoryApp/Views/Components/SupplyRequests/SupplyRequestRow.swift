import SwiftUI

struct SupplyRequestRow: View {
    let request: SupplyRequest

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
            Text("Request \(request.id.prefix(8))")
                .font(.headline)
            Text("\(request.priority) · \(Int(request.totalVolumeVU)) VU")
                .font(.subheadline)
                .foregroundStyle(.secondary)
            HStack {
                FactoryStatusBadge(text: request.state)
                Spacer()
                if let target = request.requestedDeliveryDate {
                    Text("Due: \(target)")
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
