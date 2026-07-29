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
            ResponsiveGridContentWrapper {
                ForEach(targets, selection: $selectedTargetManifestId) { target in
                    VStack(alignment: .leading) {
                        Text(target.truckPlate.isEmpty ? String(target.truckId.prefix(8)) : target.truckPlate)
                            .font(.headline)
                        Text("\(Int(target.totalVolumeVU)) / \(Int(target.maxCapacityVU)) VU")
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                    }
                    .tag(target.id)
                }
            }
            .navigationTitle("Move Transfer")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel", action: onCancel)
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Move") {
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
