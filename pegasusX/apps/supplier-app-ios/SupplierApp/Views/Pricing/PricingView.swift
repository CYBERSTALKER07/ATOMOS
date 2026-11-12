import SwiftUI

struct PricingView: View {
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    @State private var viewModel = PricingViewModel()

    var body: some View {
        @Bindable var vm = viewModel
        Group {
            if viewModel.loading && viewModel.products.isEmpty {
                SupplierLoadingView(title: "mobile_supplier.loading.pricing")
            } else if let error = viewModel.error, viewModel.products.isEmpty {
                SupplierErrorView(message: error) { Task { await viewModel.load() } }
            } else if viewModel.products.isEmpty {
                SupplierEmptyView(
                    title: "mobile_supplier.empty.no_products_to_price",
                    message: "mobile_supplier.empty.pricing_message"
                )
            } else if viewModel.filtered.isEmpty {
                SupplierEmptyView(
                    title: "mobile_supplier.empty.no_matches",
                    message: L10n.format("mobile_supplier.empty.no_products_match", viewModel.query)
                )
            } else {
                ResponsiveGridContentWrapper {
                    ForEach(viewModel.filtered) { product in
                        NavigationLink {
                            ProductPricingDetailView(product: product) {
                                Task { await viewModel.load(silent: true) }
                            }
                        } label: {
                            VStack(alignment: .leading, spacing: 4) {
                                Text(product.name).font(.headline)
                                Text(Self.formatPrice(product))
                                    .font(.caption.monospaced())
                                    .foregroundStyle(.secondary)
                            }
                            .accessibilityElement(children: .combine)
                        }
                    }
                }
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("portal.nav.pricing")
        .searchable(text: $vm.query, prompt: "mobile_supplier.chrome.product_name")
        .task { await viewModel.load() }
        .refreshable { await viewModel.load(silent: true) }
        .silentRealtimeRefresh(
            refreshEpoch: realtimeHub.refreshEpoch,
            reconnectEpoch: realtimeHub.reconnectEpoch
        ) { silent in
            Task { await viewModel.load(silent: silent) }
        }
    }

    static func formatPrice(_ product: CatalogProduct) -> String {
        MoneyFormat.minor(product.priceMinor, currency: product.currency)
    }
}
