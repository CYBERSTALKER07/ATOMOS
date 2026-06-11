import SwiftUI

struct MoreHubView: View {
    var body: some View {
        List {
            Section("Fulfillment") {
                NavigationLink { ManifestsView() } label: {
                    Label("Manifests", systemImage: "doc.text")
                }
                NavigationLink { DispatchPreviewView() } label: {
                    Label("Dispatch preview", systemImage: "paperplane")
                }
                NavigationLink { FleetOrdersView() } label: {
                    Label("Fleet orders", systemImage: "truck.box")
                }
            }
            Section("Exceptions") {
                NavigationLink { ExceptionsView() } label: {
                    Label("Exceptions", systemImage: "exclamationmark.triangle")
                }
                NavigationLink { ShopClosedView() } label: {
                    Label("Shop closed", systemImage: "storefront")
                }
                // Quantity negotiation disabled ecosystem-wide.
            }
            Section("Insights") {
                NavigationLink { ActivityView() } label: {
                    Label("Activity", systemImage: "clock.arrow.circlepath")
                }
                NavigationLink { LedgerView() } label: {
                    Label("Payment ledger", systemImage: "banknote")
                }
                NavigationLink { OperationsView() } label: {
                    Label("Operations", systemImage: "wrench.and.screwdriver")
                }
            }
            Section("Account") {
                NavigationLink { InventoryView() } label: {
                    Label("Inventory", systemImage: "archivebox")
                }
                NavigationLink { PromotionsView() } label: {
                    Label("Promotions", systemImage: "tag")
                }
                NavigationLink { EarningsView() } label: {
                    Label("Earnings", systemImage: "chart.line.uptrend.xyaxis")
                }
                NavigationLink { ProfileView() } label: {
                    Label("Profile", systemImage: "building.2")
                }
            }
        }
        .navigationTitle("More")
    }
}
