import SwiftUI

struct OrdersView: View {
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @State private var orders: [SupplierOrder] = []
    @State private var loading = true
    @State private var error: String?
    @State private var statusFilter = "PENDING"
    @State private var selection: SupplierOrder?

    private let filters = ["", "PENDING", "AWAITING_REVIEW", "COMPLETED"]

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
        .task(id: statusFilter) { await load() }
    }

    private var phoneContent: some View {
        Group {
            if loading {
                SupplierLoadingView(
                    title: "Loading orders",
                    message: "Fetching your supplier order queue."
                )
            } else if let error {
                SupplierErrorView(message: error) { Task { await load() } }
            } else if orders.isEmpty {
                SupplierEmptyView(title: "No orders", message: "Nothing in this queue.")
            } else {
                List(orders) { order in
                    OrderRow(order: order)
                }
                .listStyle(.insetGrouped)
            }
        }
        .refreshable { await load(silent: true) }
    }

    private var splitContent: some View {
        NavigationSplitView {
            ordersList
                .navigationTitle("Orders")
                .toolbar { ordersToolbar }
        } detail: {
            if let selection {
                OrderDetailPanel(order: selection)
            } else {
                ContentUnavailableView("Select an order", systemImage: "shippingbox")
            }
        }
    }

    private var ordersList: some View {
        Group {
            if loading {
                SupplierLoadingView(
                    title: "Loading orders",
                    message: "Fetching your supplier order queue."
                )
            } else if let error {
                SupplierErrorView(message: error) { Task { await load() } }
            } else if orders.isEmpty {
                SupplierEmptyView(title: "No orders", message: "Nothing in this queue.")
            } else {
                List(orders, selection: $selection) { order in
                    OrderRow(order: order)
                        .tag(order)
                }
                .listStyle(.sidebar)
            }
        }
        .refreshable { await load(silent: true) }
    }

    @ToolbarContentBuilder
    private var ordersToolbar: some ToolbarContent {
        ToolbarItem(placement: .topBarTrailing) {
            Menu {
                ForEach(filters, id: \.self) { filter in
                    Button {
                        statusFilter = filter
                    } label: {
                        if filter == statusFilter {
                            Label(filter.isEmpty ? "All" : filter, systemImage: "checkmark")
                        } else {
                            Text(filter.isEmpty ? "All" : filter)
                        }
                    }
                }
            } label: {
                Label("Filter", systemImage: "line.3.horizontal.decrease.circle")
            }
        }
        ToolbarItem(placement: .topBarTrailing) {
            Button("Refresh", systemImage: "arrow.clockwise") {
                Task { await load(silent: true) }
            }
            .labelStyle(.iconOnly)
        }
    }

    @MainActor
    private func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        do {
            let response = try await SupplierService.orders(
                status: statusFilter.isEmpty ? nil : statusFilter,
                limit: 500
            )
            orders = response.orders
            if selection == nil { selection = orders.first }
        } catch {
            if !silent { self.error = error.localizedDescription }
        }
        loading = false
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

    var body: some View {
        NavigationStack {
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
            }
            .navigationTitle("Order")
        }
    }
}
