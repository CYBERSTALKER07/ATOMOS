import SwiftUI

struct FactoriesView: View {
    @State private var vm = TopologyNodesListViewModel()

    var body: some View {
        @Bindable var vm = vm
        Group {
            if vm.loading && vm.factories.isEmpty {
                SupplierLoadingView(title: "mobile_supplier.loading.factories")
            } else if let error = vm.error, vm.factories.isEmpty {
                SupplierErrorView(message: error) { Task { await vm.load() } }
            } else if vm.factories.isEmpty {
                TopologyCenteredEmptyState(
                    title: "mobile_supplier.empty.no_factories",
                    message: "mobile_supplier.empty.factories_message",
                    actionLabel: "mobile_supplier.empty.add_first_factory"
                ) {
                    PegasusHaptics.light()
                    vm.showAdd = true
                }
            } else {
                FactoryList(factories: vm.factories)
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("portal.nav.factories")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("supplier_portal.factories.components.factory_form.text.add_factory", systemImage: "plus") {
                    PegasusHaptics.light()
                    vm.showAdd = true
                }
                .labelStyle(.iconOnly)
                .frame(minWidth: 44, minHeight: 44)
            }
        }
        .sheet(isPresented: $vm.showAdd) {
            AddFactorySheet {
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
