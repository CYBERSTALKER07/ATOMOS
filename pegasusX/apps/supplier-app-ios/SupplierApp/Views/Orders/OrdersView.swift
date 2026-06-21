import SwiftUI

struct OrdersView: View {
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    @State private var vm = OrdersViewModel()

    var body: some View {
        Group {
            if horizontalSizeClass == .regular {
                splitContent
            } else {
                NavigationStack {
                    phoneContent
                        .navigationTitle("Orders")
                        .toolbar { ordersToolbar }
                }
            }
        }
        .background(SupplierTheme.background)
        .task(id: vm.statusFilter) { await vm.load() }
        .onChange(of: realtimeHub.refreshEpoch) { _, _ in
            Task { await vm.load(silent: true) }
        }
        .onChange(of: realtimeHub.reconnectEpoch) { _, _ in
            Task { await vm.load(silent: true) }
        }
    }

    private var phoneContent: some View {
        VStack(spacing: 0) {
            filterTabs
            Group {
                if vm.loading {
                    SupplierLoadingView(
                        title: "Loading orders",
                        message: "Fetching your supplier order queue."
                    )
                } else if let error = vm.error {
                    SupplierErrorView(message: error) { Task { await vm.load() } }
                } else if vm.orders.isEmpty {
                    SupplierEmptyView(title: "No orders", message: "Nothing in this queue.")
                } else {
                    List(vm.orders) { order in
                        NavigationLink {
                            OrderDetailPanel(order: order, vm: vm)
                        } label: {
                            OrderRow(order: order)
                        }
                    }
                    .listStyle(.insetGrouped)
                }
            }
            .refreshable { await vm.load(silent: true) }
        }
    }

    private var splitContent: some View {
        NavigationSplitView {
            VStack(spacing: 0) {
                filterTabs
                ordersList
            }
            .navigationTitle("Orders")
            .toolbar { ordersToolbar }
        } detail: {
            if let selection = vm.selection {
                OrderDetailPanel(order: selection, vm: vm)
            } else {
                ContentUnavailableView("Select an order", systemImage: "shippingbox")
            }
        }
    }

    private var filterTabs: some View {
        Picker("Filter", selection: $vm.statusFilter) {
            ForEach(vm.filters, id: \.id) { filter in
                Text(filter.label).tag(filter.id)
            }
        }
        .pickerStyle(.segmented)
        .padding(.horizontal)
        .padding(.vertical, SupplierTheme.spacingSM)
    }

    private var ordersList: some View {
        Group {
            if vm.loading {
                SupplierLoadingView(
                    title: "Loading orders",
                    message: "Fetching your supplier order queue."
                )
            } else if let error = vm.error {
                SupplierErrorView(message: error) { Task { await vm.load() } }
            } else if vm.orders.isEmpty {
                SupplierEmptyView(title: "No orders", message: "Nothing in this queue.")
            } else {
                List(vm.orders, selection: $vm.selection) { order in
                    OrderRow(order: order)
                        .tag(order)
                }
                .listStyle(.sidebar)
            }
        }
        .refreshable { await vm.load(silent: true) }
    }

    @ToolbarContentBuilder
    private var ordersToolbar: some ToolbarContent {
        ToolbarItem(placement: .topBarTrailing) {
            Button("Refresh", systemImage: "arrow.clockwise") {
                Task { await vm.load(silent: true) }
            }
            .labelStyle(.iconOnly)
        }
    }
}

struct OrderRow: View {
    let order: SupplierOrder
    var showWarehouseMenu: Bool = false
    var onDelay: (() -> Void)?
    var onReject: (() -> Void)?

    var body: some View {
        VStack(alignment: .leading, spacing: SupplierTheme.spacingXS) {
            HStack {
                Text(order.retailerId.isEmpty ? String(order.orderId.prefix(12)) : order.retailerId)
                    .font(.subheadline.weight(.semibold))
                    .lineLimit(1)
                Spacer()
                SupplierStatusBadge(text: order.status)
            }
            Text(MoneyFormat.minor(order.totalMinor, currency: order.currency))
                .font(.caption)
                .foregroundStyle(.secondary)
            Text(order.orderId)
                .font(.caption2.monospaced())
                .foregroundStyle(.secondary)
        }
        .padding(.vertical, SupplierTheme.spacingXS)
        .contextMenu {
            if showWarehouseMenu {
                if let onDelay {
                    Button("Delay delivery") { onDelay() }
                }
                if let onReject {
                    Button("Reject", role: .destructive) { onReject() }
                }
            }
        }
    }
}

struct OrderDetailPanel: View {
    let order: SupplierOrder
    @Bindable var vm: OrdersViewModel
    @State private var warehouseDetail: WarehouseOrderDetail?
    @State private var opsReason = ""
    @State private var proposeDate = Date()
    @State private var showProposeSheet = false
    @State private var showRejectDialog = false

    var body: some View {
        List {
            Section("Order") {
                LabeledContent("ID", value: order.orderId)
                LabeledContent("Retailer", value: warehouseDetail?.retailerName ?? order.retailerId)
                LabeledContent("Status") {
                    SupplierStatusBadge(text: warehouseDetail?.state ?? warehouseDetail?.status ?? order.status)
                }
                if let decision = order.decision, !decision.isEmpty {
                    LabeledContent("Decision", value: decision)
                }
                LabeledContent("Total", value: MoneyFormat.minor(order.totalMinor, currency: order.currency))
                LabeledContent("Updated", value: order.updatedAt)
            }

            if let items = warehouseDetail?.lineItems, !items.isEmpty {
                Section("Line items") {
                    ForEach(items) { item in
                        VStack(alignment: .leading) {
                            Text(item.productName ?? item.productId ?? "—")
                            Text("Qty \(item.quantity ?? 0)")
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                    }
                }
            }

            if vm.canWarehouseOps(for: order) {
                Section("Warehouse admin") {
                    Button("Delay delivery") { showProposeSheet = true }
                    Button("Cancel order", role: .destructive) { showRejectDialog = true }
                }
            }
        }
        .navigationTitle("Order")
        .task { await loadWarehouseDetail() }
        .sheet(isPresented: $showProposeSheet) {
            NavigationStack {
                Form {
                    DatePicker("New delivery date", selection: $proposeDate, displayedComponents: .date)
                    TextField("Reason (required)", text: $opsReason, axis: .vertical)
                }
                .navigationTitle("Delay delivery")
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) { Button("Cancel") { showProposeSheet = false } }
                    ToolbarItem(placement: .confirmationAction) {
                        Button("Notify retailer") {
                            Task {
                                await vm.proposeWarehouseOrder(
                                    order,
                                    proposedDeliveryDate: isoDeliveryDate(from: proposeDate),
                                    reason: opsReason,
                                )
                                opsReason = ""
                                showProposeSheet = false
                                await loadWarehouseDetail()
                            }
                        }
                        .disabled(opsReason.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                    }
                }
            }
            .presentationDetents([.medium])
        }
        .alert("Cancel order", isPresented: $showRejectDialog) {
            Button("Reject", role: .destructive) {
                Task {
                    await vm.rejectWarehouseOrder(order, reason: opsReason)
                    opsReason = ""
                    await loadWarehouseDetail()
                }
            }
            Button("Cancel", role: .cancel) { opsReason = "" }
        } message: {
            Text("Reason is required for reject.")
        }
    }

    private func isoDeliveryDate(from date: Date) -> String {
        var components = Calendar.current.dateComponents(in: TimeZone(secondsFromGMT: 5 * 3600)!, from: date)
        components.hour = 12
        components.minute = 0
        components.second = 0
        let noon = Calendar.current.date(from: components) ?? date
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime]
        formatter.timeZone = TimeZone(secondsFromGMT: 5 * 3600)
        return formatter.string(from: noon)
    }

    private func loadWarehouseDetail() async {
        guard let warehouseId = order.warehouseId, !warehouseId.isEmpty else { return }
        do {
            warehouseDetail = try await SupplierOperationsService.getWarehouseOrder(
                orderId: order.orderId,
                warehouseId: warehouseId
            )
        } catch {
            warehouseDetail = nil
        }
    }
}
