import SwiftUI

struct LocationSettingsView: View {
    @State private var vm = LocationSettingsViewModel()

    var body: some View {
        @Bindable var vm = vm
        Group {
            if vm.loading {
                ProgressView().frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let error = vm.error {
                ContentUnavailableView {
                    Label("mobile_warehouse.ui.error", systemImage: "exclamationmark.triangle")
                } description: {
                    Text(error)
                } actions: {
                    Button("common.action.retry") {
                        PegasusHaptics.light()
                        Task { await vm.load() }
                    }
                    .frame(minWidth: 44, minHeight: 44)
                    .contentShape(Rectangle())
                }
            } else {
                Form {
                    if !vm.warehouseName.isEmpty {
                        Section { Text(vm.warehouseName) }
                    }
                    if !vm.packCountry.isEmpty {
                        Section("Pack country") {
                            Text(vm.packCountry)
                        }
                    }
                    Section("Depot address") {
                        Text("mobile_warehouse.ui.used_for_smart_dispatch_routing_coordinates_stay_hidden_from_dai")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                        AddressLocationField(value: $vm.location, label: "Warehouse address")
                    }
                    if let saveMessage = vm.saveMessage {
                        Section {
                            Text(saveMessage)
                                .foregroundStyle(saveMessage.contains("saved") ? .green : .red)
                        }
                    }
                    Section {
                        Button(vm.saving ? "Saving…" : "Save location") {
                            PegasusHaptics.medium()
                            Task { await vm.save() }
                        }
                        .disabled(vm.saving || vm.location.address.isEmpty)
                        .frame(minHeight: 44)
                        .contentShape(Rectangle())
                        .accessibilityLabel("Save warehouse location")
                    }
                }
                .scrollDismissesKeyboard(.interactively)
            }
        }
        .navigationTitle("warehouse_portal.settings.text.depot_location")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") {
                    PegasusHaptics.light()
                    Task { await vm.load() }
                }
                .frame(minWidth: 44, minHeight: 44)
                .contentShape(Rectangle())
                .accessibilityLabel("Refresh depot location")
            }
        }
        .task { await vm.load() }
    }
}
