import SwiftUI

struct ReturnPolicySettingsView: View {
    @State private var vm = ReturnPolicySettingsViewModel()

    var body: some View {
        @Bindable var vm = vm
        Group {
            if vm.loading {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else {
                Form {
                    Section {
                        Text("mobile_warehouse.ui.reverse_dock_sla_and_optional_retailer_claim_window_override_ove")
                            .font(.footnote)
                            .foregroundStyle(.secondary)
                    }

                    if let error = vm.error {
                        Section {
                            Text(error).foregroundStyle(.red)
                        }
                    }
                    if vm.saved {
                        Section {
                            Text("supplier_portal.settings.return_policy.text.return_policy_saved")
                                .foregroundStyle(.green)
                        }
                    }

                    Section("Reverse dock") {
                        TextField("mobile_warehouse.ui.sla_hours", text: $vm.reverseSla)
                            .keyboardType(.numberPad)
                            .onChange(of: vm.reverseSla) { _, newValue in
                                vm.sanitizeReverseSla(newValue)
                            }
                    }

                    Section("Retailer window") {
                        Toggle("Override retailer claim filing window (lengthen only)", isOn: $vm.canOverride)
                            .frame(minHeight: 44)
                            .onChange(of: vm.canOverride) { _, _ in
                                PegasusHaptics.selection()
                            }
                        if vm.canOverride {
                            TextField("mobile_warehouse.ui.retailer_file_window_hours", text: $vm.retailerWindow)
                                .keyboardType(.numberPad)
                                .onChange(of: vm.retailerWindow) { _, newValue in
                                    vm.sanitizeRetailerWindow(newValue)
                                }
                        }
                    }

                    Section {
                        Button(vm.saving ? "Saving…" : "Save return policy") {
                            PegasusHaptics.medium()
                            Task { await vm.save() }
                        }
                        .disabled(vm.saving)
                        .frame(minHeight: 44)
                        .contentShape(Rectangle())
                        .accessibilityLabel("Save return policy")
                    }
                }
                .scrollDismissesKeyboard(.interactively)
            }
        }
        .navigationTitle("warehouse_portal.settings.return_policy_settings_section.text.returns_and_reverse_sla")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") {
                    PegasusHaptics.light()
                    Task { await vm.load() }
                }
                .frame(minWidth: 44, minHeight: 44)
                .contentShape(Rectangle())
                .accessibilityLabel("Refresh return policy settings")
            }
        }
        .task { await vm.load() }
    }
}
