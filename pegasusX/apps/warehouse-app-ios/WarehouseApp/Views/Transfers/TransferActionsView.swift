import SwiftUI

struct TransferActionsView: View {
    @Environment(WarehouseRealtimeHub.self) private var realtimeHub
    @State private var vm = TransferActionsViewModel()

    var body: some View {
        @Bindable var vm = vm
        ScrollView {
            VStack(alignment: .leading, spacing: LabTheme.spacingLG) {
                WarehouseSectionHeader(
                    title: "Factory inbound transfers",
                    subtitle: "Emergency payload, force receive, and inbound receive controls"
                )

                Form {
                    Section("Payload") {
                        TextField("warehouse_portal.transfers.text.volume_vu", text: $vm.volumeInput)
                            .keyboardType(.decimalPad)
                        TextField("supplier_portal.returns.text.notes_optional", text: $vm.notesInput)
                    }
                    Section("Actions") {
                        Button("mobile_warehouse.ui.create_emergency_transfer") {
                            PegasusHaptics.medium()
                            Task { await vm.createEmergencyTransfer() }
                        }
                        .frame(minHeight: 44)
                        .contentShape(Rectangle())
                        .disabled(vm.busy)

                        Button("mobile_warehouse.ui.force_receive_payload") {
                            PegasusHaptics.heavy()
                            Task { await vm.forceReceivePayload() }
                        }
                        .frame(minHeight: 44)
                        .contentShape(Rectangle())
                        .disabled(vm.busy)
                    }
                    Section("Receive inbound") {
                        TextField("mobile_warehouse.ui.transfer_id", text: $vm.transferIdInput)
                            .textInputAutocapitalization(.never)
                        Button("mobile_warehouse.ui.receive_transfer") {
                            PegasusHaptics.medium()
                            Task { await vm.receiveInboundTransfer() }
                        }
                        .frame(minHeight: 44)
                        .contentShape(Rectangle())
                        .disabled(vm.busy)
                    }
                    Section("WMS putaway (lots)") {
                        TextField("factory_portal.analytics.text.product_id", text: $vm.putawayProduct)
                            .textInputAutocapitalization(.never)
                        TextField("mobile_warehouse.ui.location_bin_id", text: $vm.putawayLocation)
                            .textInputAutocapitalization(.never)
                        TextField("warehouse_portal.inventory.inventory_stock_list.text.quantity", text: $vm.putawayQty)
                            .keyboardType(.numberPad)
                        TextField("mobile_warehouse.ui.expiry_yyyy_mm_dd", text: $vm.putawayExpiry)
                            .textInputAutocapitalization(.never)
                        Button("mobile_warehouse.ui.ensure_stage_bin") {
                            PegasusHaptics.light()
                            Task { await vm.ensureStageBin() }
                        }
                        .frame(minHeight: 44)
                        .contentShape(Rectangle())
                        .disabled(vm.busy)

                        Button("mobile_warehouse.ui.putaway_lot") {
                            PegasusHaptics.medium()
                            Task { await vm.performPutawayLot() }
                        }
                        .frame(minHeight: 44)
                        .contentShape(Rectangle())
                        .disabled(vm.busy)
                    }
                    Section("WMS pick waves") {
                        TextField("mobile_warehouse.ui.manifest_id", text: $vm.pickManifestId)
                            .textInputAutocapitalization(.never)
                        Button("mobile_warehouse.ui.create_pick_wave") {
                            PegasusHaptics.medium()
                            Task { await vm.createPickWave() }
                        }
                        .frame(minHeight: 44)
                        .contentShape(Rectangle())
                        .disabled(vm.busy)

                        Button("mobile_warehouse.ui.load_pick_waves") {
                            PegasusHaptics.light()
                            Task { await vm.loadPickWaves() }
                        }
                        .frame(minHeight: 44)
                        .contentShape(Rectangle())
                        .disabled(vm.busy)

                        EANBarcodeScannerView(
                            onBarcode: { code in
                                Task { await vm.onPickBarcode(code) }
                            },
                            enabled: !vm.busy,
                            previewHeight: 180
                        )
                        TextField("mobile_warehouse.ui.scan_product_lot_id", text: $vm.pickScan)
                            .textInputAutocapitalization(.never)
                            .onSubmit {
                                let code = vm.pickScan
                                Task { await vm.onPickBarcode(code) }
                            }
                        if let wave = vm.activeWave {
                            Text(L10n.format("mobile_warehouse.ui.wave_prefix_status", "\(wave.waveId.prefix(8))", "\(wave.status)"))
                                .font(.footnote)
                            if let task = vm.nextPendingTask {
                                Text(L10n.format("mobile_warehouse.ui.next_productid_locationid_qty_quantityrequested", "\(task.productId)", "\(task.locationId ?? "—")", "\(task.quantityRequested)"))
                                    .font(.caption)
                                Text("Scanned \(vm.pickScannedQty) / \(task.quantityRequested)")
                                    .font(.caption)
                                    .foregroundStyle(.tint)
                                Button("mobile_warehouse.ui.confirm_pick") {
                                    PegasusHaptics.medium()
                                    Task { await vm.confirmCurrentPick(wave: wave, task: task) }
                                }
                                .frame(minHeight: 44)
                                .contentShape(Rectangle())
                                .disabled(vm.busy)
                            }
                        }
                    }
                    Section("WMS cycle counts") {
                        TextField("mobile_warehouse.ui.count_location_id", text: $vm.putawayLocation)
                            .textInputAutocapitalization(.never)
                        TextField("mobile_warehouse.ui.count_product_id", text: $vm.putawayProduct)
                            .textInputAutocapitalization(.never)
                        Button("mobile_warehouse.ui.create_cycle_count") {
                            PegasusHaptics.medium()
                            Task { await vm.createCycleCount() }
                        }
                        .frame(minHeight: 44)
                        .contentShape(Rectangle())
                        .disabled(vm.busy)

                        Button("mobile_warehouse.ui.submit_first_open_expected") {
                            PegasusHaptics.medium()
                            Task { await vm.submitFirstOpenExpected() }
                        }
                        .frame(minHeight: 44)
                        .contentShape(Rectangle())
                        .disabled(vm.busy)
                    }
                    if let statusMessage = vm.statusMessage {
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
        .scrollDismissesKeyboard(.interactively)
        .background(LabTheme.background)
        .navigationTitle("warehouse_portal.transfers.text.transfer_actions")
        .onChange(of: realtimeHub.reconnectEpoch) { _, _ in
            vm.handleReconnect()
        }
    }
}
