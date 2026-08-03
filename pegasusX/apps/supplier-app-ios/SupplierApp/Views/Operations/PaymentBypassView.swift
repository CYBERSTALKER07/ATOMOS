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
                TextField("Order ID (AWAITING_PAYMENT)", text: $orderId)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                TextField("Reason (optional)", text: $bypassReason)
                Button(bypassing ? "Issuing…" : "Issue bypass token") {
                    showBypassConfirm = true
                }
                .disabled(bypassing || orderId.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                if let bypassToken {
                    Text("Driver token: \(bypassToken)")
                        .font(.footnote.monospaced())
                }
            }
        }
    }
}
