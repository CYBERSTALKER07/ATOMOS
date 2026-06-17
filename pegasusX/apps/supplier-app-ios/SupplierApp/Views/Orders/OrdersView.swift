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

private struct OrderRow: View {
    let order: SupplierOrder

    var body: some View {
        VStack(alignment: .leading, spacing: SupplierTheme.spacingXS) {
            HStack {
                Text(order.orderId)
                    .font(.subheadline.monospaced())
                    .lineLimit(1)
                Spacer()
                SupplierStatusBadge(text: order.status)
            }
            Text(MoneyFormat.minor(order.totalMinor, currency: order.currency))
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .padding(.vertical, SupplierTheme.spacingXS)
    }
}

private struct OrderDetailPanel: View {
    let order: SupplierOrder
    @Bindable var vm: OrdersViewModel
    @State private var note = ""

    private var canVet: Bool {
        ["PENDING", "AWAITING_REVIEW"].contains(order.status.uppercased())
    }

    var body: some View {
        List {
            Section("Order") {
                LabeledContent("ID", value: order.orderId)
                LabeledContent("Retailer", value: order.retailerId)
                LabeledContent("Status") {
                    SupplierStatusBadge(text: order.status)
                }
                if let decision = order.decision, !decision.isEmpty {
                    LabeledContent("Decision", value: decision)
                }
                LabeledContent("Total", value: MoneyFormat.minor(order.totalMinor, currency: order.currency))
                LabeledContent("Updated", value: order.updatedAt)
            }

            if canVet {
                Section("Vet decision") {
                    TextField("Note (optional)", text: $note)
                    HStack {
                        Button("Approve") {
                            Task { await vm.vet(order: order, decision: "APPROVED", note: note) }
                        }
                        .buttonStyle(.borderedProminent)
                        .disabled(vm.vettingOrderId == order.orderId)

                        Button("Reject", role: .destructive) {
                            Task { await vm.vet(order: order, decision: "REJECTED", note: note) }
                        }
                        .disabled(vm.vettingOrderId == order.orderId)
                    }
                }
            }
        }
        .navigationTitle("Order")
    }
}
