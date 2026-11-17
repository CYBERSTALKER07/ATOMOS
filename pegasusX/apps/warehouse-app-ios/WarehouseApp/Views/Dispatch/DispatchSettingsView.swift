import SwiftUI

struct DispatchSettingsView: View {
    @State private var vm = DispatchSettingsViewModel()

    var body: some View {
        @Bindable var vm = vm
        Group {
            if vm.loading {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
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
                    Section {
                        Text("warehouse_portal.residual.text.configure_warehouse_auto_dispatch_policy_for_this_node")
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                    }
                    if let saveMessage = vm.saveMessage {
                        Section {
                            Text(saveMessage)
                                .foregroundStyle(
                                    saveMessage.hasPrefix("Auto dispatch") ? .green : .red
                                )
                        }
                    }
                    Section("Auto dispatch") {
                        Toggle("Enable auto dispatch", isOn: $vm.autoDispatchEnabled)
                            .frame(minHeight: 44)
                            .disabled(vm.saving || !vm.hasLoaded)
                            .accessibilityLabel("Enable auto dispatch")
                            .onChange(of: vm.autoDispatchEnabled) { _, newValue in
                                Task { await vm.handleToggleChange(newValue) }
                            }
                        Text("mobile_warehouse.ui.when_enabled_the_ai_worker_may_auto_assign_pending_orders")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }
                .scrollDismissesKeyboard(.interactively)
            }
        }
        .navigationTitle("warehouse_portal.dispatch_settings.text.dispatch_settings")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") {
                    PegasusHaptics.light()
                    Task { await vm.load() }
                }
                .frame(minWidth: 44, minHeight: 44)
                .contentShape(Rectangle())
                .accessibilityLabel("Refresh dispatch settings")
            }
        }
        .task { await vm.load() }
    }
}
