import SwiftUI

struct OverrideTransferRow: View {
    let transfer: ManifestTransfer
    let isProcessing: Bool
    let canMove: Bool
    let onMove: () -> Void
    let onRelease: () -> Void

    var body: some View {
        let titleText = transfer.productName.isEmpty ? "Transfer \(transfer.id.prefix(8))" : transfer.productName

        VStack(spacing: LabTheme.spacingSM) {
            HStack {
                VStack(alignment: .leading) {
                    Text(titleText)
                        .font(.subheadline.bold())
                    Text(transfer.id.prefix(8))
                        .font(.caption.monospaced())
                        .foregroundStyle(.secondary)
                }
                Spacer()
                FactoryStatusBadge(text: transfer.state)
            }
            .accessibilityElement(children: .combine)
            .accessibilityLabel("\(titleText), status \(transfer.state)")

            HStack {
                MetricBox(label: "Qty", value: "\(transfer.quantity)")
                MetricBox(label: "Volume", value: "\(Int(transfer.volumeVU)) VU")
            }

            HStack {
                Button("mobile_factory.ui.move") {
                    PegasusHaptics.selection()
                    onMove()
                }
                .buttonStyle(.borderedProminent)
                .frame(minWidth: 44, minHeight: 44)
                .disabled(isProcessing || !canMove)
                .accessibilityLabel("Move \(titleText) to another manifest")

                Button("mobile_factory.ui.release") {
                    PegasusHaptics.warning()
                    onRelease()
                }
                .buttonStyle(.bordered)
                .frame(minWidth: 44, minHeight: 44)
                .disabled(isProcessing)
                .accessibilityLabel("Release \(titleText) from manifest")
            }
        }
        .padding()
        .background(Color(uiColor: .systemBackground))
        .clipShape(RoundedRectangle(cornerRadius: LabTheme.radiusMD, style: .continuous))
    }
}
