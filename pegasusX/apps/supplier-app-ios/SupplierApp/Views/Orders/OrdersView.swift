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
        .onChange(of: vm.reassignMessage) { _, newValue in
            if let msg = newValue {
                // simple print or we could show an alert. But since we dismiss it, 
                // typically we just print it or you could add a snackbar equivalent in swiftui
                print("Reassign message: \(msg)")
            }
        }
    }

    private var phoneContent: some View {
        VStack(spacing: 0) {
            filterTabs
            OrdersList(vm: vm)
        }
    }

    private var splitContent: some View {
        NavigationSplitView {
            VStack(spacing: 0) {
                filterTabs
                OrdersList(vm: vm)
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
