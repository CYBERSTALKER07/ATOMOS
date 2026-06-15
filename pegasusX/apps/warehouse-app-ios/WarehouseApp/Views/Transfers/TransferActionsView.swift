import SwiftUI

struct TransferActionsView: View {
    @Environment(WarehouseRealtimeHub.self) private var realtimeHub
    @State private var volumeInput = "20"
    @State private var transferIdInput = ""
    @State private var notesInput = ""
    @State private var busy = false
    @State private var statusMessage: String?

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: LabTheme.spacingLG) {
                WarehouseSectionHeader(
                    title: "Factory inbound transfers",
                    subtitle: "Emergency payload, force receive, and inbound receive controls"
                )

                Form {
                    Section("Payload") {
                        TextField("Volume (VU)", text: $volumeInput)
                            .keyboardType(.decimalPad)
                        TextField("Notes (optional)", text: $notesInput)
                    }
                    Section("Actions") {
                        Button("Create emergency transfer") {
                            run(label: "Emergency transfer") {
                                try await WarehouseOperationsService.emergencyTransfer(
                                    totalVolumeVu: Double(volumeInput) ?? 20,
                                    notes: notesInput.isEmpty ? nil : notesInput
                                )
                            }
                        }
                        .disabled(busy)
                        Button("Force receive payload") {
                            run(label: "Force receive") {
                                try await WarehouseOperationsService.forceReceive(
                                    totalVolumeVu: Double(volumeInput) ?? 20,
                                    notes: notesInput.isEmpty ? nil : notesInput
                                )
                            }
                        }
                        .disabled(busy)
                    }
                    Section("Receive inbound") {
                        TextField("Transfer ID", text: $transferIdInput)
                            .textInputAutocapitalization(.never)
                        Button("Receive transfer") {
                            let id = transferIdInput.trimmingCharacters(in: .whitespacesAndNewlines)
                            guard !id.isEmpty else {
                                statusMessage = "Transfer ID required"
                                return
                            }
                            run(label: "Receive transfer") {
                                try await WarehouseOperationsService.receiveTransfer(transferId: id)
                            }
                        }
                        .disabled(busy)
                    }
                    if let statusMessage {
                        Section {
                            Text(statusMessage)
                                .font(.footnote)
                                .foregroundStyle(.secondary)
                        }
                    }
                }
                .scrollContentBackground(.hidden)
            }
            .labReadableWidth()
            .padding()
        }
        .background(LabTheme.background)
        .navigationTitle("Transfer actions")
        .onChange(of: realtimeHub.reconnectEpoch) { _, _ in
            if busy {
                busy = false
                statusMessage = "Connection restored — verify status before retrying."
            }
        }
    }

    private func run(label: String, _ block: @escaping () async throws -> TransferMutationResponse) {
        busy = true
        statusMessage = nil
        Task {
            defer { busy = false }
            do {
                let response = try await block()
                statusMessage = "\(label) succeeded · \(response.state)"
            } catch {
                statusMessage = error.localizedDescription
            }
        }
    }
}
