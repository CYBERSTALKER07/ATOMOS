import SwiftUI

struct OpsSettingsView: View {
    @State private var policy = "REJECT"
    @State private var scheduleJSON = "{\n  \"is_24h\": true\n}"
    @State private var loading = true
    @State private var saving = false
    @State private var error: String?
    @State private var scheduleError: String?
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
                        Text("Stock policy for retailer checkout and display operating hours.")
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                    }

                    Section("Out-of-stock orders") {
                        Picker("Policy", selection: $policy) {
                            Text("Reject").tag("REJECT")
                            Text("Accept backorder").tag("ACCEPT_BACKORDER")
                        }
                        .pickerStyle(.inline)
                    }

                    Section("Operating hours (display only)") {
                        Text("Shown to retailers for planning. Dispatch and delivery are not blocked outside these hours.")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                        TextEditor(text: $scheduleJSON)
                            .font(.system(.caption, design: .monospaced))
                            .frame(minHeight: 140)
                        if let scheduleError {
                            Text(scheduleError).foregroundStyle(.red).font(.caption)
                        }
                    }

                    if let saveMessage {
                        Section {
                            Text(saveMessage)
                                .foregroundStyle(saveMessage.contains("saved") ? .green : .red)
                        }
                    }

                    Section {
                        Button(saving ? "Saving…" : "Save settings") { save() }
                            .disabled(saving)
                    }
                }
            }
        }
        .navigationTitle("Ops settings")
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
                let settings = try await WarehouseService.opsSettings()
                policy = settings.defaultOutOfStockPolicy == "ACCEPT_BACKORDER" ? "ACCEPT_BACKORDER" : "REJECT"
                if let schedule = settings.operatingSchedule {
                    scheduleJSON = schedule.prettyJSONString()
                }
            } catch {
                self.error = error.localizedDescription
            }
            loading = false
        }
    }

    private func save() {
        saving = true
        saveMessage = nil
        scheduleError = nil
        guard let data = scheduleJSON.data(using: .utf8),
              let object = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else {
            scheduleError = "Operating schedule must be valid JSON"
            saving = false
            return
        }
        let schedule = object.mapValues { AnyCodable($0) }
        Task {
            do {
                try await WarehouseService.patchOpsSettings(policy: policy, operatingSchedule: schedule)
                saveMessage = "Warehouse settings saved"
                load()
            } catch {
                saveMessage = error.localizedDescription
            }
            saving = false
        }
    }
}
