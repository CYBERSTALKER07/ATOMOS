import SwiftUI

struct DispatchPreviewView: View {
    @State private var preview: SupplierDispatchPreview?
    @State private var warehouses: [SupplierTopologyWarehouse] = []
    @State private var selectedWarehouseId: String?
    @State private var loading = true
    @State private var error: String?
    @State private var showExecuteConfirm = false

    var body: some View {
        Group {
            if loading && preview == nil {
                SupplierLoadingView(title: "Loading dispatch preview…")
            } else if let error, preview == nil {
                SupplierErrorView(message: error) { Task { await load() } }
            } else {
                Form {
                    if !warehouses.isEmpty {
                        Section("Warehouse scope") {
                            ForEach(warehouses) { wh in
                                Button {
                                    selectedWarehouseId = selectedWarehouseId == wh.warehouseId ? nil : wh.warehouseId
                                    Task { await load() }
                                } label: {
                                    HStack {
                                        Text(wh.name.isEmpty ? wh.warehouseId : wh.name)
                                        Spacer()
                                        if selectedWarehouseId == wh.warehouseId {
                                            Image(systemName: "checkmark")
                                        }
                                    }
                                }
                            }
                        }
                    }
                    if let preview {
                        Section("Snapshot") {
                            LabeledContent("Pending orders", value: "\(preview.pendingCount ?? 0)")
                            LabeledContent("Available drivers", value: "\(preview.availableDriverCount ?? 0)")
                            if let undispatched = preview.undispatchedOrderCount {
                                LabeledContent("Undispatched bucket", value: "\(undispatched)")
                            }
                        }
                    }
                    Section {
                        Button("Execute auto-dispatch") { showExecuteConfirm = true }
                            .disabled(loading || preview == nil)
                        Button("Refresh preview") { Task { await load() } }
                            .disabled(loading)
                    }
                }
            }
        }
        .navigationTitle("Dispatch preview")
        .task { await load() }
        .alert("Execute dispatch?", isPresented: $showExecuteConfirm) {
            Button("Confirm", role: .destructive) { Task { await execute() } }
            Button("Cancel", role: .cancel) {}
        } message: {
            Text("Assign pending orders to available drivers.")
        }
    }

    private func execute() async {
        loading = true
        defer { loading = false }
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
