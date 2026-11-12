import SwiftUI

struct WarehousesView: View {
    @State private var vm = TopologyNodesListViewModel()

    var body: some View {
        @Bindable var vm = vm
        Group {
            if vm.loading && vm.warehouses.isEmpty {
                SupplierLoadingView(title: "mobile_supplier.loading.warehouses")
            } else if let error = vm.error, vm.warehouses.isEmpty {
                SupplierErrorView(message: error) { Task { await vm.load() } }
            } else if vm.warehouses.isEmpty {
                TopologyCenteredEmptyState(
                    title: "mobile_supplier.empty.no_warehouses",
                    message: "mobile_supplier.empty.warehouses_message",
                    actionLabel: "mobile_supplier.empty.add_first_warehouse"
                ) {
                    PegasusHaptics.light()
                    vm.showAdd = true
                }
            } else {
                VStack(alignment: .leading, spacing: 12) {
                    Text("mobile_supplier.chrome.pin_stores_desktop")
                        .font(.footnote)
                        .foregroundStyle(.secondary)
                        .padding(.horizontal)
                    WarehouseList(warehouses: vm.warehouses)
                }
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("portal.nav.warehouses")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    PegasusHaptics.light()
                    vm.showAdd = true
                } label: {
                    Image(systemName: "plus")
                        .font(.body.weight(.semibold))
                        .foregroundStyle(SupplierTheme.label)
                        .frame(minWidth: 44, minHeight: 44)
                }
                .accessibilityLabel(Text("supplier_portal.warehouses.components.warehouse_form.text.add_warehouse"))
            }
        }
        .sheet(isPresented: $vm.showAdd) {
            AddWarehouseSheet {
                Task { await vm.load(silent: true) }
            }
        }
        .task { await vm.load() }
        .refreshable {
            PegasusHaptics.light()
            await vm.load(silent: true)
        }
    }
}
