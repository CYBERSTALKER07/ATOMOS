import SwiftUI

struct CatalogView: View {
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    @State private var vm = CatalogViewModel()

    var body: some View {
        Group {
            if vm.loading && vm.products.isEmpty {
                SupplierLoadingView(title: "mobile_supplier.loading.catalog")
            } else if let error = vm.error, vm.products.isEmpty {
                SupplierErrorView(message: error) { Task { await vm.load() } }
            } else if vm.filtered.isEmpty {
                SupplierEmptyView(
                    title: "mobile_supplier.empty.no_products",
                    message: vm.query.isEmpty
                        ? "mobile_supplier.empty.catalog_empty"
                        : L10n.format("mobile_supplier.empty.no_products_match", vm.query)
                )
            } else {
                CatalogList(
                    products: vm.products,
                    query: vm.query,
                    draftVU: $vm.draftVU,
                    draftBarcode: $vm.draftBarcode,
                    savingId: vm.savingId,
                    imageSavingId: vm.imageSavingId,
                    onSaveUnitVolume: { await vm.saveUnitVolume($0) },
                    onSaveProductImage: { await vm.saveProductImage($0, imageData: $1) }
                )
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .background(SupplierTheme.background)
        .navigationTitle("portal.nav.catalog")
        .searchable(text: $vm.query, prompt: "mobile_supplier.chrome.product_name")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    PegasusHaptics.light()
                    vm.showCreate = true
                } label: {
                    Image(systemName: "plus")
                        .font(.body.weight(.semibold))
                        .foregroundStyle(SupplierTheme.label)
                        .frame(minWidth: 44, minHeight: 44)
                }
                .accessibilityLabel(Text("mobile_supplier.ui.add_product"))
            }
        }
        .sheet(isPresented: $vm.showCreate) {
            CatalogCreateSheet {
                Task { await vm.load(silent: true) }
            }
        }
        .task { await vm.load() }
        .refreshable {
            PegasusHaptics.light()
            await vm.load(silent: true)
        }
        .silentRealtimeRefresh(
            refreshEpoch: realtimeHub.refreshEpoch,
            reconnectEpoch: realtimeHub.reconnectEpoch
        ) { silent in
            Task { await vm.load(silent: silent) }
        }
    }
}
