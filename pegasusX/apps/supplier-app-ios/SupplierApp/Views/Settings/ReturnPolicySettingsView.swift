import SwiftUI

struct ReturnPolicySettingsView: View {
    @State private var vm = ReturnPolicySettingsViewModel()

    var body: some View {
        @Bindable var vm = vm
        Group {
            if vm.loading && !vm.hasLoaded {
                SupplierLoadingView(title: "mobile_supplier.loading.return_policy")
            } else {
                Form {
                    Section {
                        Text("supplier_portal.residual.text.claim_filing_windows_applied_when_orders_complete_retailers_see_")
                            .font(.footnote)
                            .foregroundStyle(.secondary)
                        if !vm.sourceHint.isEmpty {
                            Text(L10n.format("mobile_supplier.ui.source_hint_sourcehint_2", "\(vm.sourceHint)"))
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                    }

                    if let error = vm.error {
                        Section {
                            Text(error).foregroundStyle(SupplierTheme.destructive)
                        }
                    }
                    if vm.saved {
                        Section {
                            Text("supplier_portal.settings.return_policy.text.return_policy_saved").foregroundStyle(SupplierTheme.success)
                        }
                    }

                    Section("mobile_supplier.chrome.default_claim_window") {
                        HStack {
                            ForEach(ReturnPolicySettingsViewModel.hourPresets, id: \.self) { preset in
                                Button(L10n.format("mobile_supplier.ui.preseth", "\(preset)")) {
                                    vm.selectPreset(preset)
                                }
                                .buttonStyle(.bordered)
                                .frame(minWidth: 44, minHeight: 44)
                                .tint(vm.hours == preset ? Color.accentColor : .secondary)
                            }
                        }
                        TextField("mobile_supplier.ui.custom_1_168", text: $vm.hoursText)
                            .keyboardType(.numberPad)
                            .onChange(of: vm.hoursText) { _, newValue in
                                vm.sanitizeHoursText(newValue)
                            }
                        Text(L10n.format("mobile_supplier.ui.preview_retailers_may_file_claims_for_isempty_hourstexth_after_delivery", "\(vm.hoursText.isEmpty ? "—" : vm.hoursText)"))
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }

                    Section("mobile_supplier.chrome.concealed_damage_optional") {
                        TextField("mobile_supplier.ui.hours_same_as_default_if_empty", text: $vm.concealed)
                            .keyboardType(.numberPad)
                            .onChange(of: vm.concealed) { _, newValue in
                                vm.sanitizeConcealed(newValue)
                            }
                    }

                    Section("mobile_supplier.chrome.claims_rules") {
                        Toggle("mobile_supplier.chrome.require_photo_evidence", isOn: $vm.requirePhoto)
                        Toggle("mobile_supplier.chrome.allow_filing_after_window", isOn: $vm.allowExpired)
                    }

                    Section {
                        Button(LocalizedStringKey(vm.saving ? "mobile_supplier.chrome.saving" : "mobile_supplier.chrome.save_return_policy")) {
                            PegasusHaptics.light()
                            Task { await vm.save() }
                        }
                        .frame(minHeight: 44)
                        .disabled(vm.saving)
                    }
                }
                .scrollDismissesKeyboard(.interactively)
            }
        }
        .navigationTitle("portal.nav.return_policy")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") {
                    PegasusHaptics.light()
                    Task { await vm.load(silent: true) }
                }
                .frame(minWidth: 44, minHeight: 44)
            }
        }
        .task { await vm.load() }
    }
}
