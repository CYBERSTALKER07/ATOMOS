import SwiftUI

struct LocationSetupView: View {
    @Environment(TokenStore.self) private var tokenStore
    @State private var vm = LocationSetupViewModel()

    private var hasAssignedWarehouse: Bool { tokenStore.hasAssignedWarehouse }

    var body: some View {
        @Bindable var vm = vm
        Group {
            if vm.loading {
                ProgressView("Loading…")
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else {
                Form {
                    Section {
                        Text(
                            hasAssignedWarehouse
                                ? "Confirm or update your depot address. Changes sync with dispatch and delivery routing."
                                : "Name your warehouse and set the depot address to start operations."
                        )
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                    }
                    if !hasAssignedWarehouse {
                        Section("warehouse_portal.residual.text.warehouse_name") {
                            TextField("warehouse_portal.residual.text.warehouse_name", text: $vm.warehouseName)
                        }
                    } else if !vm.warehouseName.isEmpty {
                        Section { Text(vm.warehouseName) }
                    }
                    if !vm.packCountry.isEmpty {
                        Section("Pack country") {
                            Text(vm.packCountry)
                        }
                    }
                    Section("Depot address") {
                        AddressLocationField(value: $vm.location, label: "Depot address")
                    }
                    if let error = vm.error {
                        Section {
                            Text(error).foregroundStyle(.red)
                        }
                    }
                    Section {
                        Button(vm.submitting ? "Saving…" : "Complete setup") {
                            PegasusHaptics.medium()
                            Task {
                                await vm.submit(tokenStore: tokenStore)
                            }
                        }
                        .frame(maxWidth: .infinity, minHeight: 44)
                        .contentShape(Rectangle())
                        .disabled(vm.submitting)
                        .accessibilityLabel(vm.submitting ? "Saving depot location" : "Complete depot setup")
                    }
                }
                .scrollDismissesKeyboard(.interactively)
            }
        }
        .navigationTitle("warehouse_portal.setup.location.text.warehouse_location")
        .task { await vm.loadExistingIfNeeded(hasAssignedWarehouse: hasAssignedWarehouse) }
    }
}
