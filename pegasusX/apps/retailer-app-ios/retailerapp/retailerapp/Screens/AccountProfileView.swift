import SwiftUI

struct AccountProfileView: View {
    @State private var vm = AccountProfileViewModel()

    var body: some View {
        @Bindable var vm = vm
        Form {
            if let errorMessage = vm.errorMessage {
                Section {
                    Text(errorMessage)
                        .font(.footnote)
                        .foregroundStyle(AppTheme.destructive)
                        .accessibilityElement(children: .combine)
                        .accessibilityLabel("Error: \(errorMessage)")
                }
            }
            if let saveMessage = vm.saveMessage {
                Section {
                    Text(saveMessage)
                        .font(.footnote)
                        .foregroundStyle(AppTheme.success)
                        .accessibilityElement(children: .combine)
                        .accessibilityLabel(saveMessage)
                }
            }

            Section {
                TextField("mobile_retailer.ui.entity_name", text: $vm.name)
                    .accessibilityLabel("Entity name")
                TextField("retailer_desktop.residual.text.company", text: $vm.company)
                    .accessibilityLabel("Company name")
                TextField("common.field.phone", text: $vm.phone)
                    .disabled(true)
                    .accessibilityLabel("Phone number")
            } header: {
                Text("supplier_portal.auth.register.steps.business")
            } footer: {
                Text("mobile_retailer.ui.receiving_hours_feed_dispatch_sla_scheduling_for_your_deliveries")
            }

            Section("Receiving window") {
                TextField("mobile_retailer.ui.opens_hh_mm", text: $vm.receivingWindowOpen)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                    .keyboardType(.numbersAndPunctuation)
                    .onChange(of: vm.receivingWindowOpen) { _, _ in vm.clearOpenWindowError() }
                    .accessibilityLabel("Receiving window opens HH:MM")

                if let openWindowError = vm.openWindowError {
                    Text(openWindowError)
                        .font(.caption)
                        .foregroundStyle(AppTheme.error)
                }

                TextField("mobile_retailer.ui.closes_hh_mm", text: $vm.receivingWindowClose)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                    .keyboardType(.numbersAndPunctuation)
                    .onChange(of: vm.receivingWindowClose) { _, _ in vm.clearCloseWindowError() }
                    .accessibilityLabel("Receiving window closes HH:MM")

                if let closeWindowError = vm.closeWindowError {
                    Text(closeWindowError)
                        .font(.caption)
                        .foregroundStyle(AppTheme.error)
                }
            }

            Section {
                Button {
                    Task { await vm.saveProfile() }
                } label: {
                    HStack {
                        Spacer()
                        if vm.isSaving {
                            ProgressView()
                        } else {
                            Text("mobile_retailer.ui.save_profile")
                                .font(.system(.body, design: .rounded, weight: .semibold))
                        }
                        Spacer()
                    }
                    .pegasusMinimumHitTarget()
                }
                .disabled(vm.isLoading || vm.isSaving)
                .accessibilityLabel("Save profile")
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .navigationTitle("supplier_portal.auth.register.steps.account")
        .navigationBarTitleDisplayMode(.inline)
        .task { await vm.loadProfile() }
        .refreshable { await vm.loadProfile() }
        .overlay {
            if vm.isLoading && vm.name.isEmpty && vm.company.isEmpty {
                ProgressView()
            }
        }
    }
}
