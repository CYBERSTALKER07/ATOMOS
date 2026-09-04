import SwiftUI

struct ProductsView: View {
    @State private var products: [Product] = []
    @State private var loading = true
    @State private var error: String?

    var body: some View {
        NavigationStack {
            Group {
                if loading {
                    ProgressView()
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else if let error {
                    ContentUnavailableView {
                        Label("mobile_warehouse.ui.error", systemImage: "exclamationmark.triangle")
                    } description: {
                        Text(error)
                    } actions: {
                        Button("common.action.retry") { load() }
                    }
                } else if products.isEmpty {
                    ContentUnavailableView("No Products", systemImage: "shippingbox", description: Text("mobile_warehouse.ui.product_catalog_is_empty"))
                } else {
                    ResponsiveGridContentWrapper {
                        ForEach(products) { product in
                            HStack {
                                VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                    Text(product.name)
                                        .font(.headline)
                                    Text(product.skuId)
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                }
                                Spacer()
                                Text(L10n.format("mobile_warehouse.ui.formatted_uzs", "\(product.priceUzs.formatted())"))
                                    .font(.subheadline)
                                    .foregroundStyle(.secondary)
                            }
                        }
                    }
                }
            }
            .background(LabTheme.background)
            .navigationTitle("portal.nav.products")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") { load() }
                }
            }
            .task { load() }
            .refreshable { load() }
        }
    }

    private func load() {
        loading = true
        error = nil
        Task {
            do {
                let resp = try await WarehouseService.products()
                products = resp.products
            } catch {
                self.error = error.localizedDescription
            }
            loading = false
        }
    }
}
