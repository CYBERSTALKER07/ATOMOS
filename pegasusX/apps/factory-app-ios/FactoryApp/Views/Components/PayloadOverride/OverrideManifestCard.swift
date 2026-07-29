import SwiftUI

struct OverrideManifestCard: View {
    let manifest: Manifest
    let isProcessing: Bool
    let manifestsCount: Int
    let onCancelManifest: () -> Void
    let onMoveTransfer: (ManifestTransfer) -> Void
    let onReleaseTransfer: (ManifestTransfer) -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingMD) {
            HStack {
                VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                    Text(manifest.truckPlate.isEmpty ? String(manifest.truckId.prefix(8)) : manifest.truckPlate)
                        .font(.headline)
                    Text("Manifest \(manifest.id.prefix(8))")
                        .font(.caption.monospaced())
                        .foregroundStyle(.secondary)
                }
                Spacer()
                Button("Cancel Manifest", role: .destructive, action: onCancelManifest)
                .buttonStyle(.bordered)
                .disabled(isProcessing)
            }

            let capacity = manifest.maxCapacityVU > 0 ? manifest.maxCapacityVU : 1.0
            let progress = min(max(manifest.totalVolumeVU / capacity, 0.0), 1.0)
            
            ProgressView(value: progress)
                .tint(progress > 0.95 ? .red : .blue)
            
            HStack {
                MetricBox(label: "Volume", value: "\(Int(manifest.totalVolumeVU)) VU")
                MetricBox(label: "Capacity", value: "\(Int(manifest.maxCapacityVU)) VU")
                MetricBox(label: "Transfers", value: "\(manifest.transfers.count)")
            }
            
            if manifest.transfers.isEmpty {
                Text("No transfers are assigned to this manifest.")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
                    .padding()
                    .frame(maxWidth: .infinity)
                    .background(Color(uiColor: .secondarySystemBackground))
                    .cornerRadius(LabTheme.radiusSM)
            } else {
                ForEach(manifest.transfers) { transfer in
                    TransferRow(
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
        .cornerRadius(LabTheme.radiusLG)
    }
}
