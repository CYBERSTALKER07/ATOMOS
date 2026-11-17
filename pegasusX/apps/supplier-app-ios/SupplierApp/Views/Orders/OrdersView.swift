import SwiftUI

struct OrdersView: View {
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    @State private var vm = OrdersViewModel()

    var body: some View {
        @Bindable var vm = vm
        Group {
            if horizontalSizeClass == .regular {
                splitContent
            } else {
                phoneContent
                    .navigationTitle("portal.nav.orders")
                    .toolbar { ordersToolbar }
            }
        }
        .background(SupplierTheme.background)
        .task(id: vm.loadIdentity) { await vm.load() }
        .onChange(of: realtimeHub.refreshEpoch) { _, _ in
            Task { await vm.load(silent: true) }
        }
        .onChange(of: realtimeHub.reconnectEpoch) { _, _ in
            Task { await vm.load(silent: true) }
        }
        .alert("mobile_supplier.chrome.order_update", isPresented: $vm.showReassignMessageAlert) {
            Button("mobile_supplier.chrome.ok", role: .cancel) { vm.reassignMessage = nil }
        } message: {
            Text(vm.reassignMessage ?? "")
        }
    }

    private var phoneContent: some View {
        VStack(spacing: 0) {
            filterTabs
            OrdersList(vm: vm)
        }
    }

    private var splitContent: some View {
        @Bindable var vm = vm
        return NavigationSplitView {
            VStack(spacing: 0) {
                filterTabs
                OrdersList(vm: vm)
            }
            .navigationTitle("portal.nav.orders")
            .toolbar { ordersToolbar }
        } detail: {
            if let selection = vm.selection {
                OrderDetailPanel(order: selection, vm: vm)
            } else {
                ContentUnavailableView("mobile_supplier.chrome.select_an_order", systemImage: "shippingbox")
            }
        }
    }

    private var filterTabs: some View {
        @Bindable var vm = vm
        return Picker("mobile_supplier.chrome.filter", selection: $vm.statusFilter) {
            ForEach(vm.filters, id: \.id) { filter in
                Text(LocalizedStringKey(filter.label)).tag(filter.id)
            }
        }
        .pickerStyle(.segmented)
        .padding(.horizontal)
        .padding(.vertical, SupplierTheme.spacingSM)
        .onChange(of: vm.statusFilter) { _, newValue in
            PegasusHaptics.selection()
            vm.setCoarseFilter(newValue)
        }
    }

    @ToolbarContentBuilder
    private var ordersToolbar: some ToolbarContent {
        ToolbarItem(placement: .topBarTrailing) {
            Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") {
                PegasusHaptics.light()
                Task { await vm.load(silent: true) }
            }
            .labelStyle(.iconOnly)
            .frame(minWidth: 44, minHeight: 44)
        }
    }
}
