import SwiftUI

struct BusinessSetupView: View {
    @Environment(TokenStore.self) private var tokenStore
    @State private var vm = OnboardingViewModel()

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    Text("mobile_supplier.ui.provide_tax_and_headquarters_information_to_complete_supplier_re")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                }

                Section("Tax & registration") {
                    TextField("supplier_portal.residual.text.tax_id_vat_tin", text: $vm.taxId)
                    TextField("supplier_portal.residual.text.company_registration_number", text: $vm.registrationNumber)
                }

                Section("Location") {
                    TextField("mobile_supplier.ui.headquarters_address", text: $vm.headquartersAddress)
                    TextField("supplier_portal.analytics.demand.signals.text.city", text: $vm.city)
                    TextField("supplier_portal.residual.text.postal_code", text: $vm.postalCode)
                }

                if let error = vm.error {
                    Section { Text(error).foregroundStyle(SupplierTheme.destructive) }
                }

                Section {
                    Button(vm.loading ? "Saving…" : "Save & continue") {
                        Task { _ = await vm.setupBusiness(tokenStore: tokenStore) }
                    }
                    .disabled(vm.loading)

                    Button("common.action.skip_for_now", role: .cancel) {
                        tokenStore.markRegistered(true)
                    }
                    .disabled(vm.loading)
                }
            }
            .navigationTitle("mobile_supplier.ui.business_setup")
        }
    }
}
