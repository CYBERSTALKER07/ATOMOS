import SwiftUI

struct BusinessSetupView: View {
    @Environment(TokenStore.self) private var tokenStore
    @State private var vm = OnboardingViewModel()

    var body: some View {
        Form {
            Section {
                Text("mobile_supplier.ui.provide_tax_and_headquarters_information_to_complete_supplier_re")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
            }

            Section("mobile_supplier.chrome.tax_and_registration") {
                HStack(spacing: SupplierTheme.spacingSM) {
                    TextField("supplier_portal.onboarding_inn.tax_id_placeholder", text: $vm.taxId)
                        .keyboardType(.numberPad)
                        .onChange(of: vm.taxId) { _, newValue in
                            if vm.innLookup?.normalizedInn != String(newValue.filter(\.isNumber)) {
                                vm.innLookup = nil
                                vm.innError = nil
                            }
                        }

                    Button {
                        Task { await vm.verifyINN() }
                    } label: {
                        Text(LocalizedStringKey(
                            vm.innChecking
                                ? "supplier_portal.onboarding_inn.verifying"
                                : "supplier_portal.onboarding_inn.verify_button"
                        ))
                        .font(.caption.weight(.semibold))
                    }
                    .buttonStyle(.bordered)
                    .disabled(vm.innChecking || vm.normalizedINN.count != 9)
                }

                if let lookup = vm.innLookup, lookup.status.uppercased() != "UNAVAILABLE" && lookup.status.uppercased() != "INVALID" && lookup.status.uppercased() != "NOT_FOUND" {
                    VStack(alignment: .leading, spacing: SupplierTheme.spacingXS) {
                        HStack(spacing: SupplierTheme.spacingXS) {
                            Image(systemName: "checkmark.seal.fill")
                                .foregroundStyle(SupplierTheme.success)
                            Text(LocalizedStringKey("supplier_portal.onboarding_inn.verified_badge"))
                                .font(.caption.weight(.semibold))
                                .foregroundStyle(SupplierTheme.success)
                        }
                        if let legalName = lookup.legalName, !legalName.isEmpty {
                            Text(legalName)
                                .font(.subheadline.weight(.semibold))
                        }
                        if let addr = lookup.address, !addr.isEmpty {
                            Text(addr)
                                .font(.caption)
                                .foregroundStyle(SupplierTheme.secondaryLabel)
                        }
                        if let existingId = lookup.existingTenantId, !existingId.isEmpty {
                            VStack(alignment: .leading, spacing: SupplierTheme.spacingXS) {
                                Text(LocalizedStringKey("supplier_portal.onboarding_inn.existing_company_title"))
                                    .font(.caption.bold())
                                    .foregroundStyle(SupplierTheme.warning)
                                Text(L10n.format(
                                    "supplier_portal.onboarding_inn.existing_company_message",
                                    lookup.existingLegalName ?? lookup.legalName ?? existingId,
                                    lookup.normalizedInn
                                ))
                                .font(.caption)
                                .foregroundStyle(SupplierTheme.secondaryLabel)
                            }
                            .padding(.top, SupplierTheme.spacingXS)
                        }
                    }
                }

                if let innError = vm.innError {
                    Text(innError)
                        .font(.caption)
                        .foregroundStyle(SupplierTheme.destructive)
                }

                if let conflict = vm.joinRequestConflict {
                    VStack(alignment: .leading, spacing: SupplierTheme.spacingXS) {
                        Text(LocalizedStringKey("supplier_portal.onboarding_inn.existing_company_title"))
                            .font(.caption.bold())
                            .foregroundStyle(SupplierTheme.warning)
                        Text(LocalizedStringKey("supplier_portal.onboarding_inn.join_request_sent"))
                            .font(.caption)
                            .foregroundStyle(SupplierTheme.label)
                        if let reqId = conflict.requestId, !reqId.isEmpty {
                            Text("Request ID: \(reqId)")
                                .font(.caption2.monospaced())
                                .foregroundStyle(SupplierTheme.secondaryLabel)
                        }
                    }
                    .padding(.vertical, SupplierTheme.spacingXS)
                }

                TextField("supplier_portal.residual.text.company_registration_number", text: $vm.registrationNumber)
            }

            Section("mobile_supplier.chrome.location") {
                TextField("mobile_supplier.ui.headquarters_address", text: $vm.headquartersAddress)
                TextField("supplier_portal.analytics.demand.signals.text.city", text: $vm.city)
                TextField("supplier_portal.residual.text.postal_code", text: $vm.postalCode)
            }

            if let error = vm.error {
                Section { Text(error).foregroundStyle(SupplierTheme.destructive) }
            }

            Section {
                Button(LocalizedStringKey(vm.loading ? "mobile_supplier.chrome.saving" : "mobile_supplier.chrome.save_and_continue")) {
                    PegasusHaptics.light()
                    Task { _ = await vm.setupBusiness(tokenStore: tokenStore) }
                }
                .frame(minHeight: 44)
                .disabled(vm.loading)

                Button("common.action.skip_for_now", role: .cancel) {
                    tokenStore.markRegistered(true)
                }
                .frame(minHeight: 44)
                .disabled(vm.loading)
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .navigationTitle("mobile_supplier.ui.business_setup")
    }
}
