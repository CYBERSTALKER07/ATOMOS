import SwiftUI

struct CatalogDetailView: View {
    let productId: String?
    @State private var vm = CatalogDetailViewModel()

    var body: some View {
        Group {
            if vm.loading && vm.product == nil {
                SupplierLoadingView(title: "mobile_supplier.loading.product")
            } else if let error = vm.error, vm.product == nil {
                SupplierErrorView(message: error) { Task { await vm.load(productId: productId) } }
            } else if let product = vm.product {
                ResponsiveGridContentWrapper {
                    Section("mobile_supplier.chrome.product") {
                        LabeledContent("mobile_supplier.chrome.id_label", value: product.productId)
                        LabeledContent("mobile_supplier.chrome.name", value: product.name)
                        LabeledContent("mobile_supplier.chrome.category", value: product.categoryId)
                        LabeledContent("mobile_supplier.chrome.price", value: MoneyFormat.minor(product.priceMinor, currency: product.currency))
                        LabeledContent("mobile_supplier.chrome.unit_vu", value: String(format: "%.2f", product.unitVolumeVu))
                        if let barcode = product.barcode, !barcode.isEmpty {
                            LabeledContent("mobile_supplier.chrome.barcode", value: barcode)
                        }
                        LabeledContent("mobile_supplier.chrome.active", value: product.isActive ? L10n.format("mobile_supplier.chrome.yes") : L10n.format("mobile_supplier.chrome.no"))
                        LabeledContent("mobile_supplier.chrome.version", value: "\(product.version)")
                    }
                }
            } else {
                SupplierEmptyView(title: "mobile_supplier.empty.product_not_found", message: "mobile_supplier.empty.product_not_found_message")
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle(vm.product?.name ?? "mobile_supplier.chrome.catalog_detail")
        .task(id: productId) { await vm.load(productId: productId) }
    }
}
