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
                    Label("Error", systemImage: "exclamationmark.triangle")
                } description: {
                    Text(error)
                } actions: {
                    Button("Retry") { load() }
                }
            } else {
                Form {
                    Section {
                        Text("Configure warehouse auto-dispatch policy for this node.")
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
                        Text("When enabled, the AI worker may auto-assign pending orders.")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }
            }
        }
        .navigationTitle("Dispatch settings")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("Refresh", systemImage: "arrow.clockwise") { load() }
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
