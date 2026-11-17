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
                        if vm.hasColdLotOrStagingAlert {
                            VStack(alignment: .leading, spacing: 6) {
                                Label("mobile_warehouse.cold_lot.banner", systemImage: "thermometer.snowflake")
                                    .font(.caption.weight(.semibold))
                                    .foregroundStyle(.orange)
                                Text("cold_lot_migrated_without_live_baseline")
                                    .font(.caption2.monospaced())
                                    .foregroundStyle(.secondary)
                                ForEach((vm.coldWarningLots.isEmpty ? vm.stagingPutawayLots : vm.coldWarningLots).prefix(5)) { lot in
                                    Button {
                                        vm.stagePutawayLot(lot)
                                    } label: {
                                        HStack {
                                            VStack(alignment: .leading, spacing: 2) {
                                                Text("\(lot.productName ?? lot.productId) · \(lot.locationId)")
                                                    .font(.caption.weight(.medium))
                                                HStack(spacing: 6) {
                                                    Text(L10n.format("mobile_warehouse.cold_lot.lot_label", lot.lotCode ?? String(lot.lotId.prefix(8))))
                                                    if let expiry = lot.expiryDate, !expiry.isEmpty {
                                                        Text(L10n.format("mobile_warehouse.cold_lot.expiry_label", expiry))
                                                    }
                                                }
                                                .font(.caption2)
                                                .foregroundStyle(.secondary)
                                            }
                                            Spacer()
                                            if let reqZone = TransferActionsViewModel.requiredZone(for: lot) {
                                                Text(reqZone)
                                                    .font(.caption2.weight(.semibold))
                                                    .padding(.horizontal, 6)
                                                    .padding(.vertical, 2)
                                                    .background(Color.blue.opacity(0.15))
                                                    .clipShape(Capsule())
                                            }
                                        }
                                    }
                                    .buttonStyle(.plain)
                                }
                            }
                            .padding(.vertical, 4)
                        }
                        if let mismatchZone = vm.putawayZoneMismatchRequiredZone {
                            Label(
                                L10n.format("mobile_warehouse.cold_lot.zone_mismatch", mismatchZone),
                                systemImage: "exclamationmark.triangle.fill"
                            )
                            .font(.caption)
                            .foregroundStyle(.red)
                        }
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
                            HStack {
                                Text(L10n.format("mobile_warehouse.ui.wave_prefix_status", "\(wave.waveId.prefix(8))", "\(wave.status)"))
                                    .font(.footnote)
                                Spacer()
                                if vm.hasShortTasks {
                                    Button("Waive shorts") {
                                        PegasusHaptics.medium()
                                        Task { await vm.waivePickShorts() }
                                    }
                                    .buttonStyle(.bordered)
                                    .disabled(vm.busy)
                                }
                            }
                            if vm.activeWavePreSealWarning {
                                VStack(alignment: .leading, spacing: 4) {
                                    Label("mobile_warehouse.cold_lot.banner", systemImage: "thermometer.snowflake")
                                        .font(.caption.weight(.semibold))
                                        .foregroundStyle(.orange)
                                    Text("cold_lot_migrated_without_live_baseline · Pre-seal cold-chain check")
                                        .font(.caption2)
                                        .foregroundStyle(.secondary)
                                }
                                .padding(8)
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .background(Color.orange.opacity(0.12))
                                .clipShape(RoundedRectangle(cornerRadius: 8))
                            }
                            if let tasks = wave.tasks, !tasks.isEmpty {
                                ForEach(tasks) { task in
                                    HStack {
                                        VStack(alignment: .leading, spacing: 2) {
                                            Text("\(task.productId) · \(task.locationId ?? "—")")
                                                .font(.caption)
                                            Text("Picked \(task.quantityPicked ?? 0)/\(task.quantityRequested)")
                                                .font(.caption2)
                                                .foregroundStyle(.secondary)
                                        }
                                        Spacer()
                                        PickTaskStatusBadge(status: task.status)
                                    }
                                }
                            }
                            if let task = vm.nextPendingTask {
                                Text(L10n.format("mobile_warehouse.ui.next_productid_locationid_qty_quantityrequested", "\(task.productId)", "\(task.locationId ?? "—")", "\(task.quantityRequested)"))
                                    .font(.caption)
                                Text("Scanned \(vm.pickScannedQty) / \(task.quantityRequested)")
                                    .font(.caption)
                                    .foregroundStyle(.tint)
                                TextField("Picked qty (0 = SHORT)", text: $vm.pickQtyInput)
                                    .keyboardType(.numberPad)
                                HStack {
                                    Button("mobile_warehouse.ui.confirm_pick") {
                                        PegasusHaptics.medium()
                                        Task { await vm.confirmCurrentPick(wave: wave, task: task) }
                                    }
                                    .frame(minHeight: 44)
                                    .contentShape(Rectangle())
                                    .disabled(vm.busy)
                                    Spacer()
                                    Button("Report short (0)") {
                                        PegasusHaptics.medium()
                                        Task { await vm.reportShortZero(wave: wave, task: task) }
                                    }
                                    .buttonStyle(.bordered)
                                    .tint(.orange)
                                    .frame(minHeight: 44)
                                    .disabled(vm.busy)
                                }
                            }
                        }
                    }
                    Section("WMS cycle counts") {
                        TextField("mobile_warehouse.ui.count_location_id", text: $vm.putawayLocation)
                            .textInputAutocapitalization(.never)
                        TextField("mobile_warehouse.ui.count_product_id", text: $vm.putawayProduct)
                            .textInputAutocapitalization(.never)
                        TextField("Expected qty (optional)", text: $vm.cycleExpectedInput)
                            .keyboardType(.numberPad)
                        HStack {
                            Button("mobile_warehouse.ui.create_cycle_count") {
                                PegasusHaptics.medium()
                                Task { await vm.createCycleCount() }
                            }
                            .frame(minHeight: 44)
                            .contentShape(Rectangle())
                            .disabled(vm.busy)
                            Spacer()
                            Button("Refresh counts") {
                                PegasusHaptics.light()
                                Task { await vm.refreshCycleCountsAndAdjustments() }
                            }
                            .frame(minHeight: 44)
                            .disabled(vm.busy)
                        }

                        ForEach(vm.cycleCounts) { count in
                            let variance = TransferActionsViewModel.variance(for: count)
                            let isSelected = count.countId == vm.selectedOpenCycleCount?.countId
                            HStack {
                                VStack(alignment: .leading, spacing: 2) {
                                    Text("\(count.locationId ?? "—") · \(count.productId) (\(count.status))")
                                        .font(.caption)
                                    if let counted = count.countedQty {
                                        let deltaText = variance.map { " · Δ\(TransferActionsViewModel.signedDeltaString($0))" } ?? ""
                                        Text("Counted \(counted)/\(count.expectedQty)\(deltaText)")
                                            .font(.caption2)
                                            .foregroundStyle(.secondary)
                                    } else {
                                        Text("Expected \(count.expectedQty)")
                                            .font(.caption2)
                                            .foregroundStyle(.secondary)
                                    }
                                }
                                Spacer()
                                if count.status == "OPEN" {
                                    Button(isSelected ? "Selected" : "Select") {
                                        vm.selectCycleCount(count)
                                    }
                                    .buttonStyle(.bordered)
                                    .disabled(vm.busy)
                                }
                            }
                        }

                        if let open = vm.selectedOpenCycleCount {
                            Text("Submit \(open.countId.prefix(8))… · \(open.productId) @ \(open.locationId ?? "—")")
                                .font(.caption)
                            TextField("Actual counted qty (0 allowed)", text: $vm.countedQtyInput)
                                .keyboardType(.numberPad)
                            TextField("Reason code (optional)", text: $vm.cycleReasonCode)
                                .textInputAutocapitalization(.never)
                            Button("Submit cycle count") {
                                PegasusHaptics.medium()
                                Task { await vm.submitCycleCount() }
                            }
                            .frame(minHeight: 44)
                            .contentShape(Rectangle())
                            .disabled(vm.busy)
                        }
                    }
                    if !vm.adjustments.isEmpty {
                        Section("Inventory adjustments") {
                            ForEach(vm.adjustments) { adj in
                                VStack(alignment: .leading, spacing: 6) {
                                    Text("\(adj.adjustmentId.prefix(8))… · \(adj.productId) · Δ\(TransferActionsViewModel.signedDeltaString(adj.deltaQty)) · \(adj.status)")
                                        .font(.caption)
                                    if adj.status == "PENDING" {
                                        HStack {
                                            Button("Approve") {
                                                PegasusHaptics.medium()
                                                Task { await vm.approveAdjustment(adj.adjustmentId) }
                                            }
                                            .buttonStyle(.borderedProminent)
                                            .disabled(vm.busy)
                                            Button("Reject") {
                                                PegasusHaptics.medium()
                                                Task { await vm.rejectAdjustment(adj.adjustmentId) }
                                            }
                                            .buttonStyle(.bordered)
                                            .tint(.red)
                                            .disabled(vm.busy)
                                        }
                                    }
                                }
                            }
                        }
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
        .task {
            await vm.refreshBinsAndLots()
            await vm.refreshCycleCountsAndAdjustments()
        }
        .onChange(of: realtimeHub.reconnectEpoch) { _, _ in
            vm.handleReconnect()
        }
    }
}

private struct PickTaskStatusBadge: View {
    let status: String

    private var tintColor: Color {
        switch status {
        case "CONFIRMED":
            return .green
        case "SHORT":
            return .orange
        case "SHORT_WAIVED":
            return .blue
        default:
            return .secondary
        }
    }

    var body: some View {
        Text(status)
            .font(.caption2.weight(.medium))
            .padding(.horizontal, 8)
            .padding(.vertical, 3)
            .background(tintColor.opacity(0.15))
            .foregroundStyle(tintColor)
            .clipShape(RoundedRectangle(cornerRadius: 6))
    }
}
