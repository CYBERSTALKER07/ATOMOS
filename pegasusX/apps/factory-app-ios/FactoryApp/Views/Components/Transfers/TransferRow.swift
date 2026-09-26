import SwiftUI

struct TransferRow: View {
    let transfer: Transfer

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
            HStack(alignment: .top, spacing: LabTheme.spacingMD) {
                VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                    Text(transfer.warehouseName.isEmpty ? String(transfer.warehouseId.prefix(8)) : transfer.warehouseName)
                        .font(.subheadline.bold())
                    Text(L10n.format("mobile_factory.ui.transfer_prefix", "\(transfer.id.prefix(8))"))
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
                TransferRowMetric(label: "Items", value: "\(transfer.totalItems)")
                TransferRowMetric(label: "Volume", value: String(format: "%.0fL", transfer.totalVolumeL))
            }
        }
        .padding(.vertical, LabTheme.spacingXS)
    }
}
