import SwiftUI

struct BillingGateView: View {
    @Environment(TokenStore.self) private var tokenStore
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @State private var bankName = ""
    @State private var accountHolder = ""
    @State private var accountNumber = ""
    @State private var swiftBic = ""
    @State private var iban = ""
    @State private var gateways: Set<String> = ["CASH"]
    @State private var loading = false
    @State private var error: String?

    private let gatewayOptions = ["GLOBAL_PAY", "ADYEN", "AIRWALLEX", "CASH"]

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    Text("mobile_supplier.ui.complete_billing_to_unlock_payouts_and_gateway_routing_you_can_s")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                }

                Section("Bank account") {
                    TextField("supplier_portal.residual.text.bank_name", text: $bankName)
                    TextField("mobile_supplier.ui.account_holder", text: $accountHolder)
                    TextField("supplier_portal.residual.text.account_number", text: $accountNumber)
                    TextField("supplier_portal.residual.text.swift_bic", text: $swiftBic)
                    TextField("mobile_supplier.ui.iban_optional", text: $iban)
                }

                Section("Payment gateways") {
                    ForEach(gatewayOptions, id: \.self) { gateway in
                        Toggle(gateway, isOn: Binding(
                            get: { gateways.contains(gateway) },
                            set: { on in
                                if on { gateways.insert(gateway) } else { gateways.remove(gateway) }
                            }
                        ))
                    }
                }

                if let error {
                    Section {
                        Text(error)
                            .foregroundStyle(SupplierTheme.destructive)
                            .font(.caption)
                    }
                }

                Section {
                    Button {
                        submit()
                    } label: {
                        HStack {
                            Spacer()
                            if loading { ProgressView() } else { Text("supplier_portal.residual.text.save_and_continue") }
                            Spacer()
                        }
                    }
                    .disabled(loading || !canSubmit)

                    Button("common.action.skip_for_now", role: .cancel) {
                        tokenStore.dismissBillingGate()
                    }
                    .disabled(loading)
                }
            }
            .navigationTitle("mobile_supplier.ui.billing_setup")
            .frame(maxWidth: horizontalSizeClass == .regular ? 560 : .infinity)
            .frame(maxWidth: .infinity)
        }
    }

    private var canSubmit: Bool {
        !bankName.isEmpty && !accountHolder.isEmpty && !accountNumber.isEmpty
            && !swiftBic.isEmpty && !gateways.isEmpty
    }

    private func submit() {
        loading = true
        error = nil
        Task {
            do {
                let resp = try await SupplierService.configureBilling(
                    BillingSetupRequest(
                        bankName: bankName,
                        accountHolder: accountHolder,
                        accountNumber: accountNumber,
                        swiftBic: swiftBic,
                        iban: iban.isEmpty ? nil : iban,
                        selectedGateways: Array(gateways).sorted()
                    )
                )
                tokenStore.markConfigured(resp.isConfigured)
            } catch {
                self.error = error.localizedDescription
            }
            loading = false
        }
    }
}
