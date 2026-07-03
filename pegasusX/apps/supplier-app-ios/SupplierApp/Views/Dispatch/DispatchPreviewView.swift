import SwiftUI

private enum DispatchMode: String, CaseIterable {
    case manual = "Manual truck"
    case auto = "Smart assign"
}

private let tetrisBuffer = 0.95

struct DispatchPreviewView: View {
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    var embeddedInHub: Bool = false
    @State private var preview: SupplierDispatchPreview?
    @State private var warehouses: [SupplierTopologyWarehouse] = []
    @State private var selectedWarehouseId: String?
    @State private var loading = true
    @State private var executing = false
    @State private var error: String?
    @State private var executeMessage: String?
    @State private var showAutoConfirm = false
    @State private var showCapacityOverride = false
    @State private var dispatchMode: DispatchMode = .manual
    @State private var selectedDriverId = ""
    @State private var selectedOrderIds = Set<String>()

    var body: some View {
        Group {
            if loading && preview == nil {
                SupplierLoadingView(title: "Loading dispatch preview…")
            } else if let error, preview == nil {
                SupplierErrorView(message: error) { Task { await load() } }
            } else {
                ScrollView {
                    VStack(alignment: .leading, spacing: SupplierTheme.spacingLG) {
                        Picker("Mode", selection: $dispatchMode) {
                            ForEach(DispatchMode.allCases, id: \.self) { mode in
                                Text(mode.rawValue).tag(mode)
                            }
                        }
                        .pickerStyle(.segmented)

                        if !warehouses.isEmpty {
                            VStack(alignment: .leading, spacing: SupplierTheme.spacingSM) {
                                Text("Warehouse scope").font(.headline)
                                ScrollView(.horizontal, showsIndicators: false) {
                                    HStack(spacing: SupplierTheme.spacingSM) {
                                        ForEach(warehouses) { wh in
                                            Button {
                                                selectedWarehouseId = selectedWarehouseId == wh.warehouseId ? nil : wh.warehouseId
                                                Task { await load() }
                                            } label: {
                                                Text(wh.name.isEmpty ? wh.warehouseId : wh.name)
                                                    .font(.subheadline.weight(.medium))
                                                    .padding(.horizontal, 12)
                                                    .padding(.vertical, 8)
                                                    .background(selectedWarehouseId == wh.warehouseId ? Color.primary : Color.clear)
                                                    .foregroundStyle(selectedWarehouseId == wh.warehouseId ? Color(.systemBackground) : Color.primary)
                                                    .clipShape(Capsule())
                                                    .overlay {
                                                        Capsule().strokeBorder(Color.primary.opacity(0.25), lineWidth: selectedWarehouseId == wh.warehouseId ? 0 : 1)
                                                    }
                                            }
                                            .buttonStyle(.plain)
                                        }
                                    }
                                }
                            }
                        }
                        if let preview {
                            if preview.planFingerprintMismatch {
                                VStack(alignment: .leading, spacing: 4) {
                                    Text("Dispatch plan drift")
                                        .font(.subheadline.weight(.semibold))
                                        .foregroundStyle(.red)
                                    Text("Supplier preview differs from warehouse floor plan. Refresh warehouse dispatch before committing.")
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                }
                                .padding()
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .background(Color.red.opacity(0.08))
                                .clipShape(RoundedRectangle(cornerRadius: SupplierTheme.radiusLG))
                            }
                            HStack(spacing: SupplierTheme.spacingMD) {
                                DispatchKpiCard(title: "Pending", value: "\(preview.pendingCount ?? 0)")
                                DispatchKpiCard(title: "Drivers", value: "\(preview.availableDriverCount ?? 0)")
                                DispatchKpiCard(title: "Undispatched", value: "\(preview.undispatchedOrderCount ?? preview.undispatchedOrders.count)")
                            }
                            if dispatchMode == .manual, !preview.undispatchedOrders.isEmpty {
                                manualAssignmentSection(preview: preview)
                            }
                            if dispatchMode == .auto, !preview.proposedRoutes.isEmpty {
                                VStack(alignment: .leading, spacing: SupplierTheme.spacingSM) {
                                    HStack {
                                        Text("Route map").font(.headline)
                                        Spacer()
                                        if let source = preview.optimizerSource, !source.isEmpty {
                                            Text("Source: \(source)")
                                                .font(.caption)
                                                .foregroundStyle(.secondary)
                                        }
                                    }
                                    DispatchPreviewMapView(routes: preview.proposedRoutes)
                                        .frame(height: 320)
                                        .clipShape(RoundedRectangle(cornerRadius: SupplierTheme.radiusLG))
                                }
                            }
                        }
                        if let executeMessage {
                            Text(executeMessage)
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                        if dispatchMode == .auto {
                            Button("Execute auto-dispatch") { showAutoConfirm = true }
                                .buttonStyle(.borderedProminent)
                                .tint(.primary)
                                .disabled(loading || executing || preview == nil)
                        } else {
                            Button("Manual dispatch (\(selectedOrderIds.count))") {
                                Task { await executeManual(forceCapacity: false) }
                            }
                            .buttonStyle(.borderedProminent)
                            .tint(.primary)
                            .disabled(loading || executing || selectedDriverId.isEmpty || selectedOrderIds.isEmpty)
                        }
                        Button("Refresh preview") { Task { await load() } }
                            .disabled(loading || executing)
                    }
                    .padding()
                }
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle(embeddedInHub ? "" : "Dispatch")
        .task { await load() }
        .onChange(of: preview?.availableDrivers.map(\.driverId)) { _, _ in
            syncDriverSelection()
        }
        .onChange(of: preview?.undispatchedOrders.map(\.orderId)) { _, ids in
            guard let ids else { return }
            let valid = Set(ids)
            selectedOrderIds = selectedOrderIds.intersection(valid)
        }
        .onChange(of: realtimeHub.reconnectEpoch) { _, _ in
            if executing {
                executing = false
                error = "Connection restored — verify dispatch status before retrying."
            }
        }
        .alert("Execute auto-dispatch?", isPresented: $showAutoConfirm) {
            Button("Confirm", role: .destructive) { Task { await executeAuto() } }
            Button("Cancel", role: .cancel) {}
        } message: {
            Text("Assign pending orders via the optimizer.")
        }
        .alert("Capacity exceeded", isPresented: $showCapacityOverride) {
            Button("Continue", role: .destructive) { Task { await executeManual(forceCapacity: true) } }
            Button("Adjust", role: .cancel) {}
        } message: {
            Text("Selected orders exceed truck capacity. Continue anyway?")
        }
    }

    @ViewBuilder
    private func manualAssignmentSection(preview: SupplierDispatchPreview) -> some View {
        let driver = preview.availableDrivers.first { $0.driverId == selectedDriverId }
        let selectedVolume = preview.undispatchedOrders
            .filter { selectedOrderIds.contains($0.orderId) }
            .reduce(0.0) { $0 + $1.volumeVu }
        let truckMax = driver?.maxVolumeVu ?? 0
        let truckEffective = truckMax * tetrisBuffer
        let capacityExceeded = truckEffective > 0 && selectedVolume > truckEffective

        VStack(alignment: .leading, spacing: SupplierTheme.spacingSM) {
            Text("Manual assignment").font(.headline)
            if truckMax > 0 {
                Text(String(format: "Selected %.1f / %.1f VU", selectedVolume, truckEffective))
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            if capacityExceeded {
                Text("Insufficient truck space for selected orders.")
                    .font(.caption)
                    .foregroundStyle(.red)
            }
            if !preview.availableDrivers.isEmpty {
                Picker("Driver", selection: $selectedDriverId) {
                    ForEach(preview.availableDrivers) { d in
                        let vu = d.maxVolumeVu.map { String(format: " · %.0f VU", $0) } ?? ""
                        Text("\(d.name.isEmpty ? d.driverId : d.name)\(vu)").tag(d.driverId)
                    }
                }
            }
            HStack {
                Text("Orders").font(.subheadline.weight(.semibold))
                Spacer()
                Button("Select all") {
                    selectedOrderIds = Set(preview.undispatchedOrders.map(\.orderId))
                }
                .font(.caption)
            }
            ForEach(preview.undispatchedOrders) { order in
                Toggle(isOn: Binding(
                    get: { selectedOrderIds.contains(order.orderId) },
                    set: { on in
                        if on { selectedOrderIds.insert(order.orderId) }
                        else { selectedOrderIds.remove(order.orderId) }
                    }
                )) {
                    HStack {
                        Text(order.orderId.prefix(12) + "…")
                            .font(.caption.monospaced())
                        Spacer()
                        Text(String(format: "%.1f VU", order.volumeVu))
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }
            }
        }
        .padding()
        .background(SupplierTheme.card)
        .clipShape(RoundedRectangle(cornerRadius: SupplierTheme.radiusLG))
    }

    private func syncDriverSelection() {
        guard let preview else { return }
        let drivers = preview.availableDrivers
        if selectedDriverId.isEmpty, let first = drivers.first {
            selectedDriverId = first.driverId
        } else if !drivers.contains(where: { $0.driverId == selectedDriverId }) {
            selectedDriverId = drivers.first?.driverId ?? ""
        }
    }

    private func executeAuto() async {
        executing = true
        defer { executing = false }
        do {
            let routeFingerprint: String
            if let preview {
                routeFingerprint = """
                {"pending":\(preview.pendingCount ?? 0),"drivers":\(preview.availableDriverCount ?? 0),"undispatched":\(preview.undispatchedOrders.count)}
                """
            } else {
                routeFingerprint = "[]"
            }
            let supplierId = TokenStore.shared.supplierId ?? "supplier"
            let warehouseId = selectedWarehouseId ?? "default"
            let key = SupplierIdempotency.dispatch(
                supplierId: supplierId,
                warehouseId: warehouseId,
                mode: "AUTO",
                routeFingerprint: routeFingerprint
            )
            _ = try await SupplierOperationsService.executeDispatch(
                warehouseId: selectedWarehouseId,
                idempotencyKey: key,
                mode: "AUTO"
            )
            executeMessage = "Auto dispatch executed"
            await load()
        } catch {
            self.error = error.localizedDescription
        }
    }

    private func executeManual(forceCapacity: Bool) async {
        let orderIds = selectedOrderIds.sorted()
        guard !selectedDriverId.isEmpty, !orderIds.isEmpty else { return }
        executing = true
        defer { executing = false }
        do {
            let routeFingerprint = """
            {"mode":"MANUAL","force_capacity":\(forceCapacity),"routes":[{"driver_id":"\(selectedDriverId)","order_ids":\(orderIds)}]}
            """
            let supplierId = TokenStore.shared.supplierId ?? "supplier"
            let warehouseId = selectedWarehouseId ?? "default"
            let key = SupplierIdempotency.dispatch(
                supplierId: supplierId,
                warehouseId: warehouseId,
                mode: "MANUAL",
                routeFingerprint: routeFingerprint
            )
            let result = try await SupplierOperationsService.executeDispatch(
                warehouseId: selectedWarehouseId,
                idempotencyKey: key,
                mode: "MANUAL",
                forceCapacity: forceCapacity,
                routes: [SupplierDispatchManualRoutePayload(driverId: selectedDriverId, orderIds: orderIds)]
            )
            if result.status == "capacity_exceeded" {
                executeMessage = "Truck capacity exceeded"
                showCapacityOverride = true
            } else {
                executeMessage = "Manual dispatch committed"
                selectedOrderIds = []
                showCapacityOverride = false
                await load()
            }
        } catch {
            self.error = error.localizedDescription
        }
    }

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let topology = try await SupplierOperationsService.topology()
            warehouses = topology.warehouses
            preview = try await SupplierOperationsService.dispatchPreview(warehouseId: selectedWarehouseId)
            syncDriverSelection()
        } catch {
            self.error = error.localizedDescription
        }
    }
}

private struct DispatchKpiCard: View {
    let title: String
    let value: String

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(title).font(.caption).foregroundStyle(.secondary)
            Text(value).font(.title2.weight(.semibold))
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding()
        .background(SupplierTheme.card)
        .clipShape(RoundedRectangle(cornerRadius: SupplierTheme.radiusLG))
        .overlay {
            RoundedRectangle(cornerRadius: SupplierTheme.radiusLG)
                .strokeBorder(Color.primary.opacity(0.12), lineWidth: 1)
        }
    }
}
