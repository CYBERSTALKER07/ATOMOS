import SwiftUI

struct TransferOverviewCard: View {
    let transfer: Transfer

    var body: some View {
        let destination = transfer.warehouseName.isEmpty ? String(transfer.warehouseId.prefix(8)) : transfer.warehouseName
        let priorityLabel = transfer.priority.isEmpty ? "STANDARD" : transfer.priority

        VStack(alignment: .leading, spacing: LabTheme.spacingMD) {
            VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                Text(destination)
                    .font(.title3.bold())
                Text(L10n.format("mobile_factory.ui.transfer_prefix", "\(transfer.id.prefix(8))"))
                    .font(.footnote)
                    .foregroundStyle(.secondary)
            }

            HStack(spacing: LabTheme.spacingSM) {
                TransferDetailTag(text: transfer.state)
                TransferDetailTag(text: priorityLabel, emphasized: false)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .labCard()
        .accessibilityElement(children: .combine)
        .accessibilityLabel("Transfer \(transfer.id.prefix(8)) to \(destination), status \(transfer.state), priority \(priorityLabel)")
    }
}

struct TransferSummaryCard: View {
    let label: String
    let value: String

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
            Text(value)
                .font(.headline)
            Text(label)
                .font(.footnote)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(LabTheme.spacingMD)
        .background(LabTheme.secondaryBackground, in: RoundedRectangle(cornerRadius: LabTheme.radiusMD, style: .continuous))
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(label): \(value)")
    }
}
