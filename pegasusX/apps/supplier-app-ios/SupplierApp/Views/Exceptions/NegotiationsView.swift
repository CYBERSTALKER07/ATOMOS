import SwiftUI

/// Quantity negotiation is product-disabled ecosystem-wide.
/// Delivery-time driver propose → supplier resolve is gated off (410 / empty list).
/// Not a substitute for shop-closed, claims, or missing-items.
struct NegotiationsView: View {
    var body: some View {
        SupplierEmptyView(
            title: "Negotiations disabled",
            message: "Quantity negotiation is not available. Use shop-closed, claims, or missing-items for delivery exceptions."
        )
        .navigationTitle("mobile_supplier.ui.negotiations")
    }
}
