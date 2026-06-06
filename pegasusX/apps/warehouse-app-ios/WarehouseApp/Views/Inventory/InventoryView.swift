import SwiftUI

struct InventoryView: View {
    @State private var items: [InventoryItem] = []
    @State private var loading = true
    @State private var error: String?
    @State private var lowOnly = false
    @State private var adjustItem: InventoryItem?

    var body: some View {
        NavigationStack {
            Group {
                if loading {
                    ProgressView()
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else if let error {
                    ContentUnavailableView {
                        Label("Error", systemImage: "exclamationmark.triangle")
                    } description: {
                        Text(error)
                    } actions: {
                        Button("Retry") { load() }
                    }
                } else if items.isEmpty {
                    ContentUnavailableView("No Inventory", systemImage: "archivebox", description: Text("Inventory is empty"))
                } else {
                    List(items) { item in
                        HStack {
                            VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                Text(item.productName)
                                    .font(.headline)
                                Text("Qty: \(item.quantity) · Reorder: \(item.reorderThreshold)")
                                    .font(.subheadline)
                                    .foregroundStyle(.secondary)
                            }
                            Spacer()
                            if item.quantity <= item.reorderThreshold {
                                Text("LOW")
                                    .font(.caption.bold())
                                    .padding(.horizontal, LabTheme.spacingSM)
                                    .padding(.vertical, LabTheme.spacingXS)
                                    .foregroundStyle(.white)
                                    .background(.red, in: Capsule())
                            }
                            Button("Adjust") { adjustItem = item }
                                .buttonStyle(.bordered)
                                .controlSize(.small)
                        }
                    }
                    .listStyle(.insetGrouped)
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
            .refreshable { load() }
            .onChange(of: lowOnly) { load() }
            .sheet(item: $adjustItem) { item in
                AdjustInventorySheet(item: item) { load() }
            }
        }
    }

    private func load() {
        loading = true
        error = nil
        Task {
            do {
                let resp = try await WarehouseService.inventory(lowStock: lowOnly)
                items = resp.items
            } catch {
                self.error = error.localizedDescription
            }
            loading = false
        }
    }
}

private struct AdjustInventorySheet: View {
    let item: InventoryItem
    let onAdjusted: () -> Void
    @Environment(\.dismiss) private var dismiss
    @State private var qty: String
    @State private var submitting = false
    @State private var error: String?

    init(item: InventoryItem, onAdjusted: @escaping () -> Void) {
        self.item = item
        self.onAdjusted = onAdjusted
        _qty = State(initialValue: "\(item.quantity)")
    }

    var body: some View {
        NavigationStack {
            Form {
                Section("Product") {
                    Text(item.productName)
                }
                Section("Quantity") {
                    TextField("New Quantity", text: $qty)
                        .keyboardType(.numberPad)
                }
                if let error {
                    Text(error).foregroundStyle(.red).font(.caption)
                }
            }
            .navigationTitle("Adjust Inventory")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Save") { save() }
                        .disabled(submitting || Int(qty) == nil)
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
                try await WarehouseService.adjustInventory(productId: item.productId, quantity: q)
                dismiss()
                onAdjusted()
            } catch {
                self.error = error.localizedDescription
            }
            submitting = false
        }
    }
}
