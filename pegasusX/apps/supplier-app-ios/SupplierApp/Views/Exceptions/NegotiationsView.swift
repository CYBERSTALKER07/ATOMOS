import SwiftUI

/// Quantity negotiation is product-deferred ecosystem-wide.
struct NegotiationsView: View {
    var body: some View {
        SupplierEmptyView(
            title: "Negotiations disabled",
            message: "Quantity negotiation is not available. Use the exceptions queue for shop-closed and delivery escalations."
        )
        .navigationTitle("Negotiations")
    }
}
