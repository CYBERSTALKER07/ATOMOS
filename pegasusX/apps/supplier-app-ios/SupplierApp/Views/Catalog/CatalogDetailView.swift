import SwiftUI

struct CatalogDetailView: View {
    let productId: String?
    @State private var product: CatalogProduct?
    @State private var loading = true
    @State private var error: String?

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading product…")
            } else if let error {
                SupplierErrorView(message: error) { Task { await load() } }
            } else if let product {
                List {
                    Section("Product") {
                        LabeledContent("ID", value: product.productId)
                        LabeledContent("Name", value: product.name)
                        LabeledContent("Category", value: product.categoryId)
                        LabeledContent("Price", value: MoneyFormat.minor(product.priceMinor, currency: product.currency))
                        LabeledContent("Unit VU", value: String(format: "%.2f", product.unitVolumeVu))
                        if let barcode = product.barcode, !barcode.isEmpty {
                            LabeledContent("Barcode", value: barcode)
                        }
                        LabeledContent("Active", value: product.isActive ? "Yes" : "No")
                        LabeledContent("Version", value: "\(product.version)")
                    }
                }
                .listStyle(.insetGrouped)
            } else {
                SupplierEmptyView(title: "Product not found", message: "Select a catalog item to view details.")
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle(product?.name ?? "Catalog detail")
        .task { await load() }
    }

    @MainActor
    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        guard let productId, !productId.isEmpty else {
            error = "No product selected."
            return
        }
        do {
            let products = try await SupplierService.catalogProducts()
            product = products.first { $0.productId == productId }
        } catch {
            self.error = error.localizedDescription
        }
    }
}
