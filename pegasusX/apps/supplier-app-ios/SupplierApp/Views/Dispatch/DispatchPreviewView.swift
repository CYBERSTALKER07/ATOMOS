import SwiftUI

struct DispatchPreviewView: View {
    @State private var preview: SupplierDispatchPreview?
    @State private var warehouses: [SupplierTopologyWarehouse] = []
    @State private var selectedWarehouseId: String?
    @State private var loading = true
    @State private var error: String?

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
                        Button("Refresh preview") { Task { await load() } }
                            .disabled(loading)
                    }
                }
            }
        }
        .navigationTitle("Dispatch preview")
        .task { await load() }
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
