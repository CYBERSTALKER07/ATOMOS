import SwiftUI

struct DispatchPreviewView: View {
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    var embeddedInHub: Bool = false
    @State private var preview: SupplierDispatchPreview?
    @State private var warehouses: [SupplierTopologyWarehouse] = []
    @State private var selectedWarehouseId: String?
    @State private var loading = true
    @State private var executing = false
    @State private var error: String?
    @State private var showExecuteConfirm = false

    var body: some View {
        Group {
            if loading && preview == nil {
                SupplierLoadingView(title: "Loading dispatch preview…")
            } else if let error, preview == nil {
                SupplierErrorView(message: error) { Task { await load() } }
            } else {
                ScrollView {
                    VStack(alignment: .leading, spacing: SupplierTheme.spacingLG) {
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
                                    Text("Supplier preview fingerprint differs from the warehouse floor plan. Refresh warehouse dispatch before committing.")
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                    if let supplier = preview.planFingerprint, let warehouse = preview.warehousePlanFingerprint {
                                        Text("supplier \(supplier.prefix(12))… · warehouse \(warehouse.prefix(12))…")
                                            .font(.caption2.monospaced())
                                            .foregroundStyle(.secondary)
                                    }
                                }
                                .padding()
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .background(Color.red.opacity(0.08))
                                .clipShape(RoundedRectangle(cornerRadius: SupplierTheme.radiusLG))
                            }
                            HStack(spacing: SupplierTheme.spacingMD) {
                                DispatchKpiCard(title: "Pending", value: "\(preview.pendingCount ?? 0)")
                                DispatchKpiCard(title: "Drivers", value: "\(preview.availableDriverCount ?? 0)")
                                DispatchKpiCard(title: "Undispatched", value: "\(preview.undispatchedOrderCount ?? 0)")
                            }
                            if !preview.proposedRoutes.isEmpty {
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
                                    ForEach(Array(preview.proposedRoutes.enumerated()), id: \.offset) { index, route in
                                        let label = route.driverName?.isEmpty == false
                                            ? route.driverName!
                                            : (route.driverId?.isEmpty == false ? route.driverId! : "Route \(index + 1)")
                                        let stops = route.stopCount ?? route.orderIds.count
                                        Text("\(label) · \(stops) stops")
                                            .font(.caption)
                                            .foregroundStyle(.secondary)
                                    }
                                }
                            }
                        }
                        VStack(spacing: SupplierTheme.spacingSM) {
                            Button("Execute auto-dispatch") { showExecuteConfirm = true }
                                .buttonStyle(.borderedProminent)
                                .tint(.primary)
                                .disabled(loading || executing || preview == nil)
                            Button("Refresh preview") { Task { await load() } }
                                .disabled(loading || executing)
                        }
                    }
                    .padding()
                }
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle(embeddedInHub ? "" : "Dispatch")
        .task { await load() }
        .onChange(of: realtimeHub.reconnectEpoch) { _, _ in
            if executing {
                executing = false
                error = "Connection restored — verify dispatch status before retrying."
            }
        }
        .alert("Execute dispatch?", isPresented: $showExecuteConfirm) {
            Button("Confirm", role: .destructive) { Task { await execute() } }
            Button("Cancel", role: .cancel) {}
        } message: {
            Text("Assign pending orders to available drivers.")
        }
    }

    private func execute() async {
        executing = true
        defer { executing = false }
        do {
            let routeFingerprint: String
            if let preview {
                routeFingerprint = """
                {"pending":\(preview.pendingCount ?? 0),"drivers":\(preview.availableDriverCount ?? 0),"undispatched":\(preview.undispatchedOrderCount ?? 0)}
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
            try await SupplierOperationsService.executeDispatch(
                warehouseId: selectedWarehouseId,
                idempotencyKey: key
            )
            await load()
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
