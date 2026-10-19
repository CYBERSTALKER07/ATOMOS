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
    @State private var pickScannedQty = 0
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
                        TextField("warehouse_portal.transfers.text.volume_vu", text: $volumeInput)
                            .keyboardType(.decimalPad)
                        TextField("supplier_portal.returns.text.notes_optional", text: $notesInput)
                    }
                    Section("Actions") {
                        Button("mobile_warehouse.ui.create_emergency_transfer") {
                            run(label: "Emergency transfer") {
                                try await WarehouseOperationsService.emergencyTransfer(
                                    totalVolumeVu: Double(volumeInput) ?? 20,
                                    notes: notesInput.isEmpty ? nil : notesInput
                                )
                            }
                        }
                        .disabled(busy)
                        Button("mobile_warehouse.ui.force_receive_payload") {
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
                        TextField("mobile_warehouse.ui.transfer_id", text: $transferIdInput)
                            .textInputAutocapitalization(.never)
                        Button("mobile_warehouse.ui.receive_transfer") {
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
                        TextField("factory_portal.analytics.text.product_id", text: $putawayProduct)
                            .textInputAutocapitalization(.never)
                        TextField("mobile_warehouse.ui.location_bin_id", text: $putawayLocation)
                            .textInputAutocapitalization(.never)
                        TextField("warehouse_portal.inventory.inventory_stock_list.text.quantity", text: $putawayQty)
                            .keyboardType(.numberPad)
                        TextField("mobile_warehouse.ui.expiry_yyyy_mm_dd", text: $putawayExpiry)
                            .textInputAutocapitalization(.never)
                        Button("mobile_warehouse.ui.ensure_stage_bin") {
                            runAny(label: "Create bin") {
                                _ = try await WarehouseService.createBin(
                                    locationId: putawayLocation.isEmpty ? nil : putawayLocation,
                                    zone: "RECV",
                                    locationType: "STAGE"
                                )
                            }
                        }
                        .disabled(busy)
                        Button("mobile_warehouse.ui.putaway_lot") {
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
                        TextField("mobile_warehouse.ui.manifest_id", text: $pickManifestId)
                            .textInputAutocapitalization(.never)
                        Button("mobile_warehouse.ui.create_pick_wave") {
                            let mid = pickManifestId.trimmingCharacters(in: .whitespacesAndNewlines)
                            guard !mid.isEmpty else {
                                statusMessage = "Manifest ID required"
                                return
                            }
                            runAny(label: "Create wave") {
                                activeWave = try await WarehouseService.createPickWave(manifestId: mid)
                                pickScannedQty = 0
                            }
                        }
                        .disabled(busy)
                        Button("mobile_warehouse.ui.load_pick_waves") {
                            runAny(label: "Load waves") {
                                let list = try await WarehouseService.listPickWaves(
                                    manifestId: pickManifestId.isEmpty ? nil : pickManifestId
                                )
                                if let first = list.waves.first {
                                    activeWave = try await WarehouseService.getPickWave(waveId: first.waveId)
                                    pickScannedQty = 0
                                } else {
                                    statusMessage = "No pick waves"
                                }
                            }
                        }
                        .disabled(busy)
                        EANBarcodeScannerView(
                            onBarcode: { code in
                                onPickBarcode(code)
                            },
                            enabled: !busy,
                            previewHeight: 180
                        )
                        TextField("mobile_warehouse.ui.scan_product_lot_id", text: $pickScan)
                            .textInputAutocapitalization(.never)
                            .onSubmit { onPickBarcode(pickScan) }
                        if let wave = activeWave {
                            Text(L10n.format("mobile_warehouse.ui.wave_prefix_status", "\(wave.waveId.prefix(8))", "\(wave.status)"))
                                .font(.footnote)
                            if let task = wave.tasks?.first(where: { t in
                                t.status == "PENDING" && (
                                    pickScan.isEmpty ||
                                    t.productId.caseInsensitiveCompare(pickScan) == .orderedSame ||
                                    t.lotId.caseInsensitiveCompare(pickScan) == .orderedSame
                                )
                            }) {
                                Text(L10n.format("mobile_warehouse.ui.next_productid_locationid_qty_quantityrequested", "\(task.productId)", "\(task.locationId ?? "—")", "\(task.quantityRequested)"))
                                    .font(.caption)
                                Text("Scanned \(pickScannedQty) / \(task.quantityRequested)")
                                    .font(.caption)
                                    .foregroundStyle(.tint)
                                Button("mobile_warehouse.ui.confirm_pick") {
                                    let qty = pickScannedQty > 0
                                        ? min(pickScannedQty, task.quantityRequested)
                                        : task.quantityRequested
                                    runAny(label: "Confirm pick") {
                                        activeWave = try await WarehouseService.confirmPickTask(
                                            waveId: wave.waveId,
                                            taskId: task.taskId,
                                            quantityPicked: qty
                                        )
                                        pickScan = ""
                                        pickScannedQty = 0
                                    }
                                }
                                .disabled(busy)
                            }
                        }
                    }
                    Section("WMS cycle counts") {
                        TextField("mobile_warehouse.ui.count_location_id", text: $putawayLocation)
                            .textInputAutocapitalization(.never)
                        TextField("mobile_warehouse.ui.count_product_id", text: $putawayProduct)
                            .textInputAutocapitalization(.never)
                        Button("mobile_warehouse.ui.create_cycle_count") {
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
                        Button("mobile_warehouse.ui.submit_first_open_expected") {
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
        .navigationTitle("warehouse_portal.transfers.text.transfer_actions")
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

    private func onPickBarcode(_ code: String) {
        let trimmed = code.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty, !busy else { return }
        pickScan = trimmed
        if putawayProduct.isEmpty {
            putawayProduct = trimmed
        }
        guard let wave = activeWave,
              let task = wave.tasks?.first(where: { t in
                  t.status == "PENDING" && (
                      t.productId.caseInsensitiveCompare(trimmed) == .orderedSame ||
                      t.lotId.caseInsensitiveCompare(trimmed) == .orderedSame
                  )
              }) else { return }
        let next = pickScannedQty + 1
        pickScannedQty = next
        if next >= task.quantityRequested {
            runAny(label: "Confirm pick") {
                activeWave = try await WarehouseService.confirmPickTask(
                    waveId: wave.waveId,
                    taskId: task.taskId,
                    quantityPicked: task.quantityRequested
                )
                pickScan = ""
                pickScannedQty = 0
            }
        } else {
            statusMessage = "Scanned \(next) / \(task.quantityRequested)"
        }
    }
}
