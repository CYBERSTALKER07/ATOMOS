import SwiftUI

struct InventoryView: View {
    @Environment(WarehouseRealtimeHub.self) private var realtimeHub
    @State private var vm = InventoryViewModel()

    var body: some View {
        @Bindable var vm = vm
        Group {
            if vm.loading && vm.items.isEmpty {
                WarehouseLoadingView(title: "Loading inventory…", message: "Fetching latest stock quantities")
            } else if let error = vm.error, vm.items.isEmpty {
                WarehouseErrorView(message: error, retry: {
                    PegasusHaptics.light()
                    Task { await vm.load() }
                })
            } else if vm.items.isEmpty {
                WarehouseEmptyView(title: "No Inventory Items", message: "There are no matching items.")
            } else {
                InventoryStockList(
                    items: vm.items,
                    policySavingId: vm.policySavingId,
                    policies: vm.policies,
                    onAdjust: { item in vm.adjustItem = item },
                    onPolicyChange: { item, policy in
                        Task { await vm.updatePolicy(item: item, policy: policy) }
                    }
                )
            }
        }
        .background(LabTheme.background)
        .navigationTitle("portal.nav.inventory")
        .toolbar {
            ToolbarItem(placement: .topBarLeading) {
                Toggle(isOn: $vm.lowOnly) {
                    Label("mobile_warehouse.ui.low_stock", systemImage: "exclamationmark.triangle")
                }
                .toggleStyle(.button)
                .controlSize(.small)
                .frame(minHeight: 44)
                .contentShape(Rectangle())
                .accessibilityLabel("Filter low stock items")
            }
            ToolbarItem(placement: .topBarTrailing) {
                Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") {
                    PegasusHaptics.light()
                    Task { await vm.load() }
                }
                .frame(minWidth: 44, minHeight: 44)
                .contentShape(Rectangle())
                .accessibilityLabel("Refresh inventory")
            }
        }
        .task { await vm.load() }
        .refreshable { await vm.load(silent: false) }
        .silentRealtimeRefresh(refreshEpoch: realtimeHub.refreshEpoch, reconnectEpoch: realtimeHub.reconnectEpoch) { silent in
            Task { await vm.load(silent: silent) }
        }
        .onChange(of: vm.lowOnly) { _, _ in
            PegasusHaptics.selection()
            Task { await vm.load() }
        }
        .sheet(item: $vm.adjustItem) { item in
            AdjustInventorySheet(item: item) {
                Task { await vm.load(silent: true) }
            }
        }
    }
}

private struct AdjustInventorySheet: View {
    let onAdjusted: () -> Void
    @Environment(\.dismiss) private var dismiss
    @Environment(WarehouseRealtimeHub.self) private var realtimeHub
    @State private var vm: AdjustInventoryViewModel

    init(item: InventoryItem, onAdjusted: @escaping () -> Void) {
        self.onAdjusted = onAdjusted
        _vm = State(initialValue: AdjustInventoryViewModel(item: item))
    }

    var body: some View {
        @Bindable var vm = vm
        NavigationStack {
            Group {
                if vm.showConfirm {
                    Form {
                        Section {
                            if let newQty = vm.parsedQuantity {
                                Text(L10n.format("mobile_warehouse.ui.change_skulabel_from_quantity_to_newqty_this_affects_retailer_availabili", "\(vm.skuLabel)", "\(vm.item.quantity)", "\(newQty)"))
                                    .font(.subheadline)
                            }
                        }
                        Section("Reason (optional)") {
                            TextField("warehouse_portal.inventory.text.e_g_cycle_count_damaged_goods", text: $vm.reason)
                        }
                        if let error = vm.error {
                            Text(error).foregroundStyle(.red).font(.caption)
                        }
                    }
                    .scrollDismissesKeyboard(.interactively)
                    .navigationTitle("mobile_warehouse.ui.confirm_change")
                    .toolbar {
                        ToolbarItem(placement: .cancellationAction) {
                            Button("common.action.back") { vm.showConfirm = false }
                                .disabled(vm.submitting)
                                .frame(minWidth: 44, minHeight: 44)
                                .contentShape(Rectangle())
                        }
                        ToolbarItem(placement: .confirmationAction) {
                            Button(vm.submitting ? "Saving…" : "Confirm") {
                                PegasusHaptics.medium()
                                Task {
                                    if await vm.save() {
                                        dismiss()
                                        onAdjusted()
                                    }
                                }
                            }
                            .disabled(vm.submitting)
                            .frame(minWidth: 44, minHeight: 44)
                            .contentShape(Rectangle())
                        }
                    }
                } else {
                    Form {
                        Section("Product") {
                            Text(vm.item.productName)
                        }
                        Section("Quantity") {
                            TextField("mobile_warehouse.ui.new_quantity", text: $vm.qty)
                                .keyboardType(.numberPad)
                        }
                    }
                    .scrollDismissesKeyboard(.interactively)
                    .navigationTitle("mobile_warehouse.ui.adjust_inventory")
                    .toolbar {
                        ToolbarItem(placement: .cancellationAction) {
                            Button("common.action.cancel") { dismiss() }
                                .frame(minWidth: 44, minHeight: 44)
                                .contentShape(Rectangle())
                        }
                        ToolbarItem(placement: .confirmationAction) {
                            Button("mobile_warehouse.ui.review") {
                                PegasusHaptics.light()
                                vm.showConfirm = true
                            }
                            .disabled(!vm.canReview)
                            .frame(minWidth: 44, minHeight: 44)
                            .contentShape(Rectangle())
                        }
                    }
                }
            }
            .onChange(of: realtimeHub.reconnectEpoch) { _, _ in
                vm.handleReconnect()
            }
        }
    }
}
