import SwiftUI

struct RegisterView: View {
    @Environment(TokenStore.self) private var tokenStore
    @Environment(\.dismiss) private var dismiss
    @State private var vm = OnboardingViewModel()

    var body: some View {
        Form {
            Section {
                Text("Step \(vm.step.rawValue + 1) of \(RegisterStep.allCases.count) — \(vm.step.title)")
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
        .navigationTitle("Register supplier")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .cancellationAction) {
                Button("Cancel") { dismiss() }
            }
            ToolbarItem(placement: .confirmationAction) {
                if vm.step == .profile {
                    Button(vm.loading ? "Creating…" : "Create") {
                        Task { _ = await vm.register(tokenStore: tokenStore) }
                    }
                    .disabled(vm.loading)
                } else {
                    Button("Continue") {
                        _ = vm.advanceStep()
                    }
                    .disabled(vm.loading)
                }
            }
            if vm.step != .identity {
                ToolbarItem(placement: .bottomBar) {
                    Button("Back") { vm.retreatStep() }
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
                    TextField("Phone number", text: $vm.phoneLocal)
                        .keyboardType(.phonePad)
                }
            }
        }
    }

    private var verificationStep: some View {
        Section("Verification code") {
            TextField("6-digit code", text: $vm.otpCode)
                .keyboardType(.numberPad)
            Text("Enter the verification code sent to \(vm.fullPhone).")
                .font(.caption)
                .foregroundStyle(.secondary)
        }
    }

    private var profileStep: some View {
        Group {
            Section("Business") {
                TextField("Legal name", text: $vm.legalName)
                TextField("Contact name", text: $vm.contactName)
                TextField("Email", text: $vm.email)
                    .textContentType(.emailAddress)
                    .keyboardType(.emailAddress)
                    .textInputAutocapitalization(.never)
            }
        }
    }
}
