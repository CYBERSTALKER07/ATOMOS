import SwiftUI

struct ProductsView: View {
    @State private var vm = ProductsViewModel()

    var body: some View {
        Group {
            if vm.loading {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let error = vm.error {
                ContentUnavailableView {
                    Label("mobile_warehouse.ui.error", systemImage: "exclamationmark.triangle")
                } description: {
                    Text(error)
                } actions: {
                    Button("common.action.retry") {
                        PegasusHaptics.light()
                        Task { await vm.load() }
                    }
                    .frame(minWidth: 44, minHeight: 44)
                    .contentShape(Rectangle())
                }
            } else if vm.products.isEmpty {
                ContentUnavailableView(
                    "No Products",
                    systemImage: "shippingbox",
                    description: Text("mobile_warehouse.ui.product_catalog_is_empty")
                )
            } else {
                ResponsiveGridContentWrapper {
                    ForEach(vm.products) { product in
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
                        .frame(minHeight: 44)
                        .accessibilityElement(children: .combine)
                        .accessibilityLabel("\(product.name), SKU \(product.skuId), \(product.priceUzs.formatted()) UZS")
                    }
                }
                .scrollDismissesKeyboard(.interactively)
            }
        }
        .background(LabTheme.background)
        .navigationTitle("portal.nav.products")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    PegasusHaptics.light()
                    Task { await vm.load() }
                } label: {
                    Label("portal.page.orders.action.refresh", systemImage: "arrow.clockwise")
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                }
                .accessibilityLabel("Refresh products")
            }
        }
        .task { await vm.load() }
        .refreshable { await vm.load() }
    }
}
