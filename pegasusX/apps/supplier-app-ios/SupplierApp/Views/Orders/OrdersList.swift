import SwiftUI

struct OrdersList: View {
    @Bindable var vm: OrdersViewModel
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass

    var body: some View {
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
                if horizontalSizeClass == .regular {
                    List(vm.orders, selection: $vm.selection) { order in
                        OrderRow(order: order).tag(order)
                    }
                    .listStyle(.sidebar)
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
        }
        .refreshable { await vm.load(silent: true) }
    }
}
