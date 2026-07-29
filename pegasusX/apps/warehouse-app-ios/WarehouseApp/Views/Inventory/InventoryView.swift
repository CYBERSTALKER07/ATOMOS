import SwiftUI

struct InventoryView: View {
    @Environment(WarehouseRealtimeHub.self) private var realtimeHub
    @State private var items: [InventoryItem] = []
    @State private var loading = true
    @State private var error: String?
    @State private var lowOnly = false
    @State private var adjustItem: InventoryItem?
    @State private var policySavingId: String?

    private let policies = ["INHERIT", "REJECT", "ACCEPT_BACKORDER"]

    var body: some View {
        NavigationStack {
            Group {
                if loading && items.isEmpty {
                    WarehouseLoadingView(title: "Loading inventory…", message: "Fetching latest stock quantities")
                } else if let error, items.isEmpty {
                    WarehouseErrorView(message: error, retry: { load() })
                } else if items.isEmpty {
                    WarehouseEmptyView(title: "No Inventory Items", message: "There are no matching items.")
                } else {
                    InventoryStockList(
                        items: items,
                        policySavingId: policySavingId,
                        policies: policies,
                        onAdjust: { item in adjustItem = item },
                        onPolicyChange: { item, policy in updatePolicy(item: item, policy: policy) }
                    )
                }
            }
            .background(LabTheme.background)
            .navigationTitle("Inventory")
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Toggle(isOn: $lowOnly) {
                        Label("Low Stock", systemImage: "exclamationmark.triangle")
                    }
                    .toggleStyle(.button)
                    .controlSize(.small)
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Refresh", systemImage: "arrow.clockwise") { load() }
                }
            }
            .task { load() }
            .refreshable { load(silent: false) }
            .silentRealtimeRefresh(refreshEpoch: realtimeHub.refreshEpoch, reconnectEpoch: realtimeHub.reconnectEpoch) { silent in
                load(silent: silent)
            }
            .onChange(of: lowOnly) { load() }
            .sheet(item: $adjustItem) { item in
                AdjustInventorySheet(item: item) { load() }
            }
        }
    }

    private func load(silent: Bool = false) {
        if !silent { loading = true }
        error = nil
        Task {
            do {
                let resp = try await WarehouseService.inventory(lowStock: lowOnly)
                items = resp.items
            } catch {
                if !silent { self.error = error.localizedDescription }
            }
            if !silent { loading = false }
        }
    }

    private func updatePolicy(item: InventoryItem, policy: String) {
        let current = item.outOfStockPolicy?.isEmpty == false ? item.outOfStockPolicy! : "INHERIT"
        guard policy != current else { return }
        policySavingId = item.productId
        Task {
            do {
                try await WarehouseService.patchInventoryPolicy(productId: item.productId, policy: policy)
                load()
            } catch {
                self.error = error.localizedDescription
            }
            policySavingId = nil
        }
    }
}

private struct AdjustInventorySheet: View {
    let item: InventoryItem
    let onAdjusted: () -> Void
    @Environment(\.dismiss) private var dismiss
    @Environment(WarehouseRealtimeHub.self) private var realtimeHub
    @State private var qty: String
    @State private var reason = ""
    @State private var showConfirm = false
    @State private var submitting = false
    @State private var error: String?

    init(item: InventoryItem, onAdjusted: @escaping () -> Void) {
        self.item = item
        self.onAdjusted = onAdjusted
        _qty = State(initialValue: "\(item.quantity)")
    }

    private var skuLabel: String {
        item.productId
    }

    var body: some View {
        NavigationStack {
            Group {
                if showConfirm {
                    Form {
                        Section {
                            if let newQty = Int(qty) {
                                Text("Change \(skuLabel) from \(item.quantity) to \(newQty)? This affects retailer availability immediately.")
                                    .font(.subheadline)
                            }
                        }
                        Section("Reason (optional)") {
                            TextField("e.g. cycle count, damaged goods", text: $reason)
                        }
                        if let error {
                            Text(error).foregroundStyle(.red).font(.caption)
                        }
                    }
                    .navigationTitle("Confirm change")
                    .toolbar {
                        ToolbarItem(placement: .cancellationAction) {
                            Button("Back") { showConfirm = false }
                                .disabled(submitting)
                        }
                        ToolbarItem(placement: .confirmationAction) {
                            Button(submitting ? "Saving…" : "Confirm") { save() }
                                .disabled(submitting)
                        }
                    }
                } else {
                    Form {
                        Section("Product") {
                            Text(item.productName)
                        }
                        Section("Quantity") {
                            TextField("New Quantity", text: $qty)
                                .keyboardType(.numberPad)
                        }
                    }
                    .navigationTitle("Adjust Inventory")
                    .toolbar {
                        ToolbarItem(placement: .cancellationAction) {
                            Button("Cancel") { dismiss() }
                        }
                        ToolbarItem(placement: .confirmationAction) {
                            Button("Review") { showConfirm = true }
                                .disabled(Int(qty) == nil || Int(qty) == item.quantity)
                        }
                    }
                }
            }
            .onChange(of: realtimeHub.reconnectEpoch) { _, _ in
                if submitting {
                    submitting = false
                    error = "Connection restored — verify status before retrying."
                }
            }
        }
    }

    private func save() {
        guard let q = Int(qty) else { return }
        submitting = true
        error = nil
        Task {
            do {
                let trimmedReason = reason.trimmingCharacters(in: .whitespacesAndNewlines)
                try await WarehouseService.adjustInventory(
                    productId: item.productId,
                    quantity: q
                )
                dismiss()
                onAdjusted()
            } catch {
                self.error = error.localizedDescription
            }
            submitting = false
        }
    }
}
