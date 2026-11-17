import SwiftUI

struct LocationSettingsView: View {
    @State private var vm = LocationSettingsViewModel()

    var body: some View {
        @Bindable var vm = vm

        Group {
            if vm.loading {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let error = vm.error {
                FactoryErrorView(message: error) {
                    Task { await vm.load() }
                }
            } else {
                Form {
                    if !vm.factoryName.isEmpty {
                        Section {
                            Text(vm.factoryName)
                                .accessibilityLabel("Factory name: \(vm.factoryName)")
                        }
                    }
                    Section("Factory address") {
                        Text("mobile_factory.ui.used_for_supply_routing_and_dispatch_coordinates_stay_hidden_fro")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                        AddressLocationField(value: $vm.location, label: "Factory address")
                    }
                    if let saveMessage = vm.saveMessage {
                        Section {
                            Text(saveMessage)
                                .foregroundStyle(saveMessage.contains("saved") ? .green : .red)
                                .accessibilityLabel(saveMessage)
                        }
                    }
                    Section {
                        Button(vm.saving ? "Saving…" : "Save location") {
                            Task { await vm.save() }
                        }
                        .frame(minWidth: 44, minHeight: 44)
                        .disabled(vm.saving || vm.location.address.isEmpty)
                        .accessibilityLabel("Save factory location")
                    }
                }
                .scrollDismissesKeyboard(.interactively)
            }
        }
        .navigationTitle("factory_portal.settings.location.text.factory_location")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") {
                    PegasusHaptics.selection()
                    Task { await vm.load() }
                }
                .frame(minWidth: 44, minHeight: 44)
                .accessibilityLabel("Refresh factory location")
            }
        }
        .task { await vm.load() }
    }
}
