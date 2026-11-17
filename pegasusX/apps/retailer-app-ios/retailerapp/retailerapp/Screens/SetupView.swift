import SwiftUI

struct SetupView: View {
    @Environment(AuthManager.self) private var auth
    @State private var vm = OnboardingViewModel()

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: AppTheme.spacingXL) {
                    header
                    progressBar

                    if vm.step == .tax {
                        taxStep
                    } else {
                        addressStep
                    }

                    if let error = vm.errorMessage {
                        Text(error)
                            .font(.system(.caption, design: .rounded, weight: .semibold))
                            .foregroundStyle(AppTheme.destructive)
                            .multilineTextAlignment(.center)
                            .padding(.horizontal)
                            .accessibilityElement(children: .combine)
                            .accessibilityLabel("Error: \(error)")
                    }
                }
                .padding(AppTheme.spacingXL)
            }
            .scrollDismissesKeyboard(.interactively)
            .background(AppTheme.background)
            .navigationTitle("mobile_retailer.ui.company_profile")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("common.action.sign_out") { auth.logout() }
                        .font(.system(.subheadline, design: .rounded, weight: .semibold))
                        .pegasusMinimumHitTarget()
                        .accessibilityLabel("Sign out")
                }
            }
        }
    }

    private var header: some View {
        VStack(spacing: AppTheme.spacingMD) {
            ZStack {
                RoundedRectangle(cornerRadius: AppTheme.radiusCard)
                    .fill(AppTheme.accent)
                    .frame(width: 64, height: 64)
                Image(systemName: "building.2.fill")
                    .font(.title.weight(.semibold))
                    .foregroundStyle(.white)
            }
            Text(vm.step == .tax ? "Provide your business identity" : "Where should we deliver?")
                .font(.system(.subheadline, design: .rounded))
                .foregroundStyle(AppTheme.textSecondary)
                .multilineTextAlignment(.center)
        }
        .accessibilityElement(children: .combine)
    }

    private var progressBar: some View {
        HStack(spacing: 8) {
            Capsule().fill(AppTheme.accent).frame(height: 4)
            Capsule().fill(vm.step == .address ? AppTheme.accent : AppTheme.separator.opacity(0.4)).frame(height: 4)
        }
    }

    private var taxStep: some View {
        VStack(spacing: AppTheme.spacingLG) {
            if vm.joinRequestSent {
                joinRequestConfirmationCard
            } else {
                Text("retailer_desktop.tax_identity.subtitle")
                    .font(.caption)
                    .foregroundStyle(AppTheme.textSecondary)
                    .frame(maxWidth: .infinity, alignment: .leading)

                informalShopToggle

                if !vm.isInformalShop {
                    innFieldRow
                    if let lookup = vm.innLookup, lookup.isVerifiedFound {
                        innVerifiedCard(lookup)
                    }
                    if let innErr = vm.innError, !innErr.isEmpty {
                        Text(innErr)
                            .font(.caption)
                            .foregroundStyle(AppTheme.error)
                            .frame(maxWidth: .infinity, alignment: .leading)
                    }
                }

                continueFromTaxButton
            }
        }
    }

    private var informalShopToggle: some View {
        Toggle(isOn: $vm.isInformalShop) {
            VStack(alignment: .leading, spacing: 2) {
                Text("retailer_desktop.tax_identity.mode_informal_phone")
                    .font(.subheadline.weight(.semibold))
                Text("retailer_desktop.tax_identity.legal_status_informal")
                    .font(.caption)
                    .foregroundStyle(AppTheme.textSecondary)
            }
        }
        .padding()
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusSM))
        .onChange(of: vm.isInformalShop) { _, isInformal in
            if isInformal {
                vm.innError = nil
                vm.innLookup = nil
                vm.errorMessage = nil
            }
        }
    }

    private var innFieldRow: some View {
        HStack(spacing: 8) {
            setupField(
                title: String(localized: "retailer_desktop.tax_identity.tax_id_label"),
                text: $vm.taxId,
                icon: "doc.text"
            )
            Button {
                Task { await vm.verifyINN() }
            } label: {
                Group {
                    if vm.innChecking {
                        ProgressView()
                            .frame(width: 80, height: 44)
                    } else {
                        Text("supplier_portal.onboarding_inn.verify_button")
                            .font(.caption.weight(.semibold))
                            .padding(.horizontal, 12)
                            .frame(height: 44)
                    }
                }
            }
            .buttonStyle(.borderedProminent)
            .disabled(vm.innChecking || vm.taxId.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
        }
    }

    private func innVerifiedCard(_ lookup: RetailerINNLookupResponse) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Label("supplier_portal.onboarding_inn.verified_badge", systemImage: "checkmark.seal.fill")
                .font(.caption.weight(.bold))
                .foregroundStyle(AppTheme.success)
            if let legal = lookup.legalName, !legal.isEmpty {
                Text(legal)
                    .font(.subheadline.weight(.semibold))
            }
            if let addr = lookup.address, !addr.isEmpty {
                Text(addr)
                    .font(.caption)
                    .foregroundStyle(AppTheme.textSecondary)
            }
            if lookup.existingJoinEligible == true {
                Text("retailer_desktop.tax_identity.existing_retailer_join_hint")
                    .font(.caption)
                    .foregroundStyle(AppTheme.warning)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding()
        .background(AppTheme.success.opacity(0.08))
        .clipShape(.rect(cornerRadius: AppTheme.radiusSM))
    }

    private var continueFromTaxButton: some View {
        Button {
            Task { _ = await vm.advanceFromTaxAsync() }
        } label: {
            Group {
                if vm.isInformalShop {
                    Text("retailer_desktop.tax_identity.save_informal")
                } else {
                    Text("supplier_portal.bulk_import_wizard.text.continue")
                }
            }
            .font(.system(.headline, design: .rounded, weight: .bold))
            .frame(maxWidth: .infinity)
            .padding(.vertical, 14)
            .background(AppTheme.accent)
            .foregroundStyle(.white)
            .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
        }
        .pegasusMinimumHitTarget()
        .accessibilityLabel("Continue to delivery address step")
    }

    private var joinRequestConfirmationCard: some View {
        VStack(alignment: .leading, spacing: 10) {
            Label("supplier_portal.onboarding_inn.existing_company_title", systemImage: "person.badge.clock")
                .font(.headline)
                .foregroundStyle(AppTheme.warning)
            Text("supplier_portal.onboarding_inn.join_request_sent")
                .font(.subheadline)
                .foregroundStyle(AppTheme.textPrimary)
            if let conflict = vm.joinRequestConflict {
                if let legal = conflict.legalName, !legal.isEmpty {
                    Text(legal)
                        .font(.subheadline.weight(.semibold))
                }
                if let reqId = conflict.requestId, !reqId.isEmpty {
                    Text("Request ID: \(reqId)")
                        .font(.caption.monospaced())
                        .foregroundStyle(AppTheme.textSecondary)
                }
            }
            Button {
                vm.joinRequestSent = false
                vm.joinRequestConflict = nil
                vm.step = .tax
            } label: {
                Text("common.action.back")
                    .font(.system(.subheadline, design: .rounded, weight: .semibold))
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 10)
                    .background(AppTheme.surfaceElevated)
                    .foregroundStyle(AppTheme.textPrimary)
                    .clipShape(.rect(cornerRadius: AppTheme.radiusSM))
            }
        }
        .padding()
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
    }

    private var addressStep: some View {
        VStack(spacing: AppTheme.spacingLG) {
            if vm.joinRequestSent {
                joinRequestConfirmationCard
            } else {
            setupField(title: "Billing Address", text: $vm.billingAddress, icon: "building")
            setupField(title: "Shipping Address", text: $vm.shippingAddress, icon: "shippingbox")
            setupField(title: "City", text: $vm.city, icon: "mappin.and.ellipse")
            setupField(title: "Postal Code", text: $vm.postalCode, icon: "number")

            HStack(spacing: 12) {
                Button {
                    Haptics.selection()
                    vm.step = .tax
                } label: {
                    Text("common.action.back")
                        .font(.system(.headline, design: .rounded, weight: .semibold))
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 14)
                        .background(AppTheme.surfaceElevated)
                        .foregroundStyle(AppTheme.textPrimary)
                        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
                }
                .pegasusMinimumHitTarget()
                .accessibilityLabel("Back to tax step")

                Button {
                    Task {
                        if await vm.submit() {
                            Haptics.success()
                        } else if vm.joinRequestSent {
                            Haptics.selection()
                        } else {
                            Haptics.error()
                        }
                    }
                } label: {
                    Group {
                        if vm.isSubmitting {
                            ProgressView().tint(.white)
                        } else {
                            Text("mobile_retailer.ui.complete_setup")
                                .font(.system(.headline, design: .rounded, weight: .bold))
                        }
                    }
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 14)
                    .background(AppTheme.accent)
                    .foregroundStyle(.white)
                    .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
                }
                .pegasusMinimumHitTarget()
                .disabled(vm.isSubmitting)
                .accessibilityLabel("Complete setup")
            }
            }
        }
    }

    private func setupField(title: String, text: Binding<String>, icon: String) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(title.uppercased())
                .font(.system(.caption2, design: .rounded, weight: .bold))
                .foregroundStyle(AppTheme.textTertiary)
            HStack(spacing: 10) {
                Image(systemName: icon)
                    .foregroundStyle(AppTheme.textTertiary)
                TextField(title, text: text)
                    .font(.system(.body, design: .rounded, weight: .semibold))
            }
            .padding(14)
            .background(AppTheme.cardBackground)
            .clipShape(.rect(cornerRadius: AppTheme.radiusSM))
            .overlay(
                RoundedRectangle(cornerRadius: AppTheme.radiusSM)
                    .stroke(AppTheme.separator.opacity(0.3), lineWidth: 1)
            )
            .accessibilityElement(children: .combine)
            .accessibilityLabel(title)
        }
    }
}

#Preview {
    SetupView()
        .environment(AuthManager.shared)
}
