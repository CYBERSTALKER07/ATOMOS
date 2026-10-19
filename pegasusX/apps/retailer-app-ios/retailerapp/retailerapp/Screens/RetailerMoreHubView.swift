import SwiftUI

/// GS-UN overflow: orders, suppliers, profile stay reachable. Not a 6th tab.
struct RetailerMoreHubView: View {
    var body: some View {
        List {
            Section("Buy") {
                NavigationLink { OrdersView() } label: {
                    Label("Orders", systemImage: "shippingbox")
                }
                NavigationLink { MySuppliersView() } label: {
                    Label("Suppliers", systemImage: "building.2")
                }
            }
            Section("Account") {
                NavigationLink { ProfileView() } label: {
                    Label("Profile", systemImage: "person.circle")
                }
            }
        }
        .navigationTitle("More")
    }
}
