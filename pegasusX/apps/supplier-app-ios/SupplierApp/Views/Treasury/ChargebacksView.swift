import SwiftUI

struct ChargebacksView: View {
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    @State private var orderId = ""
    @State private var retailerId = ""
    @State private var gateway = "ADYEN"
    @State private var amount = ""
    @State private var currency = "UZS"
    @State private var sessionId = ""
    @State private var busy = false
    @State private var chargebackMessage: String?
    @State private var reversalMessage: String?
    @State private var error: String?

    private let gateways = ["ADYEN", "GLOBAL_PAY", "STRIPE", "PAYME", "CLICK", "CASH"]

    var body: some View {
        Form {
            Section {
                Text("Record payment disputes and reversals against the durable finance ledger.")
                    .font(.footnote)
                    .foregroundStyle(.secondary)
            }

            Section("Record chargeback") {
                TextField("Order ID", text: $orderId)
                    .textInputAutocapitalization(.never)
                TextField("Retailer ID", text: $retailerId)
                    .textInputAutocapitalization(.never)
                Picker("Gateway", selection: $gateway) {
                    ForEach(gateways, id: \.self) { Text($0).tag($0) }
                }
                TextField("Amount (minor units)", text: $amount)
                    .keyboardType(.numberPad)
                TextField("Currency", text: $currency)
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
                TextField("Session ID", text: $sessionId)
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
        .navigationTitle("Chargebacks")
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
                idempotencyKey: UUID().uuidString
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
                idempotencyKey: UUID().uuidString
            )
            reversalMessage = "Reversal recorded (\(response.status))."
            sessionId = ""
        } catch {
            self.error = error.localizedDescription
        }
    }
}
