import SwiftUI

struct BusinessSetupView: View {
    @Environment(TokenStore.self) private var tokenStore
    @State private var vm = OnboardingViewModel()

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    Text("Provide tax and headquarters information to complete supplier registration.")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                }

                Section("Tax & registration") {
                    TextField("Tax ID (VAT / TIN)", text: $vm.taxId)
                    TextField("Company registration number", text: $vm.registrationNumber)
                }

                Section("Location") {
                    TextField("Headquarters address", text: $vm.headquartersAddress)
                    TextField("City", text: $vm.city)
                    TextField("Postal code", text: $vm.postalCode)
                }

                if let error = vm.error {
                    Section { Text(error).foregroundStyle(SupplierTheme.destructive) }
                }

                Section {
                    Button(vm.loading ? "Saving…" : "Save & continue") {
                        Task { _ = await vm.setupBusiness(tokenStore: tokenStore) }
                    }
                    .disabled(vm.loading)

                    Button("Skip for now", role: .cancel) {
                        tokenStore.markRegistered(true)
                    }
                    .disabled(vm.loading)
                }
            }
            .navigationTitle("Business setup")
        }
    }
}
