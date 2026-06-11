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
                        .toolbar { filterMenu }
                }
            }
        }
        .task(id: statusFilter) { await load() }
    }

    private var phoneContent: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading orders…")
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
    }

    private var splitContent: some View {
        NavigationSplitView {
            ordersList
                .navigationTitle("Orders")
                .toolbar { filterMenu }
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
                SupplierLoadingView(title: "Loading orders…")
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
    }

    @ToolbarContentBuilder
    private var filterMenu: some ToolbarContent {
        ToolbarItem(placement: .topBarTrailing) {
            Menu {
                ForEach(filters, id: \.self) { filter in
                    Button(filter.isEmpty ? "All" : filter) {
                        statusFilter = filter
                    }
                }
            } label: {
                Label("Filter", systemImage: "line.3.horizontal.decrease.circle")
            }
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
        VStack(alignment: .leading, spacing: 4) {
            HStack {
                Text(order.orderId)
                    .font(.subheadline.monospaced())
                    .lineLimit(1)
                Spacer()
                Text(order.status)
                    .font(.caption.bold())
                    .padding(.horizontal, 8)
                    .padding(.vertical, 2)
                    .background(SupplierTheme.tertiaryBackground, in: Capsule())
            }
            Text(MoneyFormat.minor(order.totalMinor, currency: order.currency))
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .padding(.vertical, 2)
    }
}

private struct OrderDetailPanel: View {
    let order: SupplierOrder

    var body: some View {
        List {
            Section("Order") {
                LabeledContent("ID", value: order.orderId)
                LabeledContent("Retailer", value: order.retailerId)
                LabeledContent("Status", value: order.status)
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
