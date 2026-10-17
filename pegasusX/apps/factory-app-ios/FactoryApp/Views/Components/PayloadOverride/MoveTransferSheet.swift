import SwiftUI

struct MoveTransferSheet: View {
    let candidate: MoveCandidate
    let manifests: [Manifest]
    @Binding var selectedTargetManifestId: String?
    let isProcessing: Bool
    let onCancel: () -> Void
    let onMove: (String) -> Void
    
    var body: some View {
        NavigationStack {
            let targets = manifests.filter { $0.id != candidate.sourceManifestId }
            List(targets, selection: $selectedTargetManifestId) { target in
                VStack(alignment: .leading) {
                    Text(target.truckPlate.isEmpty ? String(target.truckId.prefix(8)) : target.truckPlate)
                        .font(.headline)
                    Text(L10n.format("mobile_factory.ui.totalvolumevu_maxcapacityvu_vu", "\(Int(target.totalVolumeVU))", "\(Int(target.maxCapacityVU))"))
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                }
                .tag(target.id)
            }
            .navigationTitle("mobile_factory.ui.move_transfer")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("common.action.cancel", action: onCancel)
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("mobile_factory.ui.move") {
                        if let targetId = selectedTargetManifestId {
                            onMove(targetId)
                        }
                    }
                    .disabled(selectedTargetManifestId == nil || isProcessing)
                }
            }
        }
        .presentationDetents([.medium])
    }
}
