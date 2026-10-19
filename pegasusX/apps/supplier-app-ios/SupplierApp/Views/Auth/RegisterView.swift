import SwiftUI

struct RegisterView: View {
    @Environment(TokenStore.self) private var tokenStore
    @Environment(\.dismiss) private var dismiss
    @State private var vm = OnboardingViewModel()

    var body: some View {
        Form {
            Section {
                Text(L10n.format("mobile_supplier.ui.step_rawvalue_1_of_count_title", "\(vm.step.rawValue + 1)", "\(RegisterStep.allCases.count)", "\(vm.step.title)"))
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
            }

            switch vm.step {
            case .identity:
                identityStep
            case .verification:
                verificationStep
            case .profile:
                profileStep
            }

            if let error = vm.error {
                Section { Text(error).foregroundStyle(SupplierTheme.destructive) }
            }
        }
        .navigationTitle("mobile_supplier.ui.register_supplier")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .cancellationAction) {
                Button("common.action.cancel") { dismiss() }
            }
            ToolbarItem(placement: .confirmationAction) {
                if vm.step == .profile {
                    Button(vm.loading ? "Creating…" : "Create") {
                        Task { _ = await vm.register(tokenStore: tokenStore) }
                    }
                    .disabled(vm.loading)
                } else {
                    Button("supplier_portal.bulk_import_wizard.text.continue") {
                        _ = vm.advanceStep()
                    }
                    .disabled(vm.loading)
                }
            }
            if vm.step != .identity {
                ToolbarItem(placement: .bottomBar) {
                    Button("common.action.back") { vm.retreatStep() }
                }
            }
        }
    }

    private var identityStep: some View {
        Group {
            Section("Country") {
                Picker("Country", selection: $vm.countryCode) {
                    ForEach(OnboardingViewModel.countries) { country in
                        Text(country.name).tag(country.id)
                    }
                }
            }
            Section("Phone") {
                HStack {
                    Text(vm.dialCode).foregroundStyle(.secondary)
                    TextField("mobile_supplier.ui.phone_number", text: $vm.phoneLocal)
                        .keyboardType(.phonePad)
                }
            }
        }
    }

    private var verificationStep: some View {
        Section("Verification code") {
            TextField("mobile_supplier.ui.6_digit_code", text: $vm.otpCode)
                .keyboardType(.numberPad)
            Text(L10n.format("mobile_supplier.ui.enter_the_verification_code_sent_to_fullphone", "\(vm.fullPhone)"))
                .font(.caption)
                .foregroundStyle(.secondary)
        }
    }

    private var profileStep: some View {
        Group {
            Section("Business") {
                TextField("supplier_portal.residual.text.legal_name", text: $vm.legalName)
                TextField("supplier_portal.residual.text.contact_name", text: $vm.contactName)
                TextField("supplier_portal.auth.login.email_label", text: $vm.email)
                    .textContentType(.emailAddress)
                    .keyboardType(.emailAddress)
                    .textInputAutocapitalization(.never)
            }
        }
    }
}
