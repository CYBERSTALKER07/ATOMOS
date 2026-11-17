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
                let plate = target.truckPlate.isEmpty ? String(target.truckId.prefix(8)) : target.truckPlate
                VStack(alignment: .leading) {
                    Text(plate)
                        .font(.headline)
                    Text(L10n.format("mobile_factory.ui.totalvolumevu_maxcapacityvu_vu", "\(Int(target.totalVolumeVU))", "\(Int(target.maxCapacityVU))"))
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                }
                .frame(minHeight: 44, alignment: .leading)
                .accessibilityElement(children: .combine)
                .accessibilityLabel("Target manifest \(plate), \(Int(target.totalVolumeVU)) of \(Int(target.maxCapacityVU)) volume units")
                .tag(target.id)
            }
            .scrollDismissesKeyboard(.interactively)
            .navigationTitle("mobile_factory.ui.move_transfer")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("common.action.cancel") {
                        PegasusHaptics.selection()
                        onCancel()
                    }
                    .frame(minWidth: 44, minHeight: 44)
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("mobile_factory.ui.move") {
                        if let targetId = selectedTargetManifestId {
                            PegasusHaptics.confirm()
                            onMove(targetId)
                        }
                    }
                    .frame(minWidth: 44, minHeight: 44)
                    .disabled(selectedTargetManifestId == nil || isProcessing)
                }
            }
        }
        .presentationDetents([.medium])
    }
}
