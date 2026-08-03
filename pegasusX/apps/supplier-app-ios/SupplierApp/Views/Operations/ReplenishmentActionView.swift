import SwiftUI

struct ReplenishmentActionView: View {
    let replenishing: Bool
    let runReplenishment: () -> Void

    var body: some View {
        Group {
            Section {
                SupplierSectionHeader(
                    title: "Replenishment",
                    subtitle: "Warehouse supply request against your primary node"
                )
            }
            Section {
                NavigationLink { ReplenishmentPoliciesView() } label: {
                    Label("View replenishment policies", systemImage: "doc.text")
                }
                Button(replenishing ? "Triggering…" : "Trigger replenishment") {
                    runReplenishment()
                }
                .disabled(replenishing)
            }
        }
    }
}
