import SwiftUI

struct ChargebacksView: View {
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    @State private var orderId = ""
    @State private var retailerId = ""
    @State private var gateway = "ADYEN"
    @State private var amount = ""
    @State private var currency = packCurrency(MarketPackStore.pack)
    @State private var sessionId = ""
    @State private var busy = false
    @State private var chargebackMessage: String?
    @State private var reversalMessage: String?
    @State private var error: String?

    private let gateways = ["ADYEN", "GLOBAL_PAY", "STRIPE", "PAYME", "CLICK", "CASH"]

    var body: some View {
        Form {
            Section {
                Text("mobile_supplier.ui.record_payment_disputes_and_reversals_against_the_durable_financ")
                    .font(.footnote)
                    .foregroundStyle(.secondary)
            }

            Section("Record chargeback") {
                TextField("supplier_portal.admin.control_center.field.order_id", text: $orderId)
                    .textInputAutocapitalization(.never)
                TextField("supplier_portal.chargebacks.text.retailer_id", text: $retailerId)
                    .textInputAutocapitalization(.never)
                Picker("Gateway", selection: $gateway) {
                    ForEach(gateways, id: \.self) { Text($0).tag($0) }
                }
                TextField("supplier_portal.chargebacks.text.amount_minor_units", text: $amount)
                    .keyboardType(.numberPad)
                TextField("supplier_portal.chargebacks.text.currency", text: $currency)
                    .textInputAutocapitalization(.characters)
                Button(busy ? "Recording…" : "Record chargeback") {
                    Task { await submitChargeback() }
                }
                .disabled(busy || orderId.isEmpty || retailerId.isEmpty || amount.isEmpty)
                if let chargebackMessage {
                    Text(chargebackMessage).foregroundStyle(SupplierTheme.success)
                }
            }

            Section("Reverse chargeback") {
                TextField("supplier_portal.admin.control_center.field.session_id", text: $sessionId)
                    .textInputAutocapitalization(.never)
                Button(busy ? "Reversing…" : "Record reversal") {
                    Task { await submitReversal() }
                }
                .disabled(busy || sessionId.isEmpty)
                if let reversalMessage {
                    Text(reversalMessage).foregroundStyle(SupplierTheme.success)
                }
            }

            if let error {
                Section { Text(error).foregroundStyle(SupplierTheme.destructive) }
            }
        }
        .navigationTitle("portal.nav.chargebacks")
        .onChange(of: realtimeHub.reconnectEpoch) { _, _ in
            if busy {
                busy = false
                error = "Connection restored — verify chargeback status before retrying."
            }
        }
    }

    private func submitChargeback() async {
        guard let amountMinor = Int64(amount) else {
            error = "Amount must be a number."
            return
        }
        busy = true
        error = nil
        chargebackMessage = nil
        defer { busy = false }
        do {
            let response = try await SupplierService.recordChargeback(
                PaymentChargebackRequest(
                    orderId: orderId,
                    retailerId: retailerId,
                    gateway: gateway,
                    amount: amountMinor,
                    currency: currency
                ),
                idempotencyKey: SupplierIdempotencyKeys.chargeback(orderId: orderId, reason: orderId)
            )
            chargebackMessage = "Chargeback recorded (\(response.status))."
            orderId = ""
            retailerId = ""
            amount = ""
        } catch {
            self.error = error.localizedDescription
        }
    }

    private func submitReversal() async {
        busy = true
        error = nil
        reversalMessage = nil
        defer { busy = false }
        do {
            let response = try await SupplierService.recordChargebackReversal(
                PaymentChargebackReversalRequest(sessionId: sessionId),
                idempotencyKey: SupplierIdempotencyKeys.chargebackReversal(chargebackId: sessionId, reason: sessionId)
            )
            reversalMessage = "Reversal recorded (\(response.status))."
            sessionId = ""
        } catch {
            self.error = error.localizedDescription
        }
    }
}
