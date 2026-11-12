import SwiftUI

struct OverrideManifestCard: View {
    let manifest: Manifest
    let isProcessing: Bool
    let manifestsCount: Int
    let onCancelManifest: () -> Void
    let onMoveTransfer: (ManifestTransfer) -> Void
    let onReleaseTransfer: (ManifestTransfer) -> Void

    var body: some View {
        let plateLabel = manifest.truckPlate.isEmpty ? String(manifest.truckId.prefix(8)) : manifest.truckPlate

        VStack(alignment: .leading, spacing: LabTheme.spacingMD) {
            HStack {
                VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                    Text(plateLabel)
                        .font(.headline)
                    Text(L10n.format("mobile_factory.ui.manifest_prefix", "\(manifest.id.prefix(8))"))
                        .font(.caption.monospaced())
                        .foregroundStyle(.secondary)
                }
                .accessibilityElement(children: .combine)

                Spacer()

                Button("mobile_factory.ui.cancel_manifest_2", role: .destructive) {
                    PegasusHaptics.warning()
                    onCancelManifest()
                }
                .buttonStyle(.bordered)
                .frame(minWidth: 44, minHeight: 44)
                .disabled(isProcessing)
                .accessibilityLabel("Cancel manifest \(plateLabel)")
            }

            let capacity = manifest.maxCapacityVU > 0 ? manifest.maxCapacityVU : 1.0
            let progress = min(max(manifest.totalVolumeVU / capacity, 0.0), 1.0)

            ProgressView(value: progress)
                .tint(progress > 0.95 ? .red : .blue)
                .accessibilityLabel("Capacity utilization")
                .accessibilityValue("\(Int((progress * 100).rounded())) percent")

            HStack {
                MetricBox(label: "Volume", value: "\(Int(manifest.totalVolumeVU)) VU")
                MetricBox(label: "Capacity", value: "\(Int(manifest.maxCapacityVU)) VU")
                MetricBox(label: "Transfers", value: "\(manifest.transfers.count)")
            }

            if manifest.transfers.isEmpty {
                Text("mobile_factory.ui.no_transfers_are_assigned_to_this_manifest")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
                    .padding()
                    .frame(maxWidth: .infinity)
                    .background(Color(uiColor: .secondarySystemBackground))
                    .clipShape(RoundedRectangle(cornerRadius: LabTheme.radiusSM, style: .continuous))
            } else {
                ForEach(manifest.transfers) { transfer in
                    OverrideTransferRow(
                        transfer: transfer,
                        isProcessing: isProcessing,
                        canMove: manifestsCount > 1,
                        onMove: { onMoveTransfer(transfer) },
                        onRelease: { onReleaseTransfer(transfer) }
                    )
                }
            }
        }
        .padding()
        .background(Color(uiColor: .secondarySystemBackground))
        .clipShape(RoundedRectangle(cornerRadius: LabTheme.radiusLG, style: .continuous))
    }
}
