import SwiftUI

struct ChargebacksView: View {
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    @State private var viewModel = ChargebacksViewModel()

    private let gateways = ["ADYEN", "GLOBAL_PAY", "STRIPE", "PAYME", "CLICK", "CASH"]

    var body: some View {
        @Bindable var vm = viewModel
        Form {
            Section {
                Text("mobile_supplier.ui.record_payment_disputes_and_reversals_against_the_durable_financ")
                    .font(.footnote)
                    .foregroundStyle(.secondary)
            }

            Section("mobile_supplier.chrome.record_chargeback") {
                TextField("supplier_portal.admin.control_center.field.order_id", text: $vm.orderId)
                    .textInputAutocapitalization(.never)
                TextField("supplier_portal.chargebacks.text.retailer_id", text: $vm.retailerId)
                    .textInputAutocapitalization(.never)
                Picker("mobile_supplier.chrome.gateway", selection: $vm.gateway) {
                    ForEach(gateways, id: \.self) { Text($0).tag($0) }
                }
                TextField("supplier_portal.chargebacks.text.amount_minor_units", text: $vm.amount)
                    .keyboardType(.numberPad)
                TextField("supplier_portal.chargebacks.text.currency", text: $vm.currency)
                    .textInputAutocapitalization(.characters)
                Button(LocalizedStringKey(viewModel.busy ? "mobile_supplier.chrome.recording" : "mobile_supplier.chrome.record_chargeback")) {
                    Task { await viewModel.submitChargeback() }
                }
                .frame(minHeight: 44)
                .disabled(viewModel.busy || viewModel.orderId.isEmpty || viewModel.retailerId.isEmpty || viewModel.amount.isEmpty)
                if let chargebackMessage = viewModel.chargebackMessage {
                    Text(chargebackMessage).foregroundStyle(SupplierTheme.success)
                }
            }

            Section("mobile_supplier.chrome.reverse_chargeback") {
                TextField("supplier_portal.admin.control_center.field.session_id", text: $vm.sessionId)
                    .textInputAutocapitalization(.never)
                Button(LocalizedStringKey(viewModel.busy ? "mobile_supplier.chrome.reversing" : "mobile_supplier.chrome.record_reversal")) {
                    Task { await viewModel.submitReversal() }
                }
                .frame(minHeight: 44)
                .disabled(viewModel.busy || viewModel.sessionId.isEmpty)
                if let reversalMessage = viewModel.reversalMessage {
                    Text(reversalMessage).foregroundStyle(SupplierTheme.success)
                }
            }

            if let error = viewModel.error {
                Section { Text(error).foregroundStyle(SupplierTheme.destructive) }
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .navigationTitle("portal.nav.chargebacks")
        .onChange(of: realtimeHub.reconnectEpoch) { _, _ in
            viewModel.handleReconnect()
        }
    }
}
