import SwiftUI

struct BayTransferCard: View {
    let transfer: Transfer

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingMD) {
            HStack(alignment: .top, spacing: LabTheme.spacingMD) {
                VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                    Text(transfer.warehouseName.isEmpty ? String(transfer.warehouseId.prefix(8)) : transfer.warehouseName)
                        .font(.subheadline.bold())
                    Text("Transfer \(transfer.id.prefix(8))")
                        .font(.footnote)
                        .foregroundStyle(.secondary)
                }
                Spacer()
                VStack(alignment: .trailing, spacing: LabTheme.spacingXS) {
                    FactoryStatusBadge(text: transfer.state)
                    FactoryStatusBadge(
                        text: transfer.priority.isEmpty ? "STANDARD" : transfer.priority,
                        emphasized: false
                    )
                }
            }

            HStack(spacing: LabTheme.spacingSM) {
                BayTransferMetric(label: "Items", value: "\(transfer.totalItems)")
                BayTransferMetric(label: "Volume", value: String(format: "%.0fL", transfer.totalVolumeL))
            }
        }
        .labCard()
    }
}

struct BayTransferMetric: View {
    let label: String
    let value: String

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
            Text(value)
                .font(.subheadline.bold())
            Text(label)
                .font(.footnote)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(LabTheme.spacingMD)
        .background(LabTheme.tertiaryBackground, in: RoundedRectangle(cornerRadius: LabTheme.radiusMD))
    }
}
