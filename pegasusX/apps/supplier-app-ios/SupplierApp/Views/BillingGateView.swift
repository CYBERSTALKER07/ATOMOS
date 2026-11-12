import SwiftUI

struct BillingGateView: View {
    @Environment(TokenStore.self) private var tokenStore
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @State private var vm = BillingGateViewModel()

    var body: some View {
        @Bindable var vm = vm
        NavigationStack {
            Form {
                Section {
                    Text("mobile_supplier.ui.complete_billing_to_unlock_payouts_and_gateway_routing_you_can_s")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                }

                Section("mobile_supplier.chrome.bank_account") {
                    TextField("supplier_portal.residual.text.bank_name", text: $vm.bankName)
                    TextField("mobile_supplier.ui.account_holder", text: $vm.accountHolder)
                    TextField("supplier_portal.residual.text.account_number", text: $vm.accountNumber)
                    TextField("supplier_portal.residual.text.swift_bic", text: $vm.swiftBic)
                    TextField("mobile_supplier.ui.iban_optional", text: $vm.iban)
                }

                Section("mobile_supplier.chrome.payment_gateways") {
                    ForEach($vm.gatewayOptions) { $option in
                        Toggle(option.id, isOn: $option.isSelected)
                            .onChange(of: option.isSelected) { _, _ in
                                PegasusHaptics.selection()
                            }
                    }
                }

                if let error = vm.error {
                    Section {
                        Text(error)
                            .foregroundStyle(SupplierTheme.destructive)
                            .font(.caption)
                    }
                }

                Section {
                    Button {
                        PegasusHaptics.light()
                        Task { await vm.submit(tokenStore: tokenStore) }
                    } label: {
                        HStack {
                            Spacer()
                            if vm.loading { ProgressView() } else { Text("supplier_portal.residual.text.save_and_continue") }
                            Spacer()
                        }
                        .frame(minHeight: 44)
                    }
                    .disabled(vm.loading || !vm.canSubmit)

                    Button("common.action.skip_for_now", role: .cancel) {
                        tokenStore.dismissBillingGate()
                    }
                    .frame(minHeight: 44)
                    .disabled(vm.loading)
                }
            }
            .scrollDismissesKeyboard(.interactively)
            .navigationTitle("mobile_supplier.ui.billing_setup")
            .frame(maxWidth: horizontalSizeClass == .regular ? 560 : .infinity)
            .frame(maxWidth: .infinity)
        }
    }
}
