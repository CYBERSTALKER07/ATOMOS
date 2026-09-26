import SwiftUI

struct PaymentBypassView: View {
    @Binding var orderId: String
    @Binding var bypassReason: String
    let bypassToken: String?
    let bypassing: Bool
    @Binding var showBypassConfirm: Bool

    var body: some View {
        Group {
            Section {
                SupplierSectionHeader(title: "Payment bypass")
            }
            Section {
                TextField("supplier_portal.operations.payment_bypass.text.order_id_awaiting_payment", text: $orderId)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                TextField("warehouse_portal.inventory.text.reason_optional", text: $bypassReason)
                Button(bypassing ? "Issuing…" : "Issue bypass token") {
                    showBypassConfirm = true
                }
                .disabled(bypassing || orderId.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                if let bypassToken {
                    Text(L10n.format("mobile_supplier.ui.driver_token_bypasstoken_2", "\(bypassToken)"))
                        .font(.footnote.monospaced())
                }
            }
        }
    }
}
