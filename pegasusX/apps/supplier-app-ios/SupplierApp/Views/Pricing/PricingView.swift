import SwiftUI

struct PricingView: View {
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    @State private var products: [CatalogProduct] = []
    @State private var loading = true
    @State private var error: String?
    @State private var query = ""

    private var filtered: [CatalogProduct] {
        guard !query.isEmpty else { return products }
        return products.filter { $0.name.localizedCaseInsensitiveContains(query) }
    }

    var body: some View {
        Group {
            if loading && products.isEmpty {
                SupplierLoadingView(title: "Loading pricing…")
            } else if let error, products.isEmpty {
                SupplierErrorView(message: error) { Task { await load() } }
            } else if products.isEmpty {
                SupplierEmptyView(
                    title: "No products to price",
                    message: "Add products in Catalog first. They will appear here for list and sale pricing."
                )
            } else if filtered.isEmpty {
                SupplierEmptyView(title: "No matches", message: "No products match \"\(query)\".")
            } else {
                List(filtered) { product in
                    NavigationLink {
                        ProductPricingDetailView(product: product) {
                            Task { await load(silent: true) }
                        }
                    } label: {
                        VStack(alignment: .leading, spacing: 4) {
                            Text(product.name).font(.headline)
                            Text(formatPrice(product))
                                .font(.caption.monospaced())
                                .foregroundStyle(.secondary)
                        }
                    }
                }
                .listStyle(.insetGrouped)
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("Pricing")
        .searchable(text: $query, prompt: "Product name")
        .task { await load() }
        .refreshable { await load(silent: true) }
        .silentRealtimeRefresh(
            refreshEpoch: realtimeHub.refreshEpoch,
            reconnectEpoch: realtimeHub.reconnectEpoch
        ) { silent in
            Task { await load(silent: silent) }
        }
    }

    private func formatPrice(_ product: CatalogProduct) -> String {
        let major = Double(product.priceMinor) / 100.0
        return String(format: "%.2f %@", major, product.currency)
    }

    @MainActor
    private func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        defer { loading = false }
        do {
            products = try await SupplierService.catalogProducts()
        } catch {
            if !silent { self.error = error.localizedDescription }
        }
    }
}
