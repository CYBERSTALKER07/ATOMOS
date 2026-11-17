import SwiftUI

struct MoveCandidate: Identifiable {
    let sourceManifestId: String
    let transfer: ManifestTransfer

    var id: String { "\(sourceManifestId):\(transfer.id)" }
}

struct CancelTransferCandidate: Identifiable {
    let manifestId: String
    let transfer: ManifestTransfer

    var id: String { "\(manifestId):\(transfer.id)" }
}

struct MetricBox: View {
    let label: String
    let value: String

    var body: some View {
        VStack(spacing: LabTheme.spacingXS) {
            Text(value)
                .font(.headline)
            Text(label)
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, LabTheme.spacingSM)
        .background(Color(uiColor: .systemBackground))
        .clipShape(RoundedRectangle(cornerRadius: LabTheme.radiusSM, style: .continuous))
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(label): \(value)")
    }
}
