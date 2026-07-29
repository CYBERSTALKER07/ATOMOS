import SwiftUI

struct TransferRow: View {
    let transfer: ManifestTransfer
    let isProcessing: Bool
    let canMove: Bool
    let onMove: () -> Void
    let onRelease: () -> Void
    
    var body: some View {
        VStack(spacing: LabTheme.spacingSM) {
            HStack {
                VStack(alignment: .leading) {
                    Text(transfer.productName.isEmpty ? "Transfer \(transfer.id.prefix(8))" : transfer.productName)
                        .font(.subheadline.bold())
                    Text(transfer.id.prefix(8))
                        .font(.caption.monospaced())
                        .foregroundStyle(.secondary)
                }
                Spacer()
                FactoryStatusBadge(text: transfer.state)
            }
            
            HStack {
                MetricBox(label: "Qty", value: "\(transfer.quantity)")
                MetricBox(label: "Volume", value: "\(Int(transfer.volumeVU)) VU")
            }
            
            HStack {
                Button("Move", action: onMove)
                .buttonStyle(.borderedProminent)
                .disabled(isProcessing || !canMove)
                
                Button("Release", action: onRelease)
                .buttonStyle(.bordered)
                .disabled(isProcessing)
            }
        }
        .padding()
        .background(Color(uiColor: .systemBackground))
        .cornerRadius(LabTheme.radiusMD)
    }
}
