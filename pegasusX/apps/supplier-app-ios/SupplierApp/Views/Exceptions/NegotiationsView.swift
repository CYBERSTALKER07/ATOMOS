import SwiftUI

/// Quantity negotiation is product-disabled ecosystem-wide.
/// Delivery-time driver propose → supplier resolve is gated off (410 / empty list).
/// Not a substitute for shop-closed, claims, or missing-items.
struct NegotiationsView: View {
    var body: some View {
        SupplierEmptyView(
            title: "mobile_supplier.empty.negotiations_disabled",
            message: "mobile_supplier.empty.negotiations_message"
        )
        .navigationTitle("mobile_supplier.ui.negotiations")
    }
}
