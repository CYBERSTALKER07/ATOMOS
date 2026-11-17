import SwiftUI

struct OrdersList: View {
    @Bindable var vm: OrdersViewModel
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass

    var body: some View {
        Group {
            if vm.loading && vm.orders.isEmpty {
                SupplierLoadingView(
                    title: "mobile_supplier.loading.orders",
                    message: "mobile_supplier.loading.orders_message"
                )
            } else if let error = vm.error, vm.orders.isEmpty {
                SupplierErrorView(message: error) { Task { await vm.load() } }
            } else if vm.orders.isEmpty {
                SupplierEmptyView(title: "mobile_supplier.empty.no_orders", message: "mobile_supplier.empty.nothing_in_queue")
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
        .safeAreaInset(edge: .top) {
            if let notice = vm.notice, !notice.isEmpty, vm.error == nil, !vm.loading {
                Text(notice)
                    .font(.footnote)
                    .foregroundStyle(.primary)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(.horizontal)
                    .padding(.vertical, 8)
                    .background(.yellow.opacity(0.25))
                    .accessibilityLabel(notice)
            }
        }
        .refreshable { await vm.load(silent: true) }
    }
}
