import SwiftUI

struct TransferActionsView: View {
    @Environment(WarehouseRealtimeHub.self) private var realtimeHub
    @State private var volumeInput = "20"
    @State private var transferIdInput = ""
    @State private var notesInput = ""
    @State private var putawayProduct = ""
    @State private var putawayLocation = ""
    @State private var putawayQty = "1"
    @State private var putawayExpiry = ""
    @State private var pickManifestId = ""
    @State private var pickScan = ""
    @State private var activeWave: PickWave?
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
                    Section("WMS putaway (lots)") {
                        TextField("Product ID", text: $putawayProduct)
                            .textInputAutocapitalization(.never)
                        TextField("Location / bin ID", text: $putawayLocation)
                            .textInputAutocapitalization(.never)
                        TextField("Quantity", text: $putawayQty)
                            .keyboardType(.numberPad)
                        TextField("Expiry YYYY-MM-DD", text: $putawayExpiry)
                            .textInputAutocapitalization(.never)
                        Button("Ensure STAGE bin") {
                            runAny(label: "Create bin") {
                                _ = try await WarehouseService.createBin(
                                    locationId: putawayLocation.isEmpty ? nil : putawayLocation,
                                    zone: "RECV",
                                    locationType: "STAGE"
                                )
                            }
                        }
                        .disabled(busy)
                        Button("Putaway lot") {
                            let pid = putawayProduct.trimmingCharacters(in: .whitespacesAndNewlines)
                            let lid = putawayLocation.trimmingCharacters(in: .whitespacesAndNewlines)
                            let qty = Int(putawayQty) ?? 0
                            guard !pid.isEmpty, !lid.isEmpty, qty > 0 else {
                                statusMessage = "Product, location, qty required"
                                return
                            }
                            runAny(label: "Putaway") {
                                _ = try await WarehouseService.putawayLot(
                                    productId: pid,
                                    locationId: lid,
                                    quantity: qty,
                                    expiryDate: putawayExpiry.isEmpty ? nil : putawayExpiry
                                )
                            }
                        }
                        .disabled(busy)
                    }
                    Section("WMS pick waves") {
                        TextField("Manifest ID", text: $pickManifestId)
                            .textInputAutocapitalization(.never)
                        Button("Create pick wave") {
                            let mid = pickManifestId.trimmingCharacters(in: .whitespacesAndNewlines)
                            guard !mid.isEmpty else {
                                statusMessage = "Manifest ID required"
                                return
                            }
                            runAny(label: "Create wave") {
                                activeWave = try await WarehouseService.createPickWave(manifestId: mid)
                            }
                        }
                        .disabled(busy)
                        Button("Load pick waves") {
                            runAny(label: "Load waves") {
                                let list = try await WarehouseService.listPickWaves(
                                    manifestId: pickManifestId.isEmpty ? nil : pickManifestId
                                )
                                if let first = list.waves.first {
                                    activeWave = try await WarehouseService.getPickWave(waveId: first.waveId)
                                } else {
                                    statusMessage = "No pick waves"
                                }
                            }
                        }
                        .disabled(busy)
                        TextField("Scan product / lot ID", text: $pickScan)
                            .textInputAutocapitalization(.never)
                        if let wave = activeWave {
                            Text("Wave \(wave.waveId.prefix(8))… · \(wave.status)")
                                .font(.footnote)
                            if let task = wave.tasks?.first(where: { t in
                                t.status == "PENDING" && (
                                    pickScan.isEmpty ||
                                    t.productId.caseInsensitiveCompare(pickScan) == .orderedSame ||
                                    t.lotId.caseInsensitiveCompare(pickScan) == .orderedSame
                                )
                            }) {
                                Text("Next: \(task.productId) @ \(task.locationId ?? "—") qty \(task.quantityRequested)")
                                    .font(.caption)
                                Button("Confirm pick") {
                                    runAny(label: "Confirm pick") {
                                        activeWave = try await WarehouseService.confirmPickTask(
                                            waveId: wave.waveId,
                                            taskId: task.taskId,
                                            quantityPicked: task.quantityRequested
                                        )
                                        pickScan = ""
                                    }
                                }
                                .disabled(busy)
                            }
                        }
                    }
                    Section("WMS cycle counts") {
                        TextField("Count location ID", text: $putawayLocation)
                            .textInputAutocapitalization(.never)
                        TextField("Count product ID", text: $putawayProduct)
                            .textInputAutocapitalization(.never)
                        Button("Create cycle count") {
                            let lid = putawayLocation.trimmingCharacters(in: .whitespacesAndNewlines)
                            let pid = putawayProduct.trimmingCharacters(in: .whitespacesAndNewlines)
                            guard !lid.isEmpty, !pid.isEmpty else {
                                statusMessage = "Location + product required"
                                return
                            }
                            runAny(label: "Create count") {
                                _ = try await WarehouseService.createCycleCount(locationId: lid, productId: pid)
                            }
                        }
                        .disabled(busy)
                        Button("Submit first OPEN @ expected") {
                            runAny(label: "Submit count") {
                                let list = try await WarehouseService.listCycleCounts()
                                guard let open = list.counts.first(where: { $0.status == "OPEN" }) else {
                                    statusMessage = "No OPEN counts"
                                    return
                                }
                                _ = try await WarehouseService.submitCycleCount(
                                    countId: open.countId,
                                    countedQty: open.expectedQty
                                )
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

    private func runAny(label: String, _ block: @escaping () async throws -> Void) {
        busy = true
        statusMessage = nil
        Task {
            defer { busy = false }
            do {
                try await block()
                if statusMessage == nil {
                    statusMessage = "\(label) succeeded"
                }
            } catch {
                statusMessage = error.localizedDescription
            }
        }
    }
}
