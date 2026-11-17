import SwiftUI

struct ReplenishmentActionView: View {
    let replenishing: Bool
    let runReplenishment: () -> Void

    var body: some View {
        Group {
            Section {
                SupplierSectionHeader(
                    title: "mobile_supplier.chrome.replenishment",
                    subtitle: "mobile_supplier.chrome.warehouse_supply_request"
                )
            }
            Section {
                NavigationLink { ReplenishmentPoliciesView() } label: {
                    Label("mobile_supplier.ui.view_replenishment_policies", systemImage: "doc.text")
                }
                Button(LocalizedStringKey(replenishing ? "mobile_supplier.chrome.triggering" : "mobile_supplier.chrome.trigger_replenishment")) {
                    runReplenishment()
                }
                .disabled(replenishing)
            }
        }
    }
}
