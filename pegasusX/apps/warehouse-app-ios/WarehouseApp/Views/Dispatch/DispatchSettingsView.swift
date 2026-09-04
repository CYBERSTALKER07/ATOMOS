import SwiftUI

struct DispatchSettingsView: View {
    @State private var autoDispatchEnabled: Bool?
    @State private var loading = true
    @State private var saving = false
    @State private var error: String?
    @State private var saveMessage: String?

    var body: some View {
        Group {
            if loading {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let error {
                ContentUnavailableView {
                    Label("mobile_warehouse.ui.error", systemImage: "exclamationmark.triangle")
                } description: {
                    Text(error)
                } actions: {
                    Button("common.action.retry") { load() }
                }
            } else {
                Form {
                    Section {
                        Text("warehouse_portal.residual.text.configure_warehouse_auto_dispatch_policy_for_this_node")
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                    }
                    if let saveMessage {
                        Section {
                            Text(saveMessage)
                                .foregroundStyle(.green)
                        }
                    }
                    Section("Auto dispatch") {
                        Toggle(
                            "Enable auto dispatch",
                            isOn: Binding(
                                get: { autoDispatchEnabled ?? false },
                                set: { newValue in save(enabled: newValue) }
                            )
                        )
                        .disabled(saving || autoDispatchEnabled == nil)
                        Text("mobile_warehouse.ui.when_enabled_the_ai_worker_may_auto_assign_pending_orders")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }
            }
        }
        .navigationTitle("warehouse_portal.dispatch_settings.text.dispatch_settings")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") { load() }
            }
        }
        .task { load() }
    }

    private func load() {
        loading = true
        error = nil
        Task {
            do {
                let settings = try await WarehouseService.dispatchSettings()
                autoDispatchEnabled = settings.autoDispatchEnabled
            } catch {
                self.error = error.localizedDescription
            }
            loading = false
        }
    }

    private func save(enabled: Bool) {
        saving = true
        saveMessage = nil
        Task {
            do {
                try await WarehouseService.patchDispatchSettings(enabled: enabled)
                autoDispatchEnabled = enabled
                saveMessage = enabled ? "Auto dispatch enabled" : "Auto dispatch disabled"
            } catch {
                saveMessage = error.localizedDescription
            }
            saving = false
        }
    }
}
