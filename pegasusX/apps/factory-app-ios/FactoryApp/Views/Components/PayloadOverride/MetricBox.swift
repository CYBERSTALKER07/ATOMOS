import SwiftUI

struct MoveCandidate {
    let sourceManifestId: String
    let transfer: ManifestTransfer
}

struct CancelTransferCandidate {
    let manifestId: String
    let transfer: ManifestTransfer
}

struct MetricBox: View {
    let label: String
    let value: String
    
    var body: some View {
        VStack(spacing: LabTheme.spacingXXS) {
            Text(value)
                .font(.headline)
            Text(label)
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, LabTheme.spacingSM)
        .background(Color(uiColor: .systemBackground))
        .cornerRadius(LabTheme.radiusSM)
    }
}
